package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/llm"
)

// fakeLLMProxy — 통합 테스트용 in-memory llm-proxy.
// setupTestServer 안에서 주입되므로 외부 네트워크 호출은 없음.
type fakeLLMProxy struct {
	mu        sync.Mutex
	students  map[string]int           // email → student id
	usage     map[int]application.ProxyUsage // proxy student id → bucket
	nextStu   int
	nextKeyID int
	revoked   []int
}

func newFakeLLMProxy() *fakeLLMProxy {
	return &fakeLLMProxy{
		students: map[string]int{},
		usage:    map[int]application.ProxyUsage{},
		nextStu:  100,
	}
}

func (f *fakeLLMProxy) CreateStudent(_ context.Context, _, _, email, _ string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextStu++
	f.students[email] = f.nextStu
	return f.nextStu, nil
}
func (f *fakeLLMProxy) FindStudentByEmail(_ context.Context, email string) (int, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.students[email]
	return id, ok, nil
}
func (f *fakeLLMProxy) IssueKey(_ context.Context, studentID int, _ string) (string, string, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextKeyID++
	plaintext := fmt.Sprintf("sk-fake-%d-%d", studentID, f.nextKeyID)
	return plaintext, plaintext[:8], f.nextKeyID, nil
}
func (f *fakeLLMProxy) RevokeKey(_ context.Context, keyID int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.revoked = append(f.revoked, keyID)
	return nil
}
func (f *fakeLLMProxy) Usage(_ context.Context, _ int) (map[int]application.ProxyUsage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[int]application.ProxyUsage, len(f.usage))
	for k, v := range f.usage {
		out[k] = v
	}
	return out, nil
}
func (f *fakeLLMProxy) Status(_ context.Context) (*application.ProxyStatus, error) {
	return &application.ProxyStatus{
		Service:       "llm-proxy",
		Version:       "test-0.0.1",
		UptimeSeconds: 3600,
		Upstream:      "ok",
		Model:         "Qwen3.6-35B.gguf",
		LatencyMs:     1.2,
		SlotsTotal:    4,
		SlotsIdle:     3,
	}, nil
}
func (f *fakeLLMProxy) setUsage(studentID int, bucket application.ProxyUsage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.usage[studentID] = bucket
}

// --- tests ---

func TestLLM_GetMyKey_AutoProvisionsOnFirstCall(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("llm1@ewha.ac.kr", "pass1234", "LLM테스트1", "2024001")

	r := ts.get("/api/llm/me", token)
	if !r.Success {
		t.Fatalf("GET /llm/me failed: %v", r.Error)
	}
	var k map[string]any
	if err := json.Unmarshal(r.Data, &k); err != nil {
		t.Fatalf("parse key: %v", err)
	}
	if k["plaintext"] == nil || k["plaintext"] == "" {
		t.Errorf("plaintext should be present on first call, got: %+v", k)
	}
	if k["proxy_key_id"].(float64) <= 0 {
		t.Errorf("proxy_key_id should be > 0, got %v", k["proxy_key_id"])
	}
}

func TestLLM_GetMyKey_SecondCallOmitsPlaintext(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("llm2@ewha.ac.kr", "pass1234", "LLM테스트2", "2024002")

	_ = ts.get("/api/llm/me", token) // first call creates key
	r := ts.get("/api/llm/me", token)
	if !r.Success {
		t.Fatalf("second GET failed: %v", r.Error)
	}
	var k map[string]any
	_ = json.Unmarshal(r.Data, &k)
	// plaintext 는 아예 없거나 ""
	if v, ok := k["plaintext"]; ok && v != "" {
		t.Errorf("plaintext leaked on second call: %v", v)
	}
}

func TestLLM_RotateMyKey_IssuesNewKeyAndRevokesOld(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("llm3@ewha.ac.kr", "pass1234", "LLM테스트3", "2024003")

	r1 := ts.get("/api/llm/me", token)
	var k1 map[string]any
	_ = json.Unmarshal(r1.Data, &k1)
	oldKeyID := int(k1["proxy_key_id"].(float64))

	r2 := ts.post("/api/llm/me/rotate", nil, token)
	if !r2.Success {
		t.Fatalf("rotate failed: %v", r2.Error)
	}
	var k2 map[string]any
	_ = json.Unmarshal(r2.Data, &k2)
	newKeyID := int(k2["proxy_key_id"].(float64))

	if newKeyID == oldKeyID {
		t.Errorf("rotate should produce new proxy_key_id")
	}
	if k2["plaintext"] == nil || k2["plaintext"] == "" {
		t.Errorf("rotate should return new plaintext")
	}
	// 기존 proxy key 는 revoke 되어야 함
	foundRevoked := false
	for _, id := range ts.llmProxy.revoked {
		if id == oldKeyID {
			foundRevoked = true
		}
	}
	if !foundRevoked {
		t.Errorf("old key %d should have been revoked: %v", oldKeyID, ts.llmProxy.revoked)
	}
}

