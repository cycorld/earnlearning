package application

import (
	"encoding/base64"
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

// GetAddress — 유저의 발급된 주소 (없으면 nil).
func (uc *MailUseCase) GetAddress(userID int) (*mail.Address, error) {
	return uc.repo.GetAddressByUserID(userID)
}

// ClaimAddress — 개인 메일 주소 발급 (한 번만, 변경 불가).
func (uc *MailUseCase) ClaimAddress(userID int, localPart string) (*mail.Address, error) {
	localPart = strings.TrimSpace(localPart)
	if err := mail.ValidateLocalPart(localPart); err != nil {
		return nil, err
	}
	if existing, err := uc.repo.GetAddressByUserID(userID); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, mail.ErrAlreadyClaimed
	}
	if taken, err := uc.repo.GetAddressByLocalPart(localPart); err != nil {
		return nil, err
	} else if taken != nil {
		return nil, mail.ErrAddressTaken
	}
	id, err := uc.repo.CreateAddress(userID, localPart)
	if err != nil {
		// UNIQUE 경합 등 → 이미 사용 중으로 처리.
		return nil, mail.ErrAddressTaken
	}
	return &mail.Address{ID: id, UserID: userID, LocalPart: localPart}, nil
}

// ListBox — box=inbox|sent 목록 (owner 스코프).
func (uc *MailUseCase) ListBox(userID int, box string, limit, offset int) ([]*mail.EmailListItem, int, error) {
	direction := mail.DirectionIn
	if box == "sent" {
		direction = mail.DirectionOut
	}
	return uc.repo.ListEmails(userID, direction, limit, offset)
}

// ListAll — 관리자 전체 메일 목록.
func (uc *MailUseCase) ListAll(limit, offset int) ([]*mail.EmailListItem, int, error) {
	return uc.repo.ListAllEmails(limit, offset)
}

// GetEmail — 상세 + 첨부. owner 또는 admin 만. owner 가 수신 메일을 열면 read=1.
func (uc *MailUseCase) GetEmail(emailID, requesterID int, isAdmin bool) (*mail.Email, []*mail.Attachment, error) {
	e, err := uc.repo.GetEmailByID(emailID)
	if err != nil {
		return nil, nil, err
	}
	if e == nil {
		return nil, nil, mail.ErrNotFound
	}
	if !isAdmin && e.OwnerUserID != requesterID {
		return nil, nil, mail.ErrForbidden
	}
	atts, err := uc.repo.ListAttachments(emailID)
	if err != nil {
		return nil, nil, err
	}
	if e.OwnerUserID == requesterID && e.Direction == mail.DirectionIn && !e.Read {
		if err := uc.repo.MarkRead(emailID); err == nil {
			e.Read = true
		}
	}
	return e, atts, nil
}

// GetAttachmentForAccess — 다운로드 권한 검증 (parent 메일 owner 또는 admin).
func (uc *MailUseCase) GetAttachmentForAccess(attID, requesterID int, isAdmin bool) (*mail.Attachment, error) {
	a, err := uc.repo.GetAttachmentByID(attID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, mail.ErrNotFound
	}
	if !isAdmin && a.OwnerUserID != requesterID {
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

// Send — 학생 개인 주소로 메일 발신. SES 실패 시 저장하지 않고 ErrSendFailed.
func (uc *MailUseCase) Send(userID int, in SendInput) (int, error) {
	addr, err := uc.repo.GetAddressByUserID(userID)
	if err != nil {
		return 0, err
	}
	if addr == nil {
		return 0, mail.ErrNoAddress
	}

	name, _, err := uc.repo.GetUserNameEmail(userID)
	if err != nil {
		return 0, err
	}
	myAddr := mail.EmailFor(addr.LocalPart)
	fromDisplay := fmt.Sprintf("%s <%s>", name, myAddr)

	var inReplyTo, references string
	if in.InReplyToID != nil {
		orig, err := uc.repo.GetEmailByID(*in.InReplyToID)
		if err != nil {
			return 0, err
		}
		if orig == nil {
			return 0, mail.ErrNotFound
		}
		if orig.OwnerUserID != userID {
			return 0, mail.ErrForbidden
		}
		inReplyTo = orig.MessageID
		references = strings.TrimSpace(orig.Refs + " " + orig.MessageID)
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
		return 0, mail.ErrSendFailed
	}

	e := &mail.Email{
		OwnerUserID: userID,
		Direction:   mail.DirectionOut,
		FromAddr:    myAddr,
		ToAddr:      in.To,
		Subject:     in.Subject,
		BodyText:    in.BodyText,
		InReplyTo:   inReplyTo,
		Refs:        references,
		Read:        true,
	}
	return uc.repo.CreateEmail(e)
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
	if addr == nil {
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

	e := &mail.Email{
		OwnerUserID: addr.UserID,
		Direction:   mail.DirectionIn,
		FromAddr:    in.From,
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

	// 소유자 알림 (WS + push + 계정메일). notif_type=mail_received, reference=mail.
	if uc.notifUC != nil {
		body := fmt.Sprintf("%s · %s", in.From, mail.Snippet(in.Subject, snippetSubjectLen))
		_ = uc.notifUC.CreateNotification(addr.UserID, notification.NotifMailReceived, "새 메일", body, "mail", emailID)
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
