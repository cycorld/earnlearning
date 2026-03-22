package handler

import (
	"net/http"
	"strconv"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
	"github.com/labstack/echo/v4"
)

type GrantHandler struct {
	uc *application.GrantUseCase
}

func NewGrantHandler(uc *application.GrantUseCase) *GrantHandler {
	return &GrantHandler{uc: uc}
}

// ListGrants godoc
//
//	@Summary		정부과제 목록
//	@Description	정부과제 목록 조회 (페이지네이션)
//	@Tags			Grant
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status	query		string	false	"상태 필터"
//	@Param			page	query		int		false	"페이지"	default(1)
//	@Param			limit	query		int		false	"크기"	default(20)
//	@Success		200		{object}	APIResponse
//	@Router			/grants [get]
func (h *GrantHandler) ListGrants(c echo.Context) error {
	status := c.QueryParam("status")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}

	grants, total, err := h.uc.ListGrants(status, page, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}

	totalPages := 0
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return c.JSON(http.StatusOK, successResp(map[string]interface{}{
		"data": grants,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	}))
}

// CreateGrant godoc
//
//	@Summary		정부과제 생성
//	@Description	관리자용: 새 정부과제 생성
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateGrantRequest	true	"과제 정보"
//	@Success		201		{object}	APIResponse
//	@Router			/admin/grants [post]
func (h *GrantHandler) CreateGrant(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var input application.CreateGrantInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청입니다"))
	}
	g, err := h.uc.CreateGrant(input, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusCreated, successResp(g))
}

// GetGrant godoc
//
//	@Summary		정부과제 상세
//	@Description	정부과제 상세 정보 조회
//	@Tags			Grant
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"과제 ID"
//	@Success		200	{object}	APIResponse
//	@Failure		404	{object}	APIResponse
//	@Router			/grants/{id} [get]
func (h *GrantHandler) GetGrant(c echo.Context) error {
	grantID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	g, err := h.uc.GetGrant(grantID)
	if err != nil {
		return c.JSON(http.StatusNotFound, errorResp("NOT_FOUND", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(g))
}

// ApplyToGrant godoc
//
//	@Summary		정부과제 지원
//	@Description	정부과제에 지원
//	@Tags			Grant
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int				true	"과제 ID"
//	@Param			body	body		ApplyGrantRequest	true	"지원 정보"
//	@Success		200		{object}	APIResponse
//	@Router			/grants/{id}/apply [post]
func (h *GrantHandler) ApplyToGrant(c echo.Context) error {
	userID := middleware.GetUserID(c)
	grantID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	var input application.ApplyGrantInput
	_ = c.Bind(&input)
	app, err := h.uc.ApplyToGrant(grantID, input, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(app))
}

// ApproveApplication godoc
//
//	@Summary		정부과제 지원 승인
//	@Description	관리자용: 정부과제 지원 승인
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int	true	"과제 ID"
//	@Param			appId	path		int	true	"지원 ID"
//	@Success		200		{object}	APIResponse
//	@Router			/admin/grants/{id}/approve/{appId} [post]
func (h *GrantHandler) ApproveApplication(c echo.Context) error {
	userID := middleware.GetUserID(c)
	grantID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	appID, err := strconv.Atoi(c.Param("appId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 지원 ID입니다"))
	}
	if err := h.uc.ApproveApplication(grantID, appID, userID); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(map[string]string{"message": "지원이 승인되었습니다"}))
}

// CloseGrant godoc
//
//	@Summary		정부과제 종료
//	@Description	관리자용: 정부과제 종료
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"과제 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/admin/grants/{id}/close [post]
func (h *GrantHandler) CloseGrant(c echo.Context) error {
	userID := middleware.GetUserID(c)
	grantID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	if err := h.uc.CloseGrant(grantID, userID); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(map[string]string{"message": "과제가 종료되었습니다"}))
}
