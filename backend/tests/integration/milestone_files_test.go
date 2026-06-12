package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
)

// uploadMilestoneFile — multipart 업로드 헬퍼. (id, apiResponse) 반환.
func (ts *testServer) uploadMilestoneFile(token, filename, content string) *apiResponse {
	ts.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", filename)
	fw.Write([]byte(content))
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

// rawGet — JSON 이 아닌 raw body(파일 다운로드)용. (status, body) 반환.
func (ts *testServer) rawGet(path, token string) (int, []byte) {
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
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

// TestMilestoneFiles — 사업계획서 비공개 첨부 (#125).
// owner+admin 만 접근, 본인 삭제 가능.
func TestMilestoneFiles(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)
	owner := ts.registerAndApprove("bp-owner@test.com", "pass1234", "사업계획서주인", "20250201")
	other := ts.registerAndApprove("bp-other@test.com", "pass1234", "다른학생", "20250202")

	parseFileID := func(r *apiResponse) int {
		var f struct {
			ID       int    `json:"id"`
			Filename string `json:"filename"`
		}
		json.Unmarshal(r.Data, &f)
		return f.ID
	}

	t.Run("owner uploads multiple files", func(t *testing.T) {
		r1 := ts.uploadMilestoneFile(owner, "plan.pdf", "PDF-CONTENT-1")
		if !r1.Success {
			t.Fatalf("upload1: %v", r1.Error)
		}
		r2 := ts.uploadMilestoneFile(owner, "appendix.docx", "DOCX-CONTENT-2")
		if !r2.Success {
			t.Fatalf("upload2: %v", r2.Error)
		}
		// list
		lr := ts.get("/api/milestones/files", owner)
		var files []struct {
			ID       int    `json:"id"`
			Filename string `json:"filename"`
		}
		json.Unmarshal(lr.Data, &files)
		if len(files) != 2 {
			t.Fatalf("expected 2 files, got %d", len(files))
		}
	})

	// upload one file to test access control on it
	upl := ts.uploadMilestoneFile(owner, "secret-plan.pdf", "TOP-SECRET-BUSINESS-PLAN")
	fileID := parseFileID(upl)
	if fileID == 0 {
		t.Fatalf("no file id from upload: %v", upl.Error)
	}
	dlPath := fmt.Sprintf("/api/milestones/files/%d", fileID)

	t.Run("owner can download own file", func(t *testing.T) {
		status, body := ts.rawGet(dlPath, owner)
		if status != 200 {
			t.Fatalf("owner download status=%d", status)
		}
		if string(body) != "TOP-SECRET-BUSINESS-PLAN" {
			t.Errorf("content mismatch: %q", string(body))
		}
	})

	t.Run("other student is forbidden (403)", func(t *testing.T) {
		status, _ := ts.rawGet(dlPath, other)
		if status != 403 {
			t.Errorf("expected 403 for non-owner, got %d", status)
		}
	})

	t.Run("admin can download any file", func(t *testing.T) {
		status, body := ts.rawGet(dlPath, adminToken)
		if status != 200 {
			t.Fatalf("admin download status=%d", status)
		}
		if string(body) != "TOP-SECRET-BUSINESS-PLAN" {
			t.Errorf("admin content mismatch: %q", string(body))
		}
	})

	t.Run("other student cannot delete owner file", func(t *testing.T) {
		r := ts.delete(dlPath, other)
		if r.Success {
			t.Error("non-owner should not delete file")
		}
	})

	t.Run("owner can delete own file", func(t *testing.T) {
		r := ts.delete(dlPath, owner)
		if !r.Success {
			t.Fatalf("owner delete failed: %v", r.Error)
		}
		// gone → download 404
		status, _ := ts.rawGet(dlPath, owner)
		if status != 404 {
			t.Errorf("expected 404 after delete, got %d", status)
		}
	})

	t.Run("non-business_plan type rejected", func(t *testing.T) {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		w.WriteField("type", "mvp1")
		fw, _ := w.CreateFormFile("file", "x.pdf")
		fw.Write([]byte("x"))
		w.Close()
		req, _ := http.NewRequest("POST", ts.url("/api/milestones/files"), &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+owner)
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		r := ts.parseResponse(resp)
		if r.Success {
			t.Error("mvp1 file upload should be rejected")
		}
	})

	t.Run("disallowed extension rejected", func(t *testing.T) {
		r := ts.uploadMilestoneFile(owner, "evil.exe", "MZ")
		if r.Success {
			t.Error(".exe should be rejected")
		}
	})
}
