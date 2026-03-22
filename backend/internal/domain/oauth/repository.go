package oauth

// Repository defines the persistence interface for OAuth entities.
type Repository interface {
	// Client operations
	CreateClient(client *Client) error
	GetClient(id string) (*Client, error)
	ListClientsByUser(userID int) ([]*Client, error)
	DeleteClient(id string) error

	// Authorization code operations
	CreateAuthorizationCode(code *AuthorizationCode) error
	GetAuthorizationCode(code string) (*AuthorizationCode, error)
	MarkCodeUsed(code string) error

	// Token operations
	CreateToken(token *Token) error
	GetTokenByAccess(accessToken string) (*Token, error)
	GetTokenByRefresh(refreshToken string) (*Token, error)
	RevokeToken(accessToken string) error
	RevokeTokensByClient(clientID string) error
}
