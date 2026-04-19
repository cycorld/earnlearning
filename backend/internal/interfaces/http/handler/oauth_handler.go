package handler

import (
	"net/http"
	"strings"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
	"github.com/labstack/echo/v4"
)

type OAuthHandler struct {
	oauthUC *application.OAuthUseCase
}

func NewOAuthHandler(uc *application.OAuthUseCase) *OAuthHandler {
	return &OAuthHandler{oauthUC: uc}
}

// RegisterClient godoc
//
//	@Summary		OAuth 앱 등록
//	@Description	외부 앱 등록 (client_id, client_secret 발급)
//	@Tags			OAuth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		application.RegisterClientInput	true	"앱 정보"
//	@Success		201		{object}	APIResponse
//	@Failure		400		{object}	APIResponse
//	@Router			/oauth/clients [post]
func (h *OAuthHandler) RegisterClient(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var input application.RegisterClientInput
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}
	result, err := h.oauthUC.RegisterClient(userID, input)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "REGISTER_FAILED", err.Error())
	}
	return successResponse(c, http.StatusCreated, result)
}

// ListClients godoc
//
//	@Summary		내 앱 목록
//	@Description	내가 등록한 OAuth 앱 목록 조회
//	@Tags			OAuth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Router			/oauth/clients [get]
func (h *OAuthHandler) ListClients(c echo.Context) error {
	userID := middleware.GetUserID(c)
	clients, err := h.oauthUC.ListMyClients(userID)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}
	return successResponse(c, http.StatusOK, clients)
}

// DeleteClient godoc
//
//	@Summary		앱 삭제
//	@Description	등록한 OAuth 앱 삭제 + 토큰 폐기
//	@Tags			OAuth
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Client ID"
//	@Success		200	{object}	APIResponse
//	@Failure		404	{object}	APIResponse
//	@Router			/oauth/clients/{id} [delete]
func (h *OAuthHandler) DeleteClient(c echo.Context) error {
	userID := middleware.GetUserID(c)
	clientID := c.Param("id")
	if err := h.oauthUC.DeleteClient(userID, clientID); err != nil {
		return errorResponse(c, http.StatusBadRequest, "DELETE_FAILED", err.Error())
	}
	return successResponse(c, http.StatusOK, map[string]string{"message": "앱이 삭제되었습니다"})
}

// AuthorizePage godoc
//
//	@Summary		인가 정보 조회
//	@Description	인가 동의 화면에 표시할 앱 정보 조회
//	@Tags			OAuth
//	@Produce		json
//	@Security		BearerAuth
//	@Param			client_id		query		string	true	"Client ID"
//	@Param			redirect_uri	query		string	true	"Redirect URI"
//	@Param			scope			query		string	true	"요청 스코프 (공백 구분)"
//	@Success		200				{object}	APIResponse
//	@Failure		400				{object}	APIResponse
//	@Router			/oauth/authorize [get]
func (h *OAuthHandler) AuthorizePage(c echo.Context) error {
	clientID := c.QueryParam("client_id")
	redirectURI := c.QueryParam("redirect_uri")
	scopeStr := c.QueryParam("scope")
	scopes := splitScopes(scopeStr)

	info, err := h.oauthUC.GetAuthorizeInfo(clientID, redirectURI, scopes)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "AUTH_INFO_FAILED", err.Error())
	}
	return successResponse(c, http.StatusOK, info)
}

// Authorize godoc
//
//	@Summary		인가 승인
//	@Description	사용자가 외부 앱에 접근 권한 승인 → 인가 코드 발급
//	@Tags			OAuth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		application.AuthorizeInput	true	"인가 정보"
//	@Success		200		{object}	APIResponse
//	@Router			/oauth/authorize [post]
func (h *OAuthHandler) Authorize(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var input application.AuthorizeInput
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}
	result, err := h.oauthUC.Authorize(userID, input)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "AUTHORIZE_FAILED", err.Error())
	}
	return successResponse(c, http.StatusOK, result)
}

