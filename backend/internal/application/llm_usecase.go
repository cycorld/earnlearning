package application

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/earnlearning/backend/internal/domain/llm"
	"github.com/earnlearning/backend/internal/domain/notification"
	"github.com/earnlearning/backend/internal/domain/user"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

// ProxyClient 는 llm.cycorld.com admin API 의 최소 추상화.
// 테스트에서는 fake 주입.
type ProxyClient interface {
	CreateStudent(ctx context.Context, name, affiliation, email, note string) (id int, err error)
	FindStudentByEmail(ctx context.Context, email string) (id int, found bool, err error)
	IssueKey(ctx context.Context, studentID int, label string) (plaintext, prefix string, keyID int, err error)
	RevokeKey(ctx context.Context, keyID int) error
	Usage(ctx context.Context, days int) (map[int]ProxyUsage, error) // keyed by proxy student id
	Status(ctx context.Context) (*ProxyStatus, error)
}

// ProxyStatus 는 학생에게 보여줄 sanitized 상태. PID / 로그 디렉토리 / DB 카운트 같은
// 내부 정보는 포함하지 않는다.
type ProxyStatus struct {
	Service         string  `json:"service"`          // llm-proxy
	Version         string  `json:"version"`
	UptimeSeconds   int64   `json:"uptime_seconds"`
	Upstream        string  `json:"upstream_status"`  // ok / down / http_500 / timeout
	Model           string  `json:"model"`            // 파일명만 (경로 제거)
	LatencyMs       float64 `json:"latency_ms,omitempty"`
	ContextWindow   int     `json:"context_window,omitempty"`
	SlotsTotal      int     `json:"slots_total,omitempty"`
	SlotsIdle       int     `json:"slots_idle,omitempty"`
	SlotsProcessing int     `json:"slots_processing,omitempty"`
}

// ProxyUsage 는 llmproxy.UsageBucket 의 의미만 추출한 스냅샷.
type ProxyUsage struct {
	Requests         int
	PromptTokens     int
	CompletionTokens int
	CacheHits        int
	CacheTokens      int // prompt 중 캐시 재사용분 (없으면 0)
	Errors           int
}

type LLMUseCase struct {
	repo       llm.Repository
	userRepo   user.Repository
	walletRepo wallet.Repository
	proxy      ProxyClient
	notifUC    *NotificationUseCase
	affiliation string // llm-proxy 에 등록할 학생 소속 (예: "이화여대")
}

func NewLLMUseCase(repo llm.Repository, userRepo user.Repository, walletRepo wallet.Repository,
	proxy ProxyClient, notifUC *NotificationUseCase, affiliation string) *LLMUseCase {
	if affiliation == "" {
		affiliation = "이화여대"
	}
	return &LLMUseCase{
		repo:        repo,
		userRepo:    userRepo,
		walletRepo:  walletRepo,
		proxy:       proxy,
		notifUC:     notifUC,
		affiliation: affiliation,
	}
}

// EnsureKey 는 학생의 활성 키를 보장한다.
//   - 이미 있으면 그대로 반환 (plaintext 는 없음)
//   - 없으면 proxy 에 student 생성/조회 → key 발급 → DB 저장 후 반환 (plaintext 포함)
//
// 자동 발급 정책이므로 호출자(HTTP handler 등)는 이 메서드 하나로 "키 보기" UX 를
// 완결할 수 있다.
func (uc *LLMUseCase) EnsureKey(ctx context.Context, userID int) (*llm.UserKey, error) {
	existing, err := uc.repo.FindActiveKeyByUserID(userID)
	if err != nil && !errors.Is(err, llm.ErrKeyNotFound) {
		return nil, err
	}
	if existing != nil {
		// plaintext 는 발급 직후 단 1회만 노출되므로, 재조회 시엔 반드시 비운다.
		existing.Plaintext = ""
		return existing, nil
	}

	u, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	if u.Email == "" {
		return nil, llm.ErrNoEmail
	}

	studentID, err := uc.ensureProxyStudent(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("ensure proxy student: %w", err)
	}

	label := fmt.Sprintf("%s (LMS user_id=%d)", u.Name, u.ID)
	plaintext, prefix, keyID, err := uc.proxy.IssueKey(ctx, studentID, label)
	if err != nil {
		return nil, fmt.Errorf("issue key: %w", err)
	}

	k := &llm.UserKey{
		UserID:         userID,
		ProxyStudentID: studentID,
		ProxyKeyID:     keyID,
		Prefix:         prefix,
		Label:          label,
		IssuedAt:       time.Now(),
		Plaintext:      plaintext,
	}
	if _, err := uc.repo.UpsertKey(k); err != nil {
		return nil, err
	}
	return k, nil
}

