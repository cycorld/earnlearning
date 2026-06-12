package integration

import (
	"fmt"
	"testing"
)

// =============================================================================
// #130 — SyncAuto 가 학생의 직접 수정/admin 결정을 덮어쓰는 버그 회귀 테스트
//
// 버그 1: 학생이 잘못 자동 집계된 MVP URL을 "다시 제출"로 고쳐도,
//         다음 /milestones/mine 조회 시 SyncAuto 가 자동 후보 URL로 되돌림.
// 버그 2: rejected 자동 항목이 같은 URL로 매 조회마다 재 Upsert 되어
//         status 가 pending 으로 리셋되고 admin_note 가 삭제됨.
// =============================================================================

// 학생이 직접 고친 MVP URL은 자동 동기화가 덮어쓰지 않는다.
func TestMilestone_ManualFixSurvivesAutoSync(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "ms-manualfix@test.com", "수정학생", "20251101", "ms_manualfix_co")

	// 잘못된 URL이 자동 집계된 상황.
	updateCompanyServiceURL(t, ts, token, cid, "https://wrong-auto.vercel.app")
	got := getMyMilestones(t, ts, token)
	mvp1 := findMilestone(t, got, "mvp1")
	if mvp1 == nil || mvp1.URL != "https://wrong-auto.vercel.app" {
		t.Fatalf("precondition: mvp1 auto-detected wrong URL, got %+v", mvp1)
	}

	// 학생이 직접 수정 제출.
	r := ts.post("/api/milestones", map[string]string{
		"type": "mvp1", "url": "https://fixed-by-student.vercel.app",
	}, token)
	if !r.Success {
		t.Fatalf("manual submit: %v", r.Error)
	}

	// 재조회 (SyncAuto 트리거) — 직접 수정한 URL이 유지되어야 함.
	got = getMyMilestones(t, ts, token)
	mvp1 = findMilestone(t, got, "mvp1")
	if mvp1 == nil {
		t.Fatal("mvp1 missing after manual fix")
	}
	if mvp1.URL != "https://fixed-by-student.vercel.app" {
		t.Errorf("mvp1.url = %q, want manually fixed URL (auto-sync overwrote it)", mvp1.URL)
	}
	if mvp1.SourceType != "manual" {
		t.Errorf("mvp1.source_type = %q, want manual", mvp1.SourceType)
	}
}

// 회사 URL 등록 전에 직접 제출한 MVP도 이후 자동 동기화가 덮어쓰지 않는다.
func TestMilestone_ManualSubmitNotOverwrittenByLaterAutoDetect(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "ms-manualfirst@test.com", "선제출", "20251102", "ms_manualfirst_co")

	r := ts.post("/api/milestones", map[string]string{
		"type": "mvp1", "url": "https://manual-first.vercel.app",
	}, token)
	if !r.Success {
		t.Fatalf("manual submit: %v", r.Error)
	}

	// 이후 회사 service_url 등록 → 자동 후보 발생.
	updateCompanyServiceURL(t, ts, token, cid, "https://auto-later.vercel.app")

	got := getMyMilestones(t, ts, token)
	mvp1 := findMilestone(t, got, "mvp1")
	if mvp1 == nil {
		t.Fatal("mvp1 missing")
	}
	if mvp1.URL != "https://manual-first.vercel.app" {
		t.Errorf("mvp1.url = %q, want manual URL preserved", mvp1.URL)
	}
}

// rejected 자동 항목은 URL이 그대로면 상태·교수 코멘트가 유지된다.
func TestMilestone_RejectedAutoRowKeepsStatusAndNote(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)
	token, cid := createUserWithCompany(t, ts, "ms-rejkeep@test.com", "반려학생", "20251103", "ms_rejkeep_co")

	updateCompanyServiceURL(t, ts, token, cid, "https://needs-work.vercel.app")
	got := getMyMilestones(t, ts, token)
	mvp1 := findMilestone(t, got, "mvp1")
	if mvp1 == nil {
		t.Fatal("precondition: mvp1 not auto-detected")
	}

	rej := ts.post(fmt.Sprintf("/api/admin/milestones/%d/reject", mvp1.ID),
		map[string]string{"admin_note": "링크 확인 필요"}, adminToken)
	if !rej.Success {
		t.Fatalf("reject: %v", rej.Error)
	}

	// 재조회 (SyncAuto, 같은 URL) — rejected 상태와 코멘트가 유지되어야 함.
	got = getMyMilestones(t, ts, token)
	mvp1 = findMilestone(t, got, "mvp1")
	if mvp1 == nil {
		t.Fatal("mvp1 missing after reject")
	}
	if mvp1.Status != "rejected" {
		t.Errorf("mvp1.status = %q, want rejected (sync reset it to pending)", mvp1.Status)
	}
	if mvp1.AdminNote != "링크 확인 필요" {
		t.Errorf("mvp1.admin_note = %q, want preserved note", mvp1.AdminNote)
	}
}

// 자동 항목의 회사 URL이 실제로 바뀌면 동기화는 계속 동작한다.
func TestMilestone_AutoURLChangeStillSyncs(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "ms-autochg@test.com", "변경학생", "20251104", "ms_autochg_co")

	updateCompanyServiceURL(t, ts, token, cid, "https://old-version.vercel.app")
	got := getMyMilestones(t, ts, token)
	if mvp1 := findMilestone(t, got, "mvp1"); mvp1 == nil || mvp1.URL != "https://old-version.vercel.app" {
		t.Fatalf("precondition: mvp1 = %+v", mvp1)
	}

	updateCompanyServiceURL(t, ts, token, cid, "https://new-version.vercel.app")
	got = getMyMilestones(t, ts, token)
	mvp1 := findMilestone(t, got, "mvp1")
	if mvp1 == nil {
		t.Fatal("mvp1 missing")
	}
	if mvp1.URL != "https://new-version.vercel.app" {
		t.Errorf("mvp1.url = %q, want updated auto URL", mvp1.URL)
	}
	if mvp1.Status != "pending" {
		t.Errorf("mvp1.status = %q, want pending (URL changed → re-review)", mvp1.Status)
	}
}
