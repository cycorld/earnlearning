package persistence

import (
	"database/sql"
	"fmt"

	"github.com/earnlearning/backend/internal/domain/dm"
)

type DMRepo struct {
	db *sql.DB
}

func NewDMRepo(db *sql.DB) *DMRepo {
	return &DMRepo{db: db}
}

func (r *DMRepo) SendMessage(msg *dm.Message) (int, error) {
	res, err := r.db.Exec(
		`INSERT INTO dm_messages (sender_id, receiver_id, content) VALUES (?, ?, ?)`,
		msg.SenderID, msg.ReceiverID, msg.Content,
	)
	if err != nil {
		return 0, fmt.Errorf("insert dm message: %w", err)
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (r *DMRepo) GetMessages(userID, peerID, limit, beforeID int) ([]*dm.Message, error) {
	query := `SELECT id, sender_id, receiver_id, content, is_read, created_at
		FROM dm_messages
		WHERE ((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?))`
	args := []interface{}{userID, peerID, peerID, userID}

	if beforeID > 0 {
		query += ` AND id < ?`
		args = append(args, beforeID)
	}
	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("get dm messages: %w", err)
	}
	defer rows.Close()

	var messages []*dm.Message
	for rows.Next() {
		m := &dm.Message{}
		var isRead int
		if err := rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.Content, &isRead, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan dm message: %w", err)
		}
		m.IsRead = isRead != 0
		messages = append(messages, m)
	}

	// Reverse to chronological order (oldest first)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

func (r *DMRepo) GetConversations(userID int) ([]*dm.Conversation, error) {
	rows, err := r.db.Query(`
		SELECT
			pairs.peer_id,
			u.name,
			u.avatar_url,
			m.content,
			m.created_at,
			COALESCE(unread.cnt, 0)
		FROM (
			SELECT
				CASE WHEN sender_id = ? THEN receiver_id ELSE sender_id END AS peer_id,
				MAX(id) AS last_msg_id
			FROM dm_messages
			WHERE sender_id = ? OR receiver_id = ?
			GROUP BY peer_id
		) pairs
		JOIN dm_messages m ON m.id = pairs.last_msg_id
		JOIN users u ON u.id = pairs.peer_id
		LEFT JOIN (
			SELECT sender_id, COUNT(*) AS cnt
			FROM dm_messages
			WHERE receiver_id = ? AND is_read = 0
			GROUP BY sender_id
		) unread ON unread.sender_id = pairs.peer_id
		ORDER BY m.created_at DESC`, userID, userID, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("get dm conversations: %w", err)
	}
	defer rows.Close()

	var convs []*dm.Conversation
	for rows.Next() {
		c := &dm.Conversation{}
		if err := rows.Scan(&c.PeerID, &c.PeerName, &c.PeerAvatarURL, &c.LastMessage, &c.LastMessageAt, &c.UnreadCount); err != nil {
			return nil, fmt.Errorf("scan dm conversation: %w", err)
		}
		convs = append(convs, c)
	}
	return convs, nil
}

func (r *DMRepo) MarkAsRead(userID, peerID int) error {
	_, err := r.db.Exec(
		`UPDATE dm_messages SET is_read = 1 WHERE receiver_id = ? AND sender_id = ? AND is_read = 0`,
		userID, peerID,
	)
	return err
}

func (r *DMRepo) GetUnreadCount(userID int) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM dm_messages WHERE receiver_id = ? AND is_read = 0`, userID).Scan(&count)
	return count, err
}