// RotateKey 는 기존 키를 revoke 하고 새 키를 발급한다.
func (uc *LLMUseCase) RotateKey(ctx context.Context, userID int) (*llm.UserKey, error) {
	existing, err := uc.repo.FindActiveKeyByUserID(userID)
	if err != nil && !errors.Is(err, llm.ErrKeyNotFound) {
		return nil, err
	}
	if existing != nil {
		if err := uc.proxy.RevokeKey(ctx, existing.ProxyKeyID); err != nil {
			// proxy 에서 이미 폐기됐거나 404 일 수 있음 → 경고만 기록하고 계속
			log.Printf("[llm] revoke key %d: %v", existing.ProxyKeyID, err)
		}
		if err := uc.repo.MarkKeyRevoked(existing.ID, time.Now()); err != nil {
			return nil, err
		}
	}
	return uc.EnsureKey(ctx, userID)
}

// GetKey 는 활성 키 메타를 반환 (plaintext 없음). 없으면 ErrKeyNotFound.
func (uc *LLMUseCase) GetKey(userID int) (*llm.UserKey, error) {
	return uc.repo.FindActiveKeyByUserID(userID)
}

// Status 는 llm-proxy 의 상태를 조회해 학생에게 보여줄 수준으로 반환.
func (uc *LLMUseCase) Status(ctx context.Context) (*ProxyStatus, error) {
	return uc.proxy.Status(ctx)
}

// ListDailyUsage 는 최근 N일 일별 사용량 레코드.
func (uc *LLMUseCase) ListDailyUsage(userID int, days int) ([]*llm.DailyUsage, error) {
	return uc.repo.ListDailyUsage(userID, days)
}

// Summary 는 누적 / 최근 7일 청구액 요약.
func (uc *LLMUseCase) Summary(userID int) (*llm.Summary, error) {
	totalCost, totalDebt, err := uc.repo.SumUsageAllTime(userID)
	if err != nil {
		return nil, err
	}
	since := time.Now().In(llm.KST).AddDate(0, 0, -7)
	weekCost, _, err := uc.repo.SumUsageSince(userID, since)
	if err != nil {
		return nil, err
	}
	return &llm.Summary{
		CumulativeCostKRW: totalCost,
		CumulativeDebtKRW: totalDebt,
		LastWeekCostKRW:   weekCost,
	}, nil
}

// BillAll 은 KST 03:33 크론에서 호출. 전체 활성 키에 대해:
//  1. proxy 에서 최근 days=1 사용량 가져오기
//  2. 학생별 원화 비용 계산
//  3. 지갑에서 차감 가능분만 차감 + 부족분을 debt 로 기록
//  4. daily_usage 테이블에 upsert (재실행 안전)
//  5. 알림 전송
//
// billingDate 는 과금 대상 달력 일자 (크론 호출 시점 기준 "전날" KST).
// 반환값: 처리된 유저 수, 에러 (개별 실패는 집계만 하고 계속 진행).
func (uc *LLMUseCase) BillAll(ctx context.Context, billingDate time.Time) (int, error) {
	keys, err := uc.repo.ListAllActiveKeys()
	if err != nil {
		return 0, fmt.Errorf("list active keys: %w", err)
	}
	if len(keys) == 0 {
		return 0, nil
	}

	usage, err := uc.proxy.Usage(ctx, 1)
	if err != nil {
		return 0, fmt.Errorf("proxy usage: %w", err)
	}

	processed := 0
	var errs []error
	for _, k := range keys {
		bucket := usage[k.ProxyStudentID]
		if bucket.Requests == 0 && bucket.PromptTokens == 0 && bucket.CompletionTokens == 0 {
			// 사용 없음 — zero-row 기록도 스킵 (DB 깨끗하게 유지)
			continue
		}
		if err := uc.billOne(k, bucket, billingDate); err != nil {
			errs = append(errs, fmt.Errorf("user %d: %w", k.UserID, err))
			continue
		}
		processed++
	}
	if len(errs) > 0 {
		return processed, fmt.Errorf("%d errors: %v", len(errs), errs)
	}
	return processed, nil
}

