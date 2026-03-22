package middleware

import (
	"net/http"
	"strings"

	"github.com/earnlearning/backend/internal/application"
	"github.com/labstack/echo/v4"
)

// OAuthBearerAuth validates OAuth2 bearer tokens.
func OAuthBearerAuth(oauthUC *application.OAuthUseCase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			auth := c.Request().Header.Get("Authorization")
			if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"success": false, "data": nil,
					"error": map[string]string{"code": "UNAUTHORIZED", "message": "OAuth 토큰이 필요합니다"},
				})
			}

			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			token, err := oauthUC.ValidateAccessToken(tokenStr)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"success": false, "data": nil,
					"error": map[string]string{"code": "INVALID_TOKEN", "message": err.Error()},
				})
			}

			c.Set("oauth_user_id", token.UserID)
			c.Set("oauth_client_id", token.ClientID)
			c.Set("oauth_scopes", token.Scopes)
			return next(c)
		}
	}
}

// RequireScope checks that the OAuth token has the required scope.
func RequireScope(scope string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			scopes, _ := c.Get("oauth_scopes").([]string)
			for _, s := range scopes {
				if s == scope {
					return next(c)
				}
			}
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"success": false, "data": nil,
				"error": map[string]string{"code": "INSUFFICIENT_SCOPE", "message": "권한이 부족합니다: " + scope},
			})
		}
	}
}
