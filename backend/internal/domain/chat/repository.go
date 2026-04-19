package chat

import "time"

type SessionRepository interface {
	Create(s *Session) (int, error)
	FindByID(id int) (*Session, error)
	FindByIDWithUser(id int) (*Session, error) // admin: user_name 포함
	ListByUser(userID, page, limit int) ([]*Session, int, error)
	// ListAll — admin 전용. query 가 있으면 title LIKE 필터.
	// userID > 0 이면 해당 유저로 필터, 아니면 전체.
	ListAll(userID int, query string, page, limit int) ([]*Session, int, error)
	UpdateTitle(id int, title string) error
	UpdateActiveSkill(id int, skillID *int) error
	UpdateLastMessageAt(id int, at time.Time, addTokens int) error
	Delete(id int) error
}

type MessageRepository interface {
	Create(m *Message) (int, error)
	ListBySession(sessionID int, limit int) ([]*Message, error)
	CountBySession(sessionID int) (int, error)
}

type SkillRepository interface {
	Create(s *Skill) (int, error)
	Upsert(s *Skill) (int, error) // slug 기준 upsert (seed 용)
	FindBySlug(slug string) (*Skill, error)
	FindByID(id int) (*Skill, error)
	List(includeDisabled, includeAdminOnly bool) ([]*Skill, error)
	Update(s *Skill) error
	Delete(id int) error
}

type WikiRepository interface {
	UpsertMeta(m *WikiDocMeta) error
	FindMeta(slug string) (*WikiDocMeta, error)
	ListMeta() ([]*WikiDocMeta, error)
	DeleteMeta(slug string) error

	// FTS5 가상 테이블 조작
	UpsertDoc(slug, title, body string) error
	DeleteDoc(slug string) error
	// Search 는 BM25 정렬된 결과 반환. scope 가 비어있으면 전체.
	Search(query string, scope []string, limit int) ([]*WikiSearchHit, error)
	// Reset clears all docs (used on full reindex)
	Reset() error
}

type UsageRepository interface {
	AddUsage(userID int, day time.Time, requests, prompt, completion, cache, costKRW int) error
	SumForRange(from, to time.Time) ([]*UsageDay, error)
	SumForMonth(year int, month time.Month) (*UsageDay, error)
	// TopUsersForRange — 비용 기준 내림차순 상위 N명. user_name 조인.
	TopUsersForRange(from, to time.Time, limit int) ([]*UserUsageTotal, error)
}
