package application

import (
	"encoding/base64"
	"errors"
	"fmt"
	netmail "net/mail"
	"os"
	"path/filepath"
	"strings"

	"github.com/earnlearning/backend/internal/domain/mail"
	"github.com/earnlearning/backend/internal/domain/notification"
	"github.com/earnlearning/backend/internal/infrastructure/email"
)

// MaxInboundTotalBytes — inbound 첨부 디코드 총량 상한 (10MB).
const MaxInboundTotalBytes = 10 * 1024 * 1024

// snippetSubjectLen — 알림 본문에 넣을 제목 축약 길이.
const snippetSubjectLen = 60

// MailSender — 임의 From 으로 메일을 보내는 발신기(DI seam).
// 프로덕션은 *email.SESService, 테스트는 스파이가 구현한다.
type MailSender interface {
	IsEnabled() bool
	SendMailFrom(m email.OutgoingMail) error
}

// MailUseCase — #166 학생 메일함.
type MailUseCase struct {
	repo              mail.Repository
	sender            MailSender
	notifUC           *NotificationUseCase
	privateUploadPath string
}

func NewMailUseCase(repo mail.Repository, sender MailSender, notifUC *NotificationUseCase, privateUploadPath string) *MailUseCase {
	return &MailUseCase{
		repo:              repo,
		sender:            sender,
		notifUC:           notifUC,
		privateUploadPath: privateUploadPath,
	}
}

// GetAddress — 유저의 개인 주소 (없으면 nil). 하위호환용 (GET /mail/address).
func (uc *MailUseCase) GetAddress(userID int) (*mail.Address, error) {
	return uc.repo.GetAddressByOwner(mail.OwnerUser, userID)
}

// ListMailboxes — 유저의 개인 주소 + 유저가 소유한 회사 주소 목록.
func (uc *MailUseCase) ListMailboxes(userID int) ([]*mail.MailboxItem, error) {
	items, err := uc.repo.ListMailboxesForUser(userID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*mail.MailboxItem{}
	}
	return items, nil
}

// ClaimPersonalAddress — 개인 메일 주소 발급/재요청. 승인 전이면 변경 가능, 승인 후엔 불변.
func (uc *MailUseCase) ClaimPersonalAddress(userID int, localPart string) (*mail.Address, error) {
	return uc.claimAddress(mail.OwnerUser, userID, userID, localPart)
}

// ClaimCompanyAddress — 회사 메일 주소 발급/재요청. 회사 소유주만 가능(아니면 ErrForbidden).
// user_id 는 회사 소유주 유저(알림 수신 책임자)로 채운다.
func (uc *MailUseCase) ClaimCompanyAddress(companyID, actorID int, localPart string) (*mail.Address, error) {
	ownerID, _, err := uc.repo.GetCompanyOwnerName(companyID)
	if err != nil {
		return nil, err
	}
	if ownerID == 0 {
		return nil, mail.ErrNotFound
	}
	if ownerID != actorID {
		return nil, mail.ErrForbidden
	}
	return uc.claimAddress(mail.OwnerCompany, companyID, ownerID, localPart)
}

