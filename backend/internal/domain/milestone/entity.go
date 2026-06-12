package milestone

import "time"

// Type — 4개 평가지표
type Type string

const (
	TypeMVP1          Type = "mvp1"          // 1차 MVP (7주차)
	TypeMVP2          Type = "mvp2"          // 2차 MVP (12주차)
	TypeBusinessPlan  Type = "business_plan" // 사업계획서 (14주차)
	TypeRetrospective Type = "retrospective" // 회고 발표 (보강 1주차)
)

// AllTypes — 4가지 평가지표 (대시보드 순서 보장).
var AllTypes = []Type{TypeMVP1, TypeMVP2, TypeBusinessPlan, TypeRetrospective}

func (t Type) Valid() bool {
	switch t {
	case TypeMVP1, TypeMVP2, TypeBusinessPlan, TypeRetrospective:
		return true
	}
	return false
}

// SourceType — 어디서 데이터가 왔는지
type SourceType string

const (
	SourceManual  SourceType = "manual"  // 학생이 직접 폼으로 제출
	SourceCompany SourceType = "company" // 회사 service_url 에서 자동 detect
	SourceGrant   SourceType = "grant"   // grant_applications.proposal 텍스트에서 추출
)

// Status — 승인 상태
type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusApproved, StatusRejected:
		return true
	}
	return false
}

// FileRef — business_plan 비공개 첨부 파일 메타 (#125).
// 업로더 본인 + 관리자만 접근 가능. data/private_uploads/ 에 저장 (static 서빙 X).
type FileRef struct {
	ID         int       `json:"id"`
	StudentID  int       `json:"student_id"`
	Type       Type      `json:"-"`
	Filename   string    `json:"filename"`     // 원본 파일명
	StoredName string    `json:"-"`            // uuid.ext (디스크 저장명)
	MimeType   string    `json:"mime_type"`
	Size       int64     `json:"size"`
	Path       string    `json:"-"`            // 디스크 전체 경로 (스트리밍용, JSON 비노출)
	CreatedAt  time.Time `json:"created_at"`
}

// StudentAsset — percentile 산정용 (학생별 승인개수 + 총자산).
type StudentAsset struct {
	StudentID     int
	ApprovedCount int
	TotalAsset    int
}

// Milestone — 학생 1명의 4가지 평가지표 중 하나.
// UNIQUE(student_id, milestone_type) — 학생별 각 type 1개만.
type Milestone struct {
	ID            int        `json:"id"`
	StudentID     int        `json:"student_id"`
	Type          Type       `json:"type"`
	SourceType    SourceType `json:"source_type"`
	SourceRefID   *int       `json:"source_ref_id,omitempty"` // company_id / grant_application_id
	URL           string     `json:"url"`                      // MVP 의 경우 자동 detect 된 URL
	Content       string     `json:"content"`                  // 사업계획서/회고 본문 또는 부가 설명
	Files         []*FileRef `json:"files,omitempty"`          // business_plan 비공개 첨부 (#125)
	Status        Status     `json:"status"`
	AdminNote     string     `json:"admin_note"`
	ApprovedBy    *int       `json:"approved_by,omitempty"`
	ApprovedAt    *time.Time `json:"approved_at,omitempty"`
	// #120 회고 에세이 AI 평가 (retrospective 만 채워짐, 그 외 nil/0)
	AIScore       *int       `json:"ai_score,omitempty"`       // 0~100, NULL = 미평가
	AIReasoning   string     `json:"ai_reasoning,omitempty"`   // LLM 한 줄 평가
	AISignals     string     `json:"ai_signals,omitempty"`     // heuristic Signals JSON
	AIEvaluatedAt *time.Time `json:"ai_evaluated_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// StudentRef — admin matrix 응답용
type StudentRef struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	StudentID  string `json:"student_id"`
	Department string `json:"department"`
}

// StudentProgress — 학생 1명 + 4개 milestone (대시보드 응답)
type StudentProgress struct {
	Student       StudentRef   `json:"student"`
	Milestones    []*Milestone `json:"milestones"`     // ordered by AllTypes
	ApprovedCount int          `json:"approved_count"` // 0~4
	Group         string       `json:"group"`          // "A" / "B" / "C" / "D" / "" (= 성적 그레이드)
	// #125 성적/자산 — 본인 대시보드(ListForStudent)에서만 채워짐.
	AssetTotal     int `json:"asset_total"`      // 본인 총자산 (Cash+Stock+CompanyEquity−Debt)
	GroupSize      int `json:"group_size"`       // 같은 그룹(A/B/C/D) 인원 수
	AssetRank      int `json:"asset_rank"`       // 같은 그룹 내 자산 순위 (1 = 최상위), 0 = 미산정
	AssetPercentile int `json:"asset_percentile"` // 같은 그룹 내 상위 N% (1~100), 0 = 미산정
}

// ClassifyGroup — approved 개수 기반 그룹.
// 4 → A / 3 → B / 2 → C / 1 → D / 0 → "".
func ClassifyGroup(approvedCount int) string {
	switch approvedCount {
	case 4:
		return "A"
	case 3:
		return "B"
	case 2:
		return "C"
	case 1:
		return "D"
	}
	return ""
}
