package handler

import (
	"net/http"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
	"github.com/labstack/echo/v4"
)

type AdminHandler struct {
	authUC *application.AuthUseCase
}

func NewAdminHandler(uc *application.AuthUseCase) *AdminHandler {
	return &AdminHandler{authUC: uc}
}

// GetPendingUsers godoc
//
//	@Summary		승인 대기 사용자 목록
//	@Description	관리자용: 승인 대기 중인 사용자 목록 조회
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Router			/admin/users/pending [get]
func (h *AdminHandler) GetPendingUsers(c echo.Context) error {
	users, err := h.authUC.AdminGetPending()
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}

	viewerRole := middleware.GetUserRole(c)
	var result []userResponse
	for _, u := range users {
		result = append(result, userToResponse(u, viewerRole))
	}

	if result == nil {
		result = []userResponse{}
	}
	return successResponse(c, http.StatusOK, result)
}

// ApproveUser godoc
//
//	@Summary		사용자 승인
//	@Description	관리자용: 사용자 가입 승인
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"사용자 ID"
//	@Success		200	{object}	APIResponse
//	@Failure		404	{object}	APIResponse
//	@Router			/admin/users/{id}/approve [put]
func (h *AdminHandler) ApproveUser(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}

	if err := h.authUC.AdminApprove(id); err != nil {
		return errorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	}

	return successResponse(c, http.StatusOK, map[string]string{"message": "승인되었습니다"})
}

// RejectUser godoc
//
//	@Summary		사용자 거절
//	@Description	관리자용: 사용자 가입 거절
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"사용자 ID"
//	@Success		200	{object}	APIResponse
//	@Failure		404	{object}	APIResponse
//	@Router			/admin/users/{id}/reject [put]
func (h *AdminHandler) RejectUser(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}

	if err := h.authUC.AdminReject(id); err != nil {
		return errorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	}

	return successResponse(c, http.StatusOK, map[string]string{"message": "거절되었습니다"})
}

// ListUsers godoc
//
//	@Summary		전체 사용자 목록
//	@Description	관리자용: 전체 사용자 목록 (페이지네이션)
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page	query		int	false	"페이지 번호"	default(1)
//	@Param			limit	query		int	false	"페이지 크기"	default(20)
//	@Success		200		{object}	APIResponse
//	@Router			/admin/users [get]
func (h *AdminHandler) ListUsers(c echo.Context) error {
	page := intQuery(c, "page", 1)
	limit := intQuery(c, "limit", 20)

	result, err := h.authUC.AdminListUsers(page, limit)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}

	viewerRole := middleware.GetUserRole(c)
	var userList []userResponse
	for _, u := range result.Users {
		userList = append(userList, userToResponse(u, viewerRole))
	}

	if userList == nil {
		userList = []userResponse{}
	}

	return successResponse(c, http.StatusOK, map[string]interface{}{
		"users":       userList,
		"total":       result.Total,
		"total_pages": result.TotalPages,
	})
}

// ImpersonateUser godoc
//
//	@Summary		사용자 대리 로그인
//	@Description	관리자용: 특정 사용자로 대리 로그인 (디버깅용)
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"사용자 ID"
//	@Success		200	{object}	APIResponse
//	@Failure		404	{object}	APIResponse
//	@Router			/admin/users/{id}/impersonate [post]
func (h *AdminHandler) ImpersonateUser(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}

	resp, err := h.authUC.ImpersonateUser(id)
	if err != nil {
		return errorResponse(c, http.StatusNotFound, "NOT_FOUND", "사용자를 찾을 수 없습니다")
	}

	return successResponse(c, http.StatusOK, resp)
}