// claimAddress — 소유 주체(owner_type/owner_id)별 주소 발급/재요청 공통 로직.
//   - 기존 행이 approved 면 변경 불가(ErrAlreadyClaimed).
//   - pending/rejected 면 local_part 변경 + status=pending 리셋.
//   - 유일성 검사는 (owner_type, owner_id) 로 자기 행을 예외 처리한다.
func (uc *MailUseCase) claimAddress(ownerType string, ownerID, userID int, localPart string) (*mail.Address, error) {
	localPart = strings.TrimSpace(localPart)
	if err := mail.ValidateLocalPart(localPart); err != nil {
		return nil, err
	}

	existing, err := uc.repo.GetAddressByOwner(ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status == mail.StatusApproved {
		return nil, mail.ErrAlreadyClaimed
	}

	// 유일성: 다른 소유 주체가 이미 쓰는 local_part 면 거부.
	if taken, err := uc.repo.GetAddressByLocalPart(localPart); err != nil {
		return nil, err
	} else if taken != nil && !(taken.OwnerType == ownerType && taken.OwnerID == ownerID) {
		return nil, mail.ErrAddressTaken
	}

	if existing != nil {
		// 재요청: 같은 행을 갱신하고 pending 으로 리셋.
		if err := uc.repo.UpdateAddressLocalPart(existing.ID, localPart); err != nil {
			return nil, mail.ErrAddressTaken
		}
		existing.LocalPart = localPart
		existing.Status = mail.StatusPending
		return existing, nil
	}

	id, err := uc.repo.CreateAddress(ownerType, ownerID, userID, localPart)
	if err != nil {
		// UNIQUE 경합 등 → 이미 사용 중으로 처리.
		return nil, mail.ErrAddressTaken
	}
	return &mail.Address{
		ID: id, OwnerType: ownerType, OwnerID: ownerID, UserID: userID,
		LocalPart: localPart, Status: mail.StatusPending,
	}, nil
}

// resolveAccessibleAddress — 주소 id 로 조회 + 접근권 검증. admin 은 전권.
func (uc *MailUseCase) resolveAccessibleAddress(addressID, userID int, isAdmin bool) (*mail.Address, error) {
	addr, err := uc.repo.GetAddressByID(addressID)
	if err != nil {
		return nil, err
	}
	if addr == nil {
		return nil, mail.ErrNotFound
	}
	if isAdmin {
		return addr, nil
	}
	ok, err := uc.userOwnsAddress(addr, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, mail.ErrForbidden
	}
	return addr, nil
}

// userOwnsAddress — 개인 주소면 owner_id==userID, 회사 주소면 현재 회사 소유주==userID,
// 공용 주소면 활성(revoked=0) grant 보유자.
func (uc *MailUseCase) userOwnsAddress(addr *mail.Address, userID int) (bool, error) {
	switch addr.OwnerType {
	case mail.OwnerUser:
		return addr.OwnerID == userID, nil
	case mail.OwnerCompany:
		ownerID, _, err := uc.repo.GetCompanyOwnerName(addr.OwnerID)
		if err != nil {
			return false, err
		}
		return ownerID == userID, nil
	case mail.OwnerShared:
		return uc.repo.HasActiveGrant(addr.ID, userID)
	}
	return false, nil
}

// ListBox — box=inbox|sent 목록. 특정 주소(메일함) 스코프. 접근권 없으면 ErrForbidden.
func (uc *MailUseCase) ListBox(userID, addressID int, box string, isAdmin bool, limit, offset int) ([]*mail.EmailListItem, int, error) {
	if _, err := uc.resolveAccessibleAddress(addressID, userID, isAdmin); err != nil {
		return nil, 0, err
	}
	direction := mail.DirectionIn
	if box == "sent" {
		direction = mail.DirectionOut
	}
	return uc.repo.ListEmails(addressID, direction, limit, offset)
}

// ListAddressesAdmin — 관리자 승인 목록. status="" 면 전체.
func (uc *MailUseCase) ListAddressesAdmin(status string) ([]*mail.AddressAdminItem, error) {
	items, err := uc.repo.ListAddressesAdmin(status)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*mail.AddressAdminItem{}
	}
	return items, nil
}

// ApproveAddress — 관리자 승인. approved 재승인은 409(ErrAlreadyApproved).
// rejected → approved 오버라이드 허용.
func (uc *MailUseCase) ApproveAddress(addressID int) (*mail.Address, error) {
	addr, err := uc.repo.GetAddressByID(addressID)
	if err != nil {
		return nil, err
	}
	if addr == nil {
		return nil, mail.ErrNotFound
	}
	if addr.Status == mail.StatusApproved {
		return nil, mail.ErrAlreadyApproved
	}
	if err := uc.repo.UpdateAddressStatus(addressID, mail.StatusApproved); err != nil {
		return nil, err
	}
	addr.Status = mail.StatusApproved
	if uc.notifUC != nil {
		body := fmt.Sprintf("%s 사용 가능", mail.EmailFor(addr.LocalPart))
		_ = uc.notifUC.CreateNotification(addr.UserID, notification.NotifMailAddressApproved,
			"메일 주소 승인", body, "mail", addr.ID)
	}
	return addr, nil
}

