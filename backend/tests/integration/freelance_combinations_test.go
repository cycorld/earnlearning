package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestFreelanceCombinations tests all combinations of job configuration:
// price_type (fixed/negotiable) × max_workers (1/N/0) × auto_approve (true/false)
func TestFreelanceCombinations(t *testing.T) {
	ts := setupTestServer(t)

	clientToken := ts.registerAndApprove("combo-client@test.com", "pass1234", "의뢰인", "2024600")
	w1Token := ts.registerAndApprove("combo-w1@test.com", "pass1234", "작업자1", "2024601")
	w2Token := ts.registerAndApprove("combo-w2@test.com", "pass1234", "작업자2", "2024602")
	w3Token := ts.registerAndApprove("combo-w3@test.com", "pass1234", "작업자3", "2024603")

	adminToken := ts.login(testAdminEmail, testAdminPass)
	ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_all": true, "amount": 100000, "description": "조합 테스트 잔고",
	}, adminToken)

	// =========================================================
	// Case 1: fixed + traditional (max_workers=1, auto_approve=false)
	// Standard fixed-price single-worker job
	// =========================================================
	t.Run("case1: fixed + traditional", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "고정/전통", "description": "고정 금액 단일 작업자",
			"budget": 300, "price_type": "fixed",
		}, clientToken)
		if !resp.Success {
			t.Fatalf("create failed: %v", resp.Error)
		}
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		// Verify defaults
		if int(job["max_workers"].(float64)) != 1 {
			t.Errorf("expected max_workers=1, got %v", job["max_workers"])
		}
		if job["auto_approve_application"] != false {
			t.Errorf("expected auto_approve=false")
		}
		if job["price_type"] != "fixed" {
			t.Errorf("expected price_type=fixed, got %v", job["price_type"])
		}

		// Worker applies with wrong price → fail
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "다른 금액", "price": 200,
		}, w1Token)
		if resp.Success {
			t.Errorf("expected rejection for non-matching price")
		}

		// Worker applies with correct price → pending
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "정확한 금액", "price": 300,
		}, w1Token)
		if !resp.Success {
			t.Fatalf("apply failed: %v", resp.Error)
		}
		var appData map[string]interface{}
		json.Unmarshal(resp.Data, &appData)
		appID := int(appData["id"].(float64))
		if appData["status"] != "pending" {
			t.Errorf("expected pending, got %v", appData["status"])
		}

		// Client accepts → in_progress
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/accept", jobID), map[string]interface{}{
			"application_id": appID,
		}, clientToken)
		if !resp.Success {
			t.Fatalf("accept failed: %v", resp.Error)
		}

		resp = ts.get(fmt.Sprintf("/api/freelance/jobs/%d", jobID), clientToken)
		var jobDetail map[string]interface{}
		json.Unmarshal(resp.Data, &jobDetail)
		if jobDetail["status"] != "in_progress" {
			t.Errorf("expected in_progress, got %v", jobDetail["status"])
		}
	})

	// =========================================================
	// Case 2: fixed + assignment (max_workers=0, auto_approve=true)
	// Fixed-price unlimited assignment mode
	// =========================================================
	t.Run("case2: fixed + assignment unlimited auto-approve", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "고정/과제무제한", "description": "고정 금액 과제 무제한",
			"budget": 500, "price_type": "fixed",
			"max_workers": 0, "auto_approve_application": true,
		}, clientToken)
		if !resp.Success {
			t.Fatalf("create failed: %v", resp.Error)
		}
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		// All workers apply with fixed price → all auto-accepted
		for i, token := range []string{w1Token, w2Token, w3Token} {
			// Wrong price → fail
			resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
				"proposal": fmt.Sprintf("다른금액%d", i), "price": 100,
			}, token)
			if resp.Success {
				t.Errorf("worker%d: expected rejection for wrong price", i+1)
			}
		}

		// Now apply with correct price
		for i, token := range []string{w1Token, w2Token, w3Token} {
			resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
				"proposal": fmt.Sprintf("정가지원%d", i), "price": 500,
			}, token)
			if !resp.Success {
				t.Fatalf("worker%d apply failed: %v", i+1, resp.Error)
			}
			var data map[string]interface{}
			json.Unmarshal(resp.Data, &data)
			if data["status"] != "accepted" {
				t.Errorf("worker%d: expected accepted, got %v", i+1, data["status"])
			}
		}

		// Job should still be open
		resp = ts.get(fmt.Sprintf("/api/freelance/jobs/%d", jobID), clientToken)
		var detail map[string]interface{}
		json.Unmarshal(resp.Data, &detail)
		if detail["status"] != "open" {
			t.Errorf("expected open, got %v", detail["status"])
		}
	})

	// =========================================================
	// Case 3: negotiable + assignment (max_workers=2, auto_approve=true)
	// Negotiable price with worker limit
	// =========================================================
	t.Run("case3: negotiable + limited assignment auto-approve", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "협의/과제2명", "description": "협의 금액 과제 2명 제한",
			"budget": 1000, "price_type": "negotiable",
			"max_workers": 2, "auto_approve_application": true,
		}, clientToken)
		if !resp.Success {
			t.Fatalf("create failed: %v", resp.Error)
		}
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		// Worker1 applies with different price → accepted (negotiable allows any price)
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "저렴하게", "price": 700,
		}, w1Token)
		if !resp.Success {
			t.Fatalf("w1 apply failed: %v", resp.Error)
		}
		var d1 map[string]interface{}
		json.Unmarshal(resp.Data, &d1)
		if d1["status"] != "accepted" {
			t.Errorf("expected accepted, got %v", d1["status"])
		}

		// Worker2 applies with budget price → accepted
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "정가", "price": 1000,
		}, w2Token)
		if !resp.Success {
			t.Fatalf("w2 apply failed: %v", resp.Error)
		}

		// Worker3 applies → should fail (max_workers=2 reached)
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "저도", "price": 800,
		}, w3Token)
		if resp.Success {
			t.Errorf("expected rejection when max_workers reached")
		}
	})

	// =========================================================
	// Case 4: negotiable + traditional (max_workers=1, auto_approve=false)
	// Standard negotiable single-worker job (default behavior)
	// =========================================================
	t.Run("case4: negotiable + traditional (default)", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "협의/전통", "description": "기본 외주",
			"budget": 2000,
		}, clientToken)
		if !resp.Success {
			t.Fatalf("create failed: %v", resp.Error)
		}
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		// Default values
		if job["price_type"] != "negotiable" {
			t.Errorf("expected negotiable, got %v", job["price_type"])
		}
		if int(job["max_workers"].(float64)) != 1 {
			t.Errorf("expected max_workers=1")
		}

		// Worker1 applies with different price → pending
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "1500에 해드림", "price": 1500,
		}, w1Token)
		if !resp.Success {
			t.Fatalf("apply failed: %v", resp.Error)
		}
		var appData map[string]interface{}
		json.Unmarshal(resp.Data, &appData)
		if appData["status"] != "pending" {
			t.Errorf("expected pending, got %v", appData["status"])
		}
	})

	// =========================================================
	// Case 5: fixed + assignment with manual approve (max_workers=0, auto_approve=false)
	// Fixed price, unlimited workers, but needs manual approval
	// =========================================================
	t.Run("case5: fixed + unlimited manual approve", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "고정/무제한/수동", "description": "고정금액 무제한 수동승인",
			"budget": 400, "price_type": "fixed",
			"max_workers": 0, "auto_approve_application": false,
		}, clientToken)
		if !resp.Success {
			t.Fatalf("create failed: %v", resp.Error)
		}
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		// Workers apply with exact price → pending (not auto-approved)
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "지원1", "price": 400,
		}, w1Token)
		if !resp.Success {
			t.Fatalf("w1 apply failed: %v", resp.Error)
		}
		var d1 map[string]interface{}
		json.Unmarshal(resp.Data, &d1)
		if d1["status"] != "pending" {
			t.Errorf("expected pending (no auto-approve), got %v", d1["status"])
		}
		app1ID := int(d1["id"].(float64))

		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "지원2", "price": 400,
		}, w2Token)
		if !resp.Success {
			t.Fatalf("w2 apply failed: %v", resp.Error)
		}

		// Client manually accepts worker1
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/accept", jobID), map[string]interface{}{
			"application_id": app1ID,
		}, clientToken)
		if !resp.Success {
			t.Fatalf("accept failed: %v", resp.Error)
		}

		// Job should still be open (assignment mode = max_workers != 1)
		resp = ts.get(fmt.Sprintf("/api/freelance/jobs/%d", jobID), clientToken)
		var detail map[string]interface{}
		json.Unmarshal(resp.Data, &detail)
		if detail["status"] != "open" {
			t.Errorf("expected open (assignment mode), got %v", detail["status"])
		}
	})

	// =========================================================
	// Case 6: negotiable + limited workers manual approve
	// =========================================================
	t.Run("case6: negotiable + limited manual approve", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "협의/3명/수동", "description": "협의금액 3명 수동승인",
			"budget": 1500, "price_type": "negotiable",
			"max_workers": 3, "auto_approve_application": false,
		}, clientToken)
		if !resp.Success {
			t.Fatalf("create failed: %v", resp.Error)
		}
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		// All three workers apply with different prices → all pending
		for i, token := range []string{w1Token, w2Token, w3Token} {
			prices := []int{1200, 1500, 1800}
			resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
				"proposal": fmt.Sprintf("지원%d", i+1), "price": prices[i],
			}, token)
			if !resp.Success {
				t.Fatalf("worker%d apply failed: %v", i+1, resp.Error)
			}
			var data map[string]interface{}
			json.Unmarshal(resp.Data, &data)
			if data["status"] != "pending" {
				t.Errorf("worker%d: expected pending, got %v", i+1, data["status"])
			}
		}
	})

	// =========================================================
	// Case 7: Complete flow - fixed + assignment + auto-approve
	// Full lifecycle: create → apply → complete → approve → payment
	// =========================================================
	t.Run("case7: full lifecycle fixed assignment auto-approve", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "전체흐름테스트", "description": "처음부터 끝까지",
			"budget": 200, "price_type": "fixed",
			"max_workers": 0, "auto_approve_application": true,
		}, clientToken)
		if !resp.Success {
			t.Fatalf("create failed: %v", resp.Error)
		}
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		// Worker1 applies (auto-accepted)
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "지원", "price": 200,
		}, w1Token)
		if !resp.Success {
			t.Fatalf("apply failed: %v", resp.Error)
		}

		// Get application ID
		resp = ts.get(fmt.Sprintf("/api/freelance/jobs/%d/applications", jobID), clientToken)
		var apps []map[string]interface{}
		json.Unmarshal(resp.Data, &apps)
		if len(apps) == 0 {
			t.Fatal("no applications found")
		}
		appID := int(apps[0]["id"].(float64))

		// Worker1 completes
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/complete", jobID), map[string]interface{}{
			"report": "완료보고", "application_id": appID,
		}, w1Token)
		if !resp.Success {
			t.Fatalf("complete failed: %v", resp.Error)
		}

		// Client approves
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/approve", jobID), map[string]interface{}{
			"application_id": appID,
		}, clientToken)
		if !resp.Success {
			t.Fatalf("approve failed: %v", resp.Error)
		}

		// Job stays open (assignment mode)
		resp = ts.get(fmt.Sprintf("/api/freelance/jobs/%d", jobID), clientToken)
		var detail map[string]interface{}
		json.Unmarshal(resp.Data, &detail)
		if detail["status"] != "open" {
			t.Errorf("expected open, got %v", detail["status"])
		}

		// Another worker can still apply
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "추가지원", "price": 200,
		}, w2Token)
		if !resp.Success {
			t.Fatalf("additional apply after approval failed: %v", resp.Error)
		}
	})

	// =========================================================
	// Case 8: Duplicate apply prevention
	// =========================================================
	t.Run("case8: duplicate apply prevented", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "중복방지", "description": "중복 지원 테스트",
			"budget": 100, "price_type": "negotiable",
			"max_workers": 0, "auto_approve_application": true,
		}, clientToken)
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		// Worker1 applies
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "첫지원", "price": 100,
		}, w1Token)
		if !resp.Success {
			t.Fatalf("first apply failed: %v", resp.Error)
		}

		// Worker1 tries to apply again → should fail
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "중복지원", "price": 100,
		}, w1Token)
		if resp.Success {
			t.Errorf("expected duplicate apply to fail")
		}
	})

	// =========================================================
	// Case 9: Client cannot apply to own job
	// =========================================================
	t.Run("case9: client cannot apply to own job", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title": "자기지원방지", "description": "테스트",
			"budget": 100,
		}, clientToken)
		var job map[string]interface{}
		json.Unmarshal(resp.Data, &job)
		jobID := int(job["id"].(float64))

		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "자기지원", "price": 100,
		}, clientToken)
		if resp.Success {
			t.Errorf("expected self-apply to fail")
		}
	})
}
