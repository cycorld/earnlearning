package handler

import (
	"net/http"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/user"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	authUC *application.AuthUseCase
}

func NewAuthHandler(uc *application.AuthUseCase) *AuthHandler {
	return &AuthHandler{authUC: uc}
}

func (h *AuthHandler) Register(c echo.Context) error {
	var input application.RegisterInput
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}

	resp, err := h.authUC.Register(input)
	if err != nil {
		switch err {
		case user.ErrDuplicateEmail:
			return errorResponse(c, http.StatusConflict, "DUPLICATE_EMAIL", err.Error())
		case user.ErrWeakPassword:
			return errorResponse(c, http.StatusBadRequest, "WEAK_PASSWORD", err.Error())
		case user.ErrInvalidStudent:
			return errorResponse(c, http.StatusBadRequest, "INVALID_STUDENT_ID", err.Error())
		default:
			return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
		}
	}

	return successResponse(c, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(c echo.Context) error {
	var input application.LoginInput
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}

	resp, err := h.authUC.Login(input)
	if err != nil {
		switch err {
		case user.ErrInvalidCreds:
			return errorResponse(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", err.Error())
		case user.ErrRejected:
			return errorResponse(c, http.StatusForbidden, "REJECTED", err.Error())
		default:
			return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
		}
	}

	return successResponse(c, http.StatusOK, resp)
}

func (h *AuthHandler) Refresh(c echo.Context) error {
	// Extract token from Authorization header
	auth := c.Request().Header.Get("Authorization")
	if len(auth) < 8 || auth[:7] != "Bearer " {
		return errorResponse(c, http.StatusUnauthorized, "NO_TOKEN", "토큰이 없습니다")
	}
	tokenStr := auth[7:]

	resp, err := h.authUC.RefreshToken(tokenStr)
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, "REFRESH_FAILED", "토큰 갱신에 실패했습니다")
	}

	return successResponse(c, http.StatusOK, resp)
}

func (h *AuthHandler) GetMe(c echo.Context) error {
	userID := middleware.GetUserID(c)
	u, err := h.authUC.GetMe(userID)
	if err != nil {
		return errorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	}

	viewerRole := middleware.GetUserRole(c)
	return successResponse(c, http.StatusOK, userToResponse(u, viewerRole))
}

func (h *AuthHandler) GetProfile(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}

	u, err := h.authUC.GetProfile(id)
	if err != nil {
		return errorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	}

	viewerRole := middleware.GetUserRole(c)
	return successResponse(c, http.StatusOK, userToResponse(u, viewerRole))
}

type userResponse struct {
	ID         int       `json:"id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	Department string    `json:"department"`
	StudentID  string    `json:"student_id"`
	Role       string    `json:"role"`
	Status     string    `json:"status"`
	Bio        string    `json:"bio"`
	AvatarURL  string    `json:"avatar_url"`
	CreatedAt  string    `json:"created_at"`
}

func userToResponse(u *user.User, viewerRole string) userResponse {
	return userResponse{
		ID:         u.ID,
		Email:      u.Email,
		Name:       u.Name,
		Department: u.Department,
		StudentID:  u.StudentIDDisplay(viewerRole),
		Role:       string(u.Role),
		Status:     string(u.Status),
		Bio:        u.Bio,
		AvatarURL:  u.AvatarURL,
		CreatedAt:  u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
