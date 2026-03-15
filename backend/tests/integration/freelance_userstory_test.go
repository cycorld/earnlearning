package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// =============================================================================
// 유저스토리 기반 외주 마켓 통합 테스트
//
// 모든 시나리오는 실제 유저 플로우를 그대로 재현합니다.
// 외주는 1인 전통 모드만 지원합니다.
// =============================================================================

// --- 공통 헬퍼 ---

func getBalance(ts *testServer, token string) int {
	ts.t.Helper()
	wr := ts.get("/api/wallet", token)
	var w map[string]interface{}
	json.Unmarshal(wr.Data, &w)
	wObj := w["wallet"].(map[string]interface{})
	return int(wObj["balance"].(float64))
}

func createJob(ts *testServer, token string, opts map[string]interface{}) (int, map[string]interface{}) {
	ts.t.Helper()
	defaults := map[string]interface{}{
		"title": "테스트 의뢰", "description": "설명",
		"budget": 1000,
	}
	for k, v := range opts {
		defaults[k] = v
	}
	resp := ts.post("/api/freelance/jobs", defaults, token)
	if !resp.Success {
		ts.t.Fatalf("createJob failed: %v", resp.Error)
	}
	var job map[string]interface{}
	json.Unmarshal(resp.Data, &job)
	return int(job["id"].(float64)), job
}

func applyJob(ts *testServer, token string, jobID int, price int) (int, *apiResponse) {
	ts.t.Helper()
	resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", jobID), map[string]interface{}{
		"proposal": "지원합니다", "price": price,
	}, token)
	if !resp.Success {
		return 0, resp
	}
	var app map[string]interface{}
	json.Unmarshal(resp.Data, &app)
	return int(app["id"].(float64)), resp
}

func getApplications(ts *testServer, token string, jobID int) []map[string]interface{} {
	ts.t.Helper()
	resp := ts.get(fmt.Sprintf("/api/freelance/jobs/%d/applications", jobID), token)
	var apps []map[string]interface{}
	json.Unmarshal(resp.Data, &apps)
	return apps
}

func getJob(ts *testServer, token string, jobID int) map[string]interface{} {
	ts.t.Helper()
	resp := ts.get(fmt.Sprintf("/api/freelance/jobs/%d", jobID), token)
	var job map[string]interface{}
	json.Unmarshal(resp.Data, &job)
	return job
}

func listJobs(ts *testServer, token string, query string) []map[string]interface{} {
	ts.t.Helper()
	resp := ts.get("/api/freelance/jobs?"+query, token)
	var result map[string]interface{}
	json.Unmarshal(resp.Data, &result)
	data, ok := result["data"].([]interface{})
	if !ok || data == nil {
		return []map[string]interface{}{}
	}
	jobs := make([]map[string]interface{}, len(data))
	for i, d := range data {
		jobs[i] = d.(map[string]interface{})
	}
	return jobs
}