func TestLLM_BillAll_DebitsWalletAndRecordsUsage(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("llm4@ewha.ac.kr", "pass1234", "LLM테스트4", "2024004")

	// 지갑에 예상 비용보다 충분한 잔액 직접 충전 (관리자 송금 로직 bypass)
	if _, err := ts.db.Exec(
		`UPDATE wallets SET balance = 100000 WHERE user_id = (SELECT id FROM users WHERE email = ?)`,
		"llm4@ewha.ac.kr"); err != nil {
		t.Fatalf("seed wallet balance: %v", err)
	}

	r := ts.get("/api/llm/me", token)
	var k map[string]any
	_ = json.Unmarshal(r.Data, &k)
	proxyStudentID := int(k["proxy_student_id"].(float64))

	// 사용량 주입: 10k input + 5k output, no cache
	// 예상 비용: (10000*15 + 5000*75)/1M × 1400 = (0.15+0.375) × 1400 ≈ 735원
	ts.llmProxy.setUsage(proxyStudentID, application.ProxyUsage{
		Requests: 10, PromptTokens: 10_000, CompletionTokens: 5_000,
	})

	billDate := time.Date(2026, 4, 17, 0, 0, 0, 0, llm.KST)
	processed, err := ts.llmUC.BillAll(context.Background(), billDate)
	if err != nil {
		t.Fatalf("BillAll: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected 1 processed, got %d", processed)
	}

	// 사용량 조회
	usageResp := ts.get("/api/llm/me/usage?days=30", token)
	if !usageResp.Success {
		t.Fatalf("usage GET failed: %+v", usageResp.Error)
	}
	var data struct {
		Daily []struct {
			CostKRW    int `json:"cost_krw"`
			DebitedKRW int `json:"debited_krw"`
			DebtKRW    int `json:"debt_krw"`
		} `json:"daily"`
		Summary struct {
			CumulativeCostKRW int `json:"cumulative_cost_krw"`
			CumulativeDebtKRW int `json:"cumulative_debt_krw"`
		} `json:"summary"`
	}
	_ = json.Unmarshal(usageResp.Data, &data)
	if len(data.Daily) != 1 {
		t.Fatalf("expected 1 daily row, got %d", len(data.Daily))
	}
	if data.Daily[0].CostKRW < 700 || data.Daily[0].CostKRW > 770 {
		t.Errorf("cost: got %d, expected ~735", data.Daily[0].CostKRW)
	}
	if data.Daily[0].DebtKRW != 0 {
		t.Errorf("no debt expected with default wallet balance")
	}
	if data.Summary.CumulativeCostKRW != data.Daily[0].CostKRW {
		t.Errorf("summary should match daily total")
	}
}

func TestLLM_GetStatus_ReturnsSanitizedProxyStatus(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("llm-status@ewha.ac.kr", "pass1234", "상태조회", "2024099")

	r := ts.get("/api/llm/status", token)
	if !r.Success {
		t.Fatalf("status GET failed: %+v", r.Error)
	}
	var s map[string]any
	_ = json.Unmarshal(r.Data, &s)
	if s["upstream_status"] != "ok" {
		t.Errorf("upstream_status: got %v", s["upstream_status"])
	}
	if s["model"] != "Qwen3.6-35B.gguf" {
		t.Errorf("model should be filename only: %v", s["model"])
	}
	// PID / logs / database 가 응답에 노출되지 않아야 함
	if _, has := s["pid"]; has {
		t.Errorf("pid must not leak to student view")
	}
	if _, has := s["logs"]; has {
		t.Errorf("logs must not leak to student view")
	}
}

func TestLLM_BillAll_RecordsDebtWhenWalletInsufficient(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("llm5@ewha.ac.kr", "pass1234", "LLM테스트5", "2024005")

	r := ts.get("/api/llm/me", token)
	var k map[string]any
	_ = json.Unmarshal(r.Data, &k)
	proxyStudentID := int(k["proxy_student_id"].(float64))

	// 지갑을 0 으로 만들기 위해 초기 자본을 모두 탕진하는 대신,
	// 극단적으로 큰 usage 를 주입해 잔액을 초과하게 함
	// 1M prompt × 15/M × 1400 = 21,000원 (default 초기자본 10,000원 기준)
	ts.llmProxy.setUsage(proxyStudentID, application.ProxyUsage{
		Requests: 1, PromptTokens: 1_000_000, CompletionTokens: 0,
	})

	_, err := ts.llmUC.BillAll(context.Background(), time.Date(2026, 4, 16, 0, 0, 0, 0, llm.KST))
	if err != nil {
		t.Fatalf("BillAll: %v", err)
	}

	usageResp := ts.get("/api/llm/me/usage?days=30", token)
	var data struct {
		Daily []struct {
			CostKRW    int `json:"cost_krw"`
			DebitedKRW int `json:"debited_krw"`
			DebtKRW    int `json:"debt_krw"`
		} `json:"daily"`
	}
	_ = json.Unmarshal(usageResp.Data, &data)
	if len(data.Daily) != 1 {
		t.Fatalf("expected 1 row")
	}
	if data.Daily[0].DebtKRW <= 0 {
		t.Errorf("expected positive debt, got %d", data.Daily[0].DebtKRW)
	}
	if data.Daily[0].DebitedKRW+data.Daily[0].DebtKRW != data.Daily[0].CostKRW {
		t.Errorf("debit + debt should equal cost: %+v", data.Daily[0])
	}
}
