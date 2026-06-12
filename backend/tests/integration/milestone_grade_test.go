package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// progressWithAsset — #125 asset/percentile 필드 포함 파싱.
type progressWithAsset struct {
	ApprovedCount   int    `json:"approved_count"`
	Group           string `json:"group"`
	AssetTotal      int    `json:"asset_total"`
	GroupSize       int    `json:"group_size"`
	AssetRank       int    `json:"asset_rank"`
	AssetPercentile int    `json:"asset_percentile"`
	Milestones      []struct {
		ID   int    `json:"id"`
		Type string `json:"type"`
	} `json:"milestones"`
}

func fetchProgress(t *testing.T, ts *testServer, token string) *progressWithAsset {
	t.Helper()
	r := ts.get("/api/milestones/mine", token)
	if !r.Success {
		t.Fatalf("get mine: %v", r.Error)
	}
	var p progressWithAsset
	if err := json.Unmarshal(r.Data, &p); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, string(r.Data))
	}
	return &p
}

// TestMilestoneAssetPercentile — 성적 그레이드 + 같은 그룹 내 자산 percentile (#125).
func TestMilestoneAssetPercentile(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)

	mkStudent := func(email, name, sid string) (string, int) {
		r := ts.register(email, "pass1234", name, sid)
		if !r.Success {
			t.Fatalf("register %s: %v", email, r.Error)
		}
		var d struct {
			User struct {
				ID int `json:"id"`
			} `json:"user"`
		}
		json.Unmarshal(r.Data, &d)
		ts.approveUser(adminToken, d.User.ID)
		return ts.login(email, "pass1234"), d.User.ID
	}
	give := func(id, amount int) {
		r := ts.post("/api/admin/wallet/transfer", map[string]interface{}{
			"target_user_ids": []int{id}, "amount": amount, "description": "test asset",
		}, adminToken)
		if !r.Success {
			t.Fatalf("transfer to %d: %v", id, r.Error)
		}
	}

	// 3명 모두 0 approved → 그룹 "" (미진입). 자산만 다르게.
	tokA, idA := mkStudent("grade-a@test.com", "에이", "20250301")
	tokB, idB := mkStudent("grade-b@test.com", "비", "20250302")
	tokC, idC := mkStudent("grade-c@test.com", "씨", "20250303")
	give(idA, 1000000)
	give(idB, 500000)
	give(idC, 200000)

	// D: 다른 그룹(2 approved → C 그룹) + 최대 자산. 그룹 격리 검증용.
	tokD, idD := mkStudent("grade-d@test.com", "디", "20250304")
	give(idD, 2000000)
	// D 가 business_plan + retrospective 제출 후 admin 승인 → 2 approved.
	ts.post("/api/milestones", map[string]interface{}{"type": "business_plan", "content": "사업계획 내용"}, tokD)
	ts.post("/api/milestones", map[string]interface{}{"type": "retrospective", "content": "회고 내용입니다 충분히 길게 작성"}, tokD)
	dProg := fetchProgress(t, ts, tokD)
	for _, m := range dProg.Milestones {
		if m.Type == "business_plan" || m.Type == "retrospective" {
			r := ts.post(fmt.Sprintf("/api/admin/milestones/%d/approve", m.ID), map[string]string{}, adminToken)
			if !r.Success {
				t.Fatalf("approve %d: %v", m.ID, r.Error)
			}
		}
	}

	t.Run("A is rank 1 of 3 in group '' (top 34%)", func(t *testing.T) {
		p := fetchProgress(t, ts, tokA)
		if p.Group != "" {
			t.Errorf("group=%q, want '' (0 approved)", p.Group)
		}
		if p.AssetTotal != 1000000 {
			t.Errorf("asset_total=%d, want 1000000", p.AssetTotal)
		}
		if p.GroupSize != 3 {
			t.Errorf("group_size=%d, want 3 (D in different group must be excluded)", p.GroupSize)
		}
		if p.AssetRank != 1 {
			t.Errorf("asset_rank=%d, want 1", p.AssetRank)
		}
		if p.AssetPercentile != 34 { // ceil(1/3*100)
			t.Errorf("asset_percentile=%d, want 34", p.AssetPercentile)
		}
	})

	t.Run("B is rank 2 of 3 (top 67%)", func(t *testing.T) {
		p := fetchProgress(t, ts, tokB)
		if p.AssetRank != 2 || p.GroupSize != 3 {
			t.Errorf("rank=%d size=%d, want 2/3", p.AssetRank, p.GroupSize)
		}
		if p.AssetPercentile != 67 { // ceil(2/3*100)
			t.Errorf("percentile=%d, want 67", p.AssetPercentile)
		}
	})

	t.Run("C is rank 3 of 3 (top 100%)", func(t *testing.T) {
		p := fetchProgress(t, ts, tokC)
		if p.AssetRank != 3 || p.AssetPercentile != 100 {
			t.Errorf("rank=%d pct=%d, want 3/100", p.AssetRank, p.AssetPercentile)
		}
	})

	t.Run("D is grade C, alone in its group (size 1, top 100%)", func(t *testing.T) {
		p := fetchProgress(t, ts, tokD)
		if p.ApprovedCount != 2 {
			t.Fatalf("D approved_count=%d, want 2", p.ApprovedCount)
		}
		if p.Group != "C" {
			t.Errorf("D group=%q, want C", p.Group)
		}
		if p.GroupSize != 1 {
			t.Errorf("D group_size=%d, want 1 (alone in C)", p.GroupSize)
		}
		if p.AssetPercentile != 100 {
			t.Errorf("D percentile=%d, want 100 (only member)", p.AssetPercentile)
		}
	})
}
