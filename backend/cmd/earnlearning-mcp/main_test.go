package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeUsesNewlineDelimitedJSONRPC(t *testing.T) {
	c, err := newAPIClient("http://localhost:8080", "token", false, http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	input := strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{}}\n")
	var output bytes.Buffer
	if err := serve(input, &output, newServer(c, log.New(io.Discard, "", 0))); err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(output.String(), "Content-Length:") {
		t.Fatalf("stdio MCP must use newline-delimited JSON, got %q", output.String())
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 1 || !json.Valid([]byte(lines[0])) {
		t.Fatalf("expected one JSON line, got %q", output.String())
	}
}

func TestAPIClientDoesNotForwardAuthorizationAcrossRedirects(t *testing.T) {
	var redirectedAuth string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectedAuth = r.Header.Get("Authorization")
		io.WriteString(w, `{"success":true}`)
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer source.Close()
	c, err := newAPIClient(source.URL, "secret", false, source.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.call("health_get", nil); err == nil {
		t.Fatal("expected redirect rejection")
	}
	if redirectedAuth != "" {
		t.Fatal("authorization header leaked to redirect target")
	}
}

func TestAPIClientRejectsOversizedResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":"`+strings.Repeat("x", (2<<20)+1)+`"}`)
	}))
	defer ts.Close()
	c, err := newAPIClient(ts.URL, "token", false, ts.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.call("health_get", nil); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected oversized response rejection, got %v", err)
	}
}

func TestToolCallRejectsUnknownArguments(t *testing.T) {
	c, err := newAPIClient("http://localhost:8080", "token", false, http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	s := newServer(c, log.New(io.Discard, "", 0))
	resp := s.handle(request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"health_get","arguments":{"unexpected":true}}`),
	})
	if resp.Error == nil || !strings.Contains(resp.Error.Message, "unsupported argument") {
		t.Fatalf("expected unsupported argument error, got %+v", resp.Error)
	}
}

func TestNewAPIClientRejectsUnsafeConfiguration(t *testing.T) {
	for _, tc := range []struct{ name, base, token string }{
		{"missing base", "", "secret"}, {"missing token", "http://localhost:8080", ""},
		{"remote", "https://earnlearning.com", "secret"}, {"userinfo", "http://secret@localhost:8080", "secret"},
		{"remote plaintext", "http://example.test", "secret"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := newAPIClient(tc.base, tc.token, false, http.DefaultClient); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
	if _, err := newAPIClient("https://example.test", "secret", true, http.DefaultClient); err != nil {
		t.Fatalf("explicit remote opt-in rejected: %v", err)
	}
}

func TestAPIClientAuthAndBoundedQuery(t *testing.T) {
	var gotAuth, gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotQuery = r.Header.Get("Authorization"), r.URL.RawQuery
		io.WriteString(w, `{"success":true,"data":{"data":[]}}`)
	}))
	defer ts.Close()
	c, err := newAPIClient(ts.URL, "test-token", false, ts.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.call("posts_list", map[string]any{"classroom_id": 1, "page": 2, "limit": 999}); err == nil {
		t.Fatal("expected invalid limit")
	}
	if _, err = c.call("posts_list", map[string]any{"classroom_id": 1, "page": 2, "limit": 25}); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("auth=%q", gotAuth)
	}
	if !strings.Contains(gotQuery, "classroom_id=1") || !strings.Contains(gotQuery, "limit=25") {
		t.Fatalf("query=%q", gotQuery)
	}
}

func TestServerInitializeListAndCall(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"success":true,"data":"ok","error":null}`)
	}))
	defer ts.Close()
	c, _ := newAPIClient(ts.URL, "token", false, ts.Client())
	s := newServer(c, log.New(io.Discard, "", 0))
	init := s.handle(request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "initialize"})
	if init.Error != nil || init.Result == nil {
		t.Fatalf("initialize: %+v", init)
	}
	list := s.handle(request{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: "tools/list"})
	b, _ := json.Marshal(list.Result)
	for _, name := range []string{"health_get", "profile_get", "grants_list", "companies_list", "posts_list"} {
		if !bytes.Contains(b, []byte(name)) {
			t.Errorf("missing %s", name)
		}
	}
	params := json.RawMessage(`{"name":"health_get","arguments":{}}`)
	call := s.handle(request{JSONRPC: "2.0", ID: json.RawMessage(`3`), Method: "tools/call", Params: params})
	if call.Error != nil {
		t.Fatalf("call: %+v", call.Error)
	}
}

func TestValidationAndTokenRedaction(t *testing.T) {
	const token = "super-secret-token"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "upstream failed", 500) }))
	defer ts.Close()
	c, _ := newAPIClient(ts.URL, token, false, ts.Client())
	var logs bytes.Buffer
	s := newServer(c, log.New(&logs, "", 0))
	bad := s.handle(request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "tools/call", Params: json.RawMessage(`{"name":"posts_list","arguments":{"limit":0}}`)})
	if bad.Error == nil {
		t.Fatal("expected structured error")
	}
	failed := s.handle(request{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: "tools/call", Params: json.RawMessage(`{"name":"health_get","arguments":{}}`)})
	out, _ := json.Marshal(failed)
	if bytes.Contains(out, []byte(token)) || strings.Contains(logs.String(), token) {
		t.Fatal("token leaked")
	}
}
