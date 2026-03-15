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
