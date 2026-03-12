package persistence

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func SeedAdmin(db *sql.DB, email, password string) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&count)
	if err != nil {
		return fmt.Errorf("check admin exists: %w", err)
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = db.Exec(`INSERT INTO users (email, password, name, department, student_id, role, status)
		VALUES (?, ?, '최용철', '관리자', '0000000000', 'admin', 'approved')`, email, string(hash))
	if err != nil {
		return fmt.Errorf("insert admin: %w", err)
	}

	// Create admin wallet
	var adminID int
	err = db.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&adminID)
	if err != nil {
		return fmt.Errorf("get admin id: %w", err)
	}

	_, err = db.Exec("INSERT OR IGNORE INTO wallets (user_id, balance) VALUES (?, 0)", adminID)
	if err != nil {
		return fmt.Errorf("create admin wallet: %w", err)
	}

	return nil
}
