package integration

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
)

// #176 — 사업계획서 첨부로 .html 허용 (다운로드 전용).
// HTML 은 공개 /uploads 정적 경로에 노출되지 않고,
// 권한 검증 다운로드 엔드포인트가 항상 attachment + octet-stream + nosniff 를 보낸다.

// uploadMilestoneFileBytes — 바이너리 내용 + Content-Type 지정 가능한 업로드 헬퍼.
func (ts *testServer) uploadMilestoneFileBytes(token, filename string, content []byte, contentType string) *apiResponse {
	ts.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="` + filename + `"`}
	if contentType != "" {
		h["Content-Type"] = []string{contentType}
	}
	fw, err := w.CreatePart(h)
	if err != nil {
		ts.t.Fatalf("create part: %v", err)
	}
	fw.Write(content)
	w.Close()
	req, _ := http.NewRequest("POST", ts.url("/api/milestones/files"), &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ts.t.Fatalf("upload file: %v", err)
	}
	defer resp.Body.Close()
	return ts.parseResponse(resp)
}

// rawGetFull — 다운로드 응답의 헤더까지 확인하기 위한 헬퍼.
func (ts *testServer) rawGetFull(path, token string) (*http.Response, []byte) {
	ts.t.Helper()
	req, _ := http.NewRequest("GET", ts.url(path), nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ts.t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	b := make([]byte, 0)
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err == nil {
		b = buf.Bytes()
	}
	return resp, b
}

// countPrivateUploads — 비공개 저장소에 남은 파일 수 (조각 파일 정리 확인용).
func (ts *testServer) countPrivateUploads() int {
	entries, err := os.ReadDir(testUploadPath + "/private")
	if err != nil {
		return 0
	}
	return len(entries)
}

func TestMilestoneHTMLAttachment(t *testing.T) {
	ts := setupTestServer(t)
	student := ts.registerAndApprove("html-bp@test.com", "pass1234", "HTML학생", "20250301")

	fileID := func(r *apiResponse) int {
		var f struct {
			ID       int    `json:"id"`
			MimeType string `json:"mime_type"`
			Size     int64  `json:"size"`
		}
		json.Unmarshal(r.Data, &f)
		return f.ID
	}

	// 평범한 사업계획서 HTML — 스크립트/스타일이 있어도 거부하면 안 된다 (실행될 일이 없음).
	planHTML := `<!DOCTYPE html>
<html lang="ko"><head><meta charset="utf-8"><title>사업계획서</title>
<script>document.title='차트'</script></head>
<body><h1>사업계획서</h1><p>시장 규모 100억 원</p></body></html>`

	var acceptedID int

	t.Run("accepts normal business-plan html", func(t *testing.T) {
		r := ts.uploadMilestoneFileBytes(student, "사업계획서.html", []byte(planHTML), "text/html")
		if !r.Success {
			t.Fatalf("html upload rejected: %v", r.Error)
		}
		acceptedID = fileID(r)
		var f struct {
			MimeType string `json:"mime_type"`
			Size     int64  `json:"size"`
		}
		json.Unmarshal(r.Data, &f)
		// 클라이언트가 보낸 text/html 을 그대로 저장하면 안 된다.
		if f.MimeType != "application/octet-stream" {
			t.Errorf("stored mime = %q, want application/octet-stream", f.MimeType)
		}
		if f.Size != int64(len(planHTML)) {
			t.Errorf("stored size = %d, want %d", f.Size, len(planHTML))
		}
	})

	t.Run("download forces attachment with safe headers", func(t *testing.T) {
		if acceptedID == 0 {
			t.Skip("upload failed")
		}
		resp, body := ts.rawGetFull("/api/milestones/files/"+strconv.Itoa(acceptedID), student)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d", resp.StatusCode)
		}
		cd := resp.Header.Get("Content-Disposition")
		if !strings.HasPrefix(cd, "attachment") {
			t.Errorf("Content-Disposition = %q, want attachment", cd)
		}
		ct := strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0]))
		if ct != "application/octet-stream" && ct != "text/plain" {
			t.Errorf("Content-Type = %q, must not be renderable html", ct)
		}
		if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
		}
		if got := resp.Header.Get("Content-Security-Policy"); got == "" {
			t.Errorf("Content-Security-Policy missing")
		}
		if string(body) != planHTML {
			t.Errorf("body not byte-identical (%d vs %d bytes)", len(body), len(planHTML))
		}
	})

	t.Run("rejects NUL byte and invalid utf-8", func(t *testing.T) {
		if r := ts.uploadMilestoneFileBytes(student, "nul.html", []byte("<html>\x00evil</html>"), "text/html"); r.Success {
			t.Error("NUL byte html accepted")
		}
		if r := ts.uploadMilestoneFileBytes(student, "bad.html", []byte{0x3c, 0x68, 0xff, 0xfe, 0x3e}, "text/html"); r.Success {
			t.Error("invalid utf-8 html accepted")
		}
	})

	// 참고: Go 의 multipart 는 Part.FileName() 에 filepath.Base 를 적용하므로
	// "../../x.html" 은 서버 도달 전에 정규화된다. 백슬래시는 리눅스에서 남으므로 여기서 막는다.
	// 파일명 검증 자체의 전수 케이스는 application 패키지 단위 테스트 참고.
	t.Run("rejects unsafe filenames", func(t *testing.T) {
		for _, name := range []string{`..\..\evil.html`, `evil\plan.html`} {
			if r := ts.uploadMilestoneFileBytes(student, name, []byte("<html>ok</html>"), "text/html"); r.Success {
				t.Errorf("unsafe filename accepted: %q", name)
			}
		}
	})

	t.Run("rejects oversize html and leaves no file", func(t *testing.T) {
		big := bytes.Repeat([]byte("a"), 10*1024*1024+1024)
		before := ts.countPrivateUploads()
		r := ts.uploadMilestoneFileBytes(student, "big.html", big, "text/html")
		if r.Success {
			t.Fatal("oversize html accepted")
		}
		if after := ts.countPrivateUploads(); after != before {
			t.Errorf("partial file left on disk: %d -> %d", before, after)
		}
	})

	t.Run("public /upload still rejects html", func(t *testing.T) {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, _ := w.CreateFormFile("file", "plan.html")
		fw.Write([]byte(planHTML))
		w.Close()
		req, _ := http.NewRequest("POST", ts.url("/api/upload"), &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+student)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("upload: %v", err)
		}
		defer resp.Body.Close()
		r := ts.parseResponse(resp)
		if r.Success {
			t.Error("public /upload accepted html — would be served same-origin from /uploads")
		}
	})

	t.Run("existing formats still work", func(t *testing.T) {
		r := ts.uploadMilestoneFile(student, "plan.pdf", "PDF-CONTENT")
		if !r.Success {
			t.Fatalf("pdf upload broke: %v", r.Error)
		}
		resp, body := ts.rawGetFull("/api/milestones/files/"+strconv.Itoa(fileID(r)), student)
		if resp.StatusCode != http.StatusOK || string(body) != "PDF-CONTENT" {
			t.Fatalf("pdf download broke: status=%d body=%q", resp.StatusCode, body)
		}
		if cd := resp.Header.Get("Content-Disposition"); !strings.HasPrefix(cd, "inline") {
			t.Errorf("pdf Content-Disposition = %q, want inline (기존 동작 유지)", cd)
		}
	})
}
