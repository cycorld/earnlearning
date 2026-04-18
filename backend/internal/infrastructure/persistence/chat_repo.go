package persistence

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/earnlearning/backend/internal/domain/chat"
)

// ============================================================================
// Session repository
// ============================================================================

type ChatSessionRepo struct{ db *sql.DB }

func NewChatSessionRepo(db *sql.DB) *ChatSessionRepo { return &ChatSessionRepo{db: db} }

func (r *ChatSessionRepo) Create(s *chat.Session) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO chat_sessions (user_id, title, active_skill_id, last_message_at)
		VALUES (?, ?, ?, ?)`,
		s.UserID, s.Title, nullableInt(s.ActiveSkillID), s.LastMessageAt)
	if err != nil {
		return 0, fmt.Errorf("create chat session: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	s.ID = int(id)
	return s.ID, nil
}

func (r *ChatSessionRepo) FindByID(id int) (*chat.Session, error) {
	row := r.db.QueryRow(`
		SELECT id, user_id, title, active_skill_id, tokens_used, created_at, last_message_at
		FROM chat_sessions WHERE id = ?`, id)
	return scanSession(row)
}

func (r *ChatSessionRepo) ListByUser(userID, page, limit int) ([]*chat.Session, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM chat_sessions WHERE user_id = ?`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(`
		SELECT id, user_id, title, active_skill_id, tokens_used, created_at, last_message_at
		FROM chat_sessions WHERE user_id = ?
		ORDER BY COALESCE(last_message_at, created_at) DESC
		LIMIT ? OFFSET ?`, userID, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*chat.Session
	for rows.Next() {
		s, err := scanSessionRows(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

func (r *ChatSessionRepo) ListAll(page, limit int) ([]*chat.Session, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM chat_sessions`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(`
		SELECT id, user_id, title, active_skill_id, tokens_used, created_at, last_message_at
		FROM chat_sessions
		ORDER BY COALESCE(last_message_at, created_at) DESC
		LIMIT ? OFFSET ?`, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*chat.Session
	for rows.Next() {
		s, err := scanSessionRows(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

func (r *ChatSessionRepo) UpdateTitle(id int, title string) error {
	_, err := r.db.Exec(`UPDATE chat_sessions SET title = ? WHERE id = ?`, title, id)
	return err
}

func (r *ChatSessionRepo) UpdateActiveSkill(id int, skillID *int) error {
	_, err := r.db.Exec(`UPDATE chat_sessions SET active_skill_id = ? WHERE id = ?`, nullableInt(skillID), id)
	return err
}

func (r *ChatSessionRepo) UpdateLastMessageAt(id int, at time.Time, addTokens int) error {
	_, err := r.db.Exec(`UPDATE chat_sessions
		SET last_message_at = ?, tokens_used = tokens_used + ?
		WHERE id = ?`, at, addTokens, id)
	return err
}

func (r *ChatSessionRepo) Delete(id int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM chat_messages WHERE session_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM chat_sessions WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// ============================================================================
// Message repository
// ============================================================================

type ChatMessageRepo struct{ db *sql.DB }

func NewChatMessageRepo(db *sql.DB) *ChatMessageRepo { return &ChatMessageRepo{db: db} }

func (r *ChatMessageRepo) Create(m *chat.Message) (int, error) {
	toolCallsJSON := "[]"
	if len(m.ToolCalls) > 0 {
		b, err := json.Marshal(m.ToolCalls)
		if err != nil {
			return 0, fmt.Errorf("marshal tool_calls: %w", err)
		}
		toolCallsJSON = string(b)
	}
	res, err := r.db.Exec(`
		INSERT INTO chat_messages (session_id, role, content, reasoning_content, model,
			prompt_tokens, completion_tokens, cache_tokens, tool_calls, tool_call_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.SessionID, string(m.Role), m.Content, m.ReasoningContent, m.Model,
		m.PromptTokens, m.CompletionTokens, m.CacheTokens, toolCallsJSON, m.ToolCallID)
	if err != nil {
		return 0, fmt.Errorf("create chat message: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	m.ID = int(id)
	return m.ID, nil
}

func (r *ChatMessageRepo) ListBySession(sessionID, limit int) ([]*chat.Message, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.db.Query(`
		SELECT id, session_id, role, content, reasoning_content, model,
			prompt_tokens, completion_tokens, cache_tokens, tool_calls, tool_call_id, created_at
		FROM chat_messages WHERE session_id = ?
		ORDER BY id ASC LIMIT ?`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*chat.Message
	for rows.Next() {
		m := &chat.Message{}
		var role, toolCallsJSON string
		err := rows.Scan(&m.ID, &m.SessionID, &role, &m.Content, &m.ReasoningContent,
			&m.Model, &m.PromptTokens, &m.CompletionTokens, &m.CacheTokens,
			&toolCallsJSON, &m.ToolCallID, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		m.Role = chat.Role(role)
		if toolCallsJSON != "" && toolCallsJSON != "[]" {
			_ = json.Unmarshal([]byte(toolCallsJSON), &m.ToolCalls)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *ChatMessageRepo) CountBySession(sessionID int) (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE session_id = ?`, sessionID).Scan(&n)
	return n, err
}

// ============================================================================
// Skill repository
// ============================================================================

type ChatSkillRepo struct{ db *sql.DB }

func NewChatSkillRepo(db *sql.DB) *ChatSkillRepo { return &ChatSkillRepo{db: db} }

func (r *ChatSkillRepo) Create(s *chat.Skill) (int, error) {
	toolsJSON, _ := json.Marshal(s.ToolsAllowed)
	scopeJSON, _ := json.Marshal(s.WikiScope)
	res, err := r.db.Exec(`
		INSERT INTO chat_skills (slug, name, description, system_prompt, default_model,
			default_reasoning_effort, tools_allowed, wiki_scope, enabled, admin_only, created_by, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		s.Slug, s.Name, s.Description, s.SystemPrompt, s.DefaultModel,
		s.DefaultReasoningEffort, string(toolsJSON), string(scopeJSON),
		boolToInt(s.Enabled), boolToInt(s.AdminOnly), nullableInt(s.CreatedBy))
	if err != nil {
		return 0, fmt.Errorf("create skill: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	s.ID = int(id)
	return s.ID, nil
}

func (r *ChatSkillRepo) Upsert(s *chat.Skill) (int, error) {
	existing, err := r.FindBySlug(s.Slug)
	if err != nil && !errors.Is(err, chat.ErrSkillNotFound) {
		return 0, err
	}
	if existing != nil {
		s.ID = existing.ID
		if err := r.Update(s); err != nil {
			return 0, err
		}
		return s.ID, nil
	}
	return r.Create(s)
}

func (r *ChatSkillRepo) FindBySlug(slug string) (*chat.Skill, error) {
	row := r.db.QueryRow(`
		SELECT id, slug, name, description, system_prompt, default_model,
			default_reasoning_effort, tools_allowed, wiki_scope, enabled, admin_only, created_by, updated_at
		FROM chat_skills WHERE slug = ?`, slug)
	return scanSkill(row)
}

func (r *ChatSkillRepo) FindByID(id int) (*chat.Skill, error) {
	row := r.db.QueryRow(`
		SELECT id, slug, name, description, system_prompt, default_model,
			default_reasoning_effort, tools_allowed, wiki_scope, enabled, admin_only, created_by, updated_at
		FROM chat_skills WHERE id = ?`, id)
	return scanSkill(row)
}

func (r *ChatSkillRepo) List(includeDisabled, includeAdminOnly bool) ([]*chat.Skill, error) {
	where := []string{"1=1"}
	if !includeDisabled {
		where = append(where, "enabled = 1")
	}
	if !includeAdminOnly {
		where = append(where, "admin_only = 0")
	}
	q := fmt.Sprintf(`
		SELECT id, slug, name, description, system_prompt, default_model,
			default_reasoning_effort, tools_allowed, wiki_scope, enabled, admin_only, created_by, updated_at
		FROM chat_skills WHERE %s ORDER BY slug`, strings.Join(where, " AND "))
	rows, err := r.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*chat.Skill
	for rows.Next() {
		s, err := scanSkillRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ChatSkillRepo) Update(s *chat.Skill) error {
	toolsJSON, _ := json.Marshal(s.ToolsAllowed)
	scopeJSON, _ := json.Marshal(s.WikiScope)
	_, err := r.db.Exec(`
		UPDATE chat_skills SET
			name = ?, description = ?, system_prompt = ?, default_model = ?,
			default_reasoning_effort = ?, tools_allowed = ?, wiki_scope = ?,
			enabled = ?, admin_only = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		s.Name, s.Description, s.SystemPrompt, s.DefaultModel,
		s.DefaultReasoningEffort, string(toolsJSON), string(scopeJSON),
		boolToInt(s.Enabled), boolToInt(s.AdminOnly), s.ID)
	return err
}

func (r *ChatSkillRepo) Delete(id int) error {
	_, err := r.db.Exec(`DELETE FROM chat_skills WHERE id = ?`, id)
	return err
}

// ============================================================================
// Wiki repository (meta + FTS5)
// ============================================================================

type ChatWikiRepo struct{ db *sql.DB }

func NewChatWikiRepo(db *sql.DB) *ChatWikiRepo { return &ChatWikiRepo{db: db} }

func (r *ChatWikiRepo) UpsertMeta(m *chat.WikiDocMeta) error {
	_, err := r.db.Exec(`
		INSERT INTO chat_wiki_meta (slug, path, title, notion_page_id, synced_at, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(slug) DO UPDATE SET
			path = excluded.path,
			title = excluded.title,
			notion_page_id = excluded.notion_page_id,
			synced_at = excluded.synced_at,
			updated_at = CURRENT_TIMESTAMP`,
		m.Slug, m.Path, m.Title, m.NotionPageID, nullableTime(m.SyncedAt))
	return err
}

func (r *ChatWikiRepo) FindMeta(slug string) (*chat.WikiDocMeta, error) {
	row := r.db.QueryRow(`
		SELECT slug, path, title, notion_page_id, synced_at, updated_at
		FROM chat_wiki_meta WHERE slug = ?`, slug)
	m := &chat.WikiDocMeta{}
	var synced sql.NullTime
	err := row.Scan(&m.Slug, &m.Path, &m.Title, &m.NotionPageID, &synced, &m.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if synced.Valid {
		m.SyncedAt = synced.Time
	}
	return m, nil
}

func (r *ChatWikiRepo) ListMeta() ([]*chat.WikiDocMeta, error) {
	rows, err := r.db.Query(`
		SELECT slug, path, title, notion_page_id, synced_at, updated_at
		FROM chat_wiki_meta ORDER BY slug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*chat.WikiDocMeta
	for rows.Next() {
		m := &chat.WikiDocMeta{}
		var synced sql.NullTime
		if err := rows.Scan(&m.Slug, &m.Path, &m.Title, &m.NotionPageID, &synced, &m.UpdatedAt); err != nil {
			return nil, err
		}
		if synced.Valid {
			m.SyncedAt = synced.Time
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *ChatWikiRepo) DeleteMeta(slug string) error {
	_, err := r.db.Exec(`DELETE FROM chat_wiki_meta WHERE slug = ?`, slug)
	return err
}

func (r *ChatWikiRepo) UpsertDoc(slug, title, body string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM chat_wiki_docs WHERE slug = ?`, slug); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO chat_wiki_docs (slug, title, body) VALUES (?, ?, ?)`, slug, title, body); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ChatWikiRepo) DeleteDoc(slug string) error {
	_, err := r.db.Exec(`DELETE FROM chat_wiki_docs WHERE slug = ?`, slug)
	return err
}

func (r *ChatWikiRepo) Reset() error {
	_, err := r.db.Exec(`DELETE FROM chat_wiki_docs`)
	return err
}

// Search 는 FTS5 MATCH 로 BM25 검색. scope 는 slug glob 리스트.
func (r *ChatWikiRepo) Search(query string, scope []string, limit int) ([]*chat.WikiSearchHit, error) {
	if limit <= 0 || limit > 50 {
		limit = 8
	}
	// FTS5 MATCH 구문: "word" OR "word2" 로 단순화 (사용자 문자열을 양쪽 " 로 감싸 OR 로 연결)
	q := buildFtsQuery(query)
	if q == "" {
		return nil, nil
	}
	// scope 는 WHERE slug LIKE ? OR slug LIKE ? 로 처리
	args := []any{q}
	where := ""
	if len(scope) > 0 {
		parts := make([]string, 0, len(scope))
		for _, s := range scope {
			parts = append(parts, "slug LIKE ?")
			args = append(args, globToLike(s))
		}
		where = " AND (" + strings.Join(parts, " OR ") + ")"
	}
	args = append(args, limit)

	rows, err := r.db.Query(`
		SELECT slug, title, snippet(chat_wiki_docs, 2, '[', ']', ' … ', 64) AS snippet,
			bm25(chat_wiki_docs) AS score
		FROM chat_wiki_docs WHERE chat_wiki_docs MATCH ?`+where+`
		ORDER BY score ASC LIMIT ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}
	defer rows.Close()
	var out []*chat.WikiSearchHit
	for rows.Next() {
		h := &chat.WikiSearchHit{}
		if err := rows.Scan(&h.Slug, &h.Title, &h.Snippet, &h.Score); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// ============================================================================
// Usage repository
// ============================================================================

type ChatUsageRepo struct{ db *sql.DB }

func NewChatUsageRepo(db *sql.DB) *ChatUsageRepo { return &ChatUsageRepo{db: db} }

func (r *ChatUsageRepo) AddUsage(userID int, day time.Time, requests, prompt, completion, cache, costKRW int) error {
	date := day.Format("2006-01-02")
	_, err := r.db.Exec(`
		INSERT INTO chat_usage (user_id, usage_date, requests, prompt_tokens, completion_tokens, cache_tokens, cost_krw)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, usage_date) DO UPDATE SET
			requests = requests + excluded.requests,
			prompt_tokens = prompt_tokens + excluded.prompt_tokens,
			completion_tokens = completion_tokens + excluded.completion_tokens,
			cache_tokens = cache_tokens + excluded.cache_tokens,
			cost_krw = cost_krw + excluded.cost_krw`,
		userID, date, requests, prompt, completion, cache, costKRW)
	return err
}

func (r *ChatUsageRepo) SumForRange(from, to time.Time) ([]*chat.UsageDay, error) {
	rows, err := r.db.Query(`
		SELECT usage_date, SUM(requests), SUM(prompt_tokens), SUM(completion_tokens),
			SUM(cache_tokens), SUM(cost_krw)
		FROM chat_usage WHERE usage_date >= ? AND usage_date <= ?
		GROUP BY usage_date ORDER BY usage_date DESC`,
		from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*chat.UsageDay
	for rows.Next() {
		d := &chat.UsageDay{}
		var date string
		if err := rows.Scan(&date, &d.Requests, &d.PromptTokens, &d.CompletionTokens,
			&d.CacheTokens, &d.CostKRW); err != nil {
			return nil, err
		}
		t, _ := time.Parse("2006-01-02", date)
		d.Date = t
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *ChatUsageRepo) SumForMonth(year int, month time.Month) (*chat.UsageDay, error) {
	from := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, -1)
	d := &chat.UsageDay{Date: from}
	err := r.db.QueryRow(`
		SELECT COALESCE(SUM(requests),0), COALESCE(SUM(prompt_tokens),0),
			COALESCE(SUM(completion_tokens),0), COALESCE(SUM(cache_tokens),0),
			COALESCE(SUM(cost_krw),0)
		FROM chat_usage WHERE usage_date >= ? AND usage_date <= ?`,
		from.Format("2006-01-02"), to.Format("2006-01-02")).Scan(
		&d.Requests, &d.PromptTokens, &d.CompletionTokens, &d.CacheTokens, &d.CostKRW)
	return d, err
}

// ============================================================================
// Helpers
// ============================================================================

func scanSession(s scanner) (*chat.Session, error) {
	sess := &chat.Session{}
	var activeSkill sql.NullInt64
	var lastAt sql.NullTime
	err := s.Scan(&sess.ID, &sess.UserID, &sess.Title, &activeSkill, &sess.TokensUsed,
		&sess.CreatedAt, &lastAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, chat.ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	if activeSkill.Valid {
		v := int(activeSkill.Int64)
		sess.ActiveSkillID = &v
	}
	if lastAt.Valid {
		sess.LastMessageAt = lastAt.Time
	}
	return sess, nil
}

func scanSessionRows(rows *sql.Rows) (*chat.Session, error) {
	sess := &chat.Session{}
	var activeSkill sql.NullInt64
	var lastAt sql.NullTime
	err := rows.Scan(&sess.ID, &sess.UserID, &sess.Title, &activeSkill, &sess.TokensUsed,
		&sess.CreatedAt, &lastAt)
	if err != nil {
		return nil, err
	}
	if activeSkill.Valid {
		v := int(activeSkill.Int64)
		sess.ActiveSkillID = &v
	}
	if lastAt.Valid {
		sess.LastMessageAt = lastAt.Time
	}
	return sess, nil
}

func scanSkill(s scanner) (*chat.Skill, error) {
	sk := &chat.Skill{}
	var toolsJSON, scopeJSON, effort string
	var enabled, adminOnly int
	var createdBy sql.NullInt64
	err := s.Scan(&sk.ID, &sk.Slug, &sk.Name, &sk.Description, &sk.SystemPrompt, &sk.DefaultModel,
		&effort, &toolsJSON, &scopeJSON, &enabled, &adminOnly, &createdBy, &sk.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, chat.ErrSkillNotFound
	}
	if err != nil {
		return nil, err
	}
	sk.DefaultReasoningEffort = effort
	sk.Enabled = enabled != 0
	sk.AdminOnly = adminOnly != 0
	if createdBy.Valid {
		v := int(createdBy.Int64)
		sk.CreatedBy = &v
	}
	_ = json.Unmarshal([]byte(toolsJSON), &sk.ToolsAllowed)
	_ = json.Unmarshal([]byte(scopeJSON), &sk.WikiScope)
	return sk, nil
}

func scanSkillRows(rows *sql.Rows) (*chat.Skill, error) {
	sk := &chat.Skill{}
	var toolsJSON, scopeJSON, effort string
	var enabled, adminOnly int
	var createdBy sql.NullInt64
	err := rows.Scan(&sk.ID, &sk.Slug, &sk.Name, &sk.Description, &sk.SystemPrompt, &sk.DefaultModel,
		&effort, &toolsJSON, &scopeJSON, &enabled, &adminOnly, &createdBy, &sk.UpdatedAt)
	if err != nil {
		return nil, err
	}
	sk.DefaultReasoningEffort = effort
	sk.Enabled = enabled != 0
	sk.AdminOnly = adminOnly != 0
	if createdBy.Valid {
		v := int(createdBy.Int64)
		sk.CreatedBy = &v
	}
	_ = json.Unmarshal([]byte(toolsJSON), &sk.ToolsAllowed)
	_ = json.Unmarshal([]byte(scopeJSON), &sk.WikiScope)
	return sk, nil
}

func nullableInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

// buildFtsQuery takes a user string and returns an FTS5 MATCH expression.
// - 단어 단위로 쪼개서 OR 로 연결
// - 각 단어를 " 로 감싸서 특수문자 이스케이프 (FTS5 "double-quoted phrase")
// - 너무 짧은(1자) 토큰은 제거
// - 2자 이상 한글/영문/숫자만 유지
func buildFtsQuery(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// split by whitespace
	parts := strings.Fields(raw)
	tokens := make([]string, 0, len(parts))
	for _, p := range parts {
		// FTS5 에서 " 는 " "" " 로 이스케이프. 하지만 사용자 쿼리에 " 거의 없을 것이라 단순화.
		cleaned := strings.ReplaceAll(p, `"`, ``)
		if len([]rune(cleaned)) < 2 {
			continue
		}
		tokens = append(tokens, `"`+cleaned+`"`)
	}
	if len(tokens) == 0 {
		return ""
	}
	return strings.Join(tokens, " OR ")
}

// globToLike converts simple glob (prefix*) to SQL LIKE pattern.
func globToLike(glob string) string {
	g := strings.ReplaceAll(glob, "%", `\%`)
	g = strings.ReplaceAll(g, "_", `\_`)
	g = strings.ReplaceAll(g, "*", "%")
	return g
}
