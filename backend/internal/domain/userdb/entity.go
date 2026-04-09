package userdb

import "time"

// UserDatabase 는 학생이 프로비저닝 받은 개인 PostgreSQL 데이터베이스 메타데이터.
// 실제 DB 생성/삭제는 PG 서버에서 일어나고, 여기에는 참조 정보만 저장.
// 비밀번호는 저장하지 않는다 (생성/재발급 응답 시 1회만 노출).
type UserDatabase struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	ProjectName string    `json:"project_name"` // "todoapp"
	DBName      string    `json:"db_name"`      // "seowon_todoapp"
	PGUsername  string    `json:"pg_username"`  // "seowon_todoapp"
	Host        string    `json:"host"`         // "db.earnlearning.com"
	Port        int       `json:"port"`         // 6432
	CreatedAt   time.Time `json:"created_at"`
	LastRotated *time.Time `json:"last_rotated,omitempty"`
}

// Credentials 는 생성/재발급 시에만 응답에 포함되는 확장 DTO. 비밀번호 노출용.
type Credentials struct {
	*UserDatabase
	Password string `json:"password"`
	URL      string `json:"url"` // postgresql://user:pass@host:port/db
}
