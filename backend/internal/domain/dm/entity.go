package dm

import (
	"errors"
	"time"
)

type Message struct {
	ID         int       `json:"id"`
	SenderID   int       `json:"sender_id"`
	ReceiverID int       `json:"receiver_id"`
	Content    string    `json:"content"`
	IsRead     bool      `json:"is_read"`
	CreatedAt  time.Time `json:"created_at"`
	// #184 첨부. null 대신 항상 [] 로 직렬화한다.
	Attachments []*Attachment `json:"attachments"`
}

// Attachment — #184 DM 첨부. 실제 파일은 비공개 경로에만 있고 Path 는 절대 노출하지 않는다.
type Attachment struct {
	ID         int       `json:"id"`
	MessageID  int       `json:"message_id"`
	Filename   string    `json:"filename"`
	StoredName string    `json:"stored_name"`
	Mime       string    `json:"mime"`
	Size       int64     `json:"size"`
	Path       string    `json:"-"`
	CreatedAt  time.Time `json:"created_at"`
}

var (
	ErrAttachmentNotFound = errors.New("첨부를 찾을 수 없습니다")
	ErrMessageNotFound    = errors.New("메시지를 찾을 수 없습니다")
	ErrForbidden          = errors.New("접근 권한이 없습니다")
)

type Conversation struct {
	PeerID        int       `json:"peer_id"`
	PeerName      string    `json:"peer_name"`
	PeerAvatarURL string    `json:"peer_avatar_url"`
	LastMessage   string    `json:"last_message"`
	LastMessageAt time.Time `json:"last_message_at"`
	UnreadCount   int       `json:"unread_count"`
}
