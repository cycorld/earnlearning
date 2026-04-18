package llmproxy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func TestClient_AttachesBearerAuth(t *testing.T) {
	var gotAuth string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})
	defer srv.Close()

	c := New(srv.URL, "secret-token")
	_, err := c.ListKeys(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	if gotAuth != "Bearer secret-token" {
		t.Errorf("auth header: got %q", gotAuth)
	}
}

func TestClient_IssueKey_PostsAndParsesPlaintext(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s", r.Method)
		}
		if r.URL.Path != "/admin/api/students/42/keys" {
			t.Errorf("path: got %s", r.URL.Path)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["label"] != "2026-1학기" {
			t.Errorf("label: got %q", body["label"])
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"key":"sk-plain","prefix":"sk-pl","label":"2026-1학기","warning":"one-time"}`))
	})
	defer srv.Close()

	c := New(srv.URL, "admin")
	out, err := c.IssueKey(context.Background(), 42, "2026-1학기")
	if err != nil {
		t.Fatalf("IssueKey: %v", err)
	}
	if out.Key != "sk-plain" || out.Prefix != "sk-pl" {
		t.Errorf("issued key: %+v", out)
	}
}

func TestClient_FindStudentByEmail_ReturnsNilIfMissing(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":1,"name":"A","email":"a@x.com","affiliation":""},{"id":2,"name":"B","email":"b@x.com","affiliation":""}]`))
	})
	defer srv.Close()

	c := New(srv.URL, "k")
	got, err := c.FindStudentByEmail(context.Background(), "missing@x.com")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestClient_FindStudentByEmail_ReturnsMatch(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":1,"name":"A","email":"a@x.com"},{"id":2,"name":"B","email":"b@x.com"}]`))
	})
	defer srv.Close()

	c := New(srv.URL, "k")
	got, err := c.FindStudentByEmail(context.Background(), "b@x.com")
	if err != nil || got == nil || got.ID != 2 {
		t.Fatalf("find: %v, %+v", err, got)
	}
}

func TestClient_Usage_ParsesByStudent(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "days=1" {
			t.Errorf("query: got %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"days":1,"by_student":[{"student_id":2,"email":"a@x.com","requests":3,"prompt_tokens":100,"completion_tokens":200,"cache_hits":1,"errors":0}]}`))
	})
	defer srv.Close()

	c := New(srv.URL, "k")
	out, err := c.Usage(context.Background(), 1)
	if err != nil {
		t.Fatalf("usage: %v", err)
	}
	if len(out.ByStudent) != 1 || out.ByStudent[0].PromptTokens != 100 {
		t.Fatalf("usage parsed wrong: %+v", out)
	}
}

func TestClient_RevokeKey_Returns4xxAsError(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"not found"}`))
	})
	defer srv.Close()

	c := New(srv.URL, "k")
	err := c.RevokeKey(context.Background(), 99)
	if err == nil {
		t.Fatalf("expected error for 404")
	}
}

func TestClient_CreateStudent_SendsBody(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["email"] != "s@ewha.ac.kr" || body["affiliation"] != "이화여대" {
			t.Errorf("body: %+v", body)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":7,"name":"S","affiliation":"이화여대","email":"s@ewha.ac.kr","created_at":"2026-04-18T00:00:00Z","active_keys":0}`))
	})
	defer srv.Close()

	c := New(srv.URL, "k")
	out, err := c.CreateStudent(context.Background(), "S", "이화여대", "s@ewha.ac.kr", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if out.ID != 7 {
		t.Fatalf("id: %+v", out)
	}
}
