package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/earnlearning/backend/internal/domain/llm"
	"github.com/earnlearning/backend/internal/domain/user"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

// --- fakes ---

type fakeProxy struct {
	students     map[string]int // email → student id
	nextID       int
	issuedKey    string
	issuedPrefix string
	issuedID     int
	revoked      []int
	usage        map[int]ProxyUsage

	issueErr error
	findErr  error
}

func newFakeProxy() *fakeProxy {
	return &fakeProxy{students: map[string]int{}, nextID: 100}
}

func (f *fakeProxy) CreateStudent(_ context.Context, _, _, email, _ string) (int, error) {
	f.nextID++
	f.students[email] = f.nextID
	return f.nextID, nil
}
func (f *fakeProxy) FindStudentByEmail(_ context.Context, email string) (int, bool, error) {
	if f.findErr != nil {
		return 0, false, f.findErr
	}
	id, ok := f.students[email]
	return id, ok, nil
}
func (f *fakeProxy) IssueKey(_ context.Context, _ int, _ string) (string, string, int, error) {
	if f.issueErr != nil {
		return "", "", 0, f.issueErr
	}
	f.issuedID++
	key := f.issuedKey
	if key == "" {
		key = "sk-fake-plaintext"
	}
	prefix := f.issuedPrefix
	if prefix == "" {
		prefix = "sk-fa"
	}
	return key, prefix, f.issuedID, nil
}
func (f *fakeProxy) RevokeKey(_ context.Context, keyID int) error {
	f.revoked = append(f.revoked, keyID)
	return nil
}
func (f *fakeProxy) Usage(_ context.Context, _ int) (map[int]ProxyUsage, error) {
	if f.usage == nil {
		return map[int]ProxyUsage{}, nil
	}
	return f.usage, nil
}

type fakeLLMRepo struct {
	keys   []*llm.UserKey
	daily  map[string]*llm.DailyUsage
	nextID int
}

func newFakeLLMRepo() *fakeLLMRepo {
	return &fakeLLMRepo{daily: map[string]*llm.DailyUsage{}}
}

func dailyKey(userID int, d time.Time) string {
	return d.Format("2006-01-02") + "|" + itoa(userID)
}
func itoa(i int) string { return time.Now().Format("") + fmtInt(i) }
func fmtInt(i int) string {
	if i == 0 {
		return "0"
	}
	var out []byte
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	for i > 0 {
		out = append([]byte{byte('0' + i%10)}, out...)
		i /= 10
	}
	if neg {
		out = append([]byte{'-'}, out...)
	}
	return string(out)
}

func (r *fakeLLMRepo) UpsertKey(k *llm.UserKey) (int, error) {
	r.nextID++
	k.ID = r.nextID
	r.keys = append(r.keys, k)
	return k.ID, nil
}
func (r *fakeLLMRepo) FindActiveKeyByUserID(userID int) (*llm.UserKey, error) {
	for i := len(r.keys) - 1; i >= 0; i-- {
		k := r.keys[i]
		if k.UserID == userID && k.RevokedAt == nil {
			return k, nil
		}
	}
	return nil, llm.ErrKeyNotFound
}
func (r *fakeLLMRepo) FindKeyByProxyKeyID(id int) (*llm.UserKey, error) {
	for _, k := range r.keys {
		if k.ProxyKeyID == id {
			return k, nil
		}
	}
	return nil, llm.ErrKeyNotFound
}
func (r *fakeLLMRepo) MarkKeyRevoked(id int, at time.Time) error {
	for _, k := range r.keys {
		if k.ID == id {
			t := at
			k.RevokedAt = &t
			return nil
		}
	}
	return llm.ErrKeyNotFound
}
func (r *fakeLLMRepo) ListAllActiveKeys() ([]*llm.UserKey, error) {
	var out []*llm.UserKey
	for _, k := range r.keys {
		if k.RevokedAt == nil {
			out = append(out, k)
		}
	}
	return out, nil
}
func (r *fakeLLMRepo) UpsertDailyUsage(u *llm.DailyUsage) error {
	r.daily[dailyKey(u.UserID, u.UsageDate)] = u
	return nil
}
func (r *fakeLLMRepo) FindDailyUsage(userID int, date time.Time) (*llm.DailyUsage, error) {
	return r.daily[dailyKey(userID, date)], nil
}
func (r *fakeLLMRepo) ListDailyUsage(userID int, days int) ([]*llm.DailyUsage, error) {
	var out []*llm.DailyUsage
	for _, u := range r.daily {
		if u.UserID == userID {
			out = append(out, u)
		}
	}
	return out, nil
}
func (r *fakeLLMRepo) SumUsageSince(userID int, _ time.Time) (int, int, error) {
	var c, d int
	for _, u := range r.daily {
		if u.UserID == userID {
			c += u.CostKRW
			d += u.DebtKRW
		}
	}
	return c, d, nil
}
func (r *fakeLLMRepo) SumUsageAllTime(userID int) (int, int, error) {
	return r.SumUsageSince(userID, time.Time{})
}

// --- minimal user + wallet fakes ---

