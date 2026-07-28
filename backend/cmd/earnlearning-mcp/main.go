package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const maxResults = 50
const maxResponseBytes = 2 << 20

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}
type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}
type apiClient struct {
	base   *url.URL
	token  string
	client *http.Client
}
type server struct {
	api    *apiClient
	logger *log.Logger
}
type tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func newAPIClient(rawBase, token string, allowRemote bool, client *http.Client) (*apiClient, error) {
	if strings.TrimSpace(rawBase) == "" {
		return nil, errors.New("EARNLEARNING_API_BASE_URL is required")
	}
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("EARNLEARNING_API_TOKEN is required")
	}
	u, err := url.Parse(rawBase)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errors.New("invalid API base URL")
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("API base URL must not contain credentials, query, or fragment")
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	local := strings.EqualFold(host, "localhost") || (ip != nil && ip.IsLoopback())
	if !local && !allowRemote {
		return nil, errors.New("remote API base URL rejected; set EARNLEARNING_MCP_ALLOW_REMOTE=true to opt in")
	}
	if !local && u.Scheme != "https" {
		return nil, errors.New("remote API base URL must use HTTPS")
	}
	if client == nil {
		client = &http.Client{}
	}
	clientCopy := *client
	clientCopy.Timeout = 10 * time.Second
	clientCopy.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	u.Path = strings.TrimRight(u.Path, "/")
	return &apiClient{base: u, token: token, client: &clientCopy}, nil
}

func intArg(args map[string]any, name string, required bool, min, max, def int) (int, error) {
	v, ok := args[name]
	if !ok {
		if required {
			return 0, fmt.Errorf("%s is required", name)
		}
		return def, nil
	}
	var n int
	switch value := v.(type) {
	case float64:
		if value != float64(int(value)) {
			return 0, fmt.Errorf("%s must be an integer between %d and %d", name, min, max)
		}
		n = int(value)
	case int:
		n = value
	default:
		return 0, fmt.Errorf("%s must be an integer between %d and %d", name, min, max)
	}
	if n < min || n > max {
		return 0, fmt.Errorf("%s must be an integer between %d and %d", name, min, max)
	}
	return n, nil
}
func stringArg(args map[string]any, name string, allowed ...string) (string, error) {
	v, ok := args[name]
	if !ok {
		return "", nil
	}
	s, ok := v.(string)
	if !ok || len(s) > 100 {
		return "", fmt.Errorf("%s must be a string of at most 100 characters", name)
	}
	if len(allowed) > 0 && s != "" {
		for _, a := range allowed {
			if s == a {
				return s, nil
			}
		}
		return "", fmt.Errorf("%s has an unsupported value", name)
	}
	return s, nil
}

func validateArgs(args map[string]any, allowed ...string) error {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		allowedSet[name] = struct{}{}
	}
	for name := range args {
		if _, ok := allowedSet[name]; !ok {
			return fmt.Errorf("unsupported argument %q", name)
		}
	}
	return nil
}

func (c *apiClient) call(name string, args map[string]any) (json.RawMessage, error) {
	if args == nil {
		args = map[string]any{}
	}
	path := ""
	q := url.Values{}
	switch name {
	case "health_get":
		if err := validateArgs(args); err != nil {
			return nil, err
		}
		path = "/api/health"
	case "profile_get":
		if err := validateArgs(args); err != nil {
			return nil, err
		}
		path = "/api/auth/me"
	case "companies_list":
		if err := validateArgs(args); err != nil {
			return nil, err
		}
		path = "/api/companies"
	case "grants_list":
		if err := validateArgs(args, "page", "limit", "status"); err != nil {
			return nil, err
		}
		path = "/api/grants"
		page, e := intArg(args, "page", false, 1, 10000, 1)
		if e != nil {
			return nil, e
		}
		limit, e := intArg(args, "limit", false, 1, maxResults, 20)
		if e != nil {
			return nil, e
		}
		status, e := stringArg(args, "status")
		if e != nil {
			return nil, e
		}
		q.Set("page", strconv.Itoa(page))
		q.Set("limit", strconv.Itoa(limit))
		if status != "" {
			q.Set("status", status)
		}
	case "posts_list":
		if err := validateArgs(args, "classroom_id", "page", "limit", "tag"); err != nil {
			return nil, err
		}
		path = "/api/posts"
		classroom, e := intArg(args, "classroom_id", true, 1, 1_000_000, 0)
		if e != nil {
			return nil, e
		}
		page, e := intArg(args, "page", false, 1, 10000, 1)
		if e != nil {
			return nil, e
		}
		limit, e := intArg(args, "limit", false, 1, maxResults, 20)
		if e != nil {
			return nil, e
		}
		tag, e := stringArg(args, "tag")
		if e != nil {
			return nil, e
		}
		q.Set("classroom_id", strconv.Itoa(classroom))
		q.Set("page", strconv.Itoa(page))
		q.Set("limit", strconv.Itoa(limit))
		if tag != "" {
			q.Set("tag", tag)
		}
	default:
		return nil, fmt.Errorf("unknown tool %q", name)
	}
	u := *c.base
	u.Path += path
	u.RawQuery = q.Encode()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, u.String(), nil)
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.New("EarnLearning API request failed")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, errors.New("EarnLearning API response could not be read")
	}
	if len(body) > maxResponseBytes {
		return nil, errors.New("EarnLearning API response is too large")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("EarnLearning API returned HTTP %d", resp.StatusCode)
	}
	if !json.Valid(body) {
		return nil, errors.New("EarnLearning API returned invalid JSON")
	}
	return json.RawMessage(body), nil
}

