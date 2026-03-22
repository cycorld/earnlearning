package persistence

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/earnlearning/backend/internal/domain/oauth"
)

type OAuthRepo struct {
	db *sql.DB
}

func NewOAuthRepo(db *sql.DB) *OAuthRepo {
	return &OAuthRepo{db: db}
}

// --- Client operations ---

func (r *OAuthRepo) CreateClient(client *oauth.Client) error {
	redirectURIs, _ := json.Marshal(client.RedirectURIs)
	scopes, _ := json.Marshal(client.Scopes)
	_, err := r.db.Exec(
		`INSERT INTO oauth_clients (id, secret_hash, user_id, name, description, redirect_uris, scopes, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		client.ID, client.SecretHash, client.UserID, client.Name, client.Description,
		string(redirectURIs), string(scopes), client.Status, client.CreatedAt, client.UpdatedAt,
	)
	return err
}

func (r *OAuthRepo) GetClient(id string) (*oauth.Client, error) {
	row := r.db.QueryRow(
		`SELECT id, secret_hash, user_id, name, description, redirect_uris, scopes, status, created_at, updated_at
		 FROM oauth_clients WHERE id = ?`, id,
	)
	return scanClient(row)
}

func (r *OAuthRepo) ListClientsByUser(userID int) ([]*oauth.Client, error) {
	rows, err := r.db.Query(
		`SELECT id, secret_hash, user_id, name, description, redirect_uris, scopes, status, created_at, updated_at
		 FROM oauth_clients WHERE user_id = ? ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []*oauth.Client
	for rows.Next() {
		c, err := scanClientRow(rows)
		if err != nil {
			return nil, err
		}
		clients = append(clients, c)
	}
	return clients, nil
}

func (r *OAuthRepo) DeleteClient(id string) error {
	_, err := r.db.Exec(`DELETE FROM oauth_clients WHERE id = ?`, id)
	return err
}

// --- Authorization code operations ---

func (r *OAuthRepo) CreateAuthorizationCode(code *oauth.AuthorizationCode) error {
	scopes, _ := json.Marshal(code.Scopes)
	_, err := r.db.Exec(
		`INSERT INTO oauth_authorization_codes (code, client_id, user_id, redirect_uri, scopes, code_challenge, code_challenge_method, expires_at, used, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?)`,
		code.Code, code.ClientID, code.UserID, code.RedirectURI,
		string(scopes), code.CodeChallenge, code.CodeChallengeMethod,
		code.ExpiresAt, code.CreatedAt,
	)
	return err
}

func (r *OAuthRepo) GetAuthorizationCode(code string) (*oauth.AuthorizationCode, error) {
	row := r.db.QueryRow(
		`SELECT code, client_id, user_id, redirect_uri, scopes, code_challenge, code_challenge_method, expires_at, used, created_at
		 FROM oauth_authorization_codes WHERE code = ?`, code,
	)
	var ac oauth.AuthorizationCode
	var scopesJSON string
	var used int
	err := row.Scan(&ac.Code, &ac.ClientID, &ac.UserID, &ac.RedirectURI,
		&scopesJSON, &ac.CodeChallenge, &ac.CodeChallengeMethod,
		&ac.ExpiresAt, &used, &ac.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, oauth.ErrCodeNotFound
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(scopesJSON), &ac.Scopes)
	ac.Used = used != 0
	return &ac, nil
}

func (r *OAuthRepo) MarkCodeUsed(code string) error {
	_, err := r.db.Exec(`UPDATE oauth_authorization_codes SET used = 1 WHERE code = ?`, code)
	return err
}

// --- Token operations ---

func (r *OAuthRepo) CreateToken(token *oauth.Token) error {
	scopes, _ := json.Marshal(token.Scopes)
	_, err := r.db.Exec(
		`INSERT INTO oauth_tokens (client_id, user_id, access_token, refresh_token, scopes, expires_at, revoked, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, 0, ?)`,
		token.ClientID, token.UserID, token.AccessToken, token.RefreshToken,
		string(scopes), token.ExpiresAt, token.CreatedAt,
	)
	return err
}

func (r *OAuthRepo) GetTokenByAccess(accessToken string) (*oauth.Token, error) {
	row := r.db.QueryRow(
		`SELECT id, client_id, user_id, access_token, refresh_token, scopes, expires_at, revoked, created_at
		 FROM oauth_tokens WHERE access_token = ?`, accessToken,
	)
	return scanToken(row)
}

func (r *OAuthRepo) GetTokenByRefresh(refreshToken string) (*oauth.Token, error) {
	row := r.db.QueryRow(
		`SELECT id, client_id, user_id, access_token, refresh_token, scopes, expires_at, revoked, created_at
		 FROM oauth_tokens WHERE refresh_token = ?`, refreshToken,
	)
	return scanToken(row)
}

func (r *OAuthRepo) RevokeToken(accessToken string) error {
	_, err := r.db.Exec(`UPDATE oauth_tokens SET revoked = 1 WHERE access_token = ?`, accessToken)
	return err
}

func (r *OAuthRepo) RevokeTokensByClient(clientID string) error {
	_, err := r.db.Exec(`UPDATE oauth_tokens SET revoked = 1 WHERE client_id = ?`, clientID)
	return err
}

// --- Scan helpers ---

func scanClient(row *sql.Row) (*oauth.Client, error) {
	var c oauth.Client
	var redirectURIsJSON, scopesJSON string
	err := row.Scan(&c.ID, &c.SecretHash, &c.UserID, &c.Name, &c.Description,
		&redirectURIsJSON, &scopesJSON, &c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, oauth.ErrClientNotFound
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(redirectURIsJSON), &c.RedirectURIs)
	json.Unmarshal([]byte(scopesJSON), &c.Scopes)
	return &c, nil
}

func scanClientRow(rows *sql.Rows) (*oauth.Client, error) {
	var c oauth.Client
	var redirectURIsJSON, scopesJSON string
	err := rows.Scan(&c.ID, &c.SecretHash, &c.UserID, &c.Name, &c.Description,
		&redirectURIsJSON, &scopesJSON, &c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(redirectURIsJSON), &c.RedirectURIs)
	json.Unmarshal([]byte(scopesJSON), &c.Scopes)
	return &c, nil
}

func scanToken(row *sql.Row) (*oauth.Token, error) {
	var t oauth.Token
	var scopesJSON string
	var revoked int
	err := row.Scan(&t.ID, &t.ClientID, &t.UserID, &t.AccessToken, &t.RefreshToken,
		&scopesJSON, &t.ExpiresAt, &revoked, &t.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, oauth.ErrTokenNotFound
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(scopesJSON), &t.Scopes)
	t.Revoked = revoked != 0
	return &t, nil
}

// Ensure OAuthRepo implements the interface.
var _ oauth.Repository = (*OAuthRepo)(nil)

// ExpiresAt helpers for token lifetime.
func AccessTokenExpiry() time.Time  { return time.Now().Add(1 * time.Hour) }
func RefreshTokenExpiry() time.Time { return time.Now().Add(30 * 24 * time.Hour) }
func AuthCodeExpiry() time.Time     { return time.Now().Add(10 * time.Minute) }