type fakeUserRepo struct{ u *user.User }

func (r *fakeUserRepo) Create(*user.User) (int, error)                 { return 0, nil }
func (r *fakeUserRepo) FindByID(int) (*user.User, error)               { return r.u, nil }
func (r *fakeUserRepo) FindByEmail(string) (*user.User, error)         { return nil, nil }
func (r *fakeUserRepo) FindByStatus(user.Status) ([]*user.User, error) { return nil, nil }
func (r *fakeUserRepo) ListAll(int, int) ([]*user.User, int, error)    { return nil, 0, nil }
func (r *fakeUserRepo) UpdateStatus(int, user.Status) error            { return nil }
func (r *fakeUserRepo) UpdateAvatarURL(int, string) error              { return nil }
func (r *fakeUserRepo) GetUserActivity(int) (*user.UserActivity, error) {
	return nil, nil
}

type fakeWalletRepo struct {
	balance int
	debits  []struct {
		amount  int
		txType  wallet.TxType
		desc    string
		refType string
		refID   int
	}
}

func (r *fakeWalletRepo) FindByUserID(int) (*wallet.Wallet, error) {
	return &wallet.Wallet{ID: 1, UserID: 1, Balance: r.balance}, nil
}
func (r *fakeWalletRepo) CreateWallet(int) (int, error) { return 0, nil }
func (r *fakeWalletRepo) Credit(int, int, wallet.TxType, string, string, int) error {
	return nil
}
func (r *fakeWalletRepo) Debit(wid, amount int, tx wallet.TxType, desc, refType string, refID int) error {
	if amount > r.balance {
		return wallet.ErrInsufficientFunds
	}
	r.balance -= amount
	r.debits = append(r.debits, struct {
		amount  int
		txType  wallet.TxType
		desc    string
		refType string
		refID   int
	}{amount, tx, desc, refType, refID})
	return nil
}
func (r *fakeWalletRepo) GetTransactions(int, wallet.TransactionFilter, int, int) ([]*wallet.Transaction, int, error) {
	return nil, 0, nil
}
func (r *fakeWalletRepo) GetRanking(int) ([]*wallet.RankEntry, error) {
	return nil, nil
}
func (r *fakeWalletRepo) GetAssetBreakdown(int) (*wallet.AssetBreakdown, error) {
	return nil, nil
}

// --- tests ---

func newUC(wBalance int) (*LLMUseCase, *fakeProxy, *fakeLLMRepo, *fakeWalletRepo) {
	proxy := newFakeProxy()
	repo := newFakeLLMRepo()
	ur := &fakeUserRepo{u: &user.User{ID: 1, Email: "s@ewha.ac.kr", Name: "홍길동", StudentID: "2025001", Department: "컴공"}}
	wr := &fakeWalletRepo{balance: wBalance}
	uc := NewLLMUseCase(repo, ur, wr, proxy, nil, "이화여대")
	return uc, proxy, repo, wr
}

func TestEnsureKey_FirstTime_CreatesStudentAndIssuesKey(t *testing.T) {
	uc, proxy, _, _ := newUC(10_000)
	k, err := uc.EnsureKey(context.Background(), 1)
	if err != nil {
		t.Fatalf("EnsureKey: %v", err)
	}
	if k.Plaintext == "" {
		t.Errorf("plaintext should be set on first issuance")
	}
	if k.ProxyStudentID == 0 || k.ProxyKeyID == 0 {
		t.Errorf("proxy ids should be set: %+v", k)
	}
	if _, ok := proxy.students["s@ewha.ac.kr"]; !ok {
		t.Errorf("proxy CreateStudent not called")
	}
}

func TestEnsureKey_SecondCall_ReturnsExistingWithoutPlaintext(t *testing.T) {
	uc, _, _, _ := newUC(10_000)
	_, _ = uc.EnsureKey(context.Background(), 1)
	k, err := uc.EnsureKey(context.Background(), 1)
	if err != nil {
		t.Fatalf("second EnsureKey: %v", err)
	}
	if k.Plaintext != "" {
		t.Errorf("plaintext must not be returned after first issuance")
	}
}

func TestEnsureKey_NoEmailFails(t *testing.T) {
	proxy := newFakeProxy()
	repo := newFakeLLMRepo()
	ur := &fakeUserRepo{u: &user.User{ID: 1, Email: ""}}
	wr := &fakeWalletRepo{balance: 100}
	uc := NewLLMUseCase(repo, ur, wr, proxy, nil, "이화여대")

	_, err := uc.EnsureKey(context.Background(), 1)
	if !errors.Is(err, llm.ErrNoEmail) {
		t.Fatalf("expected ErrNoEmail, got %v", err)
	}
}

func TestRotateKey_RevokesOldAndIssuesNew(t *testing.T) {
	uc, proxy, _, _ := newUC(10_000)
	first, _ := uc.EnsureKey(context.Background(), 1)
	second, err := uc.RotateKey(context.Background(), 1)
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	if second.ProxyKeyID == first.ProxyKeyID {
		t.Errorf("expected new proxy key id, got same %d", second.ProxyKeyID)
	}
	if len(proxy.revoked) != 1 || proxy.revoked[0] != first.ProxyKeyID {
		t.Errorf("old key should be revoked: %v", proxy.revoked)
	}
}