// RejectAddress — 관리자 반려. approved 는 반려 불가(409, 불변성 유지).
func (uc *MailUseCase) RejectAddress(addressID int) (*mail.Address, error) {
	addr, err := uc.repo.GetAddressByID(addressID)
	if err != nil {
		return nil, err
	}
	if addr == nil {
		return nil, mail.ErrNotFound
	}
	if addr.Status == mail.StatusApproved {
		return nil, mail.ErrAlreadyApproved
	}
	if err := uc.repo.UpdateAddressStatus(addressID, mail.StatusRejected); err != nil {
		return nil, err
	}
	addr.Status = mail.StatusRejected
	if uc.notifUC != nil {
		body := fmt.Sprintf("%s 주소 요청이 반려되었습니다. 다시 신청해 주세요.", mail.EmailFor(addr.LocalPart))
		_ = uc.notifUC.CreateNotification(addr.UserID, notification.NotifMailAddressRejected,
			"메일 주소 반려", body, "mail", addr.ID)
	}
	return addr, nil
}

// ListAll — 관리자 전체 메일 목록.
func (uc *MailUseCase) ListAll(limit, offset int) ([]*mail.EmailListItem, int, error) {
	return uc.repo.ListAllEmails(limit, offset)
}

// hasAddressAccess — 주소 id 로 접근권 판정(admin 전권). 주소가 없으면 false.
func (uc *MailUseCase) hasAddressAccess(addressID, requesterID int, isAdmin bool) (bool, error) {
	if isAdmin {
		return true, nil
	}
	addr, err := uc.repo.GetAddressByID(addressID)
	if err != nil {
		return false, err
	}
	if addr == nil {
		return false, nil
	}
	return uc.userOwnsAddress(addr, requesterID)
}

// GetEmail — 상세 + 첨부. 주소 접근권자(개인/회사 소유·공용 권한) 또는 admin 만.
// 접근권자가 수신 메일을 열면 read=1 (admin 열람은 read 로 표시하지 않음).
func (uc *MailUseCase) GetEmail(emailID, requesterID int, isAdmin bool) (*mail.Email, []*mail.Attachment, error) {
	e, err := uc.repo.GetEmailByID(emailID)
	if err != nil {
		return nil, nil, err
	}
	if e == nil {
		return nil, nil, mail.ErrNotFound
	}
	access, err := uc.hasAddressAccess(e.AddressID, requesterID, isAdmin)
	if err != nil {
		return nil, nil, err
	}
	if !access {
		return nil, nil, mail.ErrForbidden
	}
	atts, err := uc.repo.ListAttachments(emailID)
	if err != nil {
		return nil, nil, err
	}
	if !isAdmin && e.Direction == mail.DirectionIn && !e.Read {
		if err := uc.repo.MarkRead(emailID); err == nil {
			e.Read = true
		}
	}
	return e, atts, nil
}

// GetAttachmentForAccess — 다운로드 권한 검증 (parent 메일 주소 접근권자 또는 admin).
func (uc *MailUseCase) GetAttachmentForAccess(attID, requesterID int, isAdmin bool) (*mail.Attachment, error) {
	a, err := uc.repo.GetAttachmentByID(attID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, mail.ErrNotFound
	}
	access, err := uc.hasAddressAccess(a.AddressID, requesterID, isAdmin)
	if err != nil {
		return nil, err
	}
	if !access {
		return nil, mail.ErrForbidden
	}
	return a, nil
}

// SendInput — 발신 요청.
type SendInput struct {
	To          string
	Subject     string
	BodyText    string
	InReplyToID *int
}

