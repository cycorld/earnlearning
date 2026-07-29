package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/earnlearning/backend/internal/domain/company"
	"github.com/earnlearning/backend/internal/domain/user"
	"github.com/earnlearning/backend/internal/domain/wallet"
	"github.com/earnlearning/backend/internal/infrastructure/rybbit"
	"github.com/earnlearning/backend/internal/infrastructure/urlcheck"
)

// validationTTL — 검증 결과 유효 기간 (#181).
// 이 시간이 지나면 Rybbit 연동 전에 다시 검증해야 한다 (주소가 바뀌었을 수 있으므로).
const validationTTL = 24 * time.Hour

// URLChecker — 서비스 URL 도달성 검사기 (테스트에서 교체 가능).
type URLChecker interface {
	Check(ctx context.Context, rawURL string) urlcheck.Result
}

// invalidURLMessages — urlcheck 사유 토큰 → 학생이 읽을 한국어 메시지 (#181).
var invalidURLMessages = map[string]string{
	urlcheck.ReasonBadSyntax:      "주소 형식이 올바르지 않아요",
	urlcheck.ReasonNotHTTPS:       "https:// 로 시작하는 주소만 등록할 수 있어요",
	urlcheck.ReasonHasCredentials: "아이디/비밀번호가 들어간 주소는 등록할 수 없어요",
	urlcheck.ReasonHasFragment:    "# 이 들어간 주소는 등록할 수 없어요",
	urlcheck.ReasonEmptyHost:      "도메인이 없는 주소예요",
	urlcheck.ReasonTooLong:        "주소가 너무 길어요",
}

func invalidURLError(err error) error {
	reason := urlcheck.ReasonOf(err)
	msg, ok := invalidURLMessages[reason]
	if !ok {
		msg = company.ErrInvalidServiceURL.Error()
	}
	return &company.InvalidServiceURLError{Reason: reason, Message: msg}
}

// CompanyServiceResponse — 프론트엔드와 동결된 응답 계약 (#181).
// url 은 정규화된 형태를 내려준다 (중복 판정 / 외부 호출과 같은 값).
type CompanyServiceResponse struct {
	ID                  int        `json:"id"`
	CompanyID           int        `json:"company_id"`
	Name                string     `json:"name"`
	URL                 string     `json:"url"`
	ValidationStatus    string     `json:"validation_status"`
	ValidationCheckedAt *time.Time `json:"validation_checked_at"`
	ValidationDetail    string     `json:"validation_detail"`
	RybbitStatus        string     `json:"rybbit_status"`
	RybbitSiteID        string     `json:"rybbit_site_id"`
	RybbitConnectedAt   *time.Time `json:"rybbit_connected_at"`
	// ConnectReady 는 저장하지 않고 매 응답마다 계산한다 (연동 버튼 활성화의 유일한 근거).
	ConnectReady bool `json:"connect_ready"`
}

// CompanyServiceInput — 서비스 등록/수정 요청 본문.
type CompanyServiceInput struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// connectReady — 검증 통과 + TTL 이내일 때만 연동 가능.
func connectReady(s *company.CompanyService) bool {
	if s.ValidationStatus != company.ValidationValid || s.ValidationCheckedAt == nil {
		return false
	}
	return time.Since(*s.ValidationCheckedAt) <= validationTTL
}

func toServiceResponse(s *company.CompanyService) *CompanyServiceResponse {
	return &CompanyServiceResponse{
		ID:                  s.ID,
		CompanyID:           s.CompanyID,
		Name:                s.Name,
		URL:                 s.NormalizedURL,
		ValidationStatus:    s.ValidationStatus,
		ValidationCheckedAt: s.ValidationCheckedAt,
		ValidationDetail:    s.ValidationDetail,
		RybbitStatus:        s.RybbitStatus,
		RybbitSiteID:        s.RybbitSiteID,
		RybbitConnectedAt:   s.RybbitConnectedAt,
		ConnectReady:        connectReady(s),
	}
}

// CompanyServiceUseCase — #181 회사 등록 서비스 (URL 검증 + Rybbit 연동).
// 권한 판정은 전부 이 계층에서 한다 (소유자 전용, admin 우회 없음).
type CompanyServiceUseCase struct {
	companyRepo company.CompanyRepository
	serviceRepo company.CompanyServiceRepository
	// userRepo — Rybbit 프로비저닝에 넣을 소유자(이메일/이름) 조회 + 학생/승인 게이트 (#181).
	userRepo    user.Repository
	walletRepo  wallet.Repository
	checker     URLChecker
	provisioner rybbit.Provisioner
}

