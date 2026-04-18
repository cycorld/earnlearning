// Package llmproxy 는 llm.cycorld.com 의 Admin API 를 호출하는 HTTP 클라이언트.
//
// 인증: Authorization: Bearer <admin-key>
// 관리자 키는 env 로만 공급되며, 프론트엔드에는 절대 노출되지 않는다.
package llmproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client 는 llm-proxy admin API 호출기.
//
// 키 구분 (#076):
//   - adminKey: `/admin/api/*` 용. admin-* 로 시작하는 관리자 키.
//   - userKey:  `/v1/*` (chat completions 등) 용. sk-stu-* 학생 키.
//     비어있으면 adminKey fallback 이지만 chat API 는 401 반환.
type Client struct {
	baseURL  string
	adminKey string
	userKey  string
	http     *http.Client
}

// New 는 새 클라이언트를 만든다. baseURL 은 `https://llm.cycorld.com` 같이
// 스킴+호스트까지. 후행 슬래시는 붙여도/안 붙여도 무관.
func New(baseURL, adminKey string) *Client {
	return &Client{
		baseURL:  baseURL,
		adminKey: adminKey,
		http:     &http.Client{Timeout: 15 * time.Second},
	}
}

// SetUserKey 는 /v1/* 호출에 쓸 학생 키를 설정한다.
func (c *Client) SetUserKey(k string) { c.userKey = k }

// Student / Key / Usage response 타입들. Swagger 스펙을 그대로 미러링.

type Student struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Affiliation string `json:"affiliation"`
	Email       string `json:"email"`
	Note        string `json:"note,omitempty"`
	CreatedAt   string `json:"created_at"`
	ActiveKeys  int    `json:"active_keys"`
}

type IssuedKey struct {
	Key     string `json:"key"` // plaintext, one-time
	Prefix  string `json:"prefix"`
	Label   string `json:"label"`
	Warning string `json:"warning"`
}

type KeyMeta struct {
	ID         int    `json:"id"`
	Prefix     string `json:"prefix"`
	Label      string `json:"label"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at"`
	RevokedAt  string `json:"revoked_at"`
}

type UsageBucket struct {
	StudentID        int    `json:"student_id"`
	Email            string `json:"email"`
	Requests         int    `json:"requests"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	CacheHits        int    `json:"cache_hits"`
	CacheTokens      int    `json:"cache_tokens"` // prompt 중 KV 캐시 재사용분
	Errors           int    `json:"errors"`
}

type UsageResponse struct {
	Days      int           `json:"days"`
	ByStudent []UsageBucket `json:"by_student"`
}

// Swagger StatusResponse 미러링.
type ServiceStatus struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	PID           int    `json:"pid"`
}

type UpstreamStatus struct {
	URL             string  `json:"url"`
	Status          string  `json:"status"`
	LatencyMs       float64 `json:"latency_ms,omitempty"`
	Model           string  `json:"model,omitempty"`
	NCtx            int     `json:"n_ctx,omitempty"`
	SlotsTotal      int     `json:"slots_total,omitempty"`
	SlotsIdle       int     `json:"slots_idle,omitempty"`
	SlotsProcessing int     `json:"slots_processing,omitempty"`
}

type DatabaseStatus struct {
	Students     int `json:"students"`
	KeysActive   int `json:"keys_active"`
	KeysRevoked  int `json:"keys_revoked"`
}

type LogsStatus struct {
	Dir                   string `json:"dir"`
	ConversationsToday    int    `json:"conversations_today"`
	MetricsFileSizeBytes  int64  `json:"metrics_file_size_bytes"`
}

type StatusResponse struct {
	Service  ServiceStatus  `json:"service"`
	Upstream UpstreamStatus `json:"upstream"`
	Database DatabaseStatus `json:"database"`
	Logs     LogsStatus     `json:"logs"`
}

// CreateStudent 는 llm-proxy 에 신규 학생을 등록한다. email unique.
func (c *Client) CreateStudent(ctx context.Context, name, affiliation, email, note string) (*Student, error) {
	body := map[string]string{
		"name":        name,
		"affiliation": affiliation,
		"email":       email,
		"note":        note,
	}
	var out Student
	if err := c.do(ctx, http.MethodPost, "/admin/api/students", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// FindStudentByEmail 은 등록된 학생 목록을 훑어 email 로 찾는다.
// 학생 수가 수백 명을 넘으면 upstream 이 페이지네이션을 도입해야 함 — 수업
// 규모에서는 O(N) 훑기로 충분.
func (c *Client) FindStudentByEmail(ctx context.Context, email string) (*Student, error) {
	var list []Student
	if err := c.do(ctx, http.MethodGet, "/admin/api/students", nil, &list); err != nil {
		return nil, err
	}
	for _, s := range list {
		if s.Email == email {
			copy := s
			return &copy, nil
		}
	}
	return nil, nil
}

// IssueKey 는 학생에 새 API 키를 발급한다. 평문 키는 이 응답에서만 1회 반환됨.
func (c *Client) IssueKey(ctx context.Context, studentID int, label string) (*IssuedKey, error) {
	body := map[string]string{"label": label}
	var out IssuedKey
	path := fmt.Sprintf("/admin/api/students/%d/keys", studentID)
	if err := c.do(ctx, http.MethodPost, path, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListKeys 는 학생의 활성·폐기 키 메타를 반환한다 (평문 없음).
func (c *Client) ListKeys(ctx context.Context, studentID int) ([]KeyMeta, error) {
	var out []KeyMeta
	path := fmt.Sprintf("/admin/api/students/%d/keys", studentID)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RevokeKey 는 키를 즉시 무효화한다.
func (c *Client) RevokeKey(ctx context.Context, keyID int) error {
	path := fmt.Sprintf("/admin/api/keys/%d/revoke", keyID)
	return c.do(ctx, http.MethodPost, path, nil, nil)
}

// Status 는 llm-proxy 의 서비스 상태를 반환한다.
func (c *Client) Status(ctx context.Context) (*StatusResponse, error) {
	var out StatusResponse
	if err := c.do(ctx, http.MethodGet, "/admin/api/status", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Usage 는 최근 N일 롤링 윈도우의 학생별 집계를 반환한다.
// 과금 크론에서 days=1 로 호출해서 "전날 24h 사용량" 을 얻어낸다.
func (c *Client) Usage(ctx context.Context, days int) (*UsageResponse, error) {
	var out UsageResponse
	path := fmt.Sprintf("/admin/api/usage?days=%d", days)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// do 는 공통 HTTP 요청기. nil body/out 허용.
func (c *Client) do(ctx context.Context, method, path string, body interface{}, out interface{}) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.adminKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("llm-proxy %s %s: %d %s", method, path, resp.StatusCode, string(raw))
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
