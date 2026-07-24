package integration

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func (ts *testServer) uploadFile(token, filename, contentType string, content []byte) *apiResponse {
	ts.t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		ts.t.Fatalf("create upload part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		ts.t.Fatalf("write upload part: %v", err)
	}
	if err := writer.Close(); err != nil {
		ts.t.Fatalf("close multipart writer: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, ts.url("/api/upload"), &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ts.t.Fatalf("upload request: %v", err)
	}
	defer resp.Body.Close()
	return ts.parseResponse(resp)
}

func TestMarkdownUpload(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("markdown-upload@test.com", "pass1234", "마크다운업로더", "20250301")

	for _, contentType := range []string{"text/markdown", "text/plain; charset=utf-8", "application/octet-stream"} {
		t.Run("accepts markdown as "+contentType, func(t *testing.T) {
			r := ts.uploadFile(token, "product-spec.md", contentType, []byte("# Product SPEC\n\n- requirement\n"))
			if !r.Success {
				t.Fatalf("markdown upload failed: %v", r.Error)
			}
			if !strings.Contains(string(r.Data), `"filename":"product-spec.md"`) || !strings.Contains(string(r.Data), `.md`) {
				t.Fatalf("markdown response is not attachment-compatible: %s", r.Data)
			}
			var uploaded struct {
				URL string `json:"url"`
			}
			if err := json.Unmarshal(r.Data, &uploaded); err != nil {
				t.Fatalf("decode upload: %v", err)
			}
			// Production serves this directory at /uploads; verify the persisted
			// bytes that the returned download URL resolves to.
			got, err := os.ReadFile(filepath.Join(testUploadPath, filepath.Base(uploaded.URL)))
			if err != nil || string(got) != "# Product SPEC\n\n- requirement\n" {
				t.Fatalf("download content mismatch: %q, %v", got, err)
			}
		})
	}

	t.Run("rejects markdown MIME on a disallowed extension", func(t *testing.T) {
		r := ts.uploadFile(token, "payload.exe", "text/markdown", []byte("# not executable"))
		if r.Success {
			t.Fatal("disallowed extension should be rejected")
		}
	})

	t.Run("rejects HTML disguised as markdown", func(t *testing.T) {
		r := ts.uploadFile(token, "payload.md", "text/markdown", []byte("<!doctype html><script>alert(1)</script>"))
		if r.Success {
			t.Fatal("mismatched MIME should be rejected")
		}
	})

	for _, tc := range []struct {
		name    string
		payload []byte
	}{
		{"invalid UTF-8 after byte 512", []byte{0xff}},
		{"NUL after byte 512", []byte{0}},
		{"HTML after byte 512", []byte("<script>alert(1)</script>")},
	} {
		t.Run("rejects and cleans up "+tc.name, func(t *testing.T) {
			before, err := filepath.Glob(filepath.Join(testUploadPath, "*"))
			if err != nil {
				t.Fatalf("list uploads before request: %v", err)
			}
			content := append(bytes.Repeat([]byte("a"), 513), tc.payload...)
			r := ts.uploadFile(token, "delayed-payload.md", "text/markdown", content)
			if r.Success {
				t.Fatal("unsafe payload after byte 512 should be rejected")
			}
			after, err := filepath.Glob(filepath.Join(testUploadPath, "*"))
			if err != nil {
				t.Fatalf("list uploads after request: %v", err)
			}
			if len(after) != len(before) {
				t.Fatalf("rejected upload was not cleaned up: before=%d after=%d", len(before), len(after))
			}
		})
	}

	// net/http's multipart parser strips directory components before the use case.
	for _, filename := range []string{"bad\x00name.md"} {
		t.Run("rejects unsafe filename", func(t *testing.T) {
			r := ts.uploadFile(token, filename, "text/markdown", []byte("# PRD"))
			if r.Success {
				t.Fatalf("unsafe filename %q should be rejected", filename)
			}
		})
	}

	t.Run("rejects file larger than limit", func(t *testing.T) {
		r := ts.uploadFile(token, "too-large.md", "text/markdown", bytes.Repeat([]byte("a"), 10*1024*1024+1))
		if r.Success {
			t.Fatal("oversized markdown should be rejected")
		}
	})
}

func TestExistingPDFUploadStillWorks(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("pdf-upload@test.com", "pass1234", "PDF업로더", "20250302")
	for _, contentType := range []string{"application/pdf", "application/octet-stream"} {
		r := ts.uploadFile(token, "assignment.pdf", contentType, []byte("%PDF-1.4\n"))
		if !r.Success {
			t.Fatalf("existing PDF upload regressed for %s: %v", contentType, r.Error)
		}
	}
}