// =============================================================================
// 유저스토리 1: 외주 전체 흐름 (1인 전통 모드)
//
// 의뢰인이 외주를 등록하고, 작업자가 지원하고, 의뢰인이 수락하면
// 에스크로가 잡히고, 작업 완료 후 승인하면 대금이 지급된다.
// =============================================================================
func TestUserStory_TraditionalFreelance(t *testing.T) {
	ts := setupTestServer(t)

	client := ts.registerAndApprove("us1-client@test.com", "pass1234", "의뢰인", "2025001")
	worker := ts.registerAndApprove("us1-worker@test.com", "pass1234", "작업자", "2025002")

	adminToken := ts.login(testAdminEmail, testAdminPass)
	ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_all": true, "amount": 50000, "description": "초기자금",
	}, adminToken)

	t.Run("1. 의뢰인이 외주를 등록한다", func(t *testing.T) {
		jobID, job := createJob(ts, client, map[string]interface{}{
			"title": "로고 디자인", "description": "회사 로고 만들어주세요",
			"budget": 5000,
		})
		if jobID == 0 {
			t.Fatal("job ID is 0")
		}
		if job["status"] != "open" {
			t.Errorf("expected open, got %v", job["status"])
		}
	})

	// 새로운 job으로 전체 흐름 테스트
	t.Run("2. 전체 흐름: 등록→지원→수락→완료→승인→대금지급", func(t *testing.T) {
		clientBefore := getBalance(ts, client)
		workerBefore := getBalance(ts, worker)

		// 등록
		jobID, _ := createJob(ts, client, map[string]interface{}{
			"title": "웹사이트 제작", "budget": 3000,
		})

		// 지원
		_, resp := applyJob(ts, worker, jobID, 2500)
		if !resp.Success {
			t.Fatalf("apply failed: %v", resp.Error)
		}

		// 지원 목록 확인
		apps := getApplications(ts, client, jobID)
		if len(apps) != 1 {
			t.Fatalf("expected 1 application, got %d", len(apps))
		}
		appID := int(apps[0]["id"].(float64))
		if apps[0]["status"] != "pending" {
			t.Errorf("expected pending, got %v", apps[0]["status"])
		}

		// 수락 → 에스크로 차감
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/accept", jobID), map[string]interface{}{
			"application_id": appID,
		}, client)
		if !resp.Success {
			t.Fatalf("accept failed: %v", resp.Error)
		}

		clientAfterAccept := getBalance(ts, client)
		if clientAfterAccept != clientBefore-2500 {
			t.Errorf("에스크로 차감 실패: before=%d, after=%d, expected=%d", clientBefore, clientAfterAccept, clientBefore-2500)
		}

		// job 상태 확인 (in_progress)
		job := getJob(ts, client, jobID)
		if job["status"] != "in_progress" {
			t.Errorf("expected in_progress, got %v", job["status"])
		}

		// 작업 완료 보고
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/complete", jobID), map[string]interface{}{
			"report": "웹사이트 완성했습니다",
		}, worker)
		if !resp.Success {
			t.Fatalf("complete failed: %v", resp.Error)
		}

		// 의뢰인 승인 → 대금 지급
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/approve", jobID), nil, client)
		if !resp.Success {
			t.Fatalf("approve failed: %v", resp.Error)
		}

		// 잔액 확인
		workerAfter := getBalance(ts, worker)
		if workerAfter != workerBefore+2500 {
			t.Errorf("작업자 대금 미지급: before=%d, after=%d, expected=%d", workerBefore, workerAfter, workerBefore+2500)
		}

		// job 상태 = completed
		job = getJob(ts, client, jobID)
		if job["status"] != "completed" {
			t.Errorf("expected completed, got %v", job["status"])
		}
	})
}

// =============================================================================
// 유저스토리 2: 의뢰 취소와 에스크로 환불
//
// 의뢰인이 의뢰를 취소하면 에스크로로 잡힌 금액이 환불된다.
// =============================================================================
func TestUserStory_CancelAndRefund(t *testing.T) {
	ts := setupTestServer(t)

	client := ts.registerAndApprove("us5-client@test.com", "pass1234", "의뢰인", "2025040")
	worker := ts.registerAndApprove("us5-worker@test.com", "pass1234", "작업자", "2025041")

	adminToken := ts.login(testAdminEmail, testAdminPass)
	ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_all": true, "amount": 50000, "description": "초기자금",
	}, adminToken)

	t.Run("수락 후 취소 시 에스크로 환불", func(t *testing.T) {
		before := getBalance(ts, client)

		jobID, _ := createJob(ts, client, map[string]interface{}{
			"title": "취소 테스트", "budget": 2000,
		})

		// 지원
		applyJob(ts, worker, jobID, 2000)

		// 수락 → 에스크로 차감
		apps := getApplications(ts, client, jobID)
		appID := int(apps[0]["id"].(float64))
		ts.post(fmt.Sprintf("/api/freelance/jobs/%d/accept", jobID), map[string]interface{}{
			"application_id": appID,
		}, client)

		afterEscrow := getBalance(ts, client)
		if afterEscrow >= before {
			t.Errorf("에스크로 차감 안됨: before=%d, after=%d", before, afterEscrow)
		}

		// 취소
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/cancel", jobID), nil, client)
		if !resp.Success {
			t.Fatalf("cancel failed: %v", resp.Error)
		}

		// 환불 확인
		afterCancel := getBalance(ts, client)
		if afterCancel != before {
			t.Errorf("에스크로 환불 실패: before=%d, afterCancel=%d", before, afterCancel)
		}

		// 상태 확인
		job := getJob(ts, client, jobID)
		if job["status"] != "cancelled" {
			t.Errorf("expected cancelled, got %v", job["status"])
		}
	})

	t.Run("대기 중 지원자가 있어도 취소 가능", func(t *testing.T) {
		jobID, _ := createJob(ts, client, map[string]interface{}{
			"title": "대기 취소", "budget": 1000,
		})

		applyJob(ts, worker, jobID, 1000)

		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/cancel", jobID), nil, client)
		if !resp.Success {
			t.Fatalf("cancel with pending apps failed: %v", resp.Error)
		}
	})

	t.Run("작업자는 의뢰를 취소할 수 없다", func(t *testing.T) {
		jobID, _ := createJob(ts, client, map[string]interface{}{
			"title": "권한 테스트", "budget": 1000,
		})

		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/cancel", jobID), nil, worker)
		if resp.Success {
			t.Error("작업자가 취소에 성공하면 안됨")
		}
	})
}