// Send — 지정한 주소(메일함)로 메일 발신. 접근권 + 승인 필요. SES 실패 시 저장하지 않고 ErrSendFailed.
func (uc *MailUseCase) Send(userID, addressID int, in SendInput) (int, error) {
	addr, err := uc.resolveAccessibleAddress(addressID, userID, false)
	if err != nil {
		return 0, err
	}
	if addr.Status != mail.StatusApproved {
		return 0, mail.ErrNotApproved
	}

	fromName, err := uc.fromDisplayName(addr)
	if err != nil {
		return 0, err
	}
	myAddr := mail.EmailFor(addr.LocalPart)
	// 표시명은 사용자/회사가 제어 → RFC 5322 quoted-string 으로 감싸 헤더 인젝션/오인 방지.
	fromDisplay := fmt.Sprintf("%s <%s>", quoteDisplayName(fromName), myAddr)

	var inReplyTo, references string
	if in.InReplyToID != nil {
		orig, err := uc.repo.GetEmailByID(*in.InReplyToID)
		if err != nil {
			return 0, err
		}
		if orig == nil {
			return 0, mail.ErrNotFound
		}
		// 답장 원본은 같은 메일함 소속이어야 한다.
		if orig.AddressID != addressID {
			return 0, mail.ErrForbidden
		}
		// 원본 message_id/refs 에 CR/LF 가 있으면 헤더 인젝션이 되므로 제거 후 사용.
		origMsgID := stripHeaderNewlines(orig.MessageID)
		inReplyTo = origMsgID
		references = strings.TrimSpace(stripHeaderNewlines(orig.Refs) + " " + origMsgID)
	}

	msg := email.OutgoingMail{
		FromDisplay: fromDisplay,
		To:          in.To,
		Subject:     in.Subject,
		TextBody:    in.BodyText,
		InReplyTo:   inReplyTo,
		References:  references,
		ReplyTo:     myAddr,
	}
	if err := uc.sender.SendMailFrom(msg); err != nil {
		// 발신기 비활성은 별도(503)로 구분 — 어느 경우든 이 아래 저장은 실행되지 않는다(거짓 성공 방지).
		if errors.Is(err, email.ErrSenderDisabled) {
			return 0, mail.ErrSendDisabled
		}
		return 0, mail.ErrSendFailed
	}

	e := &mail.Email{
		AddressID:   addressID,
		OwnerUserID: userID, // 실제 발신자
		Direction:   mail.DirectionOut,
		FromAddr:    myAddr,
		// 표시용 헤더 From (#171): 발신은 내 주소/이름이 곧 헤더 From.
		HeaderFrom:     myAddr,
		HeaderFromName: fromName,
		ToAddr:      in.To,
		Subject:     in.Subject,
		BodyText:    in.BodyText,
		InReplyTo:   inReplyTo,
		Refs:        references,
		Read:        true,
	}
	return uc.repo.CreateEmail(e)
}

// stripHeaderNewlines — 헤더 값에서 CR/LF 제거 (헤더 인젝션 방지).
func stripHeaderNewlines(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}