// Token godoc
//
//	@Summary		토큰 교환
//	@Description	인가 코드 → 액세스 토큰 교환, 또는 리프레시 토큰으로 갱신.
//	@Description
//	@Description	**PKCE (RFC 7636) 퍼블릭 클라이언트** 는 `client_secret` 없이 `code_verifier` 만으로 교환 가능.
//	@Description	**refresh_token** grant 도 PKCE 로 발급된 토큰이면 `client_secret` 옵셔널.
//	@Description	응답 (RFC 6749 §5.1): `access_token`, `refresh_token`, `token_type` (Bearer), `expires_in` (초), `scopes`.
//	@Tags			OAuth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		OAuthTokenRequest	true	"토큰 교환/갱신 요청"
//	@Success		200		{object}	OAuthTokenResponse
//	@Failure		400		{object}	APIResponse
//	@Router			/oauth/token [post]
func (h *OAuthHandler) Token(c echo.Context) error {
	var body struct {
		GrantType    string `json:"grant_type"`
		Code         string `json:"code"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RedirectURI  string `json:"redirect_uri"`
		CodeVerifier string `json:"code_verifier"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.Bind(&body); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}

	switch body.GrantType {
	case "authorization_code":
		result, err := h.oauthUC.ExchangeCode(application.ExchangeCodeInput{
			Code:         body.Code,
			ClientID:     body.ClientID,
			ClientSecret: body.ClientSecret,
			RedirectURI:  body.RedirectURI,
			CodeVerifier: body.CodeVerifier,
		})
		if err != nil {
			return errorResponse(c, http.StatusBadRequest, "TOKEN_EXCHANGE_FAILED", err.Error())
		}
		return successResponse(c, http.StatusOK, result)

	case "refresh_token":
		result, err := h.oauthUC.RefreshAccessToken(application.RefreshTokenInput{
			RefreshToken: body.RefreshToken,
			ClientID:     body.ClientID,
			ClientSecret: body.ClientSecret,
		})
		if err != nil {
			return errorResponse(c, http.StatusBadRequest, "REFRESH_FAILED", err.Error())
		}
		return successResponse(c, http.StatusOK, result)

	default:
		return errorResponse(c, http.StatusBadRequest, "UNSUPPORTED_GRANT", "지원하지 않는 grant_type입니다")
	}
}

// Revoke godoc
//
//	@Summary		토큰 폐기
//	@Description	액세스 토큰 또는 리프레시 토큰 폐기
//	@Tags			OAuth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		RevokeTokenInput	true	"폐기할 토큰"
//	@Success		200		{object}	APIResponse
//	@Router			/oauth/revoke [post]
func (h *OAuthHandler) Revoke(c echo.Context) error {
	var input application.RevokeTokenInput
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}
	if err := h.oauthUC.RevokeToken(input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "REVOKE_FAILED", err.Error())
	}
	return successResponse(c, http.StatusOK, map[string]string{"message": "토큰이 폐기되었습니다"})
}

// UserInfo godoc
//
//	@Summary		사용자 정보 (OAuth)
//	@Description	OAuth 액세스 토큰으로 사용자 정보 조회. 응답: id, email, name, department, bio, avatar_url
//	@Tags			OAuth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	OAuthUserInfoResponse
//	@Router			/oauth/userinfo [get]
func (h *OAuthHandler) UserInfo(c echo.Context) error {
	userID, ok := c.Get("oauth_user_id").(int)
	if !ok || userID == 0 {
		return errorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "인증이 필요합니다")
	}
	info, err := h.oauthUC.GetUserInfo(userID)
	if err != nil {
		return errorResponse(c, http.StatusNotFound, "NOT_FOUND", "사용자를 찾을 수 없습니다")
	}
	return successResponse(c, http.StatusOK, info)
}

// RevokeTokenInput swagger model for revoke endpoint.
type RevokeTokenInput struct {
	Token string `json:"token" example:"access_token_or_refresh_token"`
}

func splitScopes(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Fields(s)
}