// =============================================================================
// 유저스토리 3: 목록 조회와 필터링
// =============================================================================
func TestUserStory_ListAndFilter(t *testing.T) {
	ts := setupTestServer(t)

	client := ts.registerAndApprove("us6-client@test.com", "pass1234", "의뢰인", "2025050")

	adminToken := ts.login(testAdminEmail, testAdminPass)
	ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_all": true, "amount": 50000, "description": "초기자금",
	}, adminToken)

	// 다양한 의뢰 등록
	createJob(ts, client, map[string]interface{}{
		"title": "외주1", "budget": 1000,
	})
	createJob(ts, client, map[string]interface{}{
		"title": "외주2", "budget": 2000,
	})
	createJob(ts, client, map[string]interface{}{
		"title": "외주3", "budget": 3000,
	})

	t.Run("목록이 올바르게 조회된다", func(t *testing.T) {
		jobs := listJobs(ts, client, "page=1&limit=10")
		if len(jobs) != 3 {
			t.Fatalf("expected 3 jobs, got %d", len(jobs))
		}
	})

	t.Run("상태 필터가 동작한다", func(t *testing.T) {
		openJobs := listJobs(ts, client, "status=open&page=1&limit=10")
		if len(openJobs) != 3 {
			t.Errorf("expected 3 open jobs, got %d", len(openJobs))
		}

		completedJobs := listJobs(ts, client, "status=completed&page=1&limit=10")
		if len(completedJobs) != 0 {
			t.Errorf("expected 0 completed jobs, got %d", len(completedJobs))
		}
	})
}

// =============================================================================
// 유저스토리 4: 자기 자신 의뢰에 지원 불가 + 중복 지원 불가
// =============================================================================
func TestUserStory_ApplyRestrictions(t *testing.T) {
	ts := setupTestServer(t)

	client := ts.registerAndApprove("us7-client@test.com", "pass1234", "의뢰인", "2025060")
	worker := ts.registerAndApprove("us7-worker@test.com", "pass1234", "작업자", "2025061")

	adminToken := ts.login(testAdminEmail, testAdminPass)
	ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_all": true, "amount": 50000, "description": "초기자금",
	}, adminToken)

	jobID, _ := createJob(ts, client, map[string]interface{}{
		"title": "제한 테스트", "budget": 1000,
	})

	t.Run("자기 의뢰에 지원 불가", func(t *testing.T) {
		_, resp := applyJob(ts, client, jobID, 1000)
		if resp.Success {
			t.Error("자기 의뢰에 지원이 성공하면 안됨")
		}
	})

	t.Run("중복 지원 불가", func(t *testing.T) {
		_, resp := applyJob(ts, worker, jobID, 1000)
		if !resp.Success {
			t.Fatalf("첫 지원 실패: %v", resp.Error)
		}

		_, resp = applyJob(ts, worker, jobID, 1000)
		if resp.Success {
			t.Error("중복 지원이 성공하면 안됨")
		}
	})
}

