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

// Register godoc
//
//	@Summary		회원가입
//	@Description	이메일, 비밀번호, 이름, 학번으로 회원가입
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		RegisterRequest	true	"회원가입 정보"
//	@Success		201		{object}	APIResponse
//	@Failure		400		{object}	APIResponse
//	@Failure		409		{object}	APIResponse
//	@Router			/auth/register [post]
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

// Login godoc
//
//	@Summary		로그인
//	@Description	이메일/비밀번호로 로그인하여 JWT 토큰 발급
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		LoginRequest	true	"로그인 정보"
//	@Success		200		{object}	APIResponse
//	@Failure		401		{object}	APIResponse
//	@Router			/auth/login [post]
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

// Refresh godoc
//
//	@Summary		토큰 갱신
//	@Description	만료 임박 JWT 토큰을 새 토큰으로 갱신
//	@Tags			Auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Failure		401	{object}	APIResponse
//	@Router			/auth/refresh [post]
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

// GetMe godoc
//
//	@Summary		내 정보 조회
//	@Description	현재 로그인한 사용자 정보 조회
//	@Tags			Auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Failure		404	{object}	APIResponse
//	@Router			/auth/me [get]
func (h *AuthHandler) GetMe(c echo.Context) error {
	userID := middleware.GetUserID(c)
	u, err := h.authUC.GetMe(userID)
	if err != nil {
		return errorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	}

	viewerRole := middleware.GetUserRole(c)
	return successResponse(c, http.StatusOK, userToResponse(u, viewerRole))
}

// GetProfile godoc
//
//	@Summary		사용자 프로필 조회
//	@Description	특정 사용자의 공개 프로필 조회
//	@Tags			Auth
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"사용자 ID"
//	@Success		200	{object}	APIResponse
//	@Failure		404	{object}	APIResponse
//	@Router			/users/{id}/profile [get]
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

// UpdateAvatar godoc
//
//	@Summary		아바타 변경
//	@Description	본인의 프로필 아바타 URL 변경 (빈 문자열이면 기본 아바타로 초기화)
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Router			/auth/avatar [put]
func (h *AuthHandler) UpdateAvatar(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var input struct {
		AvatarURL string `json:"avatar_url"`
	}
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}
	if err := h.authUC.UpdateAvatar(userID, input.AvatarURL); err != nil {
		return errorResponse(c, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return successResponse(c, http.StatusOK, map[string]string{"avatar_url": input.AvatarURL})
}

// GetUserActivity godoc
//
//	@Summary		사용자 활동 조회
//	@Description	특정 사용자의 포스트, 프리랜서 잡, 정부과제 지원 내역
//	@Tags			Auth
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"사용자 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/users/{id}/activity [get]
func (h *AuthHandler) GetUserActivity(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}

	activity, err := h.authUC.GetUserActivity(id)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return successResponse(c, http.StatusOK, activity)
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
