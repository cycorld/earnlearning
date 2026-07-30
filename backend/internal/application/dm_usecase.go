package application

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/earnlearning/backend/internal/domain/dm"
	"github.com/earnlearning/backend/internal/domain/notification"
	"github.com/earnlearning/backend/internal/domain/user"
)

type DMUseCase struct {
	repo     dm.Repository
	userRepo user.Repository
	hub      WSBroadcaster
	notifUC  *NotificationUseCase
	// #184 첨부 저장 루트 (비공개). 실제 파일은 <privateUploadPath>/dm 아래.
	privateUploadPath string
}

func NewDMUseCase(repo dm.Repository, userRepo user.Repository, hub WSBroadcaster, privateUploadPath string) *DMUseCase {
	return &DMUseCase{repo: repo, userRepo: userRepo, hub: hub, privateUploadPath: privateUploadPath}
}

// #184 DM 첨부 제한. nginx client_max_body_size 50M 아래로 잡는다.
const (
	MaxDMAttachments          = 4
	MaxDMAttachmentTotalBytes = 20 * 1024 * 1024
)

// allowedDMAttachmentExt — .md/.txt/.json 은 제외 (전체 버퍼 검증이 필요해 DM 에서는 지원하지 않음).
var allowedDMAttachmentExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true, ".zip": true, ".mp4": true, ".mp3": true,
	".html": true, ".htm": true,
}

// inlineDMImageMIME — inline 렌더를 허용하는 안전한 이미지 타입만.
var inlineDMImageMIME = map[string]string{
	".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png",
	".gif": "image/gif", ".webp": "image/webp",
}

// DMAttachmentInlineMIME — 저장 파일명(확장자)으로 서버가 직접 판단한 inline MIME.
// DB mime 컬럼은 업로드 당시 클라이언트 입력이라 신뢰하지 않는다. inline 불가면 "".
func DMAttachmentInlineMIME(storedName string) string {
	return inlineDMImageMIME[strings.ToLower(filepath.Ext(storedName))]
}

func (uc *DMUseCase) SetNotificationUseCase(notifUC *NotificationUseCase) {
	uc.notifUC = notifUC
}

type SendDMInput struct {
	ReceiverID int    `json:"receiver_id"`
	Content    string `json:"content"`
}

var (
	ErrCannotDMSelf = errors.New("자기 자신에게 메시지를 보낼 수 없습니다")
	ErrEmptyMessage = errors.New("메시지 내용을 입력하세요")
	ErrUserNotFound = errors.New("사용자를 찾을 수 없습니다")
)

func (uc *DMUseCase) SendMessage(senderID int, input SendDMInput) (*dm.Message, error) {
	return uc.SendMessageWithAttachments(senderID, input, nil, nil)
}

// SendMessageWithAttachments — 텍스트만 / 첨부만 / 둘 다 모두 허용 (#184).
// 디스크에 쓴 파일은 이후 어떤 단계에서 실패하든 전부 지운다 (고아 파일 금지).
func (uc *DMUseCase) SendMessageWithAttachments(senderID int, input SendDMInput, files []*multipart.FileHeader, uuidGen func() string) (*dm.Message, error) {
	if senderID == input.ReceiverID {
		return nil, ErrCannotDMSelf
	}
	if strings.TrimSpace(input.Content) == "" && len(files) == 0 {
		return nil, ErrEmptyMessage
	}
	if len(files) > MaxDMAttachments {
		return nil, fmt.Errorf("첨부는 최대 %d개까지 가능합니다", MaxDMAttachments)
	}
	var total int64
	for _, f := range files {
		total += f.Size
	}
	if total > MaxDMAttachmentTotalBytes {
		return nil, fmt.Errorf("첨부 합계 크기는 최대 20MB까지 허용됩니다")
	}

	// Verify receiver exists
	if _, err := uc.userRepo.FindByID(input.ReceiverID); err != nil {
		return nil, ErrUserNotFound
	}

	attachments, cleanup, err := uc.storeAttachments(files, uuidGen)
	if err != nil {
		cleanup()
		return nil, err
	}

	msg := &dm.Message{
		SenderID:    senderID,
		ReceiverID:  input.ReceiverID,
		Content:     input.Content,
		Attachments: attachments,
	}
	id, err := uc.repo.SendMessage(msg)
	if err != nil {
		cleanup()
		return nil, err
	}
	msg.ID = id
	msg.CreatedAt = time.Now()

	// Send real-time notification via WebSocket to both parties
	if uc.hub != nil {
		wsMsg := map[string]interface{}{
			"event": "dm",
			"data":  msg,
		}
		uc.hub.SendToUser(input.ReceiverID, wsMsg)
		uc.hub.SendToUser(senderID, wsMsg)
	}

	// Send push/email notification to receiver
	if uc.notifUC != nil {
		sender, _ := uc.userRepo.FindByID(senderID)
		senderName := "알 수 없음"
		if sender != nil {
			senderName = sender.Name
		}
		preview := input.Content
		if strings.TrimSpace(preview) == "" && len(msg.Attachments) > 0 {
			preview = dmAttachmentPreview(msg.Attachments)
		}
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		_ = uc.notifUC.CreateNotification(
			input.ReceiverID,
			notification.NotifNewDM,
			fmt.Sprintf("%s님의 새 메시지", senderName),
			preview,
			"dm", senderID,
		)
	}

	return msg, nil
}

