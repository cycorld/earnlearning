package llmproxy

import (
	"context"
	"path/filepath"

	"github.com/earnlearning/backend/internal/application"
)

// UseCaseAdapter wraps Client to match application.ProxyClient without leaking HTTP types.
type UseCaseAdapter struct {
	c *Client
}

func NewUseCaseAdapter(c *Client) *UseCaseAdapter {
	return &UseCaseAdapter{c: c}
}

func (a *UseCaseAdapter) CreateStudent(ctx context.Context, name, affiliation, email, note string) (int, error) {
	s, err := a.c.CreateStudent(ctx, name, affiliation, email, note)
	if err != nil {
		return 0, err
	}
	return s.ID, nil
}

func (a *UseCaseAdapter) FindStudentByEmail(ctx context.Context, email string) (int, bool, error) {
	s, err := a.c.FindStudentByEmail(ctx, email)
	if err != nil {
		return 0, false, err
	}
	if s == nil {
		return 0, false, nil
	}
	return s.ID, true, nil
}

func (a *UseCaseAdapter) IssueKey(ctx context.Context, studentID int, label string) (string, string, int, error) {
	out, err := a.c.IssueKey(ctx, studentID, label)
	if err != nil {
		return "", "", 0, err
	}
	// Swagger 스펙상 IssuedKey 에는 key/prefix/label/warning 만 있음.
	// key_id 는 응답에 명시되어 있지 않아, 발급 직후 ListKeys 로 찾아낸다.
	// (prefix 로 매칭)
	keys, err := a.c.ListKeys(ctx, studentID)
	if err != nil {
		return "", "", 0, err
	}
	var id int
	for _, k := range keys {
		if k.Prefix == out.Prefix && k.RevokedAt == "" {
			id = k.ID
			break
		}
	}
	return out.Key, out.Prefix, id, nil
}

func (a *UseCaseAdapter) RevokeKey(ctx context.Context, keyID int) error {
	return a.c.RevokeKey(ctx, keyID)
}

func (a *UseCaseAdapter) Status(ctx context.Context) (*application.ProxyStatus, error) {
	s, err := a.c.Status(ctx)
	if err != nil {
		return nil, err
	}
	// 모델 경로는 파일명만 노출 (파일시스템 경로 유출 방지).
	model := s.Upstream.Model
	if model != "" {
		model = filepath.Base(model)
	}
	return &application.ProxyStatus{
		Service:         s.Service.Name,
		Version:         s.Service.Version,
		UptimeSeconds:   s.Service.UptimeSeconds,
		Upstream:        s.Upstream.Status,
		Model:           model,
		LatencyMs:       s.Upstream.LatencyMs,
		ContextWindow:   s.Upstream.NCtx,
		SlotsTotal:      s.Upstream.SlotsTotal,
		SlotsIdle:       s.Upstream.SlotsIdle,
		SlotsProcessing: s.Upstream.SlotsProcessing,
	}, nil
}

func (a *UseCaseAdapter) Usage(ctx context.Context, days int) (map[int]application.ProxyUsage, error) {
	out, err := a.c.Usage(ctx, days)
	if err != nil {
		return nil, err
	}
	m := make(map[int]application.ProxyUsage, len(out.ByStudent))
	for _, b := range out.ByStudent {
		m[b.StudentID] = application.ProxyUsage{
			Requests:         b.Requests,
			PromptTokens:     b.PromptTokens,
			CompletionTokens: b.CompletionTokens,
			CacheHits:        b.CacheHits,
			CacheTokens:      b.CacheTokens,
			Errors:           b.Errors,
		}
	}
	return m, nil
}
