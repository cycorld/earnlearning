package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestGrantApplicationEditDelete(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)

	// Create classroom and student
	crResp := ts.post("/api/classrooms", map[string]interface{}{
		"name": "과제수정테스트반", "initial_capital": 1000000,
	}, adminToken)
	var cr struct {
		ID   int    `json:"id"`
		Code string `json:"code"`
	}
	json.Unmarshal(crResp.Data, &cr)

	studentToken := ts.registerAndApprove("grant-edit@test.com", "pass1234", "수정학생", "2024020")
	ts.post("/api/classrooms/join", map[string]string{"code": cr.Code}, studentToken)

	student2Token := ts.registerAndApprove("grant-edit2@test.com", "pass1234", "다른학생", "2024021")
	ts.post("/api/classrooms/join", map[string]string{"code": cr.Code}, student2Token)

	// Create grant
	grantID, _ := createGrant(ts, adminToken, map[string]interface{}{
		"title": "수정삭제 테스트 과제", "reward": 5000,
	})

	// Student applies
	appID, _ := applyGrant(ts, studentToken, grantID)

	t.Run("본인이 pending 지원서를 수정할 수 있다", func(t *testing.T) {
		r := ts.put(fmt.Sprintf("/api/grants/%d/applications/%d", grantID, appID),
			map[string]string{"proposal": "수정된 지원서 내용"}, studentToken)
		if !r.Success {
			t.Fatalf("update should succeed: %v", r.Error)
		}

		// Verify updated
		g := getGrant(ts, studentToken, grantID)
		apps := g["applications"].([]interface{})
		app := apps[0].(map[string]interface{})
		if app["proposal"] != "수정된 지원서 내용" {
			t.Errorf("expected updated proposal, got %s", app["proposal"])
		}
	})

	t.Run("다른 사용자가 지원서를 수정할 수 없다", func(t *testing.T) {
		r := ts.put(fmt.Sprintf("/api/grants/%d/applications/%d", grantID, appID),
			map[string]string{"proposal": "해킹 시도"}, student2Token)
		if r.Success {
			t.Error("should not allow other user to update")
		}
	})

	t.Run("본인이 pending 지원서를 삭제할 수 있다", func(t *testing.T) {
		// Student2 applies then deletes
		app2ID, _ := applyGrant(ts, student2Token, grantID)

		r := ts.delete(fmt.Sprintf("/api/grants/%d/applications/%d", grantID, app2ID), student2Token)
		if !r.Success {
			t.Fatalf("delete should succeed: %v", r.Error)
		}

		// Verify deleted
		g := getGrant(ts, studentToken, grantID)
		apps := g["applications"].([]interface{})
		if len(apps) != 1 {
			t.Errorf("expected 1 application after delete, got %d", len(apps))
		}
	})

	t.Run("승인된 지원서는 수정할 수 없다", func(t *testing.T) {
		// Approve the application
		ts.post(fmt.Sprintf("/api/admin/grants/%d/approve/%d", grantID, appID), nil, adminToken)

		r := ts.put(fmt.Sprintf("/api/grants/%d/applications/%d", grantID, appID),
			map[string]string{"proposal": "승인 후 수정 시도"}, studentToken)
		if r.Success {
			t.Error("should not allow editing approved application")
		}
	})

	t.Run("승인된 지원서는 삭제할 수 없다", func(t *testing.T) {
		r := ts.delete(fmt.Sprintf("/api/grants/%d/applications/%d", grantID, appID), studentToken)
		if r.Success {
			t.Error("should not allow deleting approved application")
		}
	})

	t.Run("관리자가 승인을 취소하면 status가 pending으로 돌아간다", func(t *testing.T) {
		r := ts.post(fmt.Sprintf("/api/admin/grants/%d/revoke/%d", grantID, appID), nil, adminToken)
		if !r.Success {
			t.Fatalf("revoke should succeed: %v", r.Error)
		}

		// Verify status reverted to pending
		g := getGrant(ts, adminToken, grantID)
		apps := g["applications"].([]interface{})
		for _, a := range apps {
			app := a.(map[string]interface{})
			if int(app["id"].(float64)) == appID {
				if app["status"] != "pending" {
					t.Errorf("expected status=pending after revoke, got %s", app["status"])
				}
			}
		}
	})

	t.Run("승인 취소 후 다시 수정할 수 있다", func(t *testing.T) {
		r := ts.put(fmt.Sprintf("/api/grants/%d/applications/%d", grantID, appID),
			map[string]string{"proposal": "승인 취소 후 수정"}, studentToken)
		if !r.Success {
			t.Fatalf("should allow editing after revoke: %v", r.Error)
		}
	})

	t.Run("pending 상태의 지원서는 승인 취소할 수 없다", func(t *testing.T) {
		r := ts.post(fmt.Sprintf("/api/admin/grants/%d/revoke/%d", grantID, appID), nil, adminToken)
		if r.Success {
			t.Error("should not allow revoking a pending application")
		}
	})
}
