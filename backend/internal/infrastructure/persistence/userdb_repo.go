package persistence

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/earnlearning/backend/internal/domain/userdb"
)

type UserDBRepo struct {
	db *sql.DB
}

func NewUserDBRepo(db *sql.DB) *UserDBRepo {
	return &UserDBRepo{db: db}
}

func (r *UserDBRepo) Create(u *userdb.UserDatabase) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO user_databases (user_id, project_name, db_name, pg_username, host, port)
		VALUES (?, ?, ?, ?, ?, ?)`,
		u.UserID, u.ProjectName, u.DBName, u.PGUsername, u.Host, u.Port,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return 0, userdb.ErrDuplicate
		}
		return 0, fmt.Errorf("create user_database: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *UserDBRepo) FindByID(id int) (*userdb.UserDatabase, error) {
	u := &userdb.UserDatabase{}
	var lastRotated sql.NullTime
	err := r.db.QueryRow(`
		SELECT id, user_id, project_name, db_name, pg_username, host, port, created_at, last_rotated
		FROM user_databases WHERE id = ?`, id).Scan(
		&u.ID, &u.UserID, &u.ProjectName, &u.DBName, &u.PGUsername, &u.Host, &u.Port, &u.CreatedAt, &lastRotated,
	)
	if err == sql.ErrNoRows {
		return nil, userdb.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user_database: %w", err)
	}
	if lastRotated.Valid {
		t := lastRotated.Time
		u.LastRotated = &t
	}
	return u, nil
}

func (r *UserDBRepo) FindByUserIDAndProject(userID int, projectName string) (*userdb.UserDatabase, error) {
	u := &userdb.UserDatabase{}
	var lastRotated sql.NullTime
	err := r.db.QueryRow(`
		SELECT id, user_id, project_name, db_name, pg_username, host, port, created_at, last_rotated
		FROM user_databases WHERE user_id = ? AND project_name = ?`, userID, projectName).Scan(
		&u.ID, &u.UserID, &u.ProjectName, &u.DBName, &u.PGUsername, &u.Host, &u.Port, &u.CreatedAt, &lastRotated,
	)
	if err == sql.ErrNoRows {
		return nil, userdb.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if lastRotated.Valid {
		t := lastRotated.Time
		u.LastRotated = &t
	}
	return u, nil
}

// FindByDBName — admin reconcile / 직접 삭제용 (#016).
func (r *UserDBRepo) FindByDBName(dbName string) (*userdb.UserDatabase, error) {
	u := &userdb.UserDatabase{}
	var lastRotated sql.NullTime
	err := r.db.QueryRow(`
		SELECT id, user_id, project_name, db_name, pg_username, host, port, created_at, last_rotated
		FROM user_databases WHERE db_name = ?`, dbName).Scan(
		&u.ID, &u.UserID, &u.ProjectName, &u.DBName, &u.PGUsername, &u.Host, &u.Port, &u.CreatedAt, &lastRotated,
	)
	if err == sql.ErrNoRows {
		return nil, userdb.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if lastRotated.Valid {
		t := lastRotated.Time
		u.LastRotated = &t
	}
	return u, nil
}

// ListAll — admin reconcile (#016).
func (r *UserDBRepo) ListAll() ([]*userdb.UserDatabase, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, project_name, db_name, pg_username, host, port, created_at, last_rotated
		FROM user_databases
		ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list all user_databases: %w", err)
	}
	defer rows.Close()
	var out []*userdb.UserDatabase
	for rows.Next() {
		u := &userdb.UserDatabase{}
		var lastRotated sql.NullTime
		if err := rows.Scan(&u.ID, &u.UserID, &u.ProjectName, &u.DBName, &u.PGUsername, &u.Host, &u.Port, &u.CreatedAt, &lastRotated); err != nil {
			return nil, err
		}
		if lastRotated.Valid {
			t := lastRotated.Time
			u.LastRotated = &t
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *UserDBRepo) ListByUserID(userID int) ([]*userdb.UserDatabase, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, project_name, db_name, pg_username, host, port, created_at, last_rotated
		FROM user_databases
		WHERE user_id = ?
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list user_databases: %w", err)
	}
	defer rows.Close()

	var out []*userdb.UserDatabase
	for rows.Next() {
		u := &userdb.UserDatabase{}
		var lastRotated sql.NullTime
		if err := rows.Scan(&u.ID, &u.UserID, &u.ProjectName, &u.DBName, &u.PGUsername, &u.Host, &u.Port, &u.CreatedAt, &lastRotated); err != nil {
			return nil, err
		}
		if lastRotated.Valid {
			t := lastRotated.Time
			u.LastRotated = &t
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *UserDBRepo) CountByUserID(userID int) (int, error) {
	var n int
	err := r.db.QueryRow("SELECT COUNT(*) FROM user_databases WHERE user_id = ?", userID).Scan(&n)
	return n, err
}

func (r *UserDBRepo) MarkRotated(id int) error {
	_, err := r.db.Exec("UPDATE user_databases SET last_rotated = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

func (r *UserDBRepo) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM user_databases WHERE id = ?", id)
	return err
}
