package persistence

import (
	"database/sql"
	"fmt"

	"github.com/earnlearning/backend/internal/domain/milestone"
)

type MilestoneRepo struct {
	db *sql.DB
}

func NewMilestoneRepo(db *sql.DB) *MilestoneRepo {
	return &MilestoneRepo{db: db}
}

// Upsert — INSERT or UPDATE on (student_id, milestone_type).
// On UPDATE: source/url/content 갱신하되 status는 항상 pending으로 reset
// (학생이 제출 내용을 바꿨으면 admin이 다시 봐야 하므로).
// admin_note 는 비움 (이전 코멘트가 새 제출에 안 어울릴 수 있음).
func (r *MilestoneRepo) Upsert(m *milestone.Milestone) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO student_milestones
			(student_id, milestone_type, source_type, source_ref_id, url, content, status, admin_note, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'pending', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(student_id, milestone_type) DO UPDATE SET
			source_type   = excluded.source_type,
			source_ref_id = excluded.source_ref_id,
			url           = excluded.url,
			content       = excluded.content,
			status        = 'pending',
			admin_note    = '',
			approved_by   = NULL,
			approved_at   = NULL,
			updated_at    = CURRENT_TIMESTAMP`,
		m.StudentID, m.Type, m.SourceType, m.SourceRefID, m.URL, m.Content,
	)
	if err != nil {
		return 0, fmt.Errorf("upsert milestone: %w", err)
	}
	// LastInsertId returns 0 on UPDATE in sqlite; fetch by lookup.
	cur, err := r.FindByStudentAndType(m.StudentID, m.Type)
	if err != nil {
		return 0, err
	}
	if cur == nil {
		// shouldn't happen
		id, _ := res.LastInsertId()
		return int(id), nil
	}
	return cur.ID, nil
}

func (r *MilestoneRepo) FindByStudentAndType(studentID int, typ milestone.Type) (*milestone.Milestone, error) {
	row := r.db.QueryRow(`
		SELECT id, student_id, milestone_type, source_type, source_ref_id,
		       url, content, status, admin_note, approved_by, approved_at,
		       ai_score, ai_reasoning, ai_signals, ai_evaluated_at,
		       created_at, updated_at
		FROM student_milestones
		WHERE student_id = ? AND milestone_type = ?`,
		studentID, typ,
	)
	return scanMilestone(row)
}

func (r *MilestoneRepo) FindByID(id int) (*milestone.Milestone, error) {
	row := r.db.QueryRow(`
		SELECT id, student_id, milestone_type, source_type, source_ref_id,
		       url, content, status, admin_note, approved_by, approved_at,
		       ai_score, ai_reasoning, ai_signals, ai_evaluated_at,
		       created_at, updated_at
		FROM student_milestones
		WHERE id = ?`, id,
	)
	m, err := scanMilestone(row)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, milestone.ErrNotFound
	}
	return m, nil
}

func (r *MilestoneRepo) ListByStudent(studentID int) ([]*milestone.Milestone, error) {
	rows, err := r.db.Query(`
		SELECT id, student_id, milestone_type, source_type, source_ref_id,
		       url, content, status, admin_note, approved_by, approved_at,
		       ai_score, ai_reasoning, ai_signals, ai_evaluated_at,
		       created_at, updated_at
		FROM student_milestones
		WHERE student_id = ?
		ORDER BY milestone_type`, studentID,
	)
	if err != nil {
		return nil, fmt.Errorf("list milestones: %w", err)
	}
	defer rows.Close()

	var out []*milestone.Milestone
	for rows.Next() {
		m, err := scanRowMilestone(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (r *MilestoneRepo) UpdateStatus(id int, status milestone.Status, adminNote string, adminID int) error {
	var res sql.Result
	var err error
	if status == milestone.StatusApproved {
		res, err = r.db.Exec(`
			UPDATE student_milestones
			SET status = ?, admin_note = ?, approved_by = ?, approved_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`, status, adminNote, adminID, id)
	} else {
		res, err = r.db.Exec(`
			UPDATE student_milestones
			SET status = ?, admin_note = ?, approved_by = NULL, approved_at = NULL, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`, status, adminNote, id)
	}
	if err != nil {
		return fmt.Errorf("update milestone status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return milestone.ErrNotFound
	}
	return nil
}

// rowScanner abstracts *sql.Row and *sql.Rows for shared scan logic.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanMilestone(row *sql.Row) (*milestone.Milestone, error) {
	m, err := scanRowMilestone(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

func scanRowMilestone(row rowScanner) (*milestone.Milestone, error) {
	m := &milestone.Milestone{}
	var sourceRefID sql.NullInt64
	var approvedBy sql.NullInt64
	var approvedAt sql.NullTime
	var aiScore sql.NullInt64
	var aiEvaluatedAt sql.NullTime
	if err := row.Scan(
		&m.ID, &m.StudentID, &m.Type, &m.SourceType, &sourceRefID,
		&m.URL, &m.Content, &m.Status, &m.AdminNote, &approvedBy, &approvedAt,
		&aiScore, &m.AIReasoning, &m.AISignals, &aiEvaluatedAt,
		&m.CreatedAt, &m.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if sourceRefID.Valid {
		v := int(sourceRefID.Int64)
		m.SourceRefID = &v
	}
	if approvedBy.Valid {
		v := int(approvedBy.Int64)
		m.ApprovedBy = &v
	}
	if approvedAt.Valid {
		t := approvedAt.Time
		m.ApprovedAt = &t
	}
	if aiScore.Valid {
		v := int(aiScore.Int64)
		m.AIScore = &v
	}
	if aiEvaluatedAt.Valid {
		t := aiEvaluatedAt.Time
		m.AIEvaluatedAt = &t
	}
	return m, nil
}

// =============================================================================
// #125 — business_plan 비공개 첨부 파일
// =============================================================================

func (r *MilestoneRepo) AddFile(f *milestone.FileRef) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO milestone_files (student_id, milestone_type, filename, stored_name, mime_type, size, path, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		f.StudentID, f.Type, f.Filename, f.StoredName, f.MimeType, f.Size, f.Path,
	)
	if err != nil {
		return 0, fmt.Errorf("add milestone file: %w", err)
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (r *MilestoneRepo) ListFiles(studentID int, typ milestone.Type) ([]*milestone.FileRef, error) {
	rows, err := r.db.Query(`
		SELECT id, student_id, milestone_type, filename, stored_name, mime_type, size, path, created_at
		FROM milestone_files
		WHERE student_id = ? AND milestone_type = ?
		ORDER BY created_at ASC, id ASC`, studentID, typ,
	)
	if err != nil {
		return nil, fmt.Errorf("list milestone files: %w", err)
	}
	defer rows.Close()
	var out []*milestone.FileRef
	for rows.Next() {
		f, err := scanFileRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

func (r *MilestoneRepo) FindFileByID(id int) (*milestone.FileRef, error) {
	row := r.db.QueryRow(`
		SELECT id, student_id, milestone_type, filename, stored_name, mime_type, size, path, created_at
		FROM milestone_files WHERE id = ?`, id,
	)
	f, err := scanFileRow(row)
	if err == sql.ErrNoRows {
		return nil, milestone.ErrFileNotFound
	}
	return f, err
}

func (r *MilestoneRepo) DeleteFile(id int) error {
	res, err := r.db.Exec(`DELETE FROM milestone_files WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete milestone file: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return milestone.ErrFileNotFound
	}
	return nil
}

