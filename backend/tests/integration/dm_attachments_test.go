package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #184 DM 파일/이미지 첨부 — 권한, 위조 검증, 실패 시 정리, 다운로드 헤더.

type dmTestFile struct {
	filename    string
	contentType string
	content     []byte
	rawHeader   bool // true 면 Content-Disposition 을 이스케이프 없이 직접 쓴다 (파일명 인젝션 테스트)
}

func pngBytes(pad int) []byte {
	b := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00")
	return append(b, bytes.Repeat([]byte{0x00}, pad)...)
}

func pdfBytes() []byte {
	return []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n%%EOF\n")
}

func zipBytes() []byte {
	return []byte("PK\x03\x04\x14\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
}

// postDMMultipart — multipart/form-data 로 DM 전송. (파싱된 envelope, HTTP status) 반환.
func (ts *testServer) postDMMultipart(token string, values map[string]string, files []dmTestFile) (*apiResponse, int) {
	ts.t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for k, v := range values {
		if err := w.WriteField(k, v); err != nil {
			ts.t.Fatalf("write field: %v", err)
		}
	}
	for _, f := range files {
		h := make(textproto.MIMEHeader)
		if f.rawHeader {
			h.Set("Content-Disposition", `form-data; name="files"; filename="`+f.filename+`"`)
		} else {
			h.Set("Content-Disposition", `form-data; name="files"; filename="`+strings.ReplaceAll(f.filename, `"`, `\"`)+`"`)
		}
		if f.contentType != "" {
			h.Set("Content-Type", f.contentType)
		}
		part, err := w.CreatePart(h)
		if err != nil {
			ts.t.Fatalf("create part: %v", err)
		}
		if _, err := part.Write(f.content); err != nil {
			ts.t.Fatalf("write part: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		ts.t.Fatalf("close writer: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, ts.url("/api/dm/messages"), &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ts.t.Fatalf("POST multipart dm: %v", err)
	}
	defer resp.Body.Close()
	status := resp.StatusCode
	return ts.parseResponse(resp), status
}

// rawGet — 바이너리 응답(다운로드) 용. 헤더/상태만 확인하고 JSON 파싱하지 않는다. 호출자가 Body 를 닫는다.
func (ts *testServer) rawGetResp(path, token string) *http.Response {
	ts.t.Helper()
	req, _ := http.NewRequest(http.MethodGet, ts.url(path), nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ts.t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

// dmStoredFileCount — 비공개 DM 저장소에 실제로 남아있는 파일 수.
func (ts *testServer) dmStoredFileCount() int {
	ts.t.Helper()
	entries, err := os.ReadDir(filepath.Join(ts.privateUploadPath, "dm"))
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		ts.t.Fatalf("read dm dir: %v", err)
	}
	return len(entries)
}

type dmMessageJSON struct {
	ID          int    `json:"id"`
	Content     string `json:"content"`
	Attachments []struct {
		ID       int    `json:"id"`
		Filename string `json:"filename"`
		Mime     string `json:"mime"`
		Size     int64  `json:"size"`
		Path     string `json:"path"`
	} `json:"attachments"`
}

func decodeDMMessage(t *testing.T, raw json.RawMessage) dmMessageJSON {
	t.Helper()
	var m dmMessageJSON
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decode dm message: %v (%s)", err, raw)
	}
	return m
}

func TestDMAttachments(t *testing.T) {
	ts := setupTestServer(t)
	senderToken := ts.registerAndApprove("dm-att-sender@test.com", "pass1234", "보내는사람", "20250401")
	receiverToken := ts.registerAndApprove("dm-att-receiver@test.com", "pass1234", "받는사람", "20250402")
	strangerToken := ts.registerAndApprove("dm-att-stranger@test.com", "pass1234", "제3자", "20250403")
	adminToken := ts.login(testAdminEmail, testAdminPass)

	var senderID, receiverID int
	if err := ts.db.QueryRow(`SELECT id FROM users WHERE email = ?`, "dm-att-sender@test.com").Scan(&senderID); err != nil {
		t.Fatalf("sender id: %v", err)
	}
	if err := ts.db.QueryRow(`SELECT id FROM users WHERE email = ?`, "dm-att-receiver@test.com").Scan(&receiverID); err != nil {
		t.Fatalf("receiver id: %v", err)
	}

	t.Run("1 text-only json send still works and returns empty attachments", func(t *testing.T) {
		r := ts.post("/api/dm/messages", map[string]interface{}{
			"receiver_id": receiverID, "content": "안녕하세요",
		}, senderToken)
		if !r.Success {
			t.Fatalf("text dm failed: %v", r.Error)
		}
		if !strings.Contains(string(r.Data), `"attachments":[]`) {
			t.Fatalf("attachments should serialize as [], got: %s", r.Data)
		}
	})

	t.Run("2 attachment-only send persists and is re-fetchable", func(t *testing.T) {
		r, status := ts.postDMMultipart(senderToken, map[string]string{
			"receiver_id": fmt.Sprint(receiverID),
		}, []dmTestFile{{filename: "photo.png", contentType: "image/png", content: pngBytes(64)}})
		if status != http.StatusCreated || !r.Success {
			t.Fatalf("attachment-only send: status=%d err=%v", status, r.Error)
		}
		msg := decodeDMMessage(t, r.Data)
		if len(msg.Attachments) != 1 || msg.Attachments[0].Filename != "photo.png" {
			t.Fatalf("unexpected attachments: %s", r.Data)
		}
		if msg.Attachments[0].Path != "" {
			t.Fatalf("path must never be exposed: %s", r.Data)
		}

		list := ts.get(fmt.Sprintf("/api/dm/messages/%d", senderID), receiverToken)
		if !list.Success {
			t.Fatalf("list messages: %v", list.Error)
		}
		var msgs []dmMessageJSON
		if err := json.Unmarshal(list.Data, &msgs); err != nil {
			t.Fatalf("decode list: %v", err)
		}
		found := false
		for _, m := range msgs {
			if m.ID == msg.ID && len(m.Attachments) == 1 {
				found = true
			}
		}
		if !found {
			t.Fatalf("attachment not persisted in message list: %s", list.Data)
		}
	})

	t.Run("zip accepts common browser MIME fallback", func(t *testing.T) {
		r, status := ts.postDMMultipart(senderToken, map[string]string{
			"receiver_id": fmt.Sprint(receiverID),
		}, []dmTestFile{{filename: "archive.zip", contentType: "application/x-zip-compressed", content: zipBytes()}})
		if status != http.StatusCreated || !r.Success {
			t.Fatalf("zip send: status=%d err=%v", status, r.Error)
		}
		msg := decodeDMMessage(t, r.Data)
		if len(msg.Attachments) != 1 || msg.Attachments[0].Filename != "archive.zip" {
			t.Fatalf("unexpected zip attachment: %s", r.Data)
		}
	})

	t.Run("3 text plus attachment", func(t *testing.T) {
		r, status := ts.postDMMultipart(senderToken, map[string]string{
			"receiver_id": fmt.Sprint(receiverID), "content": "자료 보냅니다",
		}, []dmTestFile{{filename: "doc.pdf", contentType: "application/pdf", content: pdfBytes()}})
		if status != http.StatusCreated || !r.Success {
			t.Fatalf("text+attachment send: status=%d err=%v", status, r.Error)
		}
		msg := decodeDMMessage(t, r.Data)
		if msg.Content != "자료 보냅니다" || len(msg.Attachments) != 1 {
			t.Fatalf("unexpected message: %s", r.Data)
		}
	})

	t.Run("4 empty message with no files is rejected", func(t *testing.T) {
		r, status := ts.postDMMultipart(senderToken, map[string]string{
			"receiver_id": fmt.Sprint(receiverID), "content": "   ",
		}, nil)
		if status != http.StatusBadRequest || r.Success {
			t.Fatalf("empty message should be 400, got %d", status)
		}
	})

	// 다운로드 대상 첨부 준비 (png + pdf)
	r, status := ts.postDMMultipart(senderToken, map[string]string{
		"receiver_id": fmt.Sprint(receiverID), "content": "다운로드용",
	}, []dmTestFile{
		{filename: "shot.png", contentType: "image/png", content: pngBytes(64)},
		{filename: "spec.pdf", contentType: "application/pdf", content: pdfBytes()},
	})
	if status != http.StatusCreated || !r.Success {
		t.Fatalf("setup download message failed: status=%d err=%v", status, r.Error)
	}
	dl := decodeDMMessage(t, r.Data)
	if len(dl.Attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %s", r.Data)
	}
	pngID, pdfID := dl.Attachments[0].ID, dl.Attachments[1].ID

	t.Run("5 sender and receiver can download", func(t *testing.T) {
		for name, token := range map[string]string{"sender": senderToken, "receiver": receiverToken} {
			resp := ts.rawGetResp(fmt.Sprintf("/api/dm/attachments/%d", pngID), token)
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("%s download: got %d want 200", name, resp.StatusCode)
			}
		}
	})

	t.Run("6 stranger and admin get 403", func(t *testing.T) {
		for name, token := range map[string]string{"stranger": strangerToken, "admin": adminToken} {
			resp := ts.rawGetResp(fmt.Sprintf("/api/dm/attachments/%d", pngID), token)
			resp.Body.Close()
			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("%s download: got %d want 403", name, resp.StatusCode)
			}
		}
	})

	t.Run("7 nonexistent attachment is 404", func(t *testing.T) {
		resp := ts.rawGetResp("/api/dm/attachments/999999", senderToken)
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("got %d want 404", resp.StatusCode)
		}
	})

	t.Run("8 spoofed png content is rejected and leaves no file", func(t *testing.T) {
		before := ts.dmStoredFileCount()
		_, status := ts.postDMMultipart(senderToken, map[string]string{
			"receiver_id": fmt.Sprint(receiverID),
		}, []dmTestFile{{filename: "evil.png", contentType: "image/png", content: []byte("<script>alert(1)</script>")}})
		if status != http.StatusBadRequest {
			t.Fatalf("spoofed png should be 400, got %d", status)
		}
		if after := ts.dmStoredFileCount(); after != before {
			t.Fatalf("leftover files: before=%d after=%d", before, after)
		}
	})

	t.Run("8b second file failing removes the first already-written file and inserts no row", func(t *testing.T) {
		beforeFiles := ts.dmStoredFileCount()
		var beforeMsgs, beforeAtts int
		ts.db.QueryRow(`SELECT COUNT(*) FROM dm_messages`).Scan(&beforeMsgs)
		ts.db.QueryRow(`SELECT COUNT(*) FROM dm_attachments`).Scan(&beforeAtts)

		_, status := ts.postDMMultipart(senderToken, map[string]string{
			"receiver_id": fmt.Sprint(receiverID),
		}, []dmTestFile{
			{filename: "good.png", contentType: "image/png", content: pngBytes(64)},
			{filename: "evil.png", contentType: "image/png", content: []byte("<script>alert(1)</script>")},
		})
		if status != http.StatusBadRequest {
			t.Fatalf("partial failure should be 400, got %d", status)
		}
		if after := ts.dmStoredFileCount(); after != beforeFiles {
			t.Fatalf("orphan file left: before=%d after=%d", beforeFiles, after)
		}
		var afterMsgs, afterAtts int
		ts.db.QueryRow(`SELECT COUNT(*) FROM dm_messages`).Scan(&afterMsgs)
		ts.db.QueryRow(`SELECT COUNT(*) FROM dm_attachments`).Scan(&afterAtts)
		if afterMsgs != beforeMsgs || afterAtts != beforeAtts {
			t.Fatalf("orphan rows: msgs %d->%d atts %d->%d", beforeMsgs, afterMsgs, beforeAtts, afterAtts)
		}
	})

	t.Run("9 disallowed extensions are rejected", func(t *testing.T) {
		for _, f := range []dmTestFile{
			{filename: "payload.exe", contentType: "application/octet-stream", content: []byte("MZ\x90\x00")},
		} {
			before := ts.dmStoredFileCount()
			_, status := ts.postDMMultipart(senderToken, map[string]string{
				"receiver_id": fmt.Sprint(receiverID),
			}, []dmTestFile{f})
			if status != http.StatusBadRequest {
				t.Fatalf("%s should be 400, got %d", f.filename, status)
			}
			if after := ts.dmStoredFileCount(); after != before {
				t.Fatalf("%s left files behind", f.filename)
			}
		}
	})

	t.Run("9b html attachment is accepted", func(t *testing.T) {
		r, status := ts.postDMMultipart(senderToken, map[string]string{
			"receiver_id": fmt.Sprint(receiverID),
		}, []dmTestFile{{filename: "prototype.html", contentType: "text/html", content: []byte("<!doctype html><title>prototype</title>")}})
		if status != http.StatusCreated || !r.Success {
			t.Fatalf("html send: status=%d err=%v", status, r.Error)
		}
	})

	t.Run("10 mime and extension mismatch is rejected", func(t *testing.T) {
		_, status := ts.postDMMultipart(senderToken, map[string]string{
			"receiver_id": fmt.Sprint(receiverID),
		}, []dmTestFile{{filename: "trick.png", contentType: "text/html", content: pngBytes(64)}})
		if status != http.StatusBadRequest {
			t.Fatalf("mime mismatch should be 400, got %d", status)
		}
	})

	t.Run("11 oversize file is rejected and leaves no file", func(t *testing.T) {
		before := ts.dmStoredFileCount()
		big := pngBytes(10*1024*1024 + 1 - len(pngBytes(0)))
		_, status := ts.postDMMultipart(senderToken, map[string]string{
			"receiver_id": fmt.Sprint(receiverID),
		}, []dmTestFile{{filename: "huge.png", contentType: "image/png", content: big}})
		if status != http.StatusBadRequest {
			t.Fatalf("oversize should be 400, got %d", status)
		}
		if after := ts.dmStoredFileCount(); after != before {
			t.Fatalf("oversize left files behind: before=%d after=%d", before, after)
		}
	})

	t.Run("12 more than four files is rejected and leaves no file", func(t *testing.T) {
		before := ts.dmStoredFileCount()
		var files []dmTestFile
		for i := 0; i < 5; i++ {
			files = append(files, dmTestFile{filename: fmt.Sprintf("f%d.png", i), contentType: "image/png", content: pngBytes(32)})
		}
		_, status := ts.postDMMultipart(senderToken, map[string]string{
			"receiver_id": fmt.Sprint(receiverID),
		}, files)
		if status != http.StatusBadRequest {
			t.Fatalf("5 files should be 400, got %d", status)
		}
		if after := ts.dmStoredFileCount(); after != before {
			t.Fatalf("too-many left files behind: before=%d after=%d", before, after)
		}
	})

	t.Run("13 download headers", func(t *testing.T) {
		img := ts.rawGetResp(fmt.Sprintf("/api/dm/attachments/%d", pngID), receiverToken)
		defer img.Body.Close()
		if img.Header.Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("png missing nosniff: %v", img.Header)
		}
		if ct := img.Header.Get("Content-Type"); !strings.HasPrefix(ct, "image/png") {
			t.Fatalf("png content-type = %q", ct)
		}
		if cd := img.Header.Get("Content-Disposition"); !strings.Contains(cd, "inline") {
			t.Fatalf("png disposition = %q", cd)
		}

		doc := ts.rawGetResp(fmt.Sprintf("/api/dm/attachments/%d", pdfID), receiverToken)
		defer doc.Body.Close()
		if ct := doc.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/octet-stream") {
			t.Fatalf("pdf content-type = %q", ct)
		}
		if cd := doc.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
			t.Fatalf("pdf disposition = %q", cd)
		}
		if doc.Header.Get("Content-Security-Policy") == "" {
			t.Fatalf("pdf missing CSP header")
		}
		if doc.Header.Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("pdf missing nosniff")
		}
	})

	t.Run("14 unsafe filenames are rejected", func(t *testing.T) {
		cases := []dmTestFile{
			{filename: `..\..\etc\passwd.png`, contentType: "image/png", content: pngBytes(32)},
			{filename: `../../etc/passwd`, contentType: "image/png", content: pngBytes(32)},
			{filename: `bad"name.png`, contentType: "image/png", content: pngBytes(32), rawHeader: true},
			{filename: "bad\nname.png", contentType: "image/png", content: pngBytes(32), rawHeader: true},
		}
		for _, f := range cases {
			before := ts.dmStoredFileCount()
			r, status := ts.postDMMultipart(senderToken, map[string]string{
				"receiver_id": fmt.Sprint(receiverID),
			}, []dmTestFile{f})
			if status < 400 || r.Success {
				t.Fatalf("filename %q should be rejected, got %d", f.filename, status)
			}
			if after := ts.dmStoredFileCount(); after != before {
				t.Fatalf("filename %q left files behind", f.filename)
			}
		}
	})

	t.Run("15b request body over the cap is rejected before parsing", func(t *testing.T) {
		before := ts.dmStoredFileCount()
		// 20MB 총합 제한보다 큰 바디 — usecase 검증 전에 거절되어야 한다 (#184).
		huge := pngBytes(22*1024*1024 - len(pngBytes(0)))
		r, status := ts.postDMMultipart(senderToken, map[string]string{
			"receiver_id": fmt.Sprint(receiverID),
		}, []dmTestFile{{filename: "toobig.png", contentType: "image/png", content: huge}})
		if status != http.StatusRequestEntityTooLarge {
			t.Fatalf("oversized body should be 413, got %d (%v)", status, r.Error)
		}
		if after := ts.dmStoredFileCount(); after != before {
			t.Fatalf("oversized body left files behind: before=%d after=%d", before, after)
		}
	})

	t.Run("15c stored attachments are private (dir 0700, files 0600)", func(t *testing.T) {
		dir := filepath.Join(ts.privateUploadPath, "dm")
		// 배포 서버에 이미 0755 로 만들어진 디렉토리가 있을 수 있다 (#184).
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("pre-create dm dir: %v", err)
		}
		if err := os.Chmod(dir, 0755); err != nil {
			t.Fatalf("chmod dm dir: %v", err)
		}

		_, status := ts.postDMMultipart(senderToken, map[string]string{
			"receiver_id": fmt.Sprint(receiverID),
		}, []dmTestFile{{filename: "perm.png", contentType: "image/png", content: pngBytes(32)}})
		if status != http.StatusCreated {
			t.Fatalf("send failed: %d", status)
		}

		di, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat dm dir: %v", err)
		}
		if di.Mode().Perm() != 0700 {
			t.Fatalf("dm dir mode = %o, want 700", di.Mode().Perm())
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read dm dir: %v", err)
		}
		if len(entries) == 0 {
			t.Fatalf("no stored attachment found")
		}
		for _, e := range entries {
			fi, err := e.Info()
			if err != nil {
				t.Fatalf("stat %s: %v", e.Name(), err)
			}
			if fi.Mode().Perm() != 0600 {
				t.Fatalf("stored file %s mode = %o, want 600", e.Name(), fi.Mode().Perm())
			}
		}
	})

	t.Run("15 attachment-only dm still creates a new_dm notification", func(t *testing.T) {
		_, status := ts.postDMMultipart(senderToken, map[string]string{
			"receiver_id": fmt.Sprint(receiverID),
		}, []dmTestFile{{filename: "notify.png", contentType: "image/png", content: pngBytes(48)}})
		if status != http.StatusCreated {
			t.Fatalf("send failed: %d", status)
		}
		notifs := ts.get("/api/notifications?limit=20", receiverToken)
		if !notifs.Success {
			t.Fatalf("get notifications: %v", notifs.Error)
		}
		if !strings.Contains(string(notifs.Data), `"new_dm"`) {
			t.Fatalf("no new_dm notification: %s", notifs.Data)
		}
	})
}
