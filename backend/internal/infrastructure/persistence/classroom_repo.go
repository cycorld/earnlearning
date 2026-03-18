package persistence

import (
	"database/sql"
	"errors"

	"github.com/earnlearning/backend/internal/domain/classroom"
)

type ClassroomRepo struct {
	db *sql.DB
}

func NewClassroomRepo(db *sql.DB) *ClassroomRepo {
	return &ClassroomRepo{db: db}
}

func (r *ClassroomRepo) Create(c *classroom.Classroom) (int, error) {
	result, err := r.db.Exec(
		`INSERT INTO classrooms (name, code, created_by, initial_capital, settings)
		 VALUES (?, ?, ?, ?, ?)`,
		c.Name, c.Code, c.CreatedBy, c.InitialCapital, c.Settings,
	)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *ClassroomRepo) FindByID(id int) (*classroom.Classroom, error) {
	c := &classroom.Classroom{}
	err := r.db.QueryRow(
		`SELECT id, name, code, created_by, initial_capital, settings, created_at
		 FROM classrooms WHERE id = ?`, id,
	).Scan(&c.ID, &c.Name, &c.Code, &c.CreatedBy, &c.InitialCapital, &c.Settings, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("클래스룸을 찾을 수 없습니다")
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *ClassroomRepo) FindByCode(code string) (*classroom.Classroom, error) {
	c := &classroom.Classroom{}
	err := r.db.QueryRow(
		`SELECT id, name, code, created_by, initial_capital, settings, created_at
		 FROM classrooms WHERE code = ?`, code,
	).Scan(&c.ID, &c.Name, &c.Code, &c.CreatedBy, &c.InitialCapital, &c.Settings, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("클래스룸을 찾을 수 없습니다")
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *ClassroomRepo) AddMember(classroomID, userID int) error {
	_, err := r.db.Exec(
		"INSERT OR IGNORE INTO classroom_members (classroom_id, user_id) VALUES (?, ?)",
		classroomID, userID,
	)
	return err
}

func (r *ClassroomRepo) IsMember(classroomID, userID int) (bool, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM classroom_members WHERE classroom_id = ? AND user_id = ?",
		classroomID, userID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *ClassroomRepo) GetMembers(classroomID int) ([]*classroom.ClassroomMember, error) {
	rows, err := r.db.Query(
		`SELECT id, classroom_id, user_id, joined_at
		 FROM classroom_members WHERE classroom_id = ? ORDER BY joined_at`, classroomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*classroom.ClassroomMember
	for rows.Next() {
		m := &classroom.ClassroomMember{}
		if err := rows.Scan(&m.ID, &m.ClassroomID, &m.UserID, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *ClassroomRepo) CreateChannel(ch *classroom.Channel) (int, error) {
	result, err := r.db.Exec(
		`INSERT INTO channels (classroom_id, name, slug, channel_type, write_role, sort_order)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		ch.ClassroomID, ch.Name, ch.Slug, ch.ChannelType, ch.WriteRole, ch.SortOrder,
	)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *ClassroomRepo) GetChannels(classroomID int) ([]*classroom.Channel, error) {
	rows, err := r.db.Query(
		`SELECT id, classroom_id, name, slug, channel_type, write_role, sort_order
		 FROM channels WHERE classroom_id = ? ORDER BY sort_order`, classroomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*classroom.Channel
	for rows.Next() {
		ch := &classroom.Channel{}
		if err := rows.Scan(&ch.ID, &ch.ClassroomID, &ch.Name, &ch.Slug, &ch.ChannelType, &ch.WriteRole, &ch.SortOrder); err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}
	return channels, rows.Err()
}

func (r *ClassroomRepo) GetMemberDashboard(classroomID int) ([]*classroom.MemberDashboard, error) {
	rows, err := r.db.Query(`
		SELECT
			u.id, u.name, u.email, u.student_id, u.department, u.avatar_url, u.status,
			cm.joined_at,
			COALESCE(w.balance, 0) AS balance,
			COALESCE(w.balance, 0)
				+ COALESCE((SELECT SUM(sh.shares * COALESCE(
					(SELECT MAX(t.price) FROM trades t WHERE t.company_id = sh.company_id), co.valuation / NULLIF(co.total_shares, 0)
				)) FROM shareholders sh JOIN companies co ON co.id = sh.company_id WHERE sh.user_id = u.id), 0)
				- COALESCE((SELECT SUM(l.remaining) FROM loans l WHERE l.borrower_id = u.id AND l.status = 'active'), 0)
			AS total_asset,
			COALESCE((SELECT COUNT(*) FROM companies WHERE owner_id = u.id AND status = 'active'), 0) AS company_count,
			COALESCE((SELECT COUNT(*) FROM loans WHERE borrower_id = u.id AND status = 'active'), 0) AS loan_count,
			COALESCE((SELECT SUM(remaining) FROM loans WHERE borrower_id = u.id AND status = 'active'), 0) AS total_debt,
			COALESCE((SELECT COUNT(*) FROM posts p JOIN channels ch ON ch.id = p.channel_id WHERE ch.classroom_id = ? AND p.author_id = u.id), 0) AS post_count,
			COALESCE((SELECT GROUP_CONCAT(name, ', ') FROM companies WHERE owner_id = u.id AND status = 'active'), '') AS company_names
		FROM classroom_members cm
		JOIN users u ON u.id = cm.user_id
		LEFT JOIN wallets w ON w.user_id = u.id
		WHERE cm.classroom_id = ? AND u.role = 'student'
		ORDER BY total_asset DESC`,
		classroomID, classroomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*classroom.MemberDashboard
	for rows.Next() {
		m := &classroom.MemberDashboard{}
		if err := rows.Scan(
			&m.UserID, &m.Name, &m.Email, &m.StudentID, &m.Department, &m.AvatarURL, &m.Status,
			&m.JoinedAt, &m.Balance, &m.TotalAsset,
			&m.CompanyCount, &m.LoanCount, &m.TotalDebt, &m.PostCount, &m.CompanyNames,
		); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *ClassroomRepo) ListByUser(userID int) ([]*classroom.Classroom, error) {
	rows, err := r.db.Query(
		`SELECT c.id, c.name, c.code, c.created_by, c.initial_capital, c.settings, c.created_at
		 FROM classrooms c
		 INNER JOIN classroom_members cm ON cm.classroom_id = c.id
		 WHERE cm.user_id = ?
		 ORDER BY c.created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var classrooms []*classroom.Classroom
	for rows.Next() {
		c := &classroom.Classroom{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Code, &c.CreatedBy, &c.InitialCapital, &c.Settings, &c.CreatedAt); err != nil {
			return nil, err
		}
		classrooms = append(classrooms, c)
	}
	return classrooms, rows.Err()
}
