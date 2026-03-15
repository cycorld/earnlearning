package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestFreelancePriceType tests the price_type field on freelance jobs.
// price_type can be 'fixed' (client sets the exact price) or 'negotiable' (workers propose their own price).
func TestFreelancePriceType(t *testing.T) {
	ts := setupTestServer(t)

	clientToken := ts.registerAndApprove("client@test.com", "pass1234", "의뢰인", "2024400")
	workerToken := ts.registerAndApprove("worker@test.com", "pass1234", "작업자", "2024401")

	adminToken := ts.login(testAdminEmail, testAdminPass)
	ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_all":  true,
		"amount":      50000,
		"description": "테스트 잔고",
	}, adminToken)

	t.Run("default price_type is negotiable", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title":       "기본 외주",
			"description": "가격 협의 가능",
			"budget":      1000,
		}, clientToken)

		if !resp.Success {
			t.Fatalf("expected success, got error: %v", resp.Error)
		}

		var data map[string]interface{}
		json.Unmarshal(resp.Data, &data)
		if data["price_type"] != "negotiable" {
			t.Errorf("expected default price_type=negotiable, got %v", data["price_type"])
		}
	})

	t.Run("create fixed price job", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title":       "고정 금액 외주",
			"description": "가격 고정",
			"budget":      500,
			"price_type":  "fixed",
		}, clientToken)

		if !resp.Success {
			t.Fatalf("expected success, got error: %v", resp.Error)
		}

		var data map[string]interface{}
		json.Unmarshal(resp.Data, &data)
		if data["price_type"] != "fixed" {
			t.Errorf("expected price_type=fixed, got %v", data["price_type"])
		}
	})

	t.Run("create negotiable price job explicitly", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title":       "협의 금액 외주",
			"description": "가격 협의",
			"budget":      800,
			"price_type":  "negotiable",
		}, clientToken)

		if !resp.Success {
			t.Fatalf("expected success, got error: %v", resp.Error)
		}

		var data map[string]interface{}
		json.Unmarshal(resp.Data, &data)
		if data["price_type"] != "negotiable" {
			t.Errorf("expected price_type=negotiable, got %v", data["price_type"])
		}
	})

	t.Run("invalid price_type rejected", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title":       "잘못된 타입",
			"description": "에러",
			"budget":      100,
			"price_type":  "invalid",
		}, clientToken)

		if resp.Success {
			t.Errorf("expected failure for invalid price_type, but got success")
		}
	})

	// Create a fixed-price job for apply tests
	resp := ts.post("/api/freelance/jobs", map[string]interface{}{
		"title":       "고정 금액 테스트",
		"description": "500원 고정",
		"budget":      500,
		"price_type":  "fixed",
	}, clientToken)
	var fixedJob map[string]interface{}
	json.Unmarshal(resp.Data, &fixedJob)
	fixedJobID := int(fixedJob["id"].(float64))

	t.Run("fixed price job: worker must apply with exact budget price", func(t *testing.T) {
		// Worker tries to apply with different price → should fail
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", fixedJobID), map[string]interface{}{
			"proposal": "다른 금액으로 지원",
			"price":    300,
		}, workerToken)

		if resp.Success {
			t.Errorf("expected failure when applying with non-budget price to fixed-price job")
		}
	})

	t.Run("fixed price job: worker applies with exact budget price succeeds", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", fixedJobID), map[string]interface{}{
			"proposal": "고정 금액으로 지원",
			"price":    500,
		}, workerToken)

		if !resp.Success {
			t.Fatalf("expected success, got error: %v", resp.Error)
		}

		var data map[string]interface{}
		json.Unmarshal(resp.Data, &data)
		if int(data["price"].(float64)) != 500 {
			t.Errorf("expected price=500, got %v", data["price"])
		}
	})

	// Create a negotiable job for apply tests
	worker2Token := ts.registerAndApprove("worker2@test.com", "pass1234", "작업자2", "2024402")
	resp2 := ts.post("/api/freelance/jobs", map[string]interface{}{
		"title":       "협의 금액 테스트",
		"description": "가격 자유",
		"budget":      1000,
		"price_type":  "negotiable",
	}, clientToken)
	var negJob map[string]interface{}
	json.Unmarshal(resp2.Data, &negJob)
	negJobID := int(negJob["id"].(float64))

	t.Run("negotiable price job: worker can apply with any price", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", negJobID), map[string]interface{}{
			"proposal": "저렴하게 해드립니다",
			"price":    700,
		}, worker2Token)

		if !resp.Success {
			t.Fatalf("expected success, got error: %v", resp.Error)
		}

		var data map[string]interface{}
		json.Unmarshal(resp.Data, &data)
		if int(data["price"].(float64)) != 700 {
			t.Errorf("expected price=700, got %v", data["price"])
		}
	})

	t.Run("price_type visible in job list", func(t *testing.T) {
		resp := ts.get("/api/freelance/jobs?limit=50", clientToken)
		var listData struct {
			Data []map[string]interface{} `json:"data"`
		}
		// The list response wraps in pagination
		json.Unmarshal(resp.Data, &listData)

		// Check that price_type field exists in listed jobs
		if len(listData.Data) == 0 {
			// Try direct array parse
			var jobs []map[string]interface{}
			json.Unmarshal(resp.Data, &jobs)
			if len(jobs) > 0 {
				if _, ok := jobs[0]["price_type"]; !ok {
					t.Errorf("expected price_type field in job list response")
				}
			}
		} else {
			if _, ok := listData.Data[0]["price_type"]; !ok {
				t.Errorf("expected price_type field in job list response")
			}
		}
	})

	t.Run("price_type with assignment mode works together", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title":                    "과제 고정금액",
			"description":              "과제 모드 + 고정 금액",
			"budget":                   200,
			"price_type":               "fixed",
			"max_workers":              0,
			"auto_approve_application": true,
		}, clientToken)

		if !resp.Success {
			t.Fatalf("expected success, got error: %v", resp.Error)
		}

		var data map[string]interface{}
		json.Unmarshal(resp.Data, &data)
		if data["price_type"] != "fixed" {
			t.Errorf("expected price_type=fixed, got %v", data["price_type"])
		}
		if int(data["max_workers"].(float64)) != 0 {
			t.Errorf("expected max_workers=0, got %v", data["max_workers"])
		}
	})
}