func NewCompanyServiceUseCase(
	cr company.CompanyRepository,
	sr company.CompanyServiceRepository,
	ur user.Repository,
	wr wallet.Repository,
	checker URLChecker,
	provisioner rybbit.Provisioner,
) *CompanyServiceUseCase {
	return &CompanyServiceUseCase{
		companyRepo: cr,
		serviceRepo: sr,
		userRepo:    ur,
		walletRepo:  wr,
		checker:     checker,
		provisioner: provisioner,
	}
}

// SetChecker / SetProvisioner — 테스트에서 외부 의존성을 주입하기 위한 훅.
func (uc *CompanyServiceUseCase) SetChecker(c URLChecker) { uc.checker = c }

func (uc *CompanyServiceUseCase) SetProvisioner(p rybbit.Provisioner) { uc.provisioner = p }

// requireOwner — 회사 존재 + 소유자 확인 (기존 UpdateCompany 정책과 동일).
func (uc *CompanyServiceUseCase) requireOwner(companyID, userID int) (*company.Company, error) {
	c, err := uc.companyRepo.FindByID(companyID)
	if err != nil {
		return nil, err
	}
	if c.OwnerID != userID {
		return nil, company.ErrNotOwner
	}
	return c, nil
}

// requireService — 서비스 조회 + 회사 소속 확인.
// 다른 회사 경로로 접근하면 존재 여부를 알려주지 않기 위해 ErrServiceNotFound.
func (uc *CompanyServiceUseCase) requireService(companyID, serviceID int) (*company.CompanyService, error) {
	s, err := uc.serviceRepo.FindByID(serviceID)
	if err != nil {
		return nil, err
	}
	if s.CompanyID != companyID {
		return nil, company.ErrServiceNotFound
	}
	return s, nil
}

// normalizeInput — 이름 길이 검사 + URL 정규화 (네트워크 호출 없음).
func normalizeInput(input CompanyServiceInput) (name, raw, normalized string, err error) {
	name = strings.TrimSpace(input.Name)
	if utf8.RuneCountInString(name) > company.MaxServiceNameLength {
		return "", "", "", company.ErrServiceNameTooLong
	}
	raw = strings.TrimSpace(input.URL)
	normalized, _, nerr := urlcheck.Normalize(raw)
	if nerr != nil {
		return "", "", "", invalidURLError(nerr)
	}
	return name, raw, normalized, nil
}

// ListServices — 승인된 사용자면 누구나 조회 가능 (회사 공개 정보).
func (uc *CompanyServiceUseCase) ListServices(companyID int) ([]*CompanyServiceResponse, error) {
	// 없는 회사는 빈 배열이 아니라 404 로 알린다.
	if _, err := uc.companyRepo.FindByID(companyID); err != nil {
		return nil, err
	}
	list, err := uc.serviceRepo.FindByCompanyID(companyID)
	if err != nil {
		return nil, err
	}
	out := make([]*CompanyServiceResponse, 0, len(list))
	for _, s := range list {
		out = append(out, toServiceResponse(s))
	}
	return out, nil
}

// CreateService — 소유자 전용. 등록 시점에는 네트워크 검사를 하지 않는다 (정규화만).
func (uc *CompanyServiceUseCase) CreateService(companyID, userID int, input CompanyServiceInput) (*CompanyServiceResponse, error) {
	if _, err := uc.requireOwner(companyID, userID); err != nil {
		return nil, err
	}
	name, raw, normalized, err := normalizeInput(input)
	if err != nil {
		return nil, err
	}
	if err := uc.checkDuplicate(companyID, normalized, 0); err != nil {
		return nil, err
	}

	id, err := uc.serviceRepo.Create(&company.CompanyService{
		CompanyID:        companyID,
		Name:             name,
		URL:              raw,
		NormalizedURL:    normalized,
		ValidationStatus: company.ValidationUnvalidated,
		RybbitStatus:     company.RybbitNotConnected,
	})
	if err != nil {
		return nil, err
	}
	created, err := uc.serviceRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return toServiceResponse(created), nil
}

