package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net"
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
	input := strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"2025-03-26\",\"capabilities\":{},\"clientInfo\":{\"name\":\"test\",\"version\":\"1\"}}}\n")
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
	s.initialized = true
	resp := s.handle(request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"health_get","arguments":{"unexpected":true}}`),
	})
	b, _ := json.Marshal(resp.Result)
	if resp.Error != nil || !bytes.Contains(b, []byte(`"isError":true`)) || !bytes.Contains(b, []byte("unsupported argument")) {
		t.Fatalf("expected tool error result, got error=%+v result=%s", resp.Error, b)
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

func TestIsPublicIPRejectsInternalRanges(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "::1", "fc00::1", "ff02::1"} {
		if isPublicIP(net.ParseIP(raw)) {
			t.Errorf("expected %s to be non-public", raw)
		}
	}
	if !isPublicIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("expected public address to be accepted")
	}
}

func TestServerRequiresInitializationAndRejectsTrailingJSON(t *testing.T) {
	c, _ := newAPIClient("http://localhost:8080", "token", false, http.DefaultClient)
	s := newServer(c, log.New(io.Discard, "", 0))
	before := s.handle(request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "tools/list"})
	if before.Error == nil || before.Error.Code != -32002 {
		t.Fatalf("expected initialization error, got %+v", before.Error)
	}
	badInit := s.handle(request{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: "initialize", Params: json.RawMessage(`{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test"}} {}`)})
	if badInit.Error == nil || badInit.Error.Code != -32602 {
		t.Fatalf("expected invalid params, got %+v", badInit.Error)
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
	init := s.handle(request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "initialize", Params: json.RawMessage(`{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1"}}`)})
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
	s.initialized = true
	bad := s.handle(request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "tools/call", Params: json.RawMessage(`{"name":"posts_list","arguments":{"limit":0}}`)})
	badResult, _ := json.Marshal(bad.Result)
	if bad.Error != nil || !bytes.Contains(badResult, []byte(`"isError":true`)) {
		t.Fatalf("expected structured tool error, got error=%+v result=%s", bad.Error, badResult)
	}
	failed := s.handle(request{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: "tools/call", Params: json.RawMessage(`{"name":"health_get","arguments":{}}`)})
	out, _ := json.Marshal(failed)
	if bytes.Contains(out, []byte(token)) || strings.Contains(logs.String(), token) {
		t.Fatal("token leaked")
	}
}

func TestMutationRequiresExactConfirmationAndBuildsAllowlistedRequest(t *testing.T) {
	var method, path string
	var body map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		io.WriteString(w, `{"success":true}`)
	}))
	defer ts.Close()
	c, _ := newAPIClient(ts.URL, "token", false, ts.Client())
	args := map[string]any{"channel_id": 7, "content": "hello"}
	if _, err := c.call("post_create", args); err == nil || !strings.Contains(err.Error(), "confirm") {
		t.Fatalf("expected confirmation rejection, got %v", err)
	}
	args["confirm"] = "post_create"
	if _, err := c.call("post_create", args); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPost || path != "/api/channels/7/posts" {
		t.Fatalf("%s %s", method, path)
	}
	if body["content"] != "hello" || body["confirm"] != nil || body["channel_id"] != nil {
		t.Fatalf("body=%v", body)
	}
	args["channel_id"] = "7/../../admin"
	if _, err := c.call("post_create", args); err == nil {
		t.Fatal("expected non-integer path rejection")
	}
}

func TestOperationsRegistryContract(t *testing.T) {
	names := map[string]bool{}
	for _, tool := range tools() {
		names[tool.Name] = true
	}
	for _, name := range []string{"health_get", "admin_users_list", "admin_user_approve", "classrooms_list", "wallet_get", "company_services_list", "investment_portfolio_get", "post_create", "assignment_submit", "company_service_create", "grant_apply", "exchange_order_create", "wallet_transfer", "loan_repay"} {
		if !names[name] {
			t.Errorf("missing %s", name)
		}
	}
	for _, excluded := range []string{"dm_send", "mail_send", "upload_create", "admin_impersonate", "investment_dividend_execute"} {
		if names[excluded] {
			t.Errorf("unsafe tool exposed: %s", excluded)
		}
	}
}
