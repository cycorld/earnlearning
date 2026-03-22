package integration

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDocs(t *testing.T) {
	ts := setupTestServer(t)

	t.Run("GET /docs returns HTML with Scalar", func(t *testing.T) {
		resp, err := http.Get(ts.url("/docs"))
		if err != nil {
			t.Fatalf("GET /docs: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		html := string(body)

		if !strings.Contains(html, "Scalar") && !strings.Contains(html, "scalar") && !strings.Contains(html, "api-reference") {
			t.Error("response does not contain Scalar reference")
		}
		if !strings.Contains(html, "openapi.json") {
			t.Error("response does not contain openapi.json reference")
		}
	})

	t.Run("GET /docs/openapi.json returns valid JSON", func(t *testing.T) {
		resp, err := http.Get(ts.url("/docs/openapi.json"))
		if err != nil {
			t.Fatalf("GET /docs/openapi.json: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("expected application/json, got %s", contentType)
		}

		body, _ := io.ReadAll(resp.Body)
		if len(body) < 100 {
			t.Error("response body too small to be valid swagger spec")
		}
		if !strings.Contains(string(body), "EarnLearning") {
			t.Error("spec does not contain expected title")
		}
	})
}
