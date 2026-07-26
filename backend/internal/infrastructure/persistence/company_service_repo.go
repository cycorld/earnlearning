package persistence

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/earnlearning/backend/internal/domain/company"
)

// CompanyServiceRepo — #181 회사 등록 서비스 영속성.
type CompanyServiceRepo struct {
	db *sql.DB
}

func NewCompanyServiceRepo(db *sql.DB) *CompanyServiceRepo {
	return &CompanyServiceRepo{db: db}
}

// companyServiceCols — 조회 공통 컬럼 (스캔 순서와 반드시 일치).
const companyServiceCols = `id, company_id, name, url, normalized_url,
	validation_status, validation_checked_at, validation_detail,
	rybbit_status, rybbit_site_id, rybbit_connected_at, created_at, updated_at`

func scanCompanyService(row interface {
	Scan(dest ...interface{}) error
}) (*company.CompanyService, error) {
	s := &company.CompanyService{}
	var checkedAt, connectedAt sql.NullTime
	err := row.Scan(
		&s.ID, &s.CompanyID, &s.Name, &s.URL, &s.NormalizedURL,
		&s.ValidationStatus, &checkedAt, &s.ValidationDetail,
		&s.RybbitStatus, &s.RybbitSiteID, &connectedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, company.ErrServiceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query company service: %w", err)
	}
	if checkedAt.Valid {
		t := checkedAt.Time
		s.ValidationCheckedAt = &t
	}
	if connectedAt.Valid {
		t := connectedAt.Time
		s.RybbitConnectedAt = &t
	}
	return s, nil
}

// isUniqueViolation — SQLite UNIQUE 인덱스 위반 판정 (중복 URL 을 409 로 매핑하기 위함).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (r *CompanyServiceRepo) Create(s *company.CompanyService) (int, error) {
	if s.ValidationStatus == "" {
		s.ValidationStatus = company.ValidationUnvalidated
	}
	if s.RybbitStatus == "" {
		s.RybbitStatus = company.RybbitNotConnected
	}
	res, err := r.db.Exec(`
		INSERT INTO company_services
			(company_id, name, url, normalized_url, validation_status,
			 validation_checked_at, validation_detail, rybbit_status, rybbit_site_id, rybbit_connected_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.CompanyID, s.Name, s.URL, s.NormalizedURL, s.ValidationStatus,
		s.ValidationCheckedAt, s.ValidationDetail, s.RybbitStatus, s.RybbitSiteID, s.RybbitConnectedAt,
	)
	if isUniqueViolation(err) {
		return 0, company.ErrDuplicateServiceURL
	}
	if err != nil {
		return 0, fmt.Errorf("insert company service: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("insert company service id: %w", err)
	}
	return int(id), nil
}

func (r *CompanyServiceRepo) FindByID(id int) (*company.CompanyService, error) {
	return scanCompanyService(r.db.QueryRow(
		`SELECT `+companyServiceCols+` FROM company_services WHERE id = ?`, id))
}

func (r *CompanyServiceRepo) FindByCompanyID(companyID int) ([]*company.CompanyService, error) {
	rows, err := r.db.Query(
		`SELECT `+companyServiceCols+` FROM company_services WHERE company_id = ? ORDER BY id ASC`,
		companyID)
	if err != nil {
		return nil, fmt.Errorf("query company services: %w", err)
	}
	defer rows.Close()

	list := make([]*company.CompanyService, 0)
	for rows.Next() {
		s, err := scanCompanyService(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *CompanyServiceRepo) FindByCompanyAndNormalizedURL(companyID int, normalizedURL string) (*company.CompanyService, error) {
	return scanCompanyService(r.db.QueryRow(
		`SELECT `+companyServiceCols+` FROM company_services
		 WHERE company_id = ? AND normalized_url = ?`, companyID, normalizedURL))
}

func (r *CompanyServiceRepo) Update(s *company.CompanyService) error {
	res, err := r.db.Exec(`
		UPDATE company_services SET
			name = ?, url = ?, normalized_url = ?,
			validation_status = ?, validation_checked_at = ?, validation_detail = ?,
			rybbit_status = ?, rybbit_site_id = ?, rybbit_connected_at = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		s.Name, s.URL, s.NormalizedURL,
		s.ValidationStatus, s.ValidationCheckedAt, s.ValidationDetail,
		s.RybbitStatus, s.RybbitSiteID, s.RybbitConnectedAt, s.ID,
	)
	if isUniqueViolation(err) {
		return company.ErrDuplicateServiceURL
	}
	if err != nil {
		return fmt.Errorf("update company service: %w", err)
	}
	return affectedOrNotFound(res)
}

func (r *CompanyServiceRepo) UpdateValidation(id int, status string, checkedAt *time.Time, detail string) error {
	res, err := r.db.Exec(`
		UPDATE company_services SET
			validation_status = ?, validation_checked_at = ?, validation_detail = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, status, checkedAt, detail, id)
	if err != nil {
		return fmt.Errorf("update company service validation: %w", err)
	}
	return affectedOrNotFound(res)
}

func (r *CompanyServiceRepo) UpdateRybbit(id int, status, siteID string, connectedAt *time.Time) error {
	res, err := r.db.Exec(`
		UPDATE company_services SET
			rybbit_status = ?, rybbit_site_id = ?, rybbit_connected_at = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, status, siteID, connectedAt, id)
	if err != nil {
		return fmt.Errorf("update company service rybbit: %w", err)
	}
	return affectedOrNotFound(res)
}

func (r *CompanyServiceRepo) Delete(id int) error {
	res, err := r.db.Exec(`DELETE FROM company_services WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete company service: %w", err)
	}
	return affectedOrNotFound(res)
}

// affectedOrNotFound — 영향 행이 0이면 ErrServiceNotFound (재삭제 → 404 매핑용).
func affectedOrNotFound(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("company service rows affected: %w", err)
	}
	if n == 0 {
		return company.ErrServiceNotFound
	}
	return nil
}
