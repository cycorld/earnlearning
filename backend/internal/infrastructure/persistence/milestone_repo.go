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
	if err := row.Scan(
		&m.ID, &m.StudentID, &m.Type, &m.SourceType, &sourceRefID,
		&m.URL, &m.Content, &m.Status, &m.AdminNote, &approvedBy, &approvedAt,
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
	return m, nil
}
