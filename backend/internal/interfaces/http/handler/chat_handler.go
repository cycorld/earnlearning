package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/chat"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
)

type ChatHandler struct {
	uc *application.ChatUseCase
}

func NewChatHandler(uc *application.ChatUseCase) *ChatHandler { return &ChatHandler{uc: uc} }

// ============================================================================
// Student endpoints
// ============================================================================

type createSessionInput struct {
	SkillSlug string `json:"skill_slug"`
}

// CreateSession godoc
//
//	@Summary		새 챗봇 세션 시작
//	@Tags			Chat
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Router			/chat/sessions [post]
func (h *ChatHandler) CreateSession(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var in createSessionInput
	_ = c.Bind(&in)
	s, err := h.uc.CreateSession(userID, in.SkillSlug)
	if err != nil {
		return chatErrorResponse(c, err)
	}
	return c.JSON(http.StatusCreated, successResp(s))
}

func (h *ChatHandler) ListSessions(c echo.Context) error {
	userID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.QueryParam("page"))
	items, total, err := h.uc.ListSessions(userID, page)
	if err != nil {
		return chatErrorResponse(c, err)
	}
	if items == nil {
		items = []*chat.Session{}
	}
	return c.JSON(http.StatusOK, successResp(map[string]any{
		"items": items, "total": total,
	}))
}

func (h *ChatHandler) GetSession(c echo.Context) error {
	userID := middleware.GetUserID(c)
	id, err := intParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	s, err := h.uc.GetSession(userID, id)
	if err != nil {
		return chatErrorResponse(c, err)
	}
	return c.JSON(http.StatusOK, successResp(s))
}

type askInput struct {
	Message   string `json:"message"`
	Mode      string `json:"mode"`       // "fast" | "deep"
	SkillSlug string `json:"skill_slug"` // optional override
}

// Ask godoc
//
//	@Summary		챗봇에 질문 (non-streaming)
//	@Tags			Chat
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Router			/chat/sessions/{id}/ask [post]
func (h *ChatHandler) Ask(c echo.Context) error {
	userID := middleware.GetUserID(c)
	isAdmin := middleware.GetUserRole(c) == "admin"
	id, err := intParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	var in askInput
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청"))
	}
	if strings.TrimSpace(in.Message) == "" {
		return c.JSON(http.StatusBadRequest, errorResp("EMPTY_MESSAGE", "메시지를 입력해주세요"))
	}
	out, err := h.uc.Ask(c.Request().Context(), application.AskInput{
		SessionID: id,
		UserID:    userID,
		IsAdmin:   isAdmin,
		Message:   in.Message,
		Mode:      chat.AskMode(in.Mode),
		SkillSlug: in.SkillSlug,
	})
	if err != nil {
		return chatErrorResponse(c, err)
	}
	return c.JSON(http.StatusOK, successResp(out))
}

func (h *ChatHandler) DeleteSession(c echo.Context) error {
	userID := middleware.GetUserID(c)
	id, err := intParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	if err := h.uc.DeleteSession(userID, id); err != nil {
		return chatErrorResponse(c, err)
	}
	return c.JSON(http.StatusOK, successResp(map[string]string{"status": "deleted"}))
}

func (h *ChatHandler) ListSkills(c echo.Context) error {
	isAdmin := middleware.GetUserRole(c) == "admin"
	items, err := h.uc.ListSkills(isAdmin)
	if err != nil {
		return chatErrorResponse(c, err)
	}
	if items == nil {
		items = []*chat.Skill{}
	}
	return c.JSON(http.StatusOK, successResp(items))
}

// ============================================================================
// Admin endpoints
// ============================================================================

type adminSkillInput struct {
	ID                     int      `json:"id"`
	Slug                   string   `json:"slug"`
	Name                   string   `json:"name"`
	Description            string   `json:"description"`
	SystemPrompt           string   `json:"system_prompt"`
	DefaultModel           string   `json:"default_model"`
	DefaultReasoningEffort string   `json:"default_reasoning_effort"`
	ToolsAllowed           []string `json:"tools_allowed"`
	WikiScope              []string `json:"wiki_scope"`
	Enabled                bool     `json:"enabled"`
	AdminOnly              bool     `json:"admin_only"`
}