// quoteDisplayName — From 표시명을 RFC 5322 quoted-string 으로 감싼다.
// CR/LF 제거 후 항상 큰따옴표로 감싸고 \ 와 " 를 이스케이프한다 (순서 중요: \ 먼저).
func quoteDisplayName(name string) string {
	name = stripHeaderNewlines(name)
	name = strings.ReplaceAll(name, `\`, `\\`)
	name = strings.ReplaceAll(name, `"`, `\"`)
	return `"` + name + `"`
}

// fromDisplayName — 발신 표시 이름: 개인=유저명, 회사=회사명, 공용=display_name.
func (uc *MailUseCase) fromDisplayName(addr *mail.Address) (string, error) {
	switch addr.OwnerType {
	case mail.OwnerCompany:
		_, name, err := uc.repo.GetCompanyOwnerName(addr.OwnerID)
		return name, err
	case mail.OwnerShared:
		return addr.DisplayName, nil
	default:
		name, _, err := uc.repo.GetUserNameEmail(addr.OwnerID)
		return name, err
	}
}

// CreateSharedAddress — 관리자 공용 주소 생성. 형식 검증만(예약어 허용), 즉시 approved.
func (uc *MailUseCase) CreateSharedAddress(adminUserID int, localPart, displayName string) (*mail.Address, error) {
	localPart = strings.TrimSpace(localPart)
	if err := mail.ValidateLocalPartFormat(localPart); err != nil {
		return nil, err
	}
	if taken, err := uc.repo.GetAddressByLocalPart(localPart); err != nil {
		return nil, err
	} else if taken != nil {
		return nil, mail.ErrAddressTaken
	}
	id, err := uc.repo.CreateSharedAddress(adminUserID, localPart, strings.TrimSpace(displayName))
	if err != nil {
		return nil, mail.ErrAddressTaken
	}
	return &mail.Address{
		ID: id, OwnerType: mail.OwnerShared, OwnerID: adminUserID, UserID: adminUserID,
		LocalPart: localPart, DisplayName: strings.TrimSpace(displayName), Status: mail.StatusApproved,
	}, nil
}

// requireSharedAddress — 공용 주소 존재 검증. 없거나 shared 가 아니면 ErrNotFound.
func (uc *MailUseCase) requireSharedAddress(addressID int) (*mail.Address, error) {
	addr, err := uc.repo.GetAddressByID(addressID)
	if err != nil {
		return nil, err
	}
	if addr == nil || addr.OwnerType != mail.OwnerShared {
		return nil, mail.ErrNotFound
	}
	return addr, nil
}

// GrantSharedAccess — 공용함 접근 권한 부여 (관리자).
func (uc *MailUseCase) GrantSharedAccess(addressID, userID int) error {
	if _, err := uc.requireSharedAddress(addressID); err != nil {
		return err
	}
	return uc.repo.GrantSharedAccess(addressID, userID)
}

// RevokeSharedAccess — 공용함 접근 권한 회수 (관리자).
func (uc *MailUseCase) RevokeSharedAccess(addressID, userID int) error {
	if _, err := uc.requireSharedAddress(addressID); err != nil {
		return err
	}
	return uc.repo.RevokeSharedAccess(addressID, userID)
}

// ListSharedAddresses — 관리자 공용함 목록 (grant 포함).
func (uc *MailUseCase) ListSharedAddresses() ([]*mail.SharedAddressItem, error) {
	items, err := uc.repo.ListSharedAddresses()
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*mail.SharedAddressItem{}
	}
	return items, nil
}

// InboundAttachment — webhook 첨부.
type InboundAttachment struct {
	Filename      string `json:"filename"`
	Mime          string `json:"mime"`
	ContentBase64 string `json:"content_base64"`
}

// InboundInput — webhook 페이로드.
type InboundInput struct {
	From        string              `json:"from"`
	To          string              `json:"to"`
	Subject     string              `json:"subject"`
	Text        string              `json:"text"`
	HTML        string              `json:"html"`
	// HeaderFrom/HeaderFromName — 파싱된 헤더 From (#171, 표시용). 봉투 From 과 다를 수 있고 위조 가능 —
	// 신뢰 판단은 봉투(From)를 쓰고 이 값은 표시에만 쓴다.
	HeaderFrom     string `json:"header_from"`
	HeaderFromName string `json:"header_from_name"`
	MessageID   string              `json:"message_id"`
	InReplyTo   string              `json:"in_reply_to"`
	References  string              `json:"references"`
	Attachments []InboundAttachment `json:"attachments"`
}

