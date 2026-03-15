package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestFreelanceAssignmentMode tests the multi-worker "assignment" mode
// where a job can accept multiple workers with auto-approve on apply.
func TestFreelanceAssignmentMode(t *testing.T) {
	ts := setupTestServer(t)

	// Setup: create client (professor) and multiple workers (students)
	clientToken := ts.registerAndApprove("prof@test.com", "pass1234", "김교수", "2024100")
	worker1Token := ts.registerAndApprove("student1@test.com", "pass1234", "학생1", "2024101")
	worker2Token := ts.registerAndApprove("student2@test.com", "pass1234", "학생2", "2024102")
	worker3Token := ts.registerAndApprove("student3@test.com", "pass1234", "학생3", "2024103")

	// Give all users enough balance
	adminToken := ts.login(testAdminEmail, testAdminPass)
	ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_all":  true,
		"amount":      10000,
		"description": "테스트 잔고",
	}, adminToken)

	t.Run("create assignment job with max_workers and auto_approve", func(t *testing.T) {
		resp := ts.post("/api/freelance/jobs", map[string]interface{}{
			"title":                    "과제1: Go 함수 작성",
			"description":              "지정된 함수를 작성하세요",
			"budget":                   100,
			"required_skills":          []string{"Go"},
			"max_workers":              0,  // unlimited
			"auto_approve_application": true,
		}, clientToken)

		if !resp.Success {
			t.Fatalf("expected success, got error: %v", resp.Error)
		}

		var data map[string]interface{}
		json.Unmarshal(resp.Data, &data)
		if int(data["max_workers"].(float64)) != 0 {
			t.Errorf("expected max_workers=0, got %v", data["max_workers"])
		}
		if data["auto_approve_application"] != true {
			t.Errorf("expected auto_approve_application=true, got %v", data["auto_approve_application"])
		}
	})

	// Create assignment job for subsequent tests
	resp := ts.post("/api/freelance/jobs", map[string]interface{}{
		"title":                    "과제2: 웹서버 만들기",
		"description":              "간단한 HTTP 서버를 작성하세요",
		"budget":                   200,
		"required_skills":          []string{"Go", "HTTP"},
		"max_workers":              0,
		"auto_approve_application": true,
	}, clientToken)
	var jobData map[string]interface{}
	json.Unmarshal(resp.Data, &jobData)
	jobID := int(jobData["id"].(float64))

	t.Run("auto_approve: worker applies and gets immediately accepted", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "열심히 하겠습니다",
			"price":    200,
		}, worker1Token)

		if !resp.Success {
			t.Fatalf("expected success, got error: %v", resp.Error)
		}

		var data map[string]interface{}
		json.Unmarshal(resp.Data, &data)
		if data["status"] != "accepted" {
			t.Errorf("expected application status=accepted, got %v", data["status"])
		}
	})

	t.Run("auto_approve: second worker also gets accepted", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "저도 하겠습니다",
			"price":    200,
		}, worker2Token)

		if !resp.Success {
			t.Fatalf("expected success, got error: %v", resp.Error)
		}

		var data map[string]interface{}
		json.Unmarshal(resp.Data, &data)
		if data["status"] != "accepted" {
			t.Errorf("expected application status=accepted, got %v", data["status"])
		}
	})

	t.Run("auto_approve: third worker also gets accepted", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "저도요",
			"price":    200,
		}, worker3Token)

		if !resp.Success {
			t.Fatalf("expected success, got error: %v", resp.Error)
		}

		var data map[string]interface{}
		json.Unmarshal(resp.Data, &data)
		if data["status"] != "accepted" {
			t.Errorf("expected status=accepted, got %v", data["status"])
		}
	})

	t.Run("job stays open after multiple accepts in assignment mode", func(t *testing.T) {
		resp := ts.get(fmt.Sprintf("/api/freelance/jobs/%d", jobID), clientToken)
		var data map[string]interface{}
		json.Unmarshal(resp.Data, &data)
		if data["status"] != "open" {
			t.Errorf("expected job status=open in assignment mode, got %v", data["status"])
		}
	})

	t.Run("worker completes and submits work via application", func(t *testing.T) {
		// Get worker1's application ID
		resp := ts.get(fmt.Sprintf("/api/freelance/jobs/%d/applications", jobID), clientToken)
		var apps []map[string]interface{}
		json.Unmarshal(resp.Data, &apps)

		var worker1AppID int
		for _, app := range apps {
			if app["status"] == "accepted" {
				worker1AppID = int(app["id"].(float64))
				break
			}
		}
		if worker1AppID == 0 {
			t.Fatal("could not find worker1's accepted application")
		}

		// Worker1 submits completion
		resp2 := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/complete", jobID), map[string]interface{}{
			"report":         "과제 완료했습니다",
			"application_id": worker1AppID,
		}, worker1Token)

		if !resp2.Success {
			t.Fatalf("expected success, got error: %v", resp2.Error)
		}
	})

	t.Run("client approves individual application", func(t *testing.T) {
		// Get worker1's application ID
		resp := ts.get(fmt.Sprintf("/api/freelance/jobs/%d/applications", jobID), clientToken)
		var apps []map[string]interface{}
		json.Unmarshal(resp.Data, &apps)

		var worker1AppID int
		for _, app := range apps {
			// After completion, status should be "completed" or still "accepted"
			worker1AppID = int(app["id"].(float64))
			break
		}

		resp2 := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/approve", jobID), map[string]interface{}{
			"application_id": worker1AppID,
		}, clientToken)

		if !resp2.Success {
			t.Fatalf("expected success, got error: %v", resp2.Error)
		}
	})

	t.Run("job still open after approving one worker in assignment mode", func(t *testing.T) {
		resp := ts.get(fmt.Sprintf("/api/freelance/jobs/%d", jobID), clientToken)
		var data map[string]interface{}
		json.Unmarshal(resp.Data, &data)
		// Job should still be open because other workers haven't completed yet
		if data["status"] != "open" {
			t.Errorf("expected job status=open, got %v", data["status"])
		}
	})
}

