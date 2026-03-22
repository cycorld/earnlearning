package oauth

import "time"

// Client represents an OAuth2 client application.
type Client struct {
	ID           string    `json:"id"`
	SecretHash   string    `json:"-"`
	UserID       int       `json:"user_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	RedirectURIs []string  `json:"redirect_uris"`
	Scopes       []string  `json:"scopes"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AuthorizationCode represents a short-lived authorization code.
type AuthorizationCode struct {
	Code                string    `json:"code"`
	ClientID            string    `json:"client_id"`
	UserID              int       `json:"user_id"`
	RedirectURI         string    `json:"redirect_uri"`
	Scopes              []string  `json:"scopes"`
	CodeChallenge       string    `json:"-"`
	CodeChallengeMethod string    `json:"-"`
	ExpiresAt           time.Time `json:"expires_at"`
	Used                bool      `json:"used"`
	CreatedAt           time.Time `json:"created_at"`
}

// Token represents an OAuth2 access/refresh token pair.
type Token struct {
	ID           int       `json:"id"`
	ClientID     string    `json:"client_id"`
	UserID       int       `json:"user_id"`
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Scopes       []string  `json:"scopes"`
	ExpiresAt    time.Time `json:"expires_at"`
	Revoked      bool      `json:"revoked"`
	CreatedAt    time.Time `json:"created_at"`
}

// ValidScopes is the list of all valid OAuth scopes.
var ValidScopes = map[string]string{
	"read:profile":       "프로필 조회",
	"write:profile":      "프로필 수정",
	"read:wallet":        "지갑 잔액/거래 조회",
	"write:wallet":       "송금",
	"read:posts":         "게시물/댓글 조회",
	"write:posts":        "게시물/댓글 작성, 좋아요",
	"read:company":       "회사 정보 조회",
	"write:company":      "회사 정보 수정",
	"read:market":        "프리랜서/거래소/투자 조회",
	"write:market":       "프리랜서 등록, 주문, 투자",
	"read:notifications": "알림 조회",
}

// IsValidScope checks if a scope is valid.
func IsValidScope(scope string) bool {
	_, ok := ValidScopes[scope]
	return ok
}
