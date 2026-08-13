package integration

import (
	"net/http"
	"testing"
)

const canonicalAdminTestEmail = "cyc@snu.ac.kr"

func TestCompanyServices_AdminReconcileRybbit_CanonicalAdminOnlyAndClassroomScoped(t *testing.T) {
	ts := setupTestServer(t)
	studentToken, companyID, classroomID := createClassroomOwnerCompany(t, ts, "admin-scope-student@test.com", "학생", "20269991", "admin_scope_company")
	fake := &fakeProvisioner{}
	ts.companyServiceUC.SetProvisioner(fake)
	ts.companyServiceUC.SetChecker(okChecker{})

	svc := ts.createService(t, companyID, studentToken, "학생앱", "https://admin-scope.example.com")
	ts.validateService(t, companyID, svc.ID, studentToken)
	status, connected := ts.svcJSON(http.MethodPost, connectPath(companyID, svc.ID), nil, studentToken)
	if status != http.StatusOK || !connected.Success {
		t.Fatalf("student connect: status=%d error=%v", status, connected.Error)
	}

	adminToken := ts.login(testAdminEmail, testAdminPass)
	var adminID int
	if err := ts.db.QueryRow(`SELECT id FROM users WHERE email = ?`, testAdminEmail).Scan(&adminID); err != nil {
		t.Fatalf("find admin: %v", err)
	}
	if _, err := ts.db.Exec(`UPDATE users SET email = ?, active_classroom_id = ? WHERE id = ?`, canonicalAdminTestEmail, classroomID, adminID); err != nil {
		t.Fatalf("scope canonical admin: %v", err)
	}

	status, reconciled := ts.svcJSON(http.MethodPost, "/api/admin/analytics/rybbit/reconcile", nil, adminToken)
	if status != http.StatusOK || !reconciled.Success {
		t.Fatalf("admin reconcile: status=%d error=%v", status, reconciled.Error)
	}
	if fake.callCount() != 2 {
		t.Fatalf("provision calls=%d, want student connect + admin reconcile", fake.callCount())
	}
	last := fake.reqs[len(fake.reqs)-1]
	if last.OwnerUserID != adminID || last.OwnerEmail != canonicalAdminTestEmail || len(last.GrantServiceIDs) != 1 || last.GrantServiceIDs[0] != svc.ID {
		t.Fatalf("canonical admin request mismatch: %+v", last)
	}

	if _, err := ts.db.Exec(`UPDATE users SET email = ? WHERE id = ?`, testAdminEmail, adminID); err != nil {
		t.Fatalf("restore generic admin email: %v", err)
	}
	status, denied := ts.svcJSON(http.MethodPost, "/api/admin/analytics/rybbit/reconcile", nil, adminToken)
	assertErrCode(t, "generic admin", status, denied, http.StatusForbidden, "NOT_OWNER")
	if fake.callCount() != 2 {
		t.Fatalf("denied generic admin caused provision call")
	}
}
