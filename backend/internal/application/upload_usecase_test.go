package application

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type failingUploadReader struct{ sent bool }

func (r *failingUploadReader) Read(p []byte) (int, error) {
	if !r.sent {
		r.sent = true
		return copy(p, "partial"), nil
	}
	return 0, errors.New("read failed")
}

func TestCopyUploadFileRemovesPartialFileOnCopyFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "partial.md")
	if _, err := copyUploadFile(path, &failingUploadReader{}); err == nil {
		t.Fatal("expected copy failure")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial upload was not removed: %v", err)
	}
}

func TestValidateMarkdownContent(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content []byte
	}{
		{"invalid UTF-8", []byte{0xff, 0xfe}},
		{"binary NUL", []byte("# title\x00payload")},
		{"HTML active content", []byte("<!doctype html><script>alert(1)</script>")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateUploadContent(".md", tc.content); err == nil {
				t.Fatal("spoofed markdown should be rejected")
			}
		})
	}
	if err := validateUploadContent(".md", []byte("# valid UTF-8 마크다운\n")); err != nil {
		t.Fatalf("valid markdown rejected: %v", err)
	}
}

func TestValidateTextContentRejectsPayloadAfterSniffWindow(t *testing.T) {
	prefix := bytes.Repeat([]byte("a"), 513)
	for _, ext := range []string{".md", ".txt", ".json"} {
		for _, tc := range []struct {
			name    string
			payload []byte
		}{
			{"invalid UTF-8", []byte{0xff}},
			{"NUL", []byte{0}},
			{"active HTML", []byte("<script>alert(1)</script>")},
		} {
			t.Run(ext+"/"+tc.name, func(t *testing.T) {
				content := append(append([]byte(nil), prefix...), tc.payload...)
				if err := validateUploadContent(ext, content); err == nil {
					t.Fatal("unsafe payload after byte 512 should be rejected")
				}
			})
		}
	}
}