func TestBillAll_NoUsage_IsNoop(t *testing.T) {
	uc, _, _, wr := newUC(10_000)
	_, _ = uc.EnsureKey(context.Background(), 1)
	n, err := uc.BillAll(context.Background(), time.Date(2026, 4, 17, 0, 0, 0, 0, llm.KST))
	if err != nil {
		t.Fatalf("BillAll: %v", err)
	}
	if n != 0 {
		t.Errorf("processed count should be 0, got %d", n)
	}
	if len(wr.debits) != 0 {
		t.Errorf("no debits expected")
	}
}

func TestBillAll_DebitsWalletAndRecordsUsage(t *testing.T) {
	uc, proxy, repo, wr := newUC(10_000)
	k, _ := uc.EnsureKey(context.Background(), 1)
	proxy.usage = map[int]ProxyUsage{
		k.ProxyStudentID: {Requests: 10, PromptTokens: 10_000, CompletionTokens: 5_000, CacheHits: 0},
	}
	billingDate := time.Date(2026, 4, 17, 0, 0, 0, 0, llm.KST)
	// expected cost: 10k * 15/M + 5k * 75/M = 0.15 + 0.375 = 0.525 USD × 1400 = 735원
	n, err := uc.BillAll(context.Background(), billingDate)
	if err != nil {
		t.Fatalf("BillAll: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 processed, got %d", n)
	}
	if len(wr.debits) != 1 {
		t.Fatalf("expected 1 debit, got %d", len(wr.debits))
	}
	if wr.debits[0].txType != wallet.TxLLMBilling {
		t.Errorf("tx type: got %s", wr.debits[0].txType)
	}
	// usage recorded
	got, _ := repo.FindDailyUsage(1, billingDate)
	if got == nil || got.CostKRW < 700 || got.CostKRW > 770 {
		t.Errorf("usage row: %+v", got)
	}
	if got.DebtKRW != 0 {
		t.Errorf("should have no debt when wallet is plenty")
	}
}

func TestBillAll_InsufficientBalance_RecordsDebt(t *testing.T) {
	uc, proxy, repo, wr := newUC(100) // 잔액 100원만
	k, _ := uc.EnsureKey(context.Background(), 1)
	proxy.usage = map[int]ProxyUsage{
		k.ProxyStudentID: {Requests: 10, PromptTokens: 10_000, CompletionTokens: 5_000},
	}
	billingDate := time.Date(2026, 4, 17, 0, 0, 0, 0, llm.KST)
	_, err := uc.BillAll(context.Background(), billingDate)
	if err != nil {
		t.Fatalf("BillAll: %v", err)
	}

	got, _ := repo.FindDailyUsage(1, billingDate)
	if got == nil {
		t.Fatalf("daily usage missing")
	}
	if got.DebitedKRW != 100 {
		t.Errorf("debited: got %d, want 100", got.DebitedKRW)
	}
	if got.DebtKRW < 600 {
		t.Errorf("debt should be ~635, got %d", got.DebtKRW)
	}
	if wr.balance != 0 {
		t.Errorf("wallet should be drained to 0, got %d", wr.balance)
	}
}

func TestBillAll_ZeroBalance_NoDebitOnlyDebt(t *testing.T) {
	uc, proxy, repo, wr := newUC(0)
	k, _ := uc.EnsureKey(context.Background(), 1)
	proxy.usage = map[int]ProxyUsage{
		k.ProxyStudentID: {Requests: 5, PromptTokens: 1000, CompletionTokens: 500},
	}
	billingDate := time.Date(2026, 4, 17, 0, 0, 0, 0, llm.KST)
	_, err := uc.BillAll(context.Background(), billingDate)
	if err != nil {
		t.Fatalf("BillAll: %v", err)
	}
	if len(wr.debits) != 0 {
		t.Errorf("no debit should happen with zero balance")
	}
	got, _ := repo.FindDailyUsage(1, billingDate)
	if got.DebitedKRW != 0 || got.DebtKRW == 0 {
		t.Errorf("expected zero debit + positive debt: %+v", got)
	}
}

func TestBillAll_Idempotent_SameDayUpsertOverwrites(t *testing.T) {
	uc, proxy, repo, _ := newUC(100_000)
	k, _ := uc.EnsureKey(context.Background(), 1)
	proxy.usage = map[int]ProxyUsage{
		k.ProxyStudentID: {Requests: 5, PromptTokens: 1000, CompletionTokens: 500},
	}
	billingDate := time.Date(2026, 4, 17, 0, 0, 0, 0, llm.KST)
	_, _ = uc.BillAll(context.Background(), billingDate)
	_, _ = uc.BillAll(context.Background(), billingDate)

	// fake repo returns by key; only 1 entry per day
	all, _ := repo.ListDailyUsage(1, 30)
	if len(all) != 1 {
		t.Errorf("upsert should keep 1 row per day, got %d", len(all))
	}
}
