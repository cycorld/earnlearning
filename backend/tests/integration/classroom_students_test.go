package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

type studentSummaryData struct {
	UserID     int    `json:"user_id"`
	Name       string `json:"name"`
	Department string `json:"department"`
	StudentID  string `json:"student_id"`
	AvatarURL  string `json:"avatar_url"`
}

func TestClassroomStudents_ScopedAndPrivacyMinimized(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)
	classroomA := ts.createClassroom(adminToken, "학생 목록 A", 1_000_000)
	classroomB := ts.createClassroom(adminToken, "학생 목록 B", 1_000_000)

	requester := ts.registerAndApprove("roster-requester@test.com", "pass1234", "요청자", "20260001")
	peer := ts.registerAndApprove("roster-peer@test.com", "pass1234", "검색학생", "20260002")
	other := ts.registerAndApprove("roster-other@test.com", "pass1234", "다른강의", "20260003")
	ts.joinClassroom(requester, classroomA.Code)
	ts.joinClassroom(peer, classroomA.Code)
	ts.joinClassroom(other, classroomB.Code)

	r := ts.get(fmt.Sprintf("/api/classrooms/%d/students", classroomA.ID), requester)
	if !r.Success {
		t.Fatalf("list students failed: %v", r.Error)
	}
	var students []studentSummaryData
	if err := json.Unmarshal(r.Data, &students); err != nil {
		t.Fatalf("decode students: %v", err)
	}
	if len(students) != 1 || students[0].Name != "검색학생" || students[0].StudentID != "20260002" {
		t.Fatalf("students=%+v, want only approved peer in classroom A", students)
	}

	var raw []map[string]interface{}
	if err := json.Unmarshal(r.Data, &raw); err != nil {
		t.Fatalf("decode raw response: %v", err)
	}
	for _, forbidden := range []string{"email", "status", "balance", "total_asset", "joined_at"} {
		if _, exists := raw[0][forbidden]; exists {
			t.Errorf("privacy-minimized DTO unexpectedly contains %q", forbidden)
		}
	}
}

func TestClassroomStudents_RequiresMembershipExceptAdmin(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)
	classroom := ts.createClassroom(adminToken, "학생 목록 권한", 1_000_000)
	member := ts.registerAndApprove("roster-member@test.com", "pass1234", "멤버", "20260004")
	nonMember := ts.registerAndApprove("roster-nonmember@test.com", "pass1234", "비멤버", "20260005")
	ts.joinClassroom(member, classroom.Code)

	if r := ts.get(fmt.Sprintf("/api/classrooms/%d/students", classroom.ID), nonMember); r.Success {
		t.Fatal("non-member must not list classroom students")
	}
	if r := ts.get(fmt.Sprintf("/api/classrooms/%d/students", classroom.ID), adminToken); !r.Success {
		t.Fatalf("admin should list students without membership: %v", r.Error)
	}
}
