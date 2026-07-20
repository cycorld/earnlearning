package integration

import (
	"os"
	"strings"
	"testing"

	"github.com/earnlearning/backend/internal/infrastructure/persistence"
)

// #159 레거시 wallets(user_id UNIQUE) 테이블을 강의실별 지갑 스키마로
// 리빌드하는 마이그레이션 검증. 프로덕션 DB 데이터 보존이 핵심.
func TestWalletMigration_RebuildPreservesData(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "earnlearning-mig-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	db, err := persistence.NewDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// 레거시 스키마 + 프로덕션 유사 데이터 구성 (RunMigrations 이전 상태 재현)
	legacy := []string{
		`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, email TEXT NOT NULL UNIQUE,
		 password_hash TEXT NOT NULL, name TEXT NOT NULL, role TEXT NOT NULL DEFAULT 'student',
		 status TEXT NOT NULL DEFAULT 'pending')`,
		`CREATE TABLE classrooms (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL,
		 code TEXT NOT NULL UNIQUE, created_by INTEGER, initial_capital INTEGER NOT NULL DEFAULT 0,
		 settings TEXT DEFAULT '{}', created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE classroom_members (id INTEGER PRIMARY KEY AUTOINCREMENT,
		 classroom_id INTEGER NOT NULL, user_id INTEGER NOT NULL,
		 joined_at DATETIME DEFAULT CURRENT_TIMESTAMP, UNIQUE(classroom_id, user_id))`,
		`CREATE TABLE wallets (id INTEGER PRIMARY KEY AUTOINCREMENT,
		 user_id INTEGER NOT NULL UNIQUE REFERENCES users(id), balance INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE transactions (id INTEGER PRIMARY KEY AUTOINCREMENT,
		 wallet_id INTEGER NOT NULL REFERENCES wallets(id), amount INTEGER NOT NULL,
		 balance_after INTEGER NOT NULL, tx_type TEXT NOT NULL, description TEXT DEFAULT '',
		 reference_type TEXT DEFAULT '', reference_id INTEGER DEFAULT 0,
		 created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
		`INSERT INTO users (id, email, password_hash, name, status) VALUES
		 (1,'a@t.com','x','A','approved'), (2,'b@t.com','x','B','approved')`,
		`INSERT INTO classrooms (id, name, code, initial_capital) VALUES (3,'스코입','ABC123',50000000)`,
		`INSERT INTO classroom_members (classroom_id, user_id) VALUES (3,1)`,
		// user1: 멤버십 있음(강의실 3), user2: 멤버십 없음
		`INSERT INTO wallets (id, user_id, balance) VALUES (5,1,777), (6,2,888)`,
		`INSERT INTO transactions (wallet_id, amount, balance_after, tx_type) VALUES (5,777,777,'initial_capital')`,
	}
	for _, stmt := range legacy {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("legacy setup %q: %v", stmt[:40], err)
		}
	}

	if err := persistence.RunMigrations(db); err != nil {
		t.Fatalf("migrations: %v", err)
	}

	// 1) classroom_id 컬럼 존재 + 멤버십 기반 백필
	var classroomID, balance int
	if err := db.QueryRow("SELECT classroom_id, balance FROM wallets WHERE id = 5").Scan(&classroomID, &balance); err != nil {
		t.Fatalf("wallet id=5 lost after rebuild: %v", err)
	}
	if classroomID != 3 || balance != 777 {
		t.Errorf("wallet id=5: classroom_id=%d balance=%d, want 3/777", classroomID, balance)
	}
	// 멤버십 없는 유저는 미배정(0)
	if err := db.QueryRow("SELECT classroom_id, balance FROM wallets WHERE id = 6").Scan(&classroomID, &balance); err != nil {
		t.Fatalf("wallet id=6 lost after rebuild: %v", err)
	}
	if classroomID != 0 || balance != 888 {
		t.Errorf("wallet id=6: classroom_id=%d balance=%d, want 0/888", classroomID, balance)
	}

	// 2) 같은 유저의 다른 강의실 지갑 허용, 같은 (user, classroom) 중복 금지
	if _, err := db.Exec("INSERT INTO wallets (user_id, classroom_id, balance) VALUES (1, 9, 0)"); err != nil {
		t.Errorf("second wallet for (user1, classroom9) must be allowed: %v", err)
	}
	if _, err := db.Exec("INSERT INTO wallets (user_id, classroom_id, balance) VALUES (1, 3, 0)"); err == nil {
		t.Errorf("duplicate (user1, classroom3) wallet must violate UNIQUE")
	}

	// 3) transactions 무결성: 기존 행 유지 + FK 가 새 wallets 테이블을 가리켜야 함
	var txCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM transactions WHERE wallet_id = 5").Scan(&txCount); err != nil || txCount != 1 {
		t.Errorf("transactions for wallet 5: count=%d err=%v, want 1", txCount, err)
	}
	var txDDL string
	if err := db.QueryRow("SELECT sql FROM sqlite_master WHERE name = 'transactions'").Scan(&txDDL); err != nil {
		t.Fatalf("read transactions DDL: %v", err)
	}
	if strings.Contains(txDDL, "wallets_legacy") {
		t.Errorf("transactions FK must reference wallets, got DDL: %s", txDDL)
	}

	// 4) 재실행 멱등성
	if err := persistence.RunMigrations(db); err != nil {
		t.Fatalf("re-run migrations: %v", err)
	}
	var cnt int
	db.QueryRow("SELECT COUNT(*) FROM wallets").Scan(&cnt)
	if cnt != 3 {
		t.Errorf("wallet count after re-run=%d, want 3 (no data loss/dup)", cnt)
	}
}
