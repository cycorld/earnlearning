package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestFreelanceEdgeCases tests critical edge cases for the traditional freelance mode.
func TestFreelanceEdgeCases(t *testing.T) {
	ts := setupTestServer(t)

	clientToken := ts.registerAndApprove("edge-client@test.com", "pass1234", "의뢰인", "2024700")
	w1Token := ts.registerAndApprove("edge-w1@test.com", "pass1234", "작업자1", "2024701")
	w2Token := ts.registerAndApprove("edge-w2@test.com", "pass1234", "작업자2", "2024702")

	adminToken := ts.login(testAdminEmail, testAdminPass)
	ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_all": true, "amount": 100000, "description": "엣지케이스 테스트",
	}, adminToken)

	// =========================================================
	// Edge 1: Cancel job should refund escrow to client
	// =========================================================
	t.Run("cancel job refunds escrow", func(t *testing.T) {
		getBalanceFn := func(token string) int {
			wr := ts.get("/api/wallet", token)
			var w map[string]interface{}
			json.Unmarshal(wr.Data, &w)
			wObj := w["wallet"].(map[string]interface{})
			return int(wObj["balance"].(float64))
		}
		balanceBefore := getBalanceFn(clientToken)

		// Create job
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "취소테스트", "description": "취소시 에스크로 환불",
			"budget": 500,
		}, clientToken)
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		// Worker1 applies
		ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "지원", "price": 500,
		}, w1Token)

		// Accept → escrow debit
		apps := ts.get(fmt.Sprintf("/api/freelance/jobs/%d/applications", jobID), clientToken)
		var appList []map[string]interface{}
		json.Unmarshal(apps.Data, &appList)
		appID := int(appList[0]["id"].(float64))

		ts.post(fmt.Sprintf("/api/freelance/jobs/%d/accept", jobID), map[string]interface{}{
			"application_id": appID,
		}, clientToken)

		balanceAfterEscrow := getBalanceFn(clientToken)
		if balanceAfterEscrow >= balanceBefore {
			t.Errorf("expected balance to decrease after escrow, before=%d after=%d", balanceBefore, balanceAfterEscrow)
		}

		// Cancel the job
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/cancel", jobID), nil, clientToken)
		if !resp.Success {
			t.Fatalf("cancel failed: %v", resp.Error)
		}

		// Balance should be restored (escrow refunded)
		balanceAfterCancel := getBalanceFn(clientToken)
		if balanceAfterCancel != balanceBefore {
			t.Errorf("expected balance restored after cancel, before=%d after=%d", balanceBefore, balanceAfterCancel)
		}
	})

	// =========================================================
	// Edge 2: Worker can see their own application status
	// =========================================================
	t.Run("worker sees own application in job detail", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "상태확인", "description": "작업자 본인 상태 확인",
			"budget": 300,
		}, clientToken)
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		// Worker applies
		ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "지원", "price": 300,
		}, w2Token)

		// Worker can see the applications list
		resp = ts.get(fmt.Sprintf("/api/freelance/jobs/%d/applications", jobID), w2Token)
		if !resp.Success {
			t.Fatalf("worker can't see applications: %v", resp.Error)
		}
		var appsList []map[string]interface{}
		json.Unmarshal(resp.Data, &appsList)
		if len(appsList) == 0 {
			t.Fatal("expected to see at least one application")
		}
	})

	// =========================================================
	// Edge 3: Cancel with pending (non-escrow) applications
	// =========================================================
	t.Run("cancel with pending applications", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "대기취소", "description": "대기 지원자가 있을 때 취소",
			"budget": 100,
		}, clientToken)
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		// Worker applies (pending, no escrow yet)
		ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "지원", "price": 100,
		}, w2Token)

		// Cancel should succeed
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/cancel", jobID), nil, clientToken)
		if !resp.Success {
			t.Fatalf("cancel failed: %v", resp.Error)
		}
	})

	// =========================================================
	// Edge 4: close endpoint should not exist anymore
	// =========================================================
	t.Run("close endpoint returns error", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "종료불가", "description": "close 없어짐",
			"budget": 100,
		}, clientToken)
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/close", jobID), nil, clientToken)
		if resp.Success {
			t.Error("close endpoint should not exist or should fail")
		}
	})
}