// =============================================================================
// 유저스토리 5: 리뷰 작성
//
// 완료된 의뢰에 대해 의뢰인과 작업자 모두 리뷰를 작성할 수 있다.
// 중복 리뷰는 불가하고, 미완료 의뢰에는 리뷰 불가.
// =============================================================================
func TestUserStory_Review(t *testing.T) {
	ts := setupTestServer(t)

	client := ts.registerAndApprove("us8-client@test.com", "pass1234", "의뢰인", "2025070")
	worker := ts.registerAndApprove("us8-worker@test.com", "pass1234", "작업자", "2025071")

	adminToken := ts.login(testAdminEmail, testAdminPass)
	ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_all": true, "amount": 50000, "description": "초기자금",
	}, adminToken)

	// 전체 흐름을 거쳐 completed 상태 만들기
	jobID, _ := createJob(ts, client, map[string]interface{}{
		"title": "리뷰 테스트", "budget": 1000,
	})
	applyJob(ts, worker, jobID, 1000)
	apps := getApplications(ts, client, jobID)
	appID := int(apps[0]["id"].(float64))

	ts.post(fmt.Sprintf("/api/freelance/jobs/%d/accept", jobID), map[string]interface{}{
		"application_id": appID,
	}, client)
	ts.post(fmt.Sprintf("/api/freelance/jobs/%d/complete", jobID), map[string]interface{}{
		"report": "완료",
	}, worker)
	ts.post(fmt.Sprintf("/api/freelance/jobs/%d/approve", jobID), nil, client)

	t.Run("의뢰인이 리뷰를 작성한다", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/review", jobID), map[string]interface{}{
			"rating": 5, "comment": "훌륭한 작업이었습니다",
		}, client)
		if !resp.Success {
			t.Fatalf("review failed: %v", resp.Error)
		}
	})

	t.Run("작업자도 리뷰를 작성한다", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/review", jobID), map[string]interface{}{
			"rating": 4, "comment": "좋은 의뢰인이었습니다",
		}, worker)
		if !resp.Success {
			t.Fatalf("review failed: %v", resp.Error)
		}
	})

	t.Run("중복 리뷰 불가", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/review", jobID), map[string]interface{}{
			"rating": 3, "comment": "다시 작성",
		}, client)
		if resp.Success {
			t.Error("중복 리뷰가 성공하면 안됨")
		}
	})

	t.Run("미완료 의뢰에 리뷰 불가", func(t *testing.T) {
		openJobID, _ := createJob(ts, client, map[string]interface{}{
			"title": "미완료 리뷰", "budget": 500,
		})
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/review", openJobID), map[string]interface{}{
			"rating": 5, "comment": "테스트",
		}, client)
		if resp.Success {
			t.Error("미완료 의뢰에 리뷰 성공하면 안됨")
		}
	})
}

// =============================================================================
// 유저스토리 6: 잔액 부족 시 에스크로 실패
// =============================================================================
func TestUserStory_InsufficientBalance(t *testing.T) {
	ts := setupTestServer(t)

	client := ts.registerAndApprove("bal-client@test.com", "pass1234", "의뢰인", "2025080")
	worker := ts.registerAndApprove("bal-worker@test.com", "pass1234", "작업자", "2025081")

	// 초기자금 없이 시작 (잔액 0)

	jobID, _ := createJob(ts, client, map[string]interface{}{
		"title": "잔액부족 테스트", "budget": 5000,
	})

	_, resp := applyJob(ts, worker, jobID, 5000)
	if !resp.Success {
		t.Fatalf("지원 실패: %v", resp.Error)
	}

	apps := getApplications(ts, client, jobID)
	appID := int(apps[0]["id"].(float64))

	t.Run("잔액 부족 시 수락 실패", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/accept", jobID), map[string]interface{}{
			"application_id": appID,
		}, client)
		if resp.Success {
			t.Error("잔액 부족인데 수락이 성공하면 안됨")
		}
	})

	t.Run("잔액 충전 후 수락 성공", func(t *testing.T) {
		adminToken := ts.login(testAdminEmail, testAdminPass)
		ts.post("/api/admin/wallet/transfer", map[string]interface{}{
			"target_all": true, "amount": 50000, "description": "충전",
		}, adminToken)

		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/accept", jobID), map[string]interface{}{
			"application_id": appID,
		}, client)
		if !resp.Success {
			t.Fatalf("충전 후 수락 실패: %v", resp.Error)
		}
	})
}

