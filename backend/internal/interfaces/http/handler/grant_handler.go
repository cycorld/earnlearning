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