// TestFreelanceMaxWorkers tests the max_workers limit.
func TestFreelanceMaxWorkers(t *testing.T) {
	ts := setupTestServer(t)

	clientToken := ts.registerAndApprove("client@test.com", "pass1234", "의뢰인", "2024200")
	worker1Token := ts.registerAndApprove("w1@test.com", "pass1234", "작업자1", "2024201")
	worker2Token := ts.registerAndApprove("w2@test.com", "pass1234", "작업자2", "2024202")
	worker3Token := ts.registerAndApprove("w3@test.com", "pass1234", "작업자3", "2024203")

	// Give all users balance
	adminToken := ts.login(testAdminEmail, testAdminPass)
	ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_all":  true,
		"amount":      10000,
		"description": "테스트",
	}, adminToken)

	// Create job with max_workers=2
	resp := ts.post("/api/freelance/jobs", map[string]interface{}{
		"title":                    "2명 제한 과제",
		"description":              "최대 2명까지",
		"budget":                   100,
		"max_workers":              2,
		"auto_approve_application": true,
	}, clientToken)
	var jobData map[string]interface{}
	json.Unmarshal(resp.Data, &jobData)
	jobID := int(jobData["id"].(float64))

	t.Run("first two workers accepted", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "1번", "price": 100,
		}, worker1Token)
		if !resp.Success {
			t.Fatalf("worker1 apply failed: %v", resp.Error)
		}

		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "2번", "price": 100,
		}, worker2Token)
		if !resp.Success {
			t.Fatalf("worker2 apply failed: %v", resp.Error)
		}
	})

	t.Run("third worker rejected - max_workers reached", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "3번", "price": 100,
		}, worker3Token)

		// Should fail because max_workers=2 already filled
		if resp.Success {
			t.Errorf("expected failure when max_workers reached, but got success")
		}
	})
}

// TestFreelanceTraditionalMode verifies backward compatibility:
// default job (max_workers=1, auto_approve=false) works exactly as before.
func TestFreelanceTraditionalMode(t *testing.T) {
	ts := setupTestServer(t)

	clientToken := ts.registerAndApprove("client@test.com", "pass1234", "의뢰인", "2024300")
	worker1Token := ts.registerAndApprove("w1@test.com", "pass1234", "작업자1", "2024301")
	_ = ts.registerAndApprove("w2@test.com", "pass1234", "작업자2", "2024302")

	adminToken := ts.login(testAdminEmail, testAdminPass)
	ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_all":  true,
		"amount":      10000,
		"description": "테스트",
	}, adminToken)

	// Create traditional job (no assignment mode fields)
	resp := ts.post("/api/freelance/jobs", map[string]interface{}{
		"title":       "일반 외주",
		"description": "기존 방식 외주",
		"budget":      500,
	}, clientToken)
	var jobData map[string]interface{}
	json.Unmarshal(resp.Data, &jobData)
	jobID := int(jobData["id"].(float64))

	// Default values check
	t.Run("default max_workers=1 and auto_approve=false", func(t *testing.T) {
		if int(jobData["max_workers"].(float64)) != 1 {
			t.Errorf("expected default max_workers=1, got %v", jobData["max_workers"])
		}
		if jobData["auto_approve_application"] != false {
			t.Errorf("expected default auto_approve=false, got %v", jobData["auto_approve_application"])
		}
	})

	t.Run("traditional flow: apply -> accept -> complete -> approve", func(t *testing.T) {
		// Worker1 applies
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
			"proposal": "지원합니다", "price": 500,
		}, worker1Token)
		var app1Data map[string]interface{}
		json.Unmarshal(resp.Data, &app1Data)
		app1ID := int(app1Data["id"].(float64))
		if app1Data["status"] != "pending" {
			t.Errorf("expected pending (no auto-approve), got %v", app1Data["status"])
		}

		// Client accepts worker1
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/accept", jobID), map[string]interface{}{
			"application_id": app1ID,
		}, clientToken)
		if !resp.Success {
			t.Fatalf("accept failed: %v", resp.Error)
		}

		// Job should be in_progress
		resp = ts.get(fmt.Sprintf("/api/freelance/jobs/%d", jobID), clientToken)
		var data map[string]interface{}
		json.Unmarshal(resp.Data, &data)
		if data["status"] != "in_progress" {
			t.Errorf("expected in_progress, got %v", data["status"])
		}

		// Worker1 completes
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/complete", jobID), map[string]interface{}{
			"report": "완료보고",
		}, worker1Token)
		if !resp.Success {
			t.Fatalf("complete failed: %v", resp.Error)
		}

		// Client approves
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/approve", jobID), nil, clientToken)
		if !resp.Success {
			t.Fatalf("approve failed: %v", resp.Error)
		}

		// Job should be completed
		resp = ts.get(fmt.Sprintf("/api/freelance/jobs/%d", jobID), clientToken)
		json.Unmarshal(resp.Data, &data)
		if data["status"] != "completed" {
			t.Errorf("expected completed, got %v", data["status"])
		}
	})
}