func (h *ChatHandler) AdminCreateSkill(c echo.Context) error {
	actorID := middleware.GetUserID(c)
	var in adminSkillInput
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청"))
	}
	sk := &chat.Skill{
		Slug:                   in.Slug,
		Name:                   in.Name,
		Description:            in.Description,
		SystemPrompt:           in.SystemPrompt,
		DefaultModel:           in.DefaultModel,
		DefaultReasoningEffort: in.DefaultReasoningEffort,
		ToolsAllowed:           in.ToolsAllowed,
		WikiScope:              in.WikiScope,
		Enabled:                in.Enabled,
		AdminOnly:              in.AdminOnly,
	}
	id, err := h.uc.AdminCreateSkill(actorID, sk)
	if err != nil {
		return chatErrorResponse(c, err)
	}
	sk.ID = id
	return c.JSON(http.StatusCreated, successResp(sk))
}

func (h *ChatHandler) AdminUpdateSkill(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	var in adminSkillInput
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청"))
	}
	sk := &chat.Skill{
		ID:                     id,
		Slug:                   in.Slug,
		Name:                   in.Name,
		Description:            in.Description,
		SystemPrompt:           in.SystemPrompt,
		DefaultModel:           in.DefaultModel,
		DefaultReasoningEffort: in.DefaultReasoningEffort,
		ToolsAllowed:           in.ToolsAllowed,
		WikiScope:              in.WikiScope,
		Enabled:                in.Enabled,
		AdminOnly:              in.AdminOnly,
	}
	if err := h.uc.AdminUpdateSkill(sk); err != nil {
		return chatErrorResponse(c, err)
	}
	return c.JSON(http.StatusOK, successResp(sk))
}

func (h *ChatHandler) AdminDeleteSkill(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	if err := h.uc.AdminDeleteSkill(id); err != nil {
		return chatErrorResponse(c, err)
	}
	return c.JSON(http.StatusOK, successResp(map[string]string{"status": "deleted"}))
}

// AdminListSessions — 모든 유저의 세션 목록. ?user_id=N 필터 가능.
func (h *ChatHandler) AdminListSessions(c echo.Context) error {
	userID, _ := strconv.Atoi(c.QueryParam("user_id"))
	page, _ := strconv.Atoi(c.QueryParam("page"))
	items, total, err := h.uc.AdminListAllSessions(userID, page)
	if err != nil {
		return chatErrorResponse(c, err)
	}
	if items == nil {
		items = []*chat.Session{}
	}
	return c.JSON(http.StatusOK, successResp(map[string]any{
		"items": items, "total": total,
	}))
}

// AdminGetSession — 임의 유저의 세션 + 전체 메시지 열람.
func (h *ChatHandler) AdminGetSession(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	s, err := h.uc.AdminGetSession(id)
	if err != nil {
		return chatErrorResponse(c, err)
	}
	return c.JSON(http.StatusOK, successResp(s))
}

func (h *ChatHandler) AdminListWiki(c echo.Context) error {
	items, err := h.uc.ListWikiDocs()
	if err != nil {
		return chatErrorResponse(c, err)
	}
	if items == nil {
		items = []*chat.WikiDocMeta{}
	}
	return c.JSON(http.StatusOK, successResp(items))
}

func (h *ChatHandler) AdminReindexWiki(c echo.Context) error {
	n, err := h.uc.AdminReindexWiki()
	if err != nil {
		return chatErrorResponse(c, err)
	}
	return c.JSON(http.StatusOK, successResp(map[string]any{
		"indexed": n, "status": "ok",
	}))
}

// ============================================================================
// Error mapping
// ============================================================================

func chatErrorResponse(c echo.Context, err error) error {
	switch {
	case errors.Is(err, chat.ErrSessionNotFound), errors.Is(err, chat.ErrSkillNotFound):
		return c.JSON(http.StatusNotFound, errorResp("NOT_FOUND", err.Error()))
	case errors.Is(err, chat.ErrForbidden), errors.Is(err, chat.ErrAdminOnly):
		return c.JSON(http.StatusForbidden, errorResp("FORBIDDEN", err.Error()))
	case errors.Is(err, chat.ErrSkillDisabled):
		return c.JSON(http.StatusGone, errorResp("DISABLED", err.Error()))
	case errors.Is(err, chat.ErrInvalidSlug):
		return c.JSON(http.StatusBadRequest, errorResp("INVALID", err.Error()))
	default:
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
}
