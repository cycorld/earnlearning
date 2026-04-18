package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/earnlearning/backend/internal/domain/llm"
)

type LLMRepo struct {
	db *sql.DB
}

func NewLLMRepo(db *sql.DB) *LLMRepo {
	return &LLMRepo{db: db}
}

// UpsertKey — user_id 에 기존 활성 키가 있으면 교체, 없으면 insert.
// 과거 키는 revoke 처리 전제(호출 측 책임).
func (r *LLMRepo) UpsertKey(k *llm.UserKey) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO llm_api_keys (user_id, proxy_student_id, proxy_key_id, prefix, label, issued_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		k.UserID, k.ProxyStudentID, k.ProxyKeyID, k.Prefix, k.Label, k.IssuedAt)
	if err != nil {
		return 0, fmt.Errorf("insert llm key: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	k.ID = int(id)
	return k.ID, nil
}

func (r *LLMRepo) FindActiveKeyByUserID(userID int) (*llm.UserKey, error) {
	row := r.db.QueryRow(`
		SELECT id, user_id, proxy_student_id, proxy_key_id, prefix, label, issued_at, revoked_at
		FROM llm_api_keys
		WHERE user_id = ? AND revoked_at IS NULL
		ORDER BY id DESC LIMIT 1`, userID)
	return scanKey(row)
}

func (r *LLMRepo) FindKeyByProxyKeyID(proxyKeyID int) (*llm.UserKey, error) {
	row := r.db.QueryRow(`
		SELECT id, user_id, proxy_student_id, proxy_key_id, prefix, label, issued_at, revoked_at
		FROM llm_api_keys WHERE proxy_key_id = ?`, proxyKeyID)
	return scanKey(row)
}

func (r *LLMRepo) MarkKeyRevoked(id int, revokedAt time.Time) error {
	_, err := r.db.Exec(`UPDATE llm_api_keys SET revoked_at = ? WHERE id = ?`, revokedAt, id)
	return err
}

func (r *LLMRepo) ListAllActiveKeys() ([]*llm.UserKey, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, proxy_student_id, proxy_key_id, prefix, label, issued_at, revoked_at
		FROM llm_api_keys
		WHERE revoked_at IS NULL
		ORDER BY user_id`)
	if err != nil {
		return nil, fmt.Errorf("list active keys: %w", err)
	}
	defer rows.Close()

	var out []*llm.UserKey
	for rows.Next() {
		k, err := scanKeyRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (r *LLMRepo) UpsertDailyUsage(u *llm.DailyUsage) error {
	date := u.UsageDate.Format("2006-01-02")
	_, err := r.db.Exec(`
		INSERT INTO llm_daily_usage
			(user_id, usage_date, prompt_tokens, completion_tokens, cache_hits, cache_tokens,
			 requests, cost_krw, debited_krw, debt_krw, billed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, usage_date) DO UPDATE SET
			prompt_tokens = excluded.prompt_tokens,
			completion_tokens = excluded.completion_tokens,
			cache_hits = excluded.cache_hits,
			cache_tokens = excluded.cache_tokens,
			requests = excluded.requests,
			cost_krw = excluded.cost_krw,
			debited_krw = excluded.debited_krw,
			debt_krw = excluded.debt_krw,
			billed_at = excluded.billed_at`,
		u.UserID, date, u.PromptTokens, u.CompletionTokens, u.CacheHits, u.CacheTokens,
		u.Requests, u.CostKRW, u.DebitedKRW, u.DebtKRW, u.BilledAt)
	if err != nil {
		return fmt.Errorf("upsert daily usage: %w", err)
	}
	return nil
}

func (r *LLMRepo) FindDailyUsage(userID int, usageDate time.Time) (*llm.DailyUsage, error) {
	date := usageDate.Format("2006-01-02")
	row := r.db.QueryRow(`
		SELECT id, user_id, usage_date, prompt_tokens, completion_tokens, cache_hits,
			   cache_tokens, requests, cost_krw, debited_krw, debt_krw, billed_at
		FROM llm_daily_usage WHERE user_id = ? AND usage_date = ?`, userID, date)
	return scanDaily(row)
}

func (r *LLMRepo) ListDailyUsage(userID int, days int) ([]*llm.DailyUsage, error) {
	if days <= 0 {
		days = 30
	}
	rows, err := r.db.Query(`
		SELECT id, user_id, usage_date, prompt_tokens, completion_tokens, cache_hits,
			   cache_tokens, requests, cost_krw, debited_krw, debt_krw, billed_at
		FROM llm_daily_usage
		WHERE user_id = ?
		ORDER BY usage_date DESC LIMIT ?`, userID, days)
	if err != nil {
		return nil, fmt.Errorf("list daily usage: %w", err)
	}
	defer rows.Close()

	var out []*llm.DailyUsage
	for rows.Next() {
		u, err := scanDailyRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *LLMRepo) SumUsageSince(userID int, since time.Time) (int, int, error) {
	sinceStr := since.Format("2006-01-02")
	var cost, debt sql.NullInt64
	err := r.db.QueryRow(`
		SELECT COALESCE(SUM(cost_krw), 0), COALESCE(SUM(debt_krw), 0)
		FROM llm_daily_usage
		WHERE user_id = ? AND usage_date >= ?`, userID, sinceStr).Scan(&cost, &debt)
	if err != nil {
		return 0, 0, err
	}
	return int(cost.Int64), int(debt.Int64), nil
}

func (r *LLMRepo) SumUsageAllTime(userID int) (int, int, error) {
	var cost, debt sql.NullInt64
	err := r.db.QueryRow(`
		SELECT COALESCE(SUM(cost_krw), 0), COALESCE(SUM(debt_krw), 0)
		FROM llm_daily_usage WHERE user_id = ?`, userID).Scan(&cost, &debt)
	if err != nil {
		return 0, 0, err
	}
	return int(cost.Int64), int(debt.Int64), nil
}

// --- helpers ---

type scanner interface {
	Scan(...any) error
}

// parseUsageDate 는 DB 가 돌려주는 DATE 값을 KST 달력 일자로 변환한다.
// SQLite go-sqlite3 드라이버는 DATETIME/DATE 컬럼을 자동으로 time.Time 포맷
// ("2006-01-02T15:04:05Z") 로 직렬화해서 반환하기도 하고, 우리가 INSERT 에 쓴
// "2006-01-02" 형태를 그대로 주기도 한다. 둘 다 수용.
func parseUsageDate(raw string) (time.Time, error) {
	// 길이 10: "2006-01-02"
	if len(raw) == 10 {
		return time.ParseInLocation("2006-01-02", raw, llm.KST)
	}
	// 그 외: RFC3339 류 → 파싱 후 KST 자정으로 정규화
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		// fallback: UTC 기준 space-separated format
		t, err = time.Parse("2006-01-02 15:04:05", raw)
		if err != nil {
			return time.Time{}, err
		}
	}
	kst := t.In(llm.KST)
	return time.Date(kst.Year(), kst.Month(), kst.Day(), 0, 0, 0, 0, llm.KST), nil
}

func scanKey(s scanner) (*llm.UserKey, error) {
	k := &llm.UserKey{}
	var revoked sql.NullTime
	err := s.Scan(&k.ID, &k.UserID, &k.ProxyStudentID, &k.ProxyKeyID, &k.Prefix, &k.Label,
		&k.IssuedAt, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, llm.ErrKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan llm key: %w", err)
	}
	if revoked.Valid {
		t := revoked.Time
		k.RevokedAt = &t
	}
	return k, nil
}

func scanKeyRows(rows *sql.Rows) (*llm.UserKey, error) {
	k := &llm.UserKey{}
	var revoked sql.NullTime
	err := rows.Scan(&k.ID, &k.UserID, &k.ProxyStudentID, &k.ProxyKeyID, &k.Prefix, &k.Label,
		&k.IssuedAt, &revoked)
	if err != nil {
		return nil, err
	}
	if revoked.Valid {
		t := revoked.Time
		k.RevokedAt = &t
	}
	return k, nil
}

func scanDaily(s scanner) (*llm.DailyUsage, error) {
	u := &llm.DailyUsage{}
	var date string
	err := s.Scan(&u.ID, &u.UserID, &date, &u.PromptTokens, &u.CompletionTokens, &u.CacheHits,
		&u.CacheTokens, &u.Requests, &u.CostKRW, &u.DebitedKRW, &u.DebtKRW, &u.BilledAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	parsed, err := parseUsageDate(date)
	if err != nil {
		return nil, fmt.Errorf("parse usage_date: %w", err)
	}
	u.UsageDate = parsed
	return u, nil
}

func scanDailyRows(rows *sql.Rows) (*llm.DailyUsage, error) {
	u := &llm.DailyUsage{}
	var date string
	err := rows.Scan(&u.ID, &u.UserID, &date, &u.PromptTokens, &u.CompletionTokens, &u.CacheHits,
		&u.CacheTokens, &u.Requests, &u.CostKRW, &u.DebitedKRW, &u.DebtKRW, &u.BilledAt)
	if err != nil {
		return nil, err
	}
	parsed, err := parseUsageDate(date)
	if err != nil {
		return nil, fmt.Errorf("parse usage_date: %w", err)
	}
	u.UsageDate = parsed
	return u, nil
}
