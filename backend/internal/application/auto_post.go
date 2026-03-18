package application

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
)

// AutoPoster creates feed posts automatically when activities happen.
type AutoPoster struct {
	db *sql.DB
}

func NewAutoPoster(db *sql.DB) *AutoPoster {
	return &AutoPoster{db: db}
}

// PostToChannel creates a post in the given channel slug for a user.
// It finds the channel across all classrooms the user belongs to.
func (ap *AutoPoster) PostToChannel(slug string, authorID int, content string, tags []string) {
	if ap == nil || ap.db == nil {
		return
	}

	var channelID int
	err := ap.db.QueryRow(`
		SELECT c.id FROM channels c
		JOIN classroom_members cm ON cm.classroom_id = c.classroom_id
		WHERE c.slug = ? AND cm.user_id = ?
		LIMIT 1`, slug, authorID).Scan(&channelID)
	if err != nil || channelID == 0 {
		return
	}

	tagsJSON, _ := json.Marshal(tags)

	_, err = ap.db.Exec(`
		INSERT INTO posts (channel_id, author_id, content, post_type, media, tags)
		VALUES (?, ?, ?, 'normal', '[]', ?)`,
		channelID, authorID, content, string(tagsJSON))
	if err != nil {
		log.Printf("auto-post: failed to post to %s channel: %v", slug, err)
	}
}

// PostToChannelAsAdmin creates a post in the given channel slug using the admin user.
// Used when the action creator may not be a classroom member (e.g., system posts).
// Returns the post ID (0 if failed).
func (ap *AutoPoster) PostToChannelAsAdmin(slug string, content string, tags []string) int {
	if ap == nil || ap.db == nil {
		return 0
	}

	var channelID, adminID int
	err := ap.db.QueryRow(`
		SELECT c.id, cm.user_id FROM channels c
		JOIN classroom_members cm ON cm.classroom_id = c.classroom_id
		JOIN users u ON u.id = cm.user_id AND u.role = 'admin'
		WHERE c.slug = ?
		LIMIT 1`, slug).Scan(&channelID, &adminID)
	if err != nil || channelID == 0 {
		return 0
	}

	tagsJSON, _ := json.Marshal(tags)

	result, err := ap.db.Exec(`
		INSERT INTO posts (channel_id, author_id, content, post_type, media, tags)
		VALUES (?, ?, ?, 'normal', '[]', ?)`,
		channelID, adminID, content, string(tagsJSON))
	if err != nil {
		log.Printf("auto-post: failed to post to %s channel as admin: %v", slug, err)
		return 0
	}
	id, _ := result.LastInsertId()
	return int(id)
}

func formatMoney(amount int) string {
	if amount >= 100000000 {
		return fmt.Sprintf("%d억원", amount/100000000)
	}
	if amount >= 10000 {
		return fmt.Sprintf("%d만원", amount/10000)
	}
	return fmt.Sprintf("%d원", amount)
}
