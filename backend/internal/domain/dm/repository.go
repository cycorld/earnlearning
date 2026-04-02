package dm

type Repository interface {
	SendMessage(msg *Message) (int, error)
	GetMessages(userID, peerID, limit, beforeID int) ([]*Message, error)
	GetConversations(userID int) ([]*Conversation, error)
	MarkAsRead(userID, peerID int) error
	GetUnreadCount(userID int) (int, error)
}
