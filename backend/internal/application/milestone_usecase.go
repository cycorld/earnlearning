package application

import (
	"fmt"
	"strings"

	"github.com/earnlearning/backend/internal/domain/company"
	"github.com/earnlearning/backend/internal/domain/grant"
	"github.com/earnlearning/backend/internal/domain/milestone"
	"github.com/earnlearning/backend/internal/domain/notification"
	"github.com/earnlearning/backend/internal/domain/user"
)

type MilestoneUseCase struct {
	repo        milestone.Repository
	userRepo    user.Repository
	companyRepo company.CompanyRepository
	grantRepo   grant.Repository
	notifUC     *NotificationUseCase
}

func NewMilestoneUseCase(
	repo milestone.Repository,
	userRepo user.Repository,
	companyRepo company.CompanyRepository,
	grantRepo grant.Repository,
	notifUC *NotificationUseCase,
) *MilestoneUseCase {
	return &MilestoneUseCase{
		repo:        repo,
		userRepo:    userRepo,
		companyRepo: companyRepo,
		grantRepo:   grantRepo,
		notifUC:     notifUC,
	}
}

// SubmitManualInput — 학생의 수동 제출 입력.
type SubmitManualInput struct {
	Type    milestone.Type `json:"type"`
	URL     string         `json:"url"`
	Content string         `json:"content"`
}

// SubmitManual — 학생이 직접 (form으로) milestone을 제출/수정.
// MVP 타입에 대해서도 학생이 직접 URL을 명시할 수 있게 허용함
// (자동 detect 가 실패할 때 fallback).
// 단, URL이 deny list 에 걸리면 거절.
func (uc *MilestoneUseCase) SubmitManual(studentID int, in SubmitManualInput) (*milestone.Milestone, error) {
	if !in.Type.Valid() {
		return nil, milestone.ErrInvalidType
	}
	url := strings.TrimSpace(in.URL)
	content := strings.TrimSpace(in.Content)

	// MVP 타입은 URL이 반드시 있어야 하고, deny list 통과해야 함.
	if in.Type == milestone.TypeMVP1 || in.Type == milestone.TypeMVP2 {
		if url == "" {
			return nil, fmt.Errorf("MVP 제출에는 배포 URL이 필요합니다")
		}
		if !milestone.IsValidMilestoneURL(url) {
			return nil, fmt.Errorf("vercel.app 또는 자체 도메인만 인정됩니다 (AI Studio·Claude·ChatGPT 등 연습용은 제외)")
		}
	} else {
		// business_plan / retrospective — content 또는 URL 둘 중 하나는 있어야 함.
		if url == "" && content == "" {
			return nil, fmt.Errorf("URL 또는 본문 중 하나는 입력해야 합니다")
		}
		// URL을 옵션으로 넣었으면 그것도 deny list 통과해야 함.
		if url != "" && !milestone.IsValidMilestoneURL(url) {
			return nil, fmt.Errorf("URL이 유효하지 않습니다 (연습용 도메인 제외)")
		}
	}

	m := &milestone.Milestone{
		StudentID:  studentID,
		Type:       in.Type,
		SourceType: milestone.SourceManual,
		URL:        url,
		Content:    content,
		Status:     milestone.StatusPending,
	}
	id, err := uc.repo.Upsert(m)
	if err != nil {
		return nil, err
	}
	return uc.repo.FindByID(id)
}

// SyncAuto — 학생의 회사 service_url + grant_applications.proposal 에서
// MVP1 / MVP2 를 자동 detect 해서 upsert.
// 이미 admin이 승인한 row 는 건드리지 않음 (URL 변경에도 승인 유지).
//
// 규칙:
// - 학생이 owner인 모든 회사의 service_url 를 모두 모아 파싱 → deny list 통과한 URL 만 추출
// - + grant_applications 의 proposal 텍스트에서 URL 추출 → 같은 필터 적용
// - 회사 등록 순서 + grant 등록 순서 (시간순)로 1번 = MVP1, 2번 = MVP2
func (uc *MilestoneUseCase) SyncAuto(studentID int) ([]*milestone.Milestone, error) {
	candidates, err := uc.collectCandidates(studentID)
	if err != nil {
		return nil, err
	}

	mvpTypes := []milestone.Type{milestone.TypeMVP1, milestone.TypeMVP2}
	for i, t := range mvpTypes {
		if i >= len(candidates) {
			break
		}
		cand := candidates[i]

		// 이미 approved 상태면 자동 갱신 스킵 (admin이 다시 검토하지 않도록).
		existing, err := uc.repo.FindByStudentAndType(studentID, t)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.Status == milestone.StatusApproved {
			// 같은 URL이면 그대로, 다른 URL이면 그래도 유지 (승인 보호).
			continue
		}

		m := &milestone.Milestone{
			StudentID:   studentID,
			Type:        t,
			SourceType:  cand.SourceType,
			SourceRefID: cand.SourceRefID,
			URL:         cand.URL,
		}
		if _, err := uc.repo.Upsert(m); err != nil {
			return nil, err
		}
	}

	return uc.repo.ListByStudent(studentID)
}

type autoCandidate struct {
	URL         string
	SourceType  milestone.SourceType
	SourceRefID *int
}

