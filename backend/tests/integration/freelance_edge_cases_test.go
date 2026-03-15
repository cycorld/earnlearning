package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestFreelanceEdgeCases tests critical edge cases found during design audit.
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
		// Check client balance before
		getBalance := func(token string) int {
			wr := ts.get("/api/wallet", token)
			var w map[string]interface{}
			json.Unmarshal(wr.Data, &w)
			wObj := w["wallet"].(map[string]interface{})
			return int(wObj["balance"].(float64))
		}
		balanceBefore := getBalance(clientToken)

		// Create job with auto-approve
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "취소테스트", "description": "취소시 에스크로 환불",
			"budget": 500, "price_type": "fixed",
			"max_workers": 0, "auto_approve_application": true,
		}, clientToken)
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		// Worker1 applies (auto-approved, escrow debited from client)
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "지원", "price": 500,
		}, w1Token)
		if !resp.Success {
			t.Fatalf("apply failed: %v", resp.Error)
		}

		// Check balance after escrow debit
		balanceAfterEscrow := getBalance(clientToken)

		if balanceAfterEscrow >= balanceBefore {
			t.Errorf("expected balance to decrease after escrow, before=%d after=%d", balanceBefore, balanceAfterEscrow)
		}

		// Cancel the job
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/cancel", jobID), nil, clientToken)
		if !resp.Success {
			t.Fatalf("cancel failed: %v", resp.Error)
		}

		// Balance should be restored (escrow refunded)
		balanceAfterCancel := getBalance(clientToken)

		if balanceAfterCancel != balanceBefore {
			t.Errorf("expected balance restored after cancel, before=%d after=%d", balanceBefore, balanceAfterCancel)
		}
	})

	// =========================================================
	// Edge 2: Client can close assignment mode job
	// =========================================================
	t.Run("client can close assignment job", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "종료테스트", "description": "과제 모드 종료",
			"budget": 200, "price_type": "fixed",
			"max_workers": 0, "auto_approve_application": true,
		}, clientToken)
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		// Worker applies and completes
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "지원", "price": 200,
		}, w1Token)
		var appData map[string]interface{}
		json.Unmarshal(resp.Data, &appData)

		// Get app ID
		appsResp := ts.get(fmt.Sprintf("/api/freelance/jobs/%d/applications", jobID), clientToken)
		var apps []map[string]interface{}
		json.Unmarshal(appsResp.Data, &apps)
		appID := int(apps[0]["id"].(float64))

		// Complete and approve
		ts.post(fmt.Sprintf("/api/freelance/jobs/%d/complete", jobID), map[string]interface{}{
			"report": "완료", "application_id": appID,
		}, w1Token)
		ts.post(fmt.Sprintf("/api/freelance/jobs/%d/approve", jobID), map[string]interface{}{
			"application_id": appID,
		}, clientToken)

		// Client closes the assignment job
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/close", jobID), nil, clientToken)
		if !resp.Success {
			t.Fatalf("close failed: %v", resp.Error)
		}

		// Job should be completed
		resp = ts.get(fmt.Sprintf("/api/freelance/jobs/%d", jobID), clientToken)
		var detail map[string]interface{}
		json.Unmarshal(resp.Data, &detail)
		if detail["status"] != "completed" {
			t.Errorf("expected completed after close, got %v", detail["status"])
		}
	})

	// =========================================================
	// Edge 3: Worker can see their own application status
	// =========================================================
	t.Run("worker sees own application in job detail", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "상태확인", "description": "작업자 본인 상태 확인",
			"budget": 300, "price_type": "negotiable",
			"max_workers": 0, "auto_approve_application": true,
		}, clientToken)
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		// Worker applies
		ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "지원", "price": 300,
		}, w2Token)

		// Worker can see the applications list (to find their own)
		resp = ts.get(fmt.Sprintf("/api/freelance/jobs/%d/applications", jobID), w2Token)
		if !resp.Success {
			t.Fatalf("worker can't see applications: %v", resp.Error)
		}
		var apps []map[string]interface{}
		json.Unmarshal(resp.Data, &apps)
		if len(apps) == 0 {
			t.Fatal("expected to see at least one application")
		}
	})

	// =========================================================
	// Edge 4: Apply succeeds even if client has low balance (escrow best-effort)
	// =========================================================
	t.Run("apply succeeds with insufficient client balance for escrow", func(t *testing.T) {
		// Create a new client with exactly 100 balance
		poorClientToken := ts.registerAndApprove("poor@test.com", "pass1234", "빈털터리", "2024703")
		ts.post("/api/admin/wallet/transfer", map[string]interface{}{
			"target_user_ids": []int{},
			"target_all":      true,
			"amount":          100,
			"description":     "소액",
		}, adminToken)

		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "잔액부족테스트", "description": "에스크로 잔액 부족",
			"budget": 99999, "price_type": "fixed",
			"max_workers": 0, "auto_approve_application": true,
		}, poorClientToken)
		if !resp.Success {
			t.Fatalf("create failed: %v", resp.Error)
		}
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		// Worker applies - should still succeed even though client can't cover escrow
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "지원", "price": 99999,
		}, w1Token)
		if !resp.Success {
			t.Fatalf("apply should succeed regardless of client balance, got error: %v", resp.Error)
		}
	})

	// =========================================================
	// Edge 5: Cancel with pending (non-escrow) applications
	// =========================================================
	t.Run("cancel with pending applications", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "대기취소", "description": "대기 지원자가 있을 때 취소",
			"budget": 100, "max_workers": 0, "auto_approve_application": false,
		}, clientToken)
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		// Worker applies (pending, no escrow)
		ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "지원", "price": 100,
		}, w2Token)

		// Cancel should succeed
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/cancel", jobID), nil, clientToken)
		if !resp.Success {
			t.Fatalf("cancel failed: %v", resp.Error)
		}
	})
}
