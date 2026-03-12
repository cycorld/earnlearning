package persistence

import (
	"database/sql"
	"strings"

	"github.com/earnlearning/backend/internal/domain/user"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(u *user.User) (int, error) {
	result, err := r.db.Exec(
		`INSERT INTO users (email, password, name, department, student_id, role, status, bio, avatar_url)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.Email, u.Password, u.Name, u.Department, u.StudentID,
		string(u.Role), string(u.Status), u.Bio, u.AvatarURL,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return 0, user.ErrDuplicateEmail
		}
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *UserRepo) FindByID(id int) (*user.User, error) {
	u := &user.User{}
	err := r.db.QueryRow(
		`SELECT id, email, password, name, department, student_id, role, status, bio, avatar_url, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Name, &u.Department, &u.StudentID,
		&u.Role, &u.Status, &u.Bio, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) FindByEmail(email string) (*user.User, error) {
	u := &user.User{}
	err := r.db.QueryRow(
		`SELECT id, email, password, name, department, student_id, role, status, bio, avatar_url, created_at, updated_at
		 FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Name, &u.Department, &u.StudentID,
		&u.Role, &u.Status, &u.Bio, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) FindByStatus(status user.Status) ([]*user.User, error) {
	rows, err := r.db.Query(
		`SELECT id, email, password, name, department, student_id, role, status, bio, avatar_url, created_at, updated_at
		 FROM users WHERE status = ? ORDER BY created_at DESC`, string(status),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanUsers(rows)
}

func (r *UserRepo) ListAll(page, limit int) ([]*user.User, int, error) {
	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(
		`SELECT id, email, password, name, department, student_id, role, status, bio, avatar_url, created_at, updated_at
		 FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users, err := scanUsers(rows)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *UserRepo) UpdateStatus(id int, status user.Status) error {
	result, err := r.db.Exec(
		"UPDATE users SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		string(status), id,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return user.ErrNotFound
	}
	return nil
}

func scanUsers(rows *sql.Rows) ([]*user.User, error) {
	var users []*user.User
	for rows.Next() {
		u := &user.User{}
		err := rows.Scan(&u.ID, &u.Email, &u.Password, &u.Name, &u.Department, &u.StudentID,
			&u.Role, &u.Status, &u.Bio, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
