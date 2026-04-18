package llm

import "time"

// UserKey 는 LMS 사용자와 llm-proxy 상의 student/key id 매핑 레코드.
// `Plaintext` 는 발급 직후 메모리에만 존재; DB 에는 저장하지 않는다.
type UserKey struct {
	ID             int        `json:"id"`
	UserID         int        `json:"-"`
	ProxyStudentID int        `json:"proxy_student_id"`
	ProxyKeyID     int        `json:"proxy_key_id"`
	Prefix         string     `json:"prefix"`
	Label          string     `json:"label"`
	IssuedAt       time.Time  `json:"issued_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`

	// 평문 키는 발급 직후 1회에 한해서만 세팅된다. DB 에는 저장되지 않고,
	// HTTP 응답으로 딱 한 번 노출된다. (이후 조회에선 항상 빈 문자열.)
	Plaintext string `json:"plaintext,omitempty"`
}

// DailyUsage 는 하루치 사용량 + 과금 결과 스냅샷.
// 재실행에 안전하도록 UNIQUE(user_id, usage_date).
type DailyUsage struct {
	ID               int       `json:"id"`
	UserID           int       `json:"-"`
	UsageDate        time.Time `json:"usage_date"` // KST 기준 달력 일자
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	CacheHits        int       `json:"cache_hits"`   // 캐시가 적중한 요청 수 (표시용)
	CacheTokens      int       `json:"cache_tokens"` // 캐시에서 재사용된 prompt 토큰 수 (과금용)
	Requests         int       `json:"requests"`
	CostKRW          int       `json:"cost_krw"`      // 계산된 총 비용
	DebitedKRW       int       `json:"debited_krw"`   // 실제 지갑에서 차감된 금액
	DebtKRW          int       `json:"debt_krw"`      // 잔액 부족으로 미차감된 부채
	BilledAt         time.Time `json:"billed_at"`
}

// Summary 는 /llm 페이지에 한눈에 보여줄 누적치 + 최근 7일 합.
type Summary struct {
	CumulativeCostKRW int `json:"cumulative_cost_krw"`
	CumulativeDebtKRW int `json:"cumulative_debt_krw"`
	LastWeekCostKRW   int `json:"last_week_cost_krw"`
}
