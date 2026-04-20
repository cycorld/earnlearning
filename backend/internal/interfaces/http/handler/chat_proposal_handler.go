package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/proposal"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
)

// ChatProposalHandler — #106 학생이 챗봇으로 제출한 교수님께의 제안.
type ChatProposalHandler struct {
	uc *application.ChatProposalUseCase
}

func NewChatProposalHandler(uc *application.ChatProposalUseCase) *ChatProposalHandler {
	return &ChatProposalHandler{uc: uc}
}

// ListMine — GET /api/chat/proposals/mine
func (h *ChatProposalHandler) ListMine(c echo.Context) error {
	userID := middleware.GetUserID(c)
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	list, err := h.uc.ListMine(userID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	if list == nil {
		list = []*proposal.Proposal{}
	}
	return c.JSON(http.StatusOK, successResp(list))
}

// Get — GET /api/chat/proposals/:id (학생 본인 또는 admin)
func (h *ChatProposalHandler) Get(c echo.Context) error {
	userID := middleware.GetUserID(c)
	isAdmin := middleware.GetUserRole(c) == "admin"
	id, err := intParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID"))
	}
	p, err := h.uc.Get(userID, id, isAdmin)
	if err != nil {
		return chatProposalErrorResp(c, err)
	}
	return c.JSON(http.StatusOK, successResp(p))
}

// AdminList — GET /api/admin/proposals?status=&category=&limit=&offset=
func (h *ChatProposalHandler) AdminList(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	filter := proposal.Filter{
		Status:   c.QueryParam("status"),
		Category: c.QueryParam("category"),
		Limit:    limit,
		Offset:   offset,
	}
	list, err := h.uc.AdminList(filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	total, _ := h.uc.AdminCount(filter)
	if list == nil {
		list = []*proposal.Proposal{}
	}
	return c.JSON(http.StatusOK, successResp(map[string]any{
		"items": list,
		"total": total,
	}))
}

// AdminUpdate — PATCH /api/admin/proposals/:id
func (h *ChatProposalHandler) AdminUpdate(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID"))
	}
	var in application.UpdateChatProposalInput
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청"))
	}
	p, err := h.uc.AdminUpdate(id, in)
	if err != nil {
		return chatProposalErrorResp(c, err)
	}
	return c.JSON(http.StatusOK, successResp(p))
}

func chatProposalErrorResp(c echo.Context, err error) error {
	if errors.Is(err, proposal.ErrNotFound) {
		return c.JSON(http.StatusNotFound, errorResp("NOT_FOUND", err.Error()))
	}
	return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
}
