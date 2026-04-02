package application

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/earnlearning/backend/internal/domain/oauth"
	"github.com/earnlearning/backend/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

type OAuthUseCase struct {
	oauthRepo oauth.Repository
	userRepo  user.Repository
}

func NewOAuthUseCase(oauthRepo oauth.Repository, userRepo user.Repository) *OAuthUseCase {
	return &OAuthUseCase{oauthRepo: oauthRepo, userRepo: userRepo}
}

// --- Input types ---

type RegisterClientInput struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
}

type RegisterClientOutput struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Name         string `json:"name"`
}

type AuthorizeInput struct {
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	Scopes              []string `json:"scopes"`
	State               string `json:"state"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
}

type AuthorizeInfoOutput struct {
	ClientName  string   `json:"client_name"`
	Scopes      []string `json:"scopes"`
	RedirectURI string   `json:"redirect_uri"`
}

type AuthorizeOutput struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirect_uri"`
	State       string `json:"state"`
}

type ExchangeCodeInput struct {
	Code         string `json:"code"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"`
	CodeVerifier string `json:"code_verifier"`
}

type TokenOutput struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scopes       []string `json:"scopes"`
}

type RefreshTokenInput struct {
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type RevokeTokenInput struct {
	Token string `json:"token"`
}

type OAuthUserInfo struct {
	ID         int    `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Department string `json:"department"`
	Bio        string `json:"bio"`
	AvatarURL  string `json:"avatar_url"`
}

// --- Client management ---

func (uc *OAuthUseCase) RegisterClient(userID int, input RegisterClientInput) (*RegisterClientOutput, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("앱 이름은 필수입니다")
	}
	if len(input.RedirectURIs) == 0 {
		return nil, fmt.Errorf("redirect_uri는 최소 1개 필요합니다")
	}
	for _, scope := range input.Scopes {
		if !oauth.IsValidScope(scope) {
			return nil, fmt.Errorf("유효하지 않은 스코프: %s", scope)
		}
	}

	clientID := generateRandomHex(16)
	clientSecret := generateRandomHex(32)
	hash, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	client := &oauth.Client{
		ID:           clientID,
		SecretHash:   string(hash),
		UserID:       userID,
		Name:         input.Name,
		Description:  input.Description,
		RedirectURIs: input.RedirectURIs,
		Scopes:       input.Scopes,
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := uc.oauthRepo.CreateClient(client); err != nil {
		return nil, err
	}

	return &RegisterClientOutput{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Name:         input.Name,
	}, nil
}

func (uc *OAuthUseCase) ListMyClients(userID int) ([]*oauth.Client, error) {
	clients, err := uc.oauthRepo.ListClientsByUser(userID)
	if err != nil {
		return nil, err
	}
	if clients == nil {
		clients = []*oauth.Client{}
	}
	return clients, nil
}

func (uc *OAuthUseCase) DeleteClient(userID int, clientID string) error {
	client, err := uc.oauthRepo.GetClient(clientID)
	if err != nil {
		return err
	}
	if client.UserID != userID {
		return oauth.ErrNotOwner
	}
	// Revoke all tokens
	uc.oauthRepo.RevokeTokensByClient(clientID)
	return uc.oauthRepo.DeleteClient(clientID)
}

// --- Authorization flow ---

func (uc *OAuthUseCase) GetAuthorizeInfo(clientID, redirectURI string, scopes []string) (*AuthorizeInfoOutput, error) {
	client, err := uc.oauthRepo.GetClient(clientID)
	if err != nil {
		return nil, err
	}
	if !containsURI(client.RedirectURIs, redirectURI) {
		return nil, oauth.ErrInvalidRedirectURI
	}
	for _, scope := range scopes {
		if !oauth.IsValidScope(scope) {
			return nil, oauth.ErrInvalidScope
		}
	}
	return &AuthorizeInfoOutput{
		ClientName:  client.Name,
		Scopes:      scopes,
		RedirectURI: redirectURI,
	}, nil
}

func (uc *OAuthUseCase) Authorize(userID int, input AuthorizeInput) (*AuthorizeOutput, error) {
	client, err := uc.oauthRepo.GetClient(input.ClientID)
	if err != nil {
		return nil, err
	}
	if !containsURI(client.RedirectURIs, input.RedirectURI) {
		return nil, oauth.ErrInvalidRedirectURI
	}
	for _, scope := range input.Scopes {
		if !oauth.IsValidScope(scope) {
			return nil, oauth.ErrInvalidScope
		}
	}

	code := generateRandomHex(32)
	authCode := &oauth.AuthorizationCode{
		Code:                code,
		ClientID:            input.ClientID,
		UserID:              userID,
		RedirectURI:         input.RedirectURI,
		Scopes:              input.Scopes,
		CodeChallenge:       input.CodeChallenge,
		CodeChallengeMethod: input.CodeChallengeMethod,
		ExpiresAt:           time.Now().Add(10 * time.Minute),
		CreatedAt:           time.Now(),
	}

	if err := uc.oauthRepo.CreateAuthorizationCode(authCode); err != nil {
		return nil, err
	}

	return &AuthorizeOutput{
		Code:        code,
		RedirectURI: input.RedirectURI,
		State:       input.State,
	}, nil
}

// --- Token exchange ---

func (uc *OAuthUseCase) ExchangeCode(input ExchangeCodeInput) (*TokenOutput, error) {
	authCode, err := uc.oauthRepo.GetAuthorizationCode(input.Code)
	if err != nil {
		return nil, err
	}
	if authCode.Used {
		return nil, oauth.ErrCodeUsed
	}
	if time.Now().After(authCode.ExpiresAt) {
		return nil, oauth.ErrCodeExpired
	}
	if authCode.ClientID != input.ClientID {
		return nil, oauth.ErrInvalidGrant
	}
	if authCode.RedirectURI != input.RedirectURI {
		return nil, oauth.ErrInvalidRedirectURI
	}

	// PKCE verification
	if authCode.CodeChallenge != "" {
		if input.CodeVerifier == "" {
			return nil, oauth.ErrInvalidPKCE
		}
		if authCode.CodeChallengeMethod == "S256" {
			hash := sha256.Sum256([]byte(input.CodeVerifier))
			challenge := base64.RawURLEncoding.EncodeToString(hash[:])
			if challenge != authCode.CodeChallenge {
				return nil, oauth.ErrInvalidPKCE
			}
		} else {
			// plain
			if input.CodeVerifier != authCode.CodeChallenge {
				return nil, oauth.ErrInvalidPKCE
			}
		}
	} else {
		// No PKCE — verify client_secret
		client, err := uc.oauthRepo.GetClient(input.ClientID)
		if err != nil {
			return nil, err
		}
		if bcrypt.CompareHashAndPassword([]byte(client.SecretHash), []byte(input.ClientSecret)) != nil {
			return nil, oauth.ErrInvalidSecret
		}
	}

	// Mark code as used
	uc.oauthRepo.MarkCodeUsed(input.Code)

	// Generate tokens
	return uc.issueTokens(authCode.ClientID, authCode.UserID, authCode.Scopes)
}

func (uc *OAuthUseCase) RefreshAccessToken(input RefreshTokenInput) (*TokenOutput, error) {
	token, err := uc.oauthRepo.GetTokenByRefresh(input.RefreshToken)
	if err != nil {
		return nil, err
	}
	if token.Revoked {
		return nil, oauth.ErrTokenRevoked
	}
	if token.ClientID != input.ClientID {
		return nil, oauth.ErrInvalidGrant
	}

	// Verify client secret
	client, err := uc.oauthRepo.GetClient(input.ClientID)
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(client.SecretHash), []byte(input.ClientSecret)) != nil {
		return nil, oauth.ErrInvalidSecret
	}

	// Revoke old token
	uc.oauthRepo.RevokeToken(token.AccessToken)

	// Issue new tokens
	return uc.issueTokens(token.ClientID, token.UserID, token.Scopes)
}

func (uc *OAuthUseCase) RevokeToken(input RevokeTokenInput) error {
	return uc.oauthRepo.RevokeToken(input.Token)
}

// --- Token validation (for middleware) ---

func (uc *OAuthUseCase) ValidateAccessToken(accessToken string) (*oauth.Token, error) {
	token, err := uc.oauthRepo.GetTokenByAccess(accessToken)
	if err != nil {
		return nil, err
	}
	if token.Revoked {
		return nil, oauth.ErrTokenRevoked
	}
	if time.Now().After(token.ExpiresAt) {
		return nil, oauth.ErrTokenExpired
	}
	return token, nil
}

// --- User info ---

func (uc *OAuthUseCase) GetUserRoleAndStatus(userID int) (string, string, error) {
	u, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return "", "", err
	}
	return string(u.Role), string(u.Status), nil
}

func (uc *OAuthUseCase) GetUserInfo(userID int) (*OAuthUserInfo, error) {
	u, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	return &OAuthUserInfo{
		ID:         u.ID,
		Email:      u.Email,
		Name:       u.Name,
		Department: u.Department,
		Bio:        u.Bio,
		AvatarURL:  u.AvatarURL,
	}, nil
}

// --- Helpers ---

func (uc *OAuthUseCase) issueTokens(clientID string, userID int, scopes []string) (*TokenOutput, error) {
	accessToken := generateRandomHex(32)
	refreshToken := generateRandomHex(32)
	expiresAt := time.Now().Add(1 * time.Hour)

	token := &oauth.Token{
		ClientID:     clientID,
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Scopes:       scopes,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
	}

	if err := uc.oauthRepo.CreateToken(token); err != nil {
		return nil, err
	}

	return &TokenOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		Scopes:       scopes,
	}, nil
}

func generateRandomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func containsURI(uris []string, uri string) bool {
	for _, u := range uris {
		if u == uri {
			return true
		}
	}
	return false
}
