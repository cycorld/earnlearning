package application

import (
	"errors"
	"testing"
	"time"

	"github.com/earnlearning/backend/internal/domain/userdb"
	"github.com/earnlearning/backend/internal/infrastructure/userdbadmin"
)

// fakeUserDBRepo — 메모리 기반 가짜 repo. AdminReconcile 테스트 용 최소 구현.
type fakeUserDBRepo struct {
	rows    []*userdb.UserDatabase
	deleted []int
}

func (f *fakeUserDBRepo) Create(*userdb.UserDatabase) (int, error)                  { return 0, nil }
func (f *fakeUserDBRepo) FindByID(id int) (*userdb.UserDatabase, error)             { return nil, userdb.ErrNotFound }
func (f *fakeUserDBRepo) FindByUserIDAndProject(int, string) (*userdb.UserDatabase, error) {
	return nil, userdb.ErrNotFound
}
func (f *fakeUserDBRepo) FindByDBName(name string) (*userdb.UserDatabase, error) {
	for _, r := range f.rows {
		if r.DBName == name {
			return r, nil
		}
	}
	return nil, userdb.ErrNotFound
}
func (f *fakeUserDBRepo) ListByUserID(int) ([]*userdb.UserDatabase, error) { return nil, nil }
func (f *fakeUserDBRepo) ListAll() ([]*userdb.UserDatabase, error)         { return f.rows, nil }
func (f *fakeUserDBRepo) CountByUserID(int) (int, error)                   { return 0, nil }
func (f *fakeUserDBRepo) MarkRotated(int) error                            { return nil }
func (f *fakeUserDBRepo) Delete(id int) error {
	f.deleted = append(f.deleted, id)
	out := f.rows[:0]
	for _, r := range f.rows {
		if r.ID != id {
			out = append(out, r)
		}
	}
	f.rows = out
	return nil
}

// fakeProvisioner — DBExists 만 의미 있게 구현.
type fakeProvisioner struct {
	exists map[string]bool
	err    error
}

func (p *fakeProvisioner) Create(string, string) (*userdbadmin.CreatedDB, error) { return nil, nil }
func (p *fakeProvisioner) Delete(string, string) error                            { return nil }
func (p *fakeProvisioner) Rotate(string) (string, error)                          { return "", nil }
func (p *fakeProvisioner) DBExists(name string) (bool, error) {
	if p.err != nil {
		return false, p.err
	}
	return p.exists[name], nil
}

func TestAdminReconcile(t *testing.T) {
	repo := &fakeUserDBRepo{
		rows: []*userdb.UserDatabase{
			{ID: 1, UserID: 10, DBName: "alice_proj1", CreatedAt: time.Now()},
			{ID: 2, UserID: 10, DBName: "alice_proj2_orphan", CreatedAt: time.Now()},
			{ID: 3, UserID: 11, DBName: "bob_proj1", CreatedAt: time.Now()},
		},
	}
	prov := &fakeProvisioner{exists: map[string]bool{
		"alice_proj1": true,
		"bob_proj1":   true,
	}}
	uc := NewUserDBUseCase(repo, prov, nil, 3)

	res, err := uc.AdminReconcile()
	if err != nil {
		t.Fatalf("AdminReconcile: %v", err)
	}
	if res.Checked != 3 {
		t.Errorf("checked = %d, want 3", res.Checked)
	}
	if res.Removed != 1 {
		t.Errorf("removed = %d, want 1", res.Removed)
	}
	if res.Errors != 0 {
		t.Errorf("errors = %d, want 0", res.Errors)
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != 2 {
		t.Errorf("deleted = %v, want [2]", repo.deleted)
	}
}

func TestAdminReconcile_ProvisionerError(t *testing.T) {
	repo := &fakeUserDBRepo{
		rows: []*userdb.UserDatabase{{ID: 1, DBName: "x"}},
	}
	prov := &fakeProvisioner{err: errors.New("pg down")}
	uc := NewUserDBUseCase(repo, prov, nil, 3)

	res, err := uc.AdminReconcile()
	if err != nil {
		t.Fatalf("AdminReconcile: %v", err)
	}
	if res.Errors != 1 || res.Removed != 0 {
		t.Errorf("res = %+v, want errors=1 removed=0", res)
	}
	if len(repo.deleted) != 0 {
		t.Errorf("should not delete on error, got %v", repo.deleted)
	}
}

func TestAdminDeleteByDBName_OrphanCase(t *testing.T) {
	repo := &fakeUserDBRepo{
		rows: []*userdb.UserDatabase{{ID: 5, DBName: "orphan_db", PGUsername: "orphan_user"}},
	}
	prov := &fakeProvisioner{exists: map[string]bool{}}
	uc := NewUserDBUseCase(repo, prov, nil, 3)

	if err := uc.AdminDeleteByDBName("orphan_db"); err != nil {
		t.Fatalf("AdminDeleteByDBName: %v", err)
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != 5 {
		t.Errorf("deleted = %v, want [5]", repo.deleted)
	}
}

func TestAdminDeleteByDBName_NotFound(t *testing.T) {
	repo := &fakeUserDBRepo{rows: nil}
	prov := &fakeProvisioner{}
	uc := NewUserDBUseCase(repo, prov, nil, 3)

	err := uc.AdminDeleteByDBName("nope")
	if !errors.Is(err, userdb.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