// ReceiveInbound — webhook 수신 처리: 저장 + 첨부 + 알림. 미지의 수신자면 ErrNotFound.
func (uc *MailUseCase) ReceiveInbound(in InboundInput) (int, error) {
	localPart, ok := parseRecipientLocalPart(in.To)
	if !ok {
		return 0, mail.ErrNotFound
	}
	addr, err := uc.repo.GetAddressByLocalPart(localPart)
	if err != nil {
		return 0, err
	}
	// 승인된 주소만 수신 (pending/rejected → 미지의 수신자로 취급 → worker 가 SMTP 거절).
	if addr == nil || addr.Status != mail.StatusApproved {
		return 0, mail.ErrNotFound
	}

	// 첨부 먼저 디코드 + 총량 검증 (초과 시 아무것도 저장하지 않음).
	type decoded struct {
		filename string
		mime     string
		data     []byte
	}
	var files []decoded
	var total int
	for _, a := range in.Attachments {
		raw, derr := base64.StdEncoding.DecodeString(strings.TrimSpace(a.ContentBase64))
		if derr != nil {
			continue // 손상된 첨부는 건너뜀
		}
		total += len(raw)
		if total > MaxInboundTotalBytes {
			return 0, mail.ErrAttachmentsTooLarge
		}
		files = append(files, decoded{filename: a.Filename, mime: a.Mime, data: raw})
	}

	// 공용함 수신은 특정 소유자가 없으므로 owner_user_id=0, 이후 권한자 전원에게 fan-out.
	ownerUserID := addr.UserID
	if addr.OwnerType == mail.OwnerShared {
		ownerUserID = 0
	}

	e := &mail.Email{
		AddressID:   addr.ID,
		OwnerUserID: ownerUserID,
		Direction:   mail.DirectionIn,
		FromAddr:    in.From,
		HeaderFrom:     strings.TrimSpace(in.HeaderFrom),
		HeaderFromName: strings.TrimSpace(in.HeaderFromName),
		ToAddr:      in.To,
		Subject:     in.Subject,
		BodyText:    in.Text,
		BodyHTML:    in.HTML,
		MessageID:   in.MessageID,
		InReplyTo:   in.InReplyTo,
		Refs:        in.References,
		Read:        false,
	}
	emailID, err := uc.repo.CreateEmail(e)
	if err != nil {
		return 0, err
	}

	// 첨부 저장: PrivateUploadPath/mail/<emailID>/<index>_<sanitized>.
	if len(files) > 0 && uc.privateUploadPath != "" {
		dir := filepath.Join(uc.privateUploadPath, "mail", fmt.Sprintf("%d", emailID))
		if err := os.MkdirAll(dir, 0755); err == nil {
			for i, f := range files {
				stored := fmt.Sprintf("%d_%s", i, sanitizeMailFilename(f.filename))
				storedPath := filepath.Join(dir, stored)
				if werr := os.WriteFile(storedPath, f.data, 0644); werr != nil {
					continue
				}
				_, _ = uc.repo.AddAttachment(&mail.Attachment{
					EmailID:    emailID,
					Filename:   f.filename,
					Mime:       f.mime,
					Size:       int64(len(f.data)),
					StoredPath: storedPath,
				})
			}
		}
	}

	// 알림 (WS + push + 계정메일). notif_type=mail_received, reference=mail.
	// 개인/회사 → 책임 유저 1명, 공용 → 활성 권한자 전원.
	if uc.notifUC != nil {
		body := fmt.Sprintf("%s · %s", in.From, mail.Snippet(in.Subject, snippetSubjectLen))
		recipients := []int{addr.UserID}
		switch addr.OwnerType {
		case mail.OwnerShared:
			ids, gerr := uc.repo.ListActiveGrantUserIDs(addr.ID)
			if gerr != nil {
				ids = nil
			}
			recipients = ids
		case mail.OwnerCompany:
			// 소유권이 바뀌었을 수 있으므로 발급 시점(addr.UserID)이 아닌 현재 소유주에게 알린다.
			if ownerID, _, oerr := uc.repo.GetCompanyOwnerName(addr.OwnerID); oerr == nil && ownerID > 0 {
				recipients = []int{ownerID}
			}
		}
		for _, uid := range recipients {
			if uid <= 0 {
				continue
			}
			_ = uc.notifUC.CreateNotification(uid, notification.NotifMailReceived, "새 메일", body, "mail", emailID)
		}
	}

	return emailID, nil
}

// parseRecipientLocalPart — "Name <local@earnlearning.com>" / "local@earnlearning.com"
// 에서 소문자 local part 추출. 도메인이 earnlearning.com 이 아니면 false.
func parseRecipientLocalPart(to string) (string, bool) {
	to = strings.TrimSpace(to)
	if parsed, err := netmail.ParseAddress(to); err == nil {
		to = parsed.Address
	}
	at := strings.LastIndex(to, "@")
	if at <= 0 {
		return "", false
	}
	local := strings.ToLower(strings.TrimSpace(to[:at]))
	domain := strings.ToLower(strings.TrimSpace(to[at+1:]))
	if local == "" || domain != mail.Domain {
		return "", false
	}
	return local, true
}

// sanitizeMailFilename — 경로 요소 제거 후 파일명만 남긴다.
func sanitizeMailFilename(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	base := strings.TrimSpace(filepath.Base(name))
	if base == "" || base == "." || base == ".." {
		return "attachment"
	}
	return base
}
