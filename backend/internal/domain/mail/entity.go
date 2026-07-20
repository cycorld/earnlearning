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

// OwnerType — 메일 주소 소유 주체. 개인(user)·회사(company)·공용(shared).
const (
	OwnerUser    = "user"    // 개인 주소 (owner_id = user_id)
	OwnerCompany = "company" // 회사 주소 (owner_id = company_id)
	OwnerShared  = "shared"  // 공용 주소 (owner_id = 생성 관리자 id, 접근은 grant 로 제어)
)

// Status — 주소 승인 상태. 발급 요청 → pending → 관리자 승인/반려.
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

// Address — 개인 또는 회사가 소유한 메일 주소.
// OwnerType/OwnerID 로 소유 주체를 식별한다. UserID 는 알림 수신 책임자
// (개인=본인, 회사=회사 소유주 유저) 로 항상 채워 둔다.
type Address struct {
	ID          int       `json:"id"`
	OwnerType   string    `json:"owner_type"`
	OwnerID     int       `json:"owner_id"`
	UserID      int       `json:"user_id"`
	LocalPart   string    `json:"local_part"`
	DisplayName string    `json:"display_name"` // shared 주소 표시명 (개인/회사는 "")
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// Grant — 공용(shared) 메일함 접근 권한. revoked=1 이면 회수됨.
type Grant struct {
	UserID   int    `json:"user_id"`
	UserName string `json:"user_name"`
	Revoked  bool   `json:"revoked"`
}

// SharedAddressItem — 관리자 공용 메일함 목록 항목 (권한 포함).
type SharedAddressItem struct {
	AddressID   int      `json:"address_id"`
	LocalPart   string   `json:"local_part"`
	DisplayName string   `json:"display_name"`
	Email       string   `json:"email"`
	Grants      []*Grant `json:"grants"`
}

// MailboxItem — GET /mail/mailboxes 응답 항목. 내 개인 주소 + 내가 소유한 회사 주소.
type MailboxItem struct {
	AddressID int    `json:"address_id"`
	Kind      string `json:"kind"`       // "user" | "company"
	CompanyID *int   `json:"company_id"` // 회사 주소면 회사 id, 개인이면 null
	Name      string `json:"name"`       // 유저 이름 또는 회사 이름
	LocalPart string `json:"local_part"`
	Email     string `json:"email"`
	Status    string `json:"status"`
}

// AddressAdminItem — 관리자 승인 목록 항목.
type AddressAdminItem struct {
	ID        int       `json:"id"`
	OwnerType string    `json:"owner_type"`
	OwnerID   int       `json:"owner_id"`
	UserID    int       `json:"user_id"`
	UserName  string    `json:"user_name"`  // 책임 유저 이름
	UserEmail string    `json:"user_email"` // 책임 유저 계정 이메일
	OwnerName string    `json:"owner_name"` // 유저 이름 또는 회사 이름
	LocalPart string    `json:"local_part"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Email 은 owner 스코프의 단일 메일 레코드.
// AddressID 로 소속 메일함(주소)을 식별하고, OwnerUserID 는 알림 수신 책임자.
type Email struct {
	ID          int       `json:"id"`
	AddressID   int       `json:"address_id"`
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
	// OwnerUserID/AddressID 는 첨부 다운로드 권한 검증용 (부모 메일 조인으로 채움).
	OwnerUserID int `json:"-"`
	AddressID   int `json:"-"`
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
	ErrAlreadyClaimed      = errors.New("이미 승인된 메일 주소는 변경할 수 없습니다")
	ErrNoAddress           = errors.New("먼저 메일 주소를 발급받아야 합니다")
	ErrNotApproved         = errors.New("메일 주소가 아직 승인되지 않았습니다. 관리자 승인 후 사용할 수 있습니다")
	ErrAlreadyApproved     = errors.New("이미 승인된 주소입니다")
	ErrForbidden           = errors.New("접근 권한이 없습니다")
	ErrNotFound            = errors.New("메일을 찾을 수 없습니다")
	ErrSendFailed          = errors.New("메일 발송에 실패했습니다")
	ErrSendDisabled        = errors.New("메일 발송 기능이 현재 비활성화되어 있습니다")
	ErrAttachmentsTooLarge = errors.New("첨부 총 용량이 10MB를 초과합니다")
)

// reservedLocalParts — 발급 불가 예약어 (운영/오남용 방지). local_part 정확히 일치 시 거부.
var reservedLocalParts = map[string]bool{
	"admin": true, "administrator": true, "postmaster": true, "abuse": true,
	"hostmaster": true, "webmaster": true, "root": true, "mail": true,
	"email": true, "noreply": true, "no-reply": true, "no_reply": true,
	"reply": true, "support": true, "help": true, "helpdesk": true,
	"hello": true, "hi": true, "info": true, "contact": true,
	"security": true, "ssl": true, "cert": true, "ceo": true,
	"cfo": true, "cto": true, "coo": true, "staff": true,
	"team": true, "official": true, "verify": true, "verification": true,
	"notification": true, "notifications": true, "alert": true, "alerts": true,
	"service": true, "services": true, "system": true, "sys": true,
	"api": true, "www": true, "ftp": true, "smtp": true,
	"imap": true, "pop": true, "pop3": true, "mx": true,
	"ns": true, "dns": true, "test": true, "testing": true,
	"demo": true, "sample": true, "example": true, "billing": true,
	"payment": true, "payments": true, "pay": true, "bank": true,
	"banking": true, "finance": true, "invoice": true, "sales": true,
	"marketing": true, "press": true, "media": true, "legal": true,
	"privacy": true, "terms": true, "dev": true, "developer": true,
	"developers": true, "ops": true, "it": true, "hr": true,
	"recruit": true, "careers": true, "jobs": true, "news": true,
	"newsletter": true, "earnlearning": true, "unlearning": true, "professor": true,
	"prof": true, "teacher": true, "ta": true, "tutor": true,
	"master": true, "moderator": true, "mod": true, "owner": true,
	"manager": true, "all": true, "everyone": true, "undisclosed": true,
	"mailer-daemon": true, "daemon": true, "bounce": true, "bounces": true,
}

// localPartRe — 소문자/숫자로 시작, 전체 3~30자, [a-z0-9._-] 만 허용.
var localPartRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,29}$`)

// ValidateLocalPartFormat — 형식만 검증 (예약어 제외). 관리자 공용 주소 발급 경로용.
func ValidateLocalPartFormat(lp string) error {
	if !localPartRe.MatchString(lp) {
		return ErrInvalidLocalPart
	}
	if strings.Contains(lp, "..") { // 연속 점 금지
		return ErrInvalidLocalPart
	}
	return nil
}

// ValidateLocalPart — 발급 요청 local_part 검증(형식 + 예약어). 규칙 위반 시 도메인 에러 반환.
func ValidateLocalPart(lp string) error {
	if err := ValidateLocalPartFormat(lp); err != nil {
		return err
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
