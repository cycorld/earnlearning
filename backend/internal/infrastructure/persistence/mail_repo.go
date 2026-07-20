package persistence

import (
	"database/sql"

	"github.com/earnlearning/backend/internal/domain/mail"
)

// MailRepo — #166 학생 메일함 영속성.
type MailRepo struct {
	db *sql.DB
}

func NewMailRepo(db *sql.DB) *MailRepo {
	return &MailRepo{db: db}
}

// snippetLen — 목록 스니펫 길이 (룬 기준).
const snippetLen = 120

func (r *MailRepo) GetAddressByUserID(userID int) (*mail.Address, error) {
	a := &mail.Address{}
	err := r.db.QueryRow(
		`SELECT id, user_id, local_part, created_at FROM mail_addresses WHERE user_id = ?`, userID,
	).Scan(&a.ID, &a.UserID, &a.LocalPart, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *MailRepo) GetAddressByLocalPart(localPart string) (*mail.Address, error) {
	a := &mail.Address{}
	err := r.db.QueryRow(
		`SELECT id, user_id, local_part, created_at FROM mail_addresses WHERE local_part = ?`, localPart,
	).Scan(&a.ID, &a.UserID, &a.LocalPart, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *MailRepo) CreateAddress(userID int, localPart string) (int, error) {
	res, err := r.db.Exec(
		`INSERT INTO mail_addresses (user_id, local_part) VALUES (?, ?)`, userID, localPart,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *MailRepo) CreateEmail(e *mail.Email) (int, error) {
	readVal := 0
	if e.Read {
		readVal = 1
	}
	res, err := r.db.Exec(
		`INSERT INTO emails (owner_user_id, direction, from_addr, to_addr, subject,
			body_text, body_html, message_id, in_reply_to, refs, read)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.OwnerUserID, e.Direction, e.FromAddr, e.ToAddr, e.Subject,
		e.BodyText, e.BodyHTML, e.MessageID, e.InReplyTo, e.Refs, readVal,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *MailRepo) GetEmailByID(id int) (*mail.Email, error) {
	e := &mail.Email{}
	var readVal int
	err := r.db.QueryRow(
		`SELECT id, owner_user_id, direction, from_addr, to_addr, subject,
			body_text, body_html, message_id, in_reply_to, refs, read, created_at
		 FROM emails WHERE id = ?`, id,
	).Scan(&e.ID, &e.OwnerUserID, &e.Direction, &e.FromAddr, &e.ToAddr, &e.Subject,
		&e.BodyText, &e.BodyHTML, &e.MessageID, &e.InReplyTo, &e.Refs, &readVal, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	e.Read = readVal == 1
	return e, nil
}

func (r *MailRepo) ListEmails(ownerUserID int, direction string, limit, offset int) ([]*mail.EmailListItem, int, error) {
	var total int
	if err := r.db.QueryRow(
		`SELECT COUNT(*) FROM emails WHERE owner_user_id = ? AND direction = ?`,
		ownerUserID, direction,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(
		`SELECT e.id, e.direction, e.from_addr, e.to_addr, e.subject, e.body_text, e.read, e.created_at,
			EXISTS(SELECT 1 FROM mail_attachments a WHERE a.email_id = e.id) AS has_attachments
		 FROM emails e
		 WHERE e.owner_user_id = ? AND e.direction = ?
		 ORDER BY e.created_at DESC, e.id DESC
		 LIMIT ? OFFSET ?`,
		ownerUserID, direction, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items, err := scanListItems(rows, false)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *MailRepo) ListAllEmails(limit, offset int) ([]*mail.EmailListItem, int, error) {
	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM emails`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(
		`SELECT e.id, e.direction, e.from_addr, e.to_addr, e.subject, e.body_text, e.read, e.created_at,
			EXISTS(SELECT 1 FROM mail_attachments a WHERE a.email_id = e.id) AS has_attachments,
			e.owner_user_id, COALESCE(u.name, '')
		 FROM emails e
		 LEFT JOIN users u ON u.id = e.owner_user_id
		 ORDER BY e.created_at DESC, e.id DESC
		 LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items, err := scanListItems(rows, true)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// scanListItems — 목록 rows 스캔. withOwner 면 owner_user_id/owner_name 컬럼 포함.
func scanListItems(rows *sql.Rows, withOwner bool) ([]*mail.EmailListItem, error) {
	var items []*mail.EmailListItem
	for rows.Next() {
		it := &mail.EmailListItem{}
		var bodyText string
		var readVal int
		var hasAttach int
		if withOwner {
			if err := rows.Scan(&it.ID, &it.Direction, &it.FromAddr, &it.ToAddr, &it.Subject,
				&bodyText, &readVal, &it.CreatedAt, &hasAttach, &it.OwnerUserID, &it.OwnerName); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(&it.ID, &it.Direction, &it.FromAddr, &it.ToAddr, &it.Subject,
				&bodyText, &readVal, &it.CreatedAt, &hasAttach); err != nil {
				return nil, err
			}
		}
		it.Snippet = mail.Snippet(bodyText, snippetLen)
		it.Read = readVal == 1
		it.HasAttachments = hasAttach == 1
		items = append(items, it)
	}
	return items, rows.Err()
}

func (r *MailRepo) MarkRead(emailID int) error {
	_, err := r.db.Exec(`UPDATE emails SET read = 1 WHERE id = ?`, emailID)
	return err
}

func (r *MailRepo) AddAttachment(a *mail.Attachment) (int, error) {
	res, err := r.db.Exec(
		`INSERT INTO mail_attachments (email_id, filename, mime, size, stored_path)
		 VALUES (?, ?, ?, ?, ?)`,
		a.EmailID, a.Filename, a.Mime, a.Size, a.StoredPath,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *MailRepo) ListAttachments(emailID int) ([]*mail.Attachment, error) {
	rows, err := r.db.Query(
		`SELECT id, email_id, filename, mime, size, stored_path
		 FROM mail_attachments WHERE email_id = ? ORDER BY id`, emailID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var atts []*mail.Attachment
	for rows.Next() {
		a := &mail.Attachment{}
		if err := rows.Scan(&a.ID, &a.EmailID, &a.Filename, &a.Mime, &a.Size, &a.StoredPath); err != nil {
			return nil, err
		}
		atts = append(atts, a)
	}
	return atts, rows.Err()
}

func (r *MailRepo) GetAttachmentByID(id int) (*mail.Attachment, error) {
	a := &mail.Attachment{}
	err := r.db.QueryRow(
		`SELECT a.id, a.email_id, a.filename, a.mime, a.size, a.stored_path, e.owner_user_id
		 FROM mail_attachments a JOIN emails e ON e.id = a.email_id
		 WHERE a.id = ?`, id,
	).Scan(&a.ID, &a.EmailID, &a.Filename, &a.Mime, &a.Size, &a.StoredPath, &a.OwnerUserID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *MailRepo) GetUserNameEmail(userID int) (string, string, error) {
	var name, email string
	err := r.db.QueryRow(`SELECT name, email FROM users WHERE id = ?`, userID).Scan(&name, &email)
	if err == sql.ErrNoRows {
		return "", "", nil
	}
	return name, email, err
}