func (uc *DMUseCase) GetMessages(userID, peerID, limit, beforeID int) ([]*dm.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	return uc.repo.GetMessages(userID, peerID, limit, beforeID)
}

func (uc *DMUseCase) GetConversations(userID int) ([]*dm.Conversation, error) {
	return uc.repo.GetConversations(userID)
}

func (uc *DMUseCase) MarkAsRead(userID, peerID int) error {
	return uc.repo.MarkAsRead(userID, peerID)
}

func (uc *DMUseCase) GetUnreadCount(userID int) (int, error) {
	return uc.repo.GetUnreadCount(userID)
}

// GetAttachmentForAccess — 해당 DM 의 발신자/수신자만 접근 가능. 관리자 우회 없음 (#184).
func (uc *DMUseCase) GetAttachmentForAccess(attID, requesterID int) (*dm.Attachment, error) {
	a, err := uc.repo.FindAttachmentByID(attID)
	if err != nil {
		return nil, err
	}
	msg, err := uc.repo.FindMessageByID(a.MessageID)
	if err != nil {
		return nil, err
	}
	if requesterID != msg.SenderID && requesterID != msg.ReceiverID {
		return nil, dm.ErrForbidden
	}
	return a, nil
}

// dmAttachmentPreview — 본문이 빈 첨부-only DM 의 알림 미리보기.
func dmAttachmentPreview(atts []*dm.Attachment) string {
	allImages := true
	for _, a := range atts {
		if DMAttachmentInlineMIME(a.StoredName) == "" {
			allImages = false
			break
		}
	}
	if allImages {
		return "사진을 보냈습니다"
	}
	return fmt.Sprintf("파일 %d개를 보냈습니다", len(atts))
}

// storeAttachments — 확장자·MIME·매직넘버 3중 검증 후 비공개 경로에 저장.
// 두 번째 반환값은 이미 쓴 파일을 모두 지우는 정리 함수 (성공 경로에서는 호출하지 않는다).
func (uc *DMUseCase) storeAttachments(files []*multipart.FileHeader, uuidGen func() string) ([]*dm.Attachment, func(), error) {
	attachments := make([]*dm.Attachment, 0, len(files))
	written := make([]string, 0, len(files))
	cleanup := func() {
		for _, p := range written {
			_ = os.Remove(p)
		}
	}
	if len(files) == 0 {
		return attachments, cleanup, nil
	}
	if uc.privateUploadPath == "" {
		return nil, cleanup, fmt.Errorf("파일 저장소가 설정되지 않았습니다")
	}
	dir := filepath.Join(uc.privateUploadPath, "dm")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, cleanup, fmt.Errorf("저장 디렉토리 생성 실패: %w", err)
	}
	// #184 DM 첨부는 비공개. MkdirAll 은 기존 디렉토리 권한을 바꾸지 않으므로 명시적으로 조인다.
	if err := os.Chmod(dir, 0700); err != nil {
		return nil, cleanup, fmt.Errorf("저장 디렉토리 권한 설정 실패: %w", err)
	}

	for _, fh := range files {
		if err := validateMilestoneFilename(fh.Filename); err != nil {
			return nil, cleanup, err
		}
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		if !allowedDMAttachmentExt[ext] {
			return nil, cleanup, fmt.Errorf("허용되지 않는 파일 형식입니다: %s", ext)
		}
		if fh.Size < 0 || fh.Size > MaxUploadSize {
			return nil, cleanup, fmt.Errorf("파일 크기는 최대 10MB까지 허용됩니다")
		}

		mimeType := fh.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		if idx := strings.Index(mimeType, ";"); idx != -1 {
			mimeType = strings.TrimSpace(mimeType[:idx])
		}
		if mimeType != "application/octet-stream" && !allowedMIMETypesByExtension[ext][mimeType] {
			return nil, cleanup, fmt.Errorf("파일 확장자와 MIME 형식이 일치하지 않습니다")
		}

		src, err := fh.Open()
		if err != nil {
			return nil, cleanup, fmt.Errorf("파일 열기 실패: %w", err)
		}
		reader := bufio.NewReader(src)
		sample, err := reader.Peek(512)
		if err != nil && err != io.EOF && err != bufio.ErrBufferFull {
			src.Close()
			return nil, cleanup, fmt.Errorf("파일 검사 실패: %w", err)
		}
		if err := validateUploadContent(ext, sample); err != nil {
			src.Close()
			return nil, cleanup, err
		}

		storedName := uuidGen() + ext
		storedPath := filepath.Join(dir, storedName)
		n, err := copyUploadFile(storedPath, io.LimitReader(reader, MaxUploadSize+1))
		src.Close()
		if err != nil {
			return nil, cleanup, fmt.Errorf("파일 저장 실패: %w", err)
		}
		written = append(written, storedPath)
		// #184 저장 파일도 소유자만 읽도록 제한 (공용 /uploads 와 달리 비공개 경로).
		if err := os.Chmod(storedPath, 0600); err != nil {
			return nil, cleanup, fmt.Errorf("파일 권한 설정 실패: %w", err)
		}
		if n > MaxUploadSize {
			return nil, cleanup, fmt.Errorf("파일 크기는 최대 10MB까지 허용됩니다")
		}

		attachments = append(attachments, &dm.Attachment{
			Filename:   fh.Filename,
			StoredName: storedName,
			Mime:       mimeType,
			Size:       n,
			Path:       storedPath,
		})
	}
	return attachments, cleanup, nil
}