// =============================================================================
// 유저스토리 7: 여러 지원자 중 1명만 수락
// =============================================================================
func TestUserStory_MultipleApplicants(t *testing.T) {
	ts := setupTestServer(t)

	client := ts.registerAndApprove("ma-client@test.com", "pass1234", "의뢰인", "2025090")
	worker1 := ts.registerAndApprove("ma-w1@test.com", "pass1234", "작업자1", "2025091")
	worker2 := ts.registerAndApprove("ma-w2@test.com", "pass1234", "작업자2", "2025092")
	worker3 := ts.registerAndApprove("ma-w3@test.com", "pass1234", "작업자3", "2025093")

	adminToken := ts.login(testAdminEmail, testAdminPass)
	ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_all": true, "amount": 50000, "description": "초기자금",
	}, adminToken)

	jobID, _ := createJob(ts, client, map[string]interface{}{
		"title": "다수 지원자", "budget": 3000,
	})

	// 3명 지원
	applyJob(ts, worker1, jobID, 2000)
	applyJob(ts, worker2, jobID, 2500)
	applyJob(ts, worker3, jobID, 3000)

	t.Run("3명 지원 확인", func(t *testing.T) {
		apps := getApplications(ts, client, jobID)
		if len(apps) != 3 {
			t.Fatalf("expected 3 applications, got %d", len(apps))
		}
	})

	t.Run("1명 수락 후 다른 지원자 수락 불가", func(t *testing.T) {
		apps := getApplications(ts, client, jobID)
		app1ID := int(apps[0]["id"].(float64))
		app2ID := int(apps[1]["id"].(float64))

		// 첫 번째 수락
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/accept", jobID), map[string]interface{}{
			"application_id": app1ID,
		}, client)
		if !resp.Success {
			t.Fatalf("첫 수락 실패: %v", resp.Error)
		}

		// 두 번째 수락 시도 — 실패해야 함
		resp = ts.post(fmt.Sprintf("/api/freelance/jobs/%d/accept", jobID), map[string]interface{}{
			"application_id": app2ID,
		}, client)
		if resp.Success {
			t.Error("이미 수락된 의뢰에 또 수락이 성공하면 안됨")
		}

		// job 상태가 in_progress
		job := getJob(ts, client, jobID)
		if job["status"] != "in_progress" {
			t.Errorf("expected in_progress, got %v", job["status"])
		}
	})
}

// =============================================================================
// 유저스토리 8: 작업 완료 전 승인 시도 불가
// =============================================================================
func TestUserStory_ApproveBeforeComplete(t *testing.T) {
	ts := setupTestServer(t)

	client := ts.registerAndApprove("abc-client@test.com", "pass1234", "의뢰인", "2025200")
	worker := ts.registerAndApprove("abc-worker@test.com", "pass1234", "작업자", "2025201")

	adminToken := ts.login(testAdminEmail, testAdminPass)
	ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_all": true, "amount": 50000, "description": "초기자금",
	}, adminToken)

	jobID, _ := createJob(ts, client, map[string]interface{}{
		"title": "승인순서 테스트", "budget": 2000,
	})

	applyJob(ts, worker, jobID, 2000)
	apps := getApplications(ts, client, jobID)
	appID := int(apps[0]["id"].(float64))

	ts.post(fmt.Sprintf("/api/freelance/jobs/%d/accept", jobID), map[string]interface{}{
		"application_id": appID,
	}, client)

	t.Run("작업 완료 전 승인 시도 실패", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/approve", jobID), nil, client)
		if resp.Success {
			t.Error("완료 보고 전에 승인이 성공하면 안됨")
		}
	})

	t.Run("작업자가 아닌 사람이 완료 보고 불가", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/complete", jobID), map[string]interface{}{
			"report": "남의 작업 완료",
		}, client)
		if resp.Success {
			t.Error("의뢰인이 완료 보고에 성공하면 안됨")
		}
	})

	t.Run("작업자가 아닌 사람이 승인 불가", func(t *testing.T) {
		// 완료 처리
		ts.post(fmt.Sprintf("/api/freelance/jobs/%d/complete", jobID), map[string]interface{}{
			"report": "완료",
		}, worker)

		resp := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/approve", jobID), nil, worker)
		if resp.Success {
			t.Error("작업자가 승인에 성공하면 안됨")
		}
	})
}
