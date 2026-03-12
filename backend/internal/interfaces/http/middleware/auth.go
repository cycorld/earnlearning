package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type JWTClaims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Status string `json:"status"`
	jwt.RegisteredClaims
}

func JWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			auth := c.Request().Header.Get("Authorization")
			if auth == "" {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"success": false, "data": nil,
					"error": map[string]string{"code": "UNAUTHORIZED", "message": "인증이 필요합니다"},
				})
			}

			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			claims := &JWTClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"success": false, "data": nil,
					"error": map[string]string{"code": "UNAUTHORIZED", "message": "유효하지 않은 토큰입니다"},
				})
			}

			c.Set("user_id", claims.UserID)
			c.Set("email", claims.Email)
			c.Set("role", claims.Role)
			c.Set("status", claims.Status)
			return next(c)
		}
	}
}

func ApprovedOnly() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			status, _ := c.Get("status").(string)
			if status != "approved" {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"success": false, "data": nil,
					"error": map[string]string{"code": "NOT_APPROVED", "message": "승인 대기 중입니다"},
				})
			}
			return next(c)
		}
	}
}

func AdminOnly() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, _ := c.Get("role").(string)
			if role != "admin" {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"success": false, "data": nil,
					"error": map[string]string{"code": "FORBIDDEN", "message": "관리자 권한이 필요합니다"},
				})
			}
			return next(c)
		}
	}
}

func GetUserID(c echo.Context) int {
	id, _ := c.Get("user_id").(int)
	return id
}

func GetUserRole(c echo.Context) string {
	role, _ := c.Get("role").(string)
	return role
}