// checkDuplicate — 같은 회사 안에서 정규화 URL 중복 검사 (excludeID 는 자기 자신 제외용).
func (uc *CompanyServiceUseCase) checkDuplicate(companyID int, normalized string, excludeID int) error {
	existing, err := uc.serviceRepo.FindByCompanyAndNormalizedURL(companyID, normalized)
	if errors.Is(err, company.ErrServiceNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.ID == excludeID {
		return nil
	}
	return company.ErrDuplicateServiceURL
}

// UpdateService — 소유자 전용. URL 이 실제로 바뀌면 검증을 초기화하고,
// 이미 연동된 서비스는 needs_reconnect 로 표시한다 (드리프트).
func (uc *CompanyServiceUseCase) UpdateService(companyID, serviceID, userID int, input CompanyServiceInput) (*CompanyServiceResponse, error) {
	if _, err := uc.requireOwner(companyID, userID); err != nil {
		return nil, err
	}
	s, err := uc.requireService(companyID, serviceID)
	if err != nil {
		return nil, err
	}
	name, raw, normalized, err := normalizeInput(input)
	if err != nil {
		return nil, err
	}
	if err := uc.checkDuplicate(companyID, normalized, s.ID); err != nil {
		return nil, err
	}

	if normalized != s.NormalizedURL {
		// 주소가 바뀌었으니 이전 검증 결과는 더 이상 근거가 못 된다.
		s.ValidationStatus = company.ValidationUnvalidated
		s.ValidationCheckedAt = nil
		s.ValidationDetail = ""
		if s.RybbitStatus == company.RybbitConnected {
			s.RybbitStatus = company.RybbitNeedsReconnect
		}
	}
	s.Name = name
	s.URL = raw
	s.NormalizedURL = normalized

	if err := uc.serviceRepo.Update(s); err != nil {
		return nil, err
	}
	updated, err := uc.serviceRepo.FindByID(s.ID)
	if err != nil {
		return nil, err
	}
	return toServiceResponse(updated), nil
}

// DeleteService — 소유자 전용.
// #181: 로컬 레코드만 지운다. Rybbit 사이트는 외부 시스템의 파괴적 작업이라 호출하지 않는다
// (다른 데이터가 딸려 사라질 수 있고, 되돌릴 수 없다).
func (uc *CompanyServiceUseCase) DeleteService(companyID, serviceID, userID int) error {
	if _, err := uc.requireOwner(companyID, userID); err != nil {
		return err
	}
	s, err := uc.requireService(companyID, serviceID)
	if err != nil {
		return err
	}
	return uc.serviceRepo.Delete(s.ID)
}

// ValidateService — 소유자 전용. 서버가 직접 검사하고 결과를 기록한다.
// 검사가 끝나면 결과가 invalid 여도 성공 응답 (클라이언트 주장은 절대 믿지 않는다).
func (uc *CompanyServiceUseCase) ValidateService(ctx context.Context, companyID, serviceID, userID int) (*CompanyServiceResponse, error) {
	if _, err := uc.requireOwner(companyID, userID); err != nil {
		return nil, err
	}
	s, err := uc.requireService(companyID, serviceID)
	if err != nil {
		return nil, err
	}

	result := uc.checker.Check(ctx, s.NormalizedURL)
	status := company.ValidationInvalid
	detail := result.Detail
	if result.OK {
		status = company.ValidationValid
		detail = ""
	}
	now := time.Now()
	if err := uc.serviceRepo.UpdateValidation(s.ID, status, &now, detail); err != nil {
		return nil, err
	}
	updated, err := uc.serviceRepo.FindByID(s.ID)
	if err != nil {
		return nil, err
	}
	return toServiceResponse(updated), nil
}

// grantServiceIDs — Rybbit 에 보낼 "이 회사의 전체 유효 접근 집합" (#181).
// Rybbit 은 grants 를 누적이 아니라 재조정(reconcile) 하므로, 클릭한 서비스 하나만
// 보내면 이전에 연동한 다른 서비스 접근이 회수된다. 따라서 항상
// (이미 프로비저닝된 서비스 전부) ∪ (클릭한 서비스) 를 보낸다.
//   - "프로비저닝됨" 판정은 RybbitSiteID 보유 여부다: connected 는 물론,
//     URL 드리프트로 needs_reconnect 가 된 서비스도 Rybbit 사이트 매핑(키)이
//     살아 있으므로 포함한다 — 빼면 그 사이트 접근이 회수된다.
//   - 아직 한 번도 연동 안 된 서비스는 Rybbit 에 키가 없어서 포함하면 전체 요청이
//     403 (CROSS_COMPANY 판정) 으로 실패하므로 제외한다.
//   - 같은 회사(serviceRepo.FindByCompanyID) 범위에서만 수집 → 교차 회사 키 원천 차단.
func grantServiceIDs(all []*company.CompanyService, clickedID int) []int {
	ids := make([]int, 0, len(all))
	for _, sv := range all { // FindByCompanyID 는 id 오름차순 → 결정적 순서
		if sv.ID == clickedID || strings.TrimSpace(sv.RybbitSiteID) != "" {
			ids = append(ids, sv.ID)
		}
	}
	return ids
}

// ConnectRybbit — 소유자(승인된 학생 또는 관리자) + 활성 강의실 일치 + 최신 검증 통과일 때만
// 사이트를 등록하고, 소유자의 회사 전체 사이트 접근을 재조정한다.
// 실패하면 상태를 전혀 바꾸지 않는다 (fail-closed: 성공 후 단일 UPDATE).
func (uc *CompanyServiceUseCase) ConnectRybbit(ctx context.Context, companyID, serviceID, userID int) (*CompanyServiceResponse, error) {
	c, err := uc.requireOwner(companyID, userID)
	if err != nil {
		return nil, err
	}
	s, err := uc.requireService(companyID, serviceID)
	if err != nil {
		return nil, err
	}

	// #181 자격 게이트 — 승인된 학생과 수업을 운영하는 관리자가 자기 회사만 연동한다.
	// JWT 클레임은 로그인 시점 값이라 오래될 수 있으므로 DB 를 다시 읽는다.
	owner, err := uc.userRepo.FindByID(c.OwnerID)
	if err != nil {
		return nil, err
	}
	if owner.Role != user.RoleStudent && owner.Role != user.RoleAdmin {
		return nil, company.ErrNotStudent
	}
	if owner.Status != user.StatusApproved {
		return nil, company.ErrNotApproved
	}

	// #159 강의실 게이트 — 캠프(강의실) 소속 학생의 회사만 연동한다.
	active, err := uc.walletRepo.GetActiveClassroomID(userID)
	if err != nil {
		return nil, err
	}
	if active == 0 {
		return nil, company.ErrNoClassroom
	}
	if active != c.ClassroomID {
		return nil, company.ErrWrongClassroom
	}

	// #181: 검증 게이트를 먼저 통과해야 한다. 이미 연동된 서비스의 멱등 처리는 그 다음
	// (프론트엔드는 connect_ready 로 버튼을 막으므로 "연동됨 + 검증 만료" 조합은 도달하지 않는다).
	if s.ValidationStatus != company.ValidationValid || s.ValidationCheckedAt == nil {
		return nil, company.ErrValidationRequired
	}
	if time.Since(*s.ValidationCheckedAt) > validationTTL {
		return nil, company.ErrValidationStale
	}

	// 이미 연동됨 → 외부 호출 없이 현재 상태를 그대로 반환 (멱등).
	// (재조정을 보내지 않으므로 Rybbit 쪽 접근 집합도 그대로 유지된다.)
	if s.RybbitStatus == company.RybbitConnected && s.RybbitSiteID != "" {
		return toServiceResponse(s), nil
	}

	_, host, err := urlcheck.Normalize(s.NormalizedURL)
	if err != nil {
		return nil, invalidURLError(err)
	}
	name := s.Name
	if name == "" {
		name = c.Name
	}

	all, err := uc.serviceRepo.FindByCompanyID(c.ID)
	if err != nil {
		return nil, err
	}

	res, err := uc.provisioner.Provision(ctx, rybbit.ProvisionRequest{
		CompanyID:       c.ID,
		CompanyName:     c.Name,
		ServiceID:       s.ID,
		ServiceName:     name,
		Domain:          host,
		OwnerUserID:     owner.ID,
		OwnerEmail:      owner.Email,
		OwnerName:       owner.Name,
		GrantServiceIDs: grantServiceIDs(all, s.ID),
	})
	if errors.Is(err, rybbit.ErrNotConfigured) {
		return nil, company.ErrRybbitNotConfigured
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", company.ErrRybbitFailed, err)
	}
	if res == nil || strings.TrimSpace(res.SiteID) == "" {
		return nil, fmt.Errorf("%w: site id 가 비어 있습니다", company.ErrRybbitFailed)
	}

	now := time.Now()
	if err := uc.serviceRepo.UpdateRybbit(s.ID, company.RybbitConnected, strings.TrimSpace(res.SiteID), &now); err != nil {
		return nil, err
	}
	updated, err := uc.serviceRepo.FindByID(s.ID)
	if err != nil {
		return nil, err
	}
	return toServiceResponse(updated), nil
}
