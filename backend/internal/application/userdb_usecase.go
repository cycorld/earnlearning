package application

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/earnlearning/backend/internal/domain/userdb"
	"github.com/earnlearning/backend/internal/infrastructure/userdbadmin"
)

// UserNameResolver 는 유저 ID 로부터 PG 사용자명(소문자/영숫자/밑줄) 를 얻는다.
// 구현체는 기존 user.Repository 를 래핑한다.
type UserNameResolver interface {
	PGSlugByUserID(userID int) (string, error)
}

type UserDBUseCase struct {
	repo        userdb.Repository
	provisioner userdbadmin.Provisioner
	users       UserNameResolver
	maxPerUser  int
}

func NewUserDBUseCase(repo userdb.Repository, prov userdbadmin.Provisioner, users UserNameResolver, maxPerUser int) *UserDBUseCase {
	if maxPerUser <= 0 {
		maxPerUser = 3
	}
	return &UserDBUseCase{repo: repo, provisioner: prov, users: users, maxPerUser: maxPerUser}
}

// --- Input types ---

type CreateUserDBInput struct {
	ProjectName string `json:"project_name"`
}

// --- Output types ---

type CreateUserDBOutput struct {
	*userdb.UserDatabase
	Password string `json:"password"`
	URL      string `json:"url"`
}

// --- Methods ---

func (uc *UserDBUseCase) List(userID int) ([]*userdb.UserDatabase, error) {
	return uc.repo.ListByUserID(userID)
}

func (uc *UserDBUseCase) Create(userID int, input CreateUserDBInput) (*CreateUserDBOutput, error) {
	if uc.provisioner == nil {
		return nil, userdb.ErrProvisionerDown
	}
	if err := userdbadmin.ValidateName(input.ProjectName); err != nil {
		return nil, userdb.ErrInvalidName
	}

	// 사용자 PG 슬러그 (이메일 local-part 나 student_id 기반)
	username, err := uc.users.PGSlugByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("사용자 정보를 가져올 수 없습니다: %w", err)
	}
	if err := userdbadmin.ValidateName(username); err != nil {
		return nil, fmt.Errorf("사용자 이름이 PG 규칙에 맞지 않습니다: %w", err)
	}

	// 쿼터 체크
	count, err := uc.repo.CountByUserID(userID)
	if err != nil {
		return nil, err
	}
	if count >= uc.maxPerUser {
		return nil, userdb.ErrQuotaExceeded
	}

	// 중복 확인 (사용자 + 프로젝트명)
	existing, err := uc.repo.FindByUserIDAndProject(userID, input.ProjectName)
	if err != nil && err != userdb.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, userdb.ErrDuplicate
	}

	// 프로비저닝 (실제 PG 에 DB/ROLE 생성)
	created, err := uc.provisioner.Create(username, input.ProjectName)
	if err != nil {
		// 도메인 에러는 그대로 전파 (핸들러에서 HTTP 상태 매핑)
		if errors.Is(err, userdb.ErrSlugConflict) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", userdb.ErrProvisionFailed, err)
	}

	// SQLite 에 메타데이터 저장
	u := &userdb.UserDatabase{
		UserID:      userID,
		ProjectName: input.ProjectName,
		DBName:      created.DBName,
		PGUsername:  created.PGUsername,
		Host:        created.Host,
		Port:        created.Port,
	}
	id, err := uc.repo.Create(u)
	if err != nil {
		// 롤백: PG 에서도 제거
		_ = uc.provisioner.Delete(created.DBName, created.PGUsername)
		return nil, err
	}
	u.ID = id

	// 응답 (비밀번호 포함 — 1회만)
	out := &CreateUserDBOutput{
		UserDatabase: u,
		Password:     created.Password,
		URL:          buildURL(created),
	}
	return out, nil
}

func (uc *UserDBUseCase) Rotate(userID, dbID int) (*CreateUserDBOutput, error) {
	if uc.provisioner == nil {
		return nil, userdb.ErrProvisionerDown
	}
	u, err := uc.repo.FindByID(dbID)
	if err != nil {
		return nil, err
	}
	if u.UserID != userID {
		return nil, userdb.ErrForbidden
	}

	newPassword, err := uc.provisioner.Rotate(u.PGUsername)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", userdb.ErrProvisionFailed, err)
	}
	_ = uc.repo.MarkRotated(dbID)

	return &CreateUserDBOutput{
		UserDatabase: u,
		Password:     newPassword,
		URL: buildURL(&userdbadmin.CreatedDB{
			DBName:     u.DBName,
			PGUsername: u.PGUsername,
			Password:   newPassword,
			Host:       u.Host,
			Port:       u.Port,
		}),
	}, nil
}

func (uc *UserDBUseCase) Delete(userID, dbID int) error {
	if uc.provisioner == nil {
		return userdb.ErrProvisionerDown
	}
	u, err := uc.repo.FindByID(dbID)
	if err != nil {
		return err
	}
	if u.UserID != userID {
		return userdb.ErrForbidden
	}

	if err := uc.provisioner.Delete(u.DBName, u.PGUsername); err != nil {
		return fmt.Errorf("%w: %v", userdb.ErrProvisionFailed, err)
	}
	return uc.repo.Delete(dbID)
}

// --- Helpers ---

// buildURL 은 학생에게 표시할 DATABASE_URL 문자열을 만든다.
func buildURL(c *userdbadmin.CreatedDB) string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
		c.PGUsername, c.Password, c.Host, c.Port, c.DBName)
}

// --- UserNameResolver 구현 (user.Repository 래핑) ---

// slugRE 는 이메일 local-part 또는 기타 입력에서 허용 문자만 남기기 위한 보수적 정규식.
var slugCleanRE = regexp.MustCompile(`[^a-z0-9_]`)

// SlugFromEmail 은 이메일 주소에서 PG 사용자명 후보를 만든다.
// 규칙: local-part → 소문자 → 비허용 문자 제거 → 영문자 시작 보장 ("u_" 접두) →
// 3~32자 패딩/절단.
func SlugFromEmail(email string) string {
	at := -1
	for i, c := range email {
		if c == '@' {
			at = i
			break
		}
	}
	local := email
	if at >= 0 {
		local = email[:at]
	}
	s := slugCleanRE.ReplaceAllString(stringToLower(local), "")
	if s == "" {
		s = "user"
	}
	// 영문자로 시작해야 함
	if !(s[0] >= 'a' && s[0] <= 'z') {
		s = "u_" + s
	}
	if len(s) < 3 {
		s = s + "_db"
	}
	if len(s) > 20 {
		s = s[:20]
	}
	return s
}

// stringToLower (표준 strings.ToLower 회피 없이 import 추가 최소화용)
func stringToLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