// collectCandidates — 회사/grant 에서 유효 URL 후보를 모아 시간순 정렬.
func (uc *MilestoneUseCase) collectCandidates(studentID int) ([]autoCandidate, error) {
	var out []autoCandidate

	// 1) 회사 service_url — created_at ASC 순으로 (FindByOwnerID 는 DESC라서 reverse)
	companies, err := uc.companyRepo.FindByOwnerID(studentID)
	if err != nil {
		return nil, err
	}
	// reverse to ASC
	for i := len(companies) - 1; i >= 0; i-- {
		c := companies[i]
		urls := milestone.ParseCommaSeparated(c.ServiceURL)
		valid := milestone.FilterValidURLs(urls)
		for _, u := range valid {
			cid := c.ID
			out = append(out, autoCandidate{
				URL:         u,
				SourceType:  milestone.SourceCompany,
				SourceRefID: &cid,
			})
		}
	}

	// 2) grant_applications.proposal 에서 URL 추출
	apps, err := uc.grantRepo.ListApplicationsByUserID(studentID)
	if err != nil {
		return nil, err
	}
	// ListApplicationsByUserID returns DESC; reverse to ASC.
	for i := len(apps) - 1; i >= 0; i-- {
		a := apps[i]
		extracted := milestone.ExtractURLsFromText(a.Proposal)
		valid := milestone.FilterValidURLs(extracted)
		for _, u := range valid {
			aid := a.ID
			out = append(out, autoCandidate{
				URL:         u,
				SourceType:  milestone.SourceGrant,
				SourceRefID: &aid,
			})
		}
	}

	return dedupCandidates(out), nil
}

// dedupCandidates — same URL 두 번 (회사+grant 양쪽 등록) 인 경우 최초 등장만.
func dedupCandidates(in []autoCandidate) []autoCandidate {
	seen := map[string]bool{}
	out := make([]autoCandidate, 0, len(in))
	for _, c := range in {
		if seen[c.URL] {
			continue
		}
		seen[c.URL] = true
		out = append(out, c)
	}
	return out
}

// ListForStudent — 학생 본인용 대시보드. 4개 type 순서대로 (없는 자리는 nil).
func (uc *MilestoneUseCase) ListForStudent(studentID int) (*milestone.StudentProgress, error) {
	u, err := uc.userRepo.FindByID(studentID)
	if err != nil {
		return nil, err
	}
	all, err := uc.repo.ListByStudent(studentID)
	if err != nil {
		return nil, err
	}
	return buildProgress(u, all), nil
}

// ListAll — 관리자용. 모든 학생 + 각자 4개 milestone 매트릭스.
func (uc *MilestoneUseCase) ListAll() ([]*milestone.StudentProgress, error) {
	// 학생은 보통 40명 이하 — 한번에 가져옴.
	students, _, err := uc.userRepo.ListAll(1, 1000)
	if err != nil {
		return nil, err
	}
	out := make([]*milestone.StudentProgress, 0, len(students))
	for _, u := range students {
		if u.Role != user.RoleStudent {
			continue
		}
		ms, err := uc.repo.ListByStudent(u.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, buildProgress(u, ms))
	}
	return out, nil
}

func buildProgress(u *user.User, ms []*milestone.Milestone) *milestone.StudentProgress {
	byType := map[milestone.Type]*milestone.Milestone{}
	approved := 0
	for _, m := range ms {
		byType[m.Type] = m
		if m.Status == milestone.StatusApproved {
			approved++
		}
	}
	ordered := make([]*milestone.Milestone, 0, len(milestone.AllTypes))
	for _, t := range milestone.AllTypes {
		ordered = append(ordered, byType[t]) // may be nil
	}
	return &milestone.StudentProgress{
		Student: milestone.StudentRef{
			ID:         u.ID,
			Name:       u.Name,
			StudentID:  u.StudentID,
			Department: u.Department,
		},
		Milestones:    ordered,
		ApprovedCount: approved,
		Group:         milestone.ClassifyGroup(approved),
	}
}

// Approve — admin 승인.
func (uc *MilestoneUseCase) Approve(milestoneID, adminID int, adminNote string) error {
	m, err := uc.repo.FindByID(milestoneID)
	if err != nil {
		return err
	}
	if err := uc.repo.UpdateStatus(milestoneID, milestone.StatusApproved, adminNote, adminID); err != nil {
		return err
	}
	if uc.notifUC != nil {
		_ = uc.notifUC.CreateNotification(
			m.StudentID,
			notification.NotifType("milestone_approved"),
			"평가지표 승인",
			fmt.Sprintf("'%s' 평가지표가 승인되었습니다.", milestoneTitle(m.Type)),
			"milestone", milestoneID,
		)
	}
	return nil
}

// Reject — admin 반려.
func (uc *MilestoneUseCase) Reject(milestoneID, adminID int, adminNote string) error {
	m, err := uc.repo.FindByID(milestoneID)
	if err != nil {
		return err
	}
	if err := uc.repo.UpdateStatus(milestoneID, milestone.StatusRejected, adminNote, adminID); err != nil {
		return err
	}
	if uc.notifUC != nil {
		_ = uc.notifUC.CreateNotification(
			m.StudentID,
			notification.NotifType("milestone_rejected"),
			"평가지표 반려",
			fmt.Sprintf("'%s' 평가지표가 반려되었습니다.", milestoneTitle(m.Type)),
			"milestone", milestoneID,
		)
	}
	return nil
}

func milestoneTitle(t milestone.Type) string {
	switch t {
	case milestone.TypeMVP1:
		return "1차 MVP 배포"
	case milestone.TypeMVP2:
		return "2차 MVP 배포"
	case milestone.TypeBusinessPlan:
		return "사업계획서 제출"
	case milestone.TypeRetrospective:
		return "한 학기 회고 발표"
	}
	return string(t)
}
