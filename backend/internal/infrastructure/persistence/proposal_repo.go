package persistence

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/earnlearning/backend/internal/domain/proposal"
)

type ProposalRepo struct {
	db *sql.DB
}

func NewProposalRepo(db *sql.DB) *ProposalRepo { return &ProposalRepo{db: db} }

func (r *ProposalRepo) Create(p *proposal.Proposal) (int, error) {
	att, _ := json.Marshal(p.Attachments)
	if att == nil {
		att = []byte("[]")
	}
	res, err := r.db.Exec(`
		INSERT INTO proposals (user_id, category, title, body, attachments, status, admin_note, ticket_link)
		VALUES (?, ?, ?, ?, ?, ?, '', '')
	`, p.UserID, string(p.Category), p.Title, p.Body, string(att), string(proposal.StatusOpen))
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (r *ProposalRepo) FindByID(id int) (*proposal.Proposal, error) {
	row := r.db.QueryRow(`
		SELECT p.id, p.user_id, p.category, p.title, p.body, p.attachments, p.status,
		       p.admin_note, p.ticket_link, p.created_at, p.updated_at,
		       u.name, COALESCE(u.student_id, ''), COALESCE(u.department, '')
		FROM proposals p
		JOIN users u ON p.user_id = u.id
		WHERE p.id = ?
	`, id)
	p := &proposal.Proposal{User: &proposal.UserRef{}}
	var attJSON, cat, status string
	if err := row.Scan(&p.ID, &p.UserID, &cat, &p.Title, &p.Body, &attJSON,
		&status, &p.AdminNote, &p.TicketLink, &p.CreatedAt, &p.UpdatedAt,
		&p.User.Name, &p.User.StudentID, &p.User.Department); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, proposal.ErrNotFound
		}
		return nil, err
	}
	p.User.ID = p.UserID
	p.Category = proposal.Category(cat)
	p.Status = proposal.Status(status)
	_ = json.Unmarshal([]byte(attJSON), &p.Attachments)
	return p, nil
}

func (r *ProposalRepo) List(f proposal.Filter) ([]*proposal.Proposal, error) {
	var clauses []string
	var args []any
	if f.UserID > 0 {
		clauses = append(clauses, "p.user_id = ?")
		args = append(args, f.UserID)
	}
	if f.Status != "" {
		clauses = append(clauses, "p.status = ?")
		args = append(args, f.Status)
	}
	if f.Category != "" {
		clauses = append(clauses, "p.category = ?")
		args = append(args, f.Category)
	}
	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	q := fmt.Sprintf(`
		SELECT p.id, p.user_id, p.category, p.title, p.body, p.attachments, p.status,
		       p.admin_note, p.ticket_link, p.created_at, p.updated_at,
		       u.name, COALESCE(u.student_id, ''), COALESCE(u.department, '')
		FROM proposals p
		JOIN users u ON p.user_id = u.id
		%s
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?
	`, where)
	args = append(args, limit, f.Offset)
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*proposal.Proposal
	for rows.Next() {
		p := &proposal.Proposal{User: &proposal.UserRef{}}
		var attJSON, cat, status string
		if err := rows.Scan(&p.ID, &p.UserID, &cat, &p.Title, &p.Body, &attJSON,
			&status, &p.AdminNote, &p.TicketLink, &p.CreatedAt, &p.UpdatedAt,
			&p.User.Name, &p.User.StudentID, &p.User.Department); err != nil {
			return nil, err
		}
		p.User.ID = p.UserID
		p.Category = proposal.Category(cat)
		p.Status = proposal.Status(status)
		_ = json.Unmarshal([]byte(attJSON), &p.Attachments)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ProposalRepo) Count(f proposal.Filter) (int, error) {
	var clauses []string
	var args []any
	if f.UserID > 0 {
		clauses = append(clauses, "user_id = ?")
		args = append(args, f.UserID)
	}
	if f.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, f.Status)
	}
	if f.Category != "" {
		clauses = append(clauses, "category = ?")
		args = append(args, f.Category)
	}
	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	var n int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM proposals "+where, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (r *ProposalRepo) UpdateStatus(id int, status proposal.Status, adminNote, ticketLink string) error {
	_, err := r.db.Exec(`
		UPDATE proposals SET status = ?, admin_note = ?, ticket_link = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, string(status), adminNote, ticketLink, id)
	return err
}