func scanFileRow(row rowScanner) (*milestone.FileRef, error) {
	f := &milestone.FileRef{}
	if err := row.Scan(
		&f.ID, &f.StudentID, &f.Type, &f.Filename, &f.StoredName,
		&f.MimeType, &f.Size, &f.Path, &f.CreatedAt,
	); err != nil {
		return nil, err
	}
	return f, nil
}

// ListStudentAssets — 전체 승인 학생의 (승인 milestone 개수, 총자산).
// 총자산 = Cash + StockValue + CompanyEquity − Debt (GetAssetBreakdown 와 동일 공식).
func (r *MilestoneRepo) ListStudentAssets() ([]milestone.StudentAsset, error) {
	rows, err := r.db.Query(`
		SELECT u.id,
		  (SELECT COUNT(*) FROM student_milestones sm
		     WHERE sm.student_id = u.id AND sm.status = 'approved') AS approved_count,
		  COALESCE((SELECT balance FROM wallets w WHERE w.user_id = u.id), 0)
		  + COALESCE((SELECT SUM(s.shares * c.valuation / c.total_shares)
		     FROM shareholders s JOIN companies c ON c.id = s.company_id
		     WHERE s.user_id = u.id AND c.status = 'active'), 0)
		  + COALESCE((SELECT SUM(cw.balance * s.shares / c.total_shares)
		     FROM shareholders s JOIN companies c ON c.id = s.company_id
		     JOIN company_wallets cw ON cw.company_id = c.id
		     WHERE s.user_id = u.id AND c.status = 'active'), 0)
		  - COALESCE((SELECT SUM(remaining) FROM loans
		     WHERE borrower_id = u.id AND status IN ('active','overdue')), 0)
		  AS total_asset
		FROM users u
		WHERE u.role = 'student' AND u.status = 'approved'`,
	)
	if err != nil {
		return nil, fmt.Errorf("list student assets: %w", err)
	}
	defer rows.Close()
	var out []milestone.StudentAsset
	for rows.Next() {
		var a milestone.StudentAsset
		if err := rows.Scan(&a.StudentID, &a.ApprovedCount, &a.TotalAsset); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// UpdateAIScore — 회고 에세이 평가 결과 저장.
func (r *MilestoneRepo) UpdateAIScore(id int, score int, reasoning, signalsJSON string) error {
	res, err := r.db.Exec(`
		UPDATE student_milestones
		SET ai_score = ?, ai_reasoning = ?, ai_signals = ?, ai_evaluated_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		score, reasoning, signalsJSON, id,
	)
	if err != nil {
		return fmt.Errorf("update ai score: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return milestone.ErrNotFound
	}
	return nil
}