func (uc *LLMUseCase) billOne(k *llm.UserKey, bucket ProxyUsage, billingDate time.Time) error {
	cost := llm.CostKRW(bucket.PromptTokens, bucket.CompletionTokens, bucket.CacheTokens)

	w, err := uc.walletRepo.FindByUserID(k.UserID)
	if err != nil {
		return fmt.Errorf("find wallet: %w", err)
	}

	// 지갑 잔액까지만 차감, 부족분은 debt 로 기록
	debit := cost
	if debit > w.Balance {
		debit = w.Balance
	}
	if debit < 0 {
		debit = 0
	}
	debt := cost - debit

	if debit > 0 {
		desc := fmt.Sprintf("LLM 사용료 %s (%d tok in / %d tok out)",
			billingDate.Format("2006-01-02"), bucket.PromptTokens, bucket.CompletionTokens)
		if err := uc.walletRepo.Debit(w.ID, debit, wallet.TxLLMBilling, desc, "llm_billing", k.ID); err != nil {
			return fmt.Errorf("debit wallet: %w", err)
		}
	}

	record := &llm.DailyUsage{
		UserID:           k.UserID,
		UsageDate:        billingDate,
		PromptTokens:     bucket.PromptTokens,
		CompletionTokens: bucket.CompletionTokens,
		CacheHits:        bucket.CacheHits,
		CacheTokens:      bucket.CacheTokens,
		Requests:         bucket.Requests,
		CostKRW:          cost,
		DebitedKRW:       debit,
		DebtKRW:          debt,
		BilledAt:         time.Now(),
	}
	if err := uc.repo.UpsertDailyUsage(record); err != nil {
		return fmt.Errorf("upsert usage: %w", err)
	}

	// 알림 — 부채 여부에 따라 메시지 분기
	if uc.notifUC != nil {
		title := fmt.Sprintf("LLM 사용료 %d원 차감", debit)
		body := fmt.Sprintf("%s 사용량: %d tok in / %d tok out → 총 %d원",
			billingDate.Format("2006-01-02"), bucket.PromptTokens, bucket.CompletionTokens, cost)
		if debt > 0 {
			title = fmt.Sprintf("LLM 사용료 %d원 차감 (부채 %d원)", debit, debt)
			body += fmt.Sprintf(" — 잔액 부족으로 %d원 부채로 기록. 지갑 충전 후 다음 과금 주기에 우선 차감됩니다.", debt)
		}
		_ = uc.notifUC.CreateNotification(k.UserID, notification.NotifLLMBilled, title, body, "wallet", 0)
	}
	return nil
}

// ensureProxyStudent — proxy 에 이미 같은 이메일로 등록됐는지 확인 후, 없으면 생성.
// 학교 이메일 재가입 등으로 email 중복이 발생하면 422 가 올 수 있어서 우선 조회 후 생성.
func (uc *LLMUseCase) ensureProxyStudent(ctx context.Context, u *user.User) (int, error) {
	id, found, err := uc.proxy.FindStudentByEmail(ctx, u.Email)
	if err != nil {
		return 0, err
	}
	if found {
		return id, nil
	}
	name := u.Name
	if name == "" {
		name = strings.Split(u.Email, "@")[0]
	}
	note := fmt.Sprintf("LMS user_id=%d, department=%s, student_id=%s", u.ID, u.Department, u.StudentID)
	return uc.proxy.CreateStudent(ctx, name, uc.affiliation, u.Email, note)
}
