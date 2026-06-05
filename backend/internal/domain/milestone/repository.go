package milestone

// Repository — student_milestones 테이블 접근.
type Repository interface {
	// Upsert — (student_id, type) 키로 INSERT 또는 UPDATE.
	// status 가 approved/rejected 였더라도 admin 코멘트는 유지, source/url/content 만 갱신.
	// admin 이 다시 reviewing 하도록 status 는 pending 으로 reset.
	Upsert(m *Milestone) (int, error)

	// FindByStudentAndType — 학생 + type 으로 단건 조회. 없으면 (nil, nil).
	FindByStudentAndType(studentID int, typ Type) (*Milestone, error)

	// FindByID — 단건 조회. 없으면 (nil, ErrNotFound).
	FindByID(id int) (*Milestone, error)

	// ListByStudent — 학생의 모든 milestone (4개 이하).
	ListByStudent(studentID int) ([]*Milestone, error)

	// UpdateStatus — admin 승인/반려.
	UpdateStatus(id int, status Status, adminNote string, adminID int) error

	// #120 회고 에세이 AI 평가 결과 저장.
	UpdateAIScore(id int, score int, reasoning, signalsJSON string) error
}
