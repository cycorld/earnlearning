// Package mail — #166 학생별 이메일 수신함.
// 학생은 <local_part>@earnlearning.com 개인 주소를 발급받고, Cloudflare Email Worker
// webhook 으로 들어온 메일을 앱에서 읽고, 기존 SES 로 답장한다.
package mail

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// Domain — 학생 개인 메일 주소 도메인.
const Domain = "earnlearning.com"

// Direction — 메일 방향.
const (
	DirectionIn  = "in"  // 수신 (webhook)
	DirectionOut = "out" // 발신 (SES)
)

// Address — 학생이 소유한 개인 메일 주소.
type Address struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	LocalPart string    `json:"local_part"`
	CreatedAt time.Time `json:"created_at"`
}

// Email 은 owner 스코프의 단일 메일 레코드.
type Email struct {
	ID          int       `json:"id"`
	OwnerUserID int       `json:"owner_user_id"`
	Direction   string    `json:"direction"`
	FromAddr    string    `json:"from_addr"`
	ToAddr      string    `json:"to_addr"`
	Subject     string    `json:"subject"`
	BodyText    string    `json:"body_text"`
	BodyHTML    string    `json:"body_html"`
	MessageID   string    `json:"message_id"`
	InReplyTo   string    `json:"in_reply_to"`
	Refs        string    `json:"refs"`
	Read        bool      `json:"read"`
	CreatedAt   time.Time `json:"created_at"`
}

// Attachment — 메일 첨부. 파일은 PrivateUploadPath 하위에 저장 (static 서빙 X).
// StoredPath 는 절대 API 응답으로 노출하지 않는다 (개인정보/경로 보호).
type Attachment struct {
	ID         int    `json:"id"`
	EmailID    int    `json:"email_id"`
	Filename   string `json:"filename"`
	Mime       string `json:"mime"`
	Size       int64  `json:"size"`
	StoredPath string `json:"-"`
	// OwnerUserID 는 첨부 다운로드 권한 검증용 (부모 메일 조인으로 채움).
	OwnerUserID int `json:"-"`
}

// EmailListItem — 목록(받은/보낸편지함, 관리자) 표시용 축약 뷰.
type EmailListItem struct {
	ID             int       `json:"id"`
	Direction      string    `json:"direction"`
	FromAddr       string    `json:"from_addr"`
	ToAddr         string    `json:"to_addr"`
	Subject        string    `json:"subject"`
	Snippet        string    `json:"snippet"`
	Read           bool      `json:"read"`
	HasAttachments bool      `json:"has_attachments"`
	CreatedAt      time.Time `json:"created_at"`
	// 관리자 목록에서만 채움.
	OwnerUserID int    `json:"owner_user_id,omitempty"`
	OwnerName   string `json:"owner_name,omitempty"`
}

// 도메인 에러.
var (
	ErrInvalidLocalPart    = errors.New("유효하지 않은 메일 주소 형식입니다")
	ErrReserved            = errors.New("사용할 수 없는(예약된) 주소입니다")
	ErrAddressTaken        = errors.New("이미 사용 중인 주소입니다")
	ErrAlreadyClaimed      = errors.New("이미 메일 주소를 발급받았습니다")
	ErrNoAddress           = errors.New("먼저 메일 주소를 발급받아야 합니다")
	ErrForbidden           = errors.New("접근 권한이 없습니다")
	ErrNotFound            = errors.New("메일을 찾을 수 없습니다")
	ErrSendFailed          = errors.New("메일 발송에 실패했습니다")
	ErrAttachmentsTooLarge = errors.New("첨부 총 용량이 10MB를 초과합니다")
)

// reservedLocalParts — 발급 불가 예약어 (운영/오남용 방지).
var reservedLocalParts = map[string]bool{
	"admin": true, "administrator": true, "postmaster": true, "abuse": true,
	"hostmaster": true, "webmaster": true, "root": true, "mail": true,
	"noreply": true, "no-reply": true, "support": true, "hello": true,
	"info": true, "security": true, "ceo": true, "contact": true,
}

// localPartRe — 소문자/숫자로 시작, 전체 3~30자, [a-z0-9._-] 만 허용.
var localPartRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,29}$`)

// ValidateLocalPart — 발급 요청 local_part 검증. 규칙 위반 시 도메인 에러 반환.
func ValidateLocalPart(lp string) error {
	if !localPartRe.MatchString(lp) {
		return ErrInvalidLocalPart
	}
	if strings.Contains(lp, "..") { // 연속 점 금지
		return ErrInvalidLocalPart
	}
	if reservedLocalParts[lp] {
		return ErrReserved
	}
	return nil
}

// EmailFor — local_part 로 전체 메일 주소를 만든다.
func EmailFor(localPart string) string {
	return localPart + "@" + Domain
}

// Snippet — 본문 앞부분을 룬(rune) 기준 n 글자로 자른다 (한글 안전).
func Snippet(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