func newServer(api *apiClient, logger *log.Logger) *server { return &server{api: api, logger: logger} }
func tools() []tool {
	empty := map[string]any{"type": "object", "properties": map[string]any{}, "additionalProperties": false}
	list := func(extra map[string]any, required ...string) map[string]any {
		s := map[string]any{"type": "object", "properties": extra, "additionalProperties": false}
		if len(required) > 0 {
			s["required"] = required
		}
		return s
	}
	page := map[string]any{"page": map[string]any{"type": "integer", "minimum": 1, "maximum": 10000}, "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": maxResults}}
	grant := map[string]any{"page": page["page"], "limit": page["limit"], "status": map[string]any{"type": "string", "maxLength": 100}}
	post := map[string]any{"classroom_id": map[string]any{"type": "integer", "minimum": 1}, "page": page["page"], "limit": page["limit"], "tag": map[string]any{"type": "string", "maxLength": 100}}
	return []tool{{"health_get", "EarnLearning API 상태를 조회합니다.", empty}, {"profile_get", "현재 인증 사용자의 /api/auth/me 정보를 조회합니다.", empty}, {"grants_list", "기존 /api/grants 과제 목록을 최대 50개 조회합니다.", list(grant)}, {"companies_list", "기존 /api/companies 기업 목록을 조회합니다.", empty}, {"posts_list", "기존 /api/posts 게시물을 클래스룸별 최대 50개 조회합니다.", list(post, "classroom_id")}}
}
func (s *server) handle(r request) response {
	out := response{JSONRPC: "2.0", ID: r.ID}
	fail := func(code int, msg string) response { out.Error = &rpcError{Code: code, Message: msg}; return out }
	if r.JSONRPC != "2.0" {
		return fail(-32600, "invalid request")
	}
	switch r.Method {
	case "initialize":
		out.Result = map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]any{"name": "earnlearning-mcp", "version": "0.1.0"}}
	case "notifications/initialized":
		out.Result = map[string]any{}
	case "tools/list":
		out.Result = map[string]any{"tools": tools()}
	case "tools/call":
		var p struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		dec := json.NewDecoder(bytes.NewReader(r.Params))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&p); err != nil || p.Name == "" {
			return fail(-32602, "invalid tools/call parameters")
		}
		if p.Arguments == nil {
			p.Arguments = map[string]any{}
		}
		data, err := s.api.call(p.Name, p.Arguments)
		if err != nil {
			s.logger.Printf("tool %s failed: %v", p.Name, err)
			return fail(-32000, err.Error())
		}
		out.Result = map[string]any{"content": []map[string]any{{"type": "text", "text": string(data)}}, "structuredContent": json.RawMessage(data)}
	default:
		return fail(-32601, "method not found")
	}
	return out
}

func serve(in io.Reader, out io.Writer, s *server) error {
	r := bufio.NewReader(in)
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var payload []byte
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			n, e := strconv.Atoi(strings.TrimSpace(strings.SplitN(line, ":", 2)[1]))
			if e != nil || n <= 0 || n > 2<<20 {
				return errors.New("invalid Content-Length")
			}
			for {
				h, e := r.ReadString('\n')
				if e != nil {
					return e
				}
				if strings.TrimSpace(h) == "" {
					break
				}
			}
			payload = make([]byte, n)
			if _, e = io.ReadFull(r, payload); e != nil {
				return e
			}
		} else {
			payload = []byte(line)
		}
		var req request
		if json.Unmarshal(payload, &req) != nil {
			writeResponse(out, response{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "parse error"}})
			continue
		}
		if len(req.ID) == 0 {
			s.handle(req)
			continue
		}
		writeResponse(out, s.handle(req))
	}
}
func writeResponse(w io.Writer, resp response) error {
	b, e := json.Marshal(resp)
	if e != nil {
		return e
	}
	_, e = fmt.Fprintf(w, "%s\n", b)
	return e
}
func main() {
	allowRemote := strings.EqualFold(os.Getenv("EARNLEARNING_MCP_ALLOW_REMOTE"), "true")
	client, err := newAPIClient(os.Getenv("EARNLEARNING_API_BASE_URL"), os.Getenv("EARNLEARNING_API_TOKEN"), allowRemote, &http.Client{Timeout: 10 * time.Second})
	if err != nil {
		log.Printf("configuration error: %v", err)
		os.Exit(1)
	}
	if err := serve(os.Stdin, os.Stdout, newServer(client, log.New(os.Stderr, "earnlearning-mcp: ", log.LstdFlags))); err != nil {
		log.Printf("server error: %v", err)
		os.Exit(1)
	}
}
