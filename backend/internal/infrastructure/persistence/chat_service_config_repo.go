package persistence

import (
	"database/sql"
	"errors"
)

// ChatServiceConfigRepo — 챗봇 서비스 레벨 설정(key-value) 저장소.
// 대표 용도: llm-proxy 서비스 키(#076).
type ChatServiceConfigRepo struct {
	db *sql.DB
}

func NewChatServiceConfigRepo(db *sql.DB) *ChatServiceConfigRepo {
	return &ChatServiceConfigRepo{db: db}
}

func (r *ChatServiceConfigRepo) Get(key string) (string, error) {
	var v string
	err := r.db.QueryRow(`SELECT value FROM chat_service_config WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

func (r *ChatServiceConfigRepo) Set(key, value string) error {
	_, err := r.db.Exec(`
		INSERT INTO chat_service_config (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`,
		key, value)
	return err
}
