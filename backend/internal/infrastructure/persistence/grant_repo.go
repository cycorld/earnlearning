package persistence

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/earnlearning/backend/internal/domain/grant"
)

type GrantRepo struct {
	db *sql.DB
}

func NewGrantRepo(db *sql.DB) *GrantRepo {
	return &GrantRepo{db: db}
}

func (r *GrantRepo) Create(g *grant.Grant) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO grants (admin_id, title, description, reward, max_applicants, status)
		VALUES (?, ?, ?, ?, ?, ?)`,
		g.AdminID, g.Title, g.Description, g.Reward, g.MaxApplicants, g.Status,
	)
	if err != nil {
		return 0, fmt.Errorf("create grant: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *GrantRepo) FindByID(id int) (*grant.Grant, error) {
	g := &grant.Grant{}
	var adminName string

	err := r.db.QueryRow(`
		SELECT g.id, g.admin_id, g.title, g.description, g.reward, g.max_applicants,
			   g.status, g.created_at, u.name AS admin_name
		FROM grants g
		JOIN users u ON u.id = g.admin_id
		WHERE g.id = ?`, id).Scan(
		&g.ID, &g.AdminID, &g.Title, &g.Description, &g.Reward, &g.MaxApplicants,
		&g.Status, &g.CreatedAt, &adminName,
	)
	if err == sql.ErrNoRows {
		return nil, grant.ErrGrantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find grant: %w", err)
	}
	g.Admin = &grant.UserRef{ID: g.AdminID, Name: adminName}
	g.AdminName = adminName
	return g, nil
}

func (r *GrantRepo) List(filter grant.GrantFilter, page, limit int) ([]*grant.Grant, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}

	if filter.Status != "" {
		where = append(where, "g.status = ?")
		args = append(args, filter.Status)
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM grants g WHERE "+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count grants: %w", err)
	}

	offset := (page - 1) * limit
	queryArgs := append(args, limit, offset)

	rows, err := r.db.Query(`
		SELECT g.id, g.admin_id, g.title, g.description, g.reward, g.max_applicants,
			   g.status, g.created_at, u.name AS admin_name,
			   (SELECT COUNT(*) FROM grant_applications WHERE grant_id = g.id) AS application_count,
			   (SELECT COUNT(*) FROM grant_applications WHERE grant_id = g.id AND status = 'approved') AS approved_count
		FROM grants g
		JOIN users u ON u.id = g.admin_id
		WHERE `+whereClause+`
		ORDER BY g.created_at DESC
		LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list grants: %w", err)
	}
	defer rows.Close()

	var grants []*grant.Grant
	for rows.Next() {
		g := &grant.Grant{}
		var adminName string
		var appCount, approvedCount int

		if err := rows.Scan(
			&g.ID, &g.AdminID, &g.Title, &g.Description, &g.Reward, &g.MaxApplicants,
			&g.Status, &g.CreatedAt, &adminName, &appCount, &approvedCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan grant: %w", err)
		}
		g.Admin = &grant.UserRef{ID: g.AdminID, Name: adminName}
		g.AdminName = adminName
		g.ApplicationCount = &appCount
		g.ApprovedCount = &approvedCount
		grants = append(grants, g)
	}
	return grants, total, nil
}

func (r *GrantRepo) UpdateStatus(id int, status grant.GrantStatus) error {
	_, err := r.db.Exec("UPDATE grants SET status = ? WHERE id = ?", status, id)
	return err
}

// --- Applications ---

func (r *GrantRepo) CreateApplication(app *grant.GrantApplication) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO grant_applications (grant_id, user_id, proposal, status)
		VALUES (?, ?, ?, ?)`,
		app.GrantID, app.UserID, app.Proposal, app.Status,
	)
	if err != nil {
		return 0, fmt.Errorf("create grant application: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *GrantRepo) FindApplicationByID(id int) (*grant.GrantApplication, error) {
	app := &grant.GrantApplication{}
	var userName string
	err := r.db.QueryRow(`
		SELECT a.id, a.grant_id, a.user_id, a.proposal, a.status, a.created_at,
			   u.name AS user_name
		FROM grant_applications a
		JOIN users u ON u.id = a.user_id
		WHERE a.id = ?`, id).Scan(
		&app.ID, &app.GrantID, &app.UserID, &app.Proposal, &app.Status, &app.CreatedAt,
		&userName,
	)
	if err == sql.ErrNoRows {
		return nil, grant.ErrApplicationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find grant application: %w", err)
	}
	app.UserName = userName
	app.User = &grant.UserRef{ID: app.UserID, Name: userName}
	return app, nil
}

func (r *GrantRepo) FindApplicationByGrantAndUser(grantID, userID int) (*grant.GrantApplication, error) {
	app := &grant.GrantApplication{}
	var userName string
	err := r.db.QueryRow(`
		SELECT a.id, a.grant_id, a.user_id, a.proposal, a.status, a.created_at,
			   u.name AS user_name
		FROM grant_applications a
		JOIN users u ON u.id = a.user_id
		WHERE a.grant_id = ? AND a.user_id = ?`, grantID, userID).Scan(
		&app.ID, &app.GrantID, &app.UserID, &app.Proposal, &app.Status, &app.CreatedAt,
		&userName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find grant application by grant and user: %w", err)
	}
	app.UserName = userName
	app.User = &grant.UserRef{ID: app.UserID, Name: userName}
	return app, nil
}

func (r *GrantRepo) ListApplicationsByGrant(grantID int) ([]*grant.GrantApplication, error) {
	rows, err := r.db.Query(`
		SELECT a.id, a.grant_id, a.user_id, a.proposal, a.status, a.created_at,
			   u.name AS user_name
		FROM grant_applications a
		JOIN users u ON u.id = a.user_id
		WHERE a.grant_id = ?
		ORDER BY a.created_at ASC`, grantID)
	if err != nil {
		return nil, fmt.Errorf("list grant applications: %w", err)
	}
	defer rows.Close()

	var apps []*grant.GrantApplication
	for rows.Next() {
		app := &grant.GrantApplication{}
		var userName string
		if err := rows.Scan(
			&app.ID, &app.GrantID, &app.UserID, &app.Proposal, &app.Status, &app.CreatedAt,
			&userName,
		); err != nil {
			return nil, fmt.Errorf("scan grant application: %w", err)
		}
		app.UserName = userName
		app.User = &grant.UserRef{ID: app.UserID, Name: userName}
		apps = append(apps, app)
	}
	return apps, nil
}

func (r *GrantRepo) UpdateApplicationStatus(id int, status grant.ApplicationStatus) error {
	_, err := r.db.Exec("UPDATE grant_applications SET status = ? WHERE id = ?", status, id)
	return err
}
