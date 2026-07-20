package persistence

import (
	"database/sql"
	"strings"

	"github.com/earnlearning/backend/internal/domain/mail"
)

// MailRepo — #166 학생/회사 메일함 영속성.
type MailRepo struct {
	db *sql.DB
}

func NewMailRepo(db *sql.DB) *MailRepo {
	return &MailRepo{db: db}
}

// snippetLen — 목록 스니펫 길이 (룬 기준).
const snippetLen = 120

// addressCols — 주소 조회 공통 컬럼.
const addressCols = `id, owner_type, owner_id, user_id, local_part, display_name, status, created_at`

func scanAddress(row interface {
	Scan(dest ...interface{}) error
}) (*mail.Address, error) {
	a := &mail.Address{}
	err := row.Scan(&a.ID, &a.OwnerType, &a.OwnerID, &a.UserID, &a.LocalPart, &a.DisplayName, &a.Status, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *MailRepo) GetAddressByID(id int) (*mail.Address, error) {
	return scanAddress(r.db.QueryRow(
		`SELECT `+addressCols+` FROM mail_addresses WHERE id = ?`, id))
}

func (r *MailRepo) GetAddressByOwner(ownerType string, ownerID int) (*mail.Address, error) {
	return scanAddress(r.db.QueryRow(
		`SELECT `+addressCols+` FROM mail_addresses WHERE owner_type = ? AND owner_id = ?`,
		ownerType, ownerID))
}

func (r *MailRepo) GetAddressByLocalPart(localPart string) (*mail.Address, error) {
	return scanAddress(r.db.QueryRow(
		`SELECT `+addressCols+` FROM mail_addresses WHERE local_part = ?`, localPart))
}

func (r *MailRepo) CreateAddress(ownerType string, ownerID, userID int, localPart string) (int, error) {
	res, err := r.db.Exec(
		`INSERT INTO mail_addresses (owner_type, owner_id, user_id, local_part, status)
		 VALUES (?, ?, ?, ?, 'pending')`,
		ownerType, ownerID, userID, localPart,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// CreateSharedAddress — 관리자 공용 주소 생성. 생성 즉시 approved.
func (r *MailRepo) CreateSharedAddress(adminUserID int, localPart, displayName string) (int, error) {
	res, err := r.db.Exec(
		`INSERT INTO mail_addresses (owner_type, owner_id, user_id, local_part, display_name, status)
		 VALUES ('shared', ?, ?, ?, ?, 'approved')`,
		adminUserID, adminUserID, localPart, displayName,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// GrantSharedAccess — 공용함 접근 권한 부여 (upsert). 재부여 시 revoked=0 으로 복구.
func (r *MailRepo) GrantSharedAccess(addressID, userID int) error {
	_, err := r.db.Exec(
		`INSERT INTO mail_address_grants (address_id, user_id, revoked) VALUES (?, ?, 0)
		 ON CONFLICT(address_id, user_id) DO UPDATE SET revoked = 0`,
		addressID, userID,
	)
	return err
}

// RevokeSharedAccess — 공용함 접근 권한 회수 (revoked=1, 삭제 안 함).
func (r *MailRepo) RevokeSharedAccess(addressID, userID int) error {
	_, err := r.db.Exec(
		`UPDATE mail_address_grants SET revoked = 1 WHERE address_id = ? AND user_id = ?`,
		addressID, userID,
	)
	return err
}

// HasActiveGrant — revoked=0 인 grant 존재 여부.
func (r *MailRepo) HasActiveGrant(addressID, userID int) (bool, error) {
	var one int
	err := r.db.QueryRow(
		`SELECT 1 FROM mail_address_grants WHERE address_id = ? AND user_id = ? AND revoked = 0`,
		addressID, userID,
	).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ListActiveGrantUserIDs — 공용함 활성 권한 유저 id 목록 (수신 알림 fan-out).
func (r *MailRepo) ListActiveGrantUserIDs(addressID int) ([]int, error) {
	rows, err := r.db.Query(
		`SELECT user_id FROM mail_address_grants WHERE address_id = ? AND revoked = 0 ORDER BY user_id`,
		addressID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var uid int
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		ids = append(ids, uid)
	}
	return ids, rows.Err()
}

// ListSharedAddresses — 관리자용 공용함 목록. 각 주소의 grant(회수 포함) 를 함께 담는다.
func (r *MailRepo) ListSharedAddresses() ([]*mail.SharedAddressItem, error) {
	rows, err := r.db.Query(
		`SELECT id, local_part, display_name FROM mail_addresses
		 WHERE owner_type = 'shared' ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*mail.SharedAddressItem
	for rows.Next() {
		it := &mail.SharedAddressItem{Grants: []*mail.Grant{}}
		if err := rows.Scan(&it.AddressID, &it.LocalPart, &it.DisplayName); err != nil {
			return nil, err
		}
		it.Email = mail.EmailFor(it.LocalPart)
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, it := range items {
		grants, err := r.listGrants(it.AddressID)
		if err != nil {
			return nil, err
		}
		it.Grants = grants
	}
	return items, nil
}

// listGrants — 특정 공용함의 grant 목록 (회수분 포함, user_name 조인).
func (r *MailRepo) listGrants(addressID int) ([]*mail.Grant, error) {
	rows, err := r.db.Query(
		`SELECT g.user_id, COALESCE(u.name, ''), g.revoked
		 FROM mail_address_grants g LEFT JOIN users u ON u.id = g.user_id
		 WHERE g.address_id = ? ORDER BY g.user_id`,
		addressID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	grants := []*mail.Grant{}
	for rows.Next() {
		g := &mail.Grant{}
		var revoked int
		if err := rows.Scan(&g.UserID, &g.UserName, &revoked); err != nil {
			return nil, err
		}
		g.Revoked = revoked == 1
		grants = append(grants, g)
	}
	return grants, rows.Err()
}

// UpdateAddressLocalPart — 재요청 시 local_part 변경 + status 를 pending 으로 리셋.
func (r *MailRepo) UpdateAddressLocalPart(id int, localPart string) error {
	_, err := r.db.Exec(
		`UPDATE mail_addresses SET local_part = ?, status = 'pending' WHERE id = ?`,
		localPart, id,
	)
	return err
}

func (r *MailRepo) UpdateAddressStatus(id int, status string) error {
	_, err := r.db.Exec(`UPDATE mail_addresses SET status = ? WHERE id = ?`, status, id)
	return err
}

// ListMailboxesForUser — 유저의 개인 주소 + 소유(companies.owner_id) 회사 주소 + 활성 권한 있는 공용 주소.
func (r *MailRepo) ListMailboxesForUser(userID int) ([]*mail.MailboxItem, error) {
	rows, err := r.db.Query(
		`SELECT a.id, a.owner_type, a.owner_id, a.local_part, a.status,
			CASE
				WHEN a.owner_type = 'company' THEN COALESCE(c.name, '')
				WHEN a.owner_type = 'shared' THEN a.display_name
				ELSE COALESCE(u.name, '')
			END AS name
		 FROM mail_addresses a
		 LEFT JOIN users u ON a.owner_type = 'user' AND u.id = a.owner_id
		 LEFT JOIN companies c ON a.owner_type = 'company' AND c.id = a.owner_id
		 WHERE (a.owner_type = 'user' AND a.owner_id = ?)
		    OR (a.owner_type = 'company' AND a.owner_id IN (SELECT id FROM companies WHERE owner_id = ?))
		    OR (a.owner_type = 'shared' AND EXISTS(
		         SELECT 1 FROM mail_address_grants g
		         WHERE g.address_id = a.id AND g.user_id = ? AND g.revoked = 0))
		 ORDER BY a.owner_type, a.id`,
		userID, userID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*mail.MailboxItem
	for rows.Next() {
		var (
			id                   int
			ownerType, localPart string
			ownerID              int
			status, name         string
		)
		if err := rows.Scan(&id, &ownerType, &ownerID, &localPart, &status, &name); err != nil {
			return nil, err
		}
		it := &mail.MailboxItem{
			AddressID: id,
			Kind:      ownerType,
			Name:      name,
			LocalPart: localPart,
			Email:     mail.EmailFor(localPart),
			Status:    status,
		}
		if ownerType == mail.OwnerCompany {
			cid := ownerID
			it.CompanyID = &cid
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// ListAddressesAdmin — 관리자 승인 목록. status="" 면 전체, 아니면 해당 상태만.
func (r *MailRepo) ListAddressesAdmin(status string) ([]*mail.AddressAdminItem, error) {
	query := `SELECT a.id, a.owner_type, a.owner_id, a.user_id,
			COALESCE(ru.name, '') AS user_name,
			COALESCE(ru.email, '') AS user_email,
			CASE WHEN a.owner_type = 'company' THEN COALESCE(c.name, '') ELSE COALESCE(ru.name, '') END AS owner_name,
			a.local_part, a.status, a.created_at
		 FROM mail_addresses a
		 LEFT JOIN users ru ON ru.id = a.user_id
		 LEFT JOIN companies c ON a.owner_type = 'company' AND c.id = a.owner_id`
	var (
		rows *sql.Rows
		err  error
	)
	if status == "" {
		query += ` ORDER BY a.created_at DESC, a.id DESC`
		rows, err = r.db.Query(query)
	} else {
		query += ` WHERE a.status = ? ORDER BY a.created_at DESC, a.id DESC`
		rows, err = r.db.Query(query, status)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*mail.AddressAdminItem
	for rows.Next() {
		it := &mail.AddressAdminItem{}
		if err := rows.Scan(&it.ID, &it.OwnerType, &it.OwnerID, &it.UserID,
			&it.UserName, &it.UserEmail, &it.OwnerName, &it.LocalPart, &it.Status, &it.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// GetCompanyOwnerName — companies 테이블에서 소유주 유저 id 와 회사명 조회. 없으면 (0,"",nil).
func (r *MailRepo) GetCompanyOwnerName(companyID int) (int, string, error) {
	var ownerID int
	var name string
	err := r.db.QueryRow(`SELECT owner_id, name FROM companies WHERE id = ?`, companyID).
		Scan(&ownerID, &name)
	if err == sql.ErrNoRows {
		return 0, "", nil
	}
	return ownerID, name, err
}

func (r *MailRepo) CreateEmail(e *mail.Email) (int, error) {
	readVal := 0
	if e.Read {
		readVal = 1
	}
	res, err := r.db.Exec(
		`INSERT INTO emails (address_id, owner_user_id, direction, from_addr, header_from, header_from_name,
			to_addr, subject, body_text, body_html, message_id, in_reply_to, refs, read)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.AddressID, e.OwnerUserID, e.Direction, e.FromAddr, e.HeaderFrom, e.HeaderFromName,
		e.ToAddr, e.Subject, e.BodyText, e.BodyHTML, e.MessageID, e.InReplyTo, e.Refs, readVal,
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
		`SELECT id, address_id, owner_user_id, direction, from_addr, header_from, header_from_name,
			to_addr, subject, body_text, body_html, message_id, in_reply_to, refs, read, created_at
		 FROM emails WHERE id = ?`, id,
	).Scan(&e.ID, &e.AddressID, &e.OwnerUserID, &e.Direction, &e.FromAddr, &e.HeaderFrom, &e.HeaderFromName,
		&e.ToAddr, &e.Subject, &e.BodyText, &e.BodyHTML, &e.MessageID, &e.InReplyTo, &e.Refs, &readVal, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	e.Read = readVal == 1
	return e, nil
}

// ListEmails — 특정 주소(메일함)의 방향별 목록.
func (r *MailRepo) ListEmails(addressID int, direction string, limit, offset int) ([]*mail.EmailListItem, int, error) {
	var total int
	if err := r.db.QueryRow(
		`SELECT COUNT(*) FROM emails WHERE address_id = ? AND direction = ?`,
		addressID, direction,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(
		`SELECT e.id, e.direction, e.from_addr, e.header_from, e.header_from_name, e.to_addr, e.subject, e.body_text, e.body_html, e.read, e.created_at,
			EXISTS(SELECT 1 FROM mail_attachments a WHERE a.email_id = e.id) AS has_attachments
		 FROM emails e
		 WHERE e.address_id = ? AND e.direction = ?
		 ORDER BY e.created_at DESC, e.id DESC
		 LIMIT ? OFFSET ?`,
		addressID, direction, limit, offset,
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
		`SELECT e.id, e.direction, e.from_addr, e.header_from, e.header_from_name, e.to_addr, e.subject, e.body_text, e.body_html, e.read, e.created_at,
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
		var bodyText, bodyHTML string
		var readVal int
		var hasAttach int
		if withOwner {
			if err := rows.Scan(&it.ID, &it.Direction, &it.FromAddr, &it.HeaderFrom, &it.HeaderFromName, &it.ToAddr, &it.Subject,
				&bodyText, &bodyHTML, &readVal, &it.CreatedAt, &hasAttach, &it.OwnerUserID, &it.OwnerName); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(&it.ID, &it.Direction, &it.FromAddr, &it.HeaderFrom, &it.HeaderFromName, &it.ToAddr, &it.Subject,
				&bodyText, &bodyHTML, &readVal, &it.CreatedAt, &hasAttach); err != nil {
				return nil, err
			}
		}
		// HTML 전용 메일(text 파트 없음)은 태그 제거 텍스트로 스니펫 폴백 (#172).
		src := bodyText
		if strings.TrimSpace(src) == "" && bodyHTML != "" {
			src = mail.StripHTMLTags(bodyHTML)
		}
		it.Snippet = mail.Snippet(src, snippetLen)
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
		`SELECT a.id, a.email_id, a.filename, a.mime, a.size, a.stored_path, e.owner_user_id, e.address_id
		 FROM mail_attachments a JOIN emails e ON e.id = a.email_id
		 WHERE a.id = ?`, id,
	).Scan(&a.ID, &a.EmailID, &a.Filename, &a.Mime, &a.Size, &a.StoredPath, &a.OwnerUserID, &a.AddressID)
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
