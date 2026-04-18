package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/llm"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
)

type LLMHandler struct {
	uc *application.LLMUseCase
}

func NewLLMHandler(uc *application.LLMUseCase) *LLMHandler {
	return &LLMHandler{uc: uc}
}

// GetMyKey godoc
//
//	@Summary		내 LLM API 키 조회 (자동 발급)
//	@Description	LLM proxy (llm.cycorld.com) 용 API 키. 최초 호출 시 자동으로 발급됨. 평문 키는 최초 1회에만 포함됨.
//	@Tags			LLM
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Router			/llm/me [get]
func (h *LLMHandler) GetMyKey(c echo.Context) error {
	userID := middleware.GetUserID(c)
	k, err := h.uc.EnsureKey(c.Request().Context(), userID)
	if err != nil {
		return llmErrorResponse(c, err)
	}
	return c.JSON(http.StatusOK, successResp(k))
}

// RotateMyKey godoc
//
//	@Summary		내 LLM API 키 재발급
//	@Description	기존 키를 즉시 폐기하고 새 키를 발급. 평문 키는 이 응답에 1회만 포함됨.
//	@Tags			LLM
//	@Produce		json
//	@Security		BearerAuth
//	@Success		201	{object}	APIResponse
//	@Router			/llm/me/rotate [post]
func (h *LLMHandler) RotateMyKey(c echo.Context) error {
	userID := middleware.GetUserID(c)
	k, err := h.uc.RotateKey(c.Request().Context(), userID)
	if err != nil {
		return llmErrorResponse(c, err)
	}
	return c.JSON(http.StatusCreated, successResp(k))
}

// GetMyUsage godoc
//
//	@Summary		내 LLM 사용량 + 과금 내역
//	@Description	최근 N일 일별 사용량 + 누적 청구액 + 최근 7일 청구액. days 쿼리 파라미터 (default 30).
//	@Tags			LLM
//	@Produce		json
//	@Security		BearerAuth
//	@Param			days	query		int	false	"일수"	default(30)
//	@Success		200		{object}	APIResponse
//	@Router			/llm/me/usage [get]
func (h *LLMHandler) GetMyUsage(c echo.Context) error {
	userID := middleware.GetUserID(c)
	days, _ := strconv.Atoi(c.QueryParam("days"))
	if days <= 0 || days > 365 {
		days = 30
	}

	daily, err := h.uc.ListDailyUsage(userID, days)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	if daily == nil {
		daily = []*llm.DailyUsage{}
	}
	summary, err := h.uc.Summary(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}

	return c.JSON(http.StatusOK, successResp(map[string]any{
		"daily":   daily,
		"summary": summary,
	}))
}

// llmErrorResponse — domain 에러를 HTTP 상태코드로 매핑.
func llmErrorResponse(c echo.Context, err error) error {
	switch {
	case errors.Is(err, llm.ErrNoEmail):
		return c.JSON(http.StatusBadRequest, errorResp("NO_EMAIL", err.Error()))
	case errors.Is(err, llm.ErrProxyUnavailable):
		return c.JSON(http.StatusServiceUnavailable, errorResp("PROXY_DOWN", err.Error()))
	default:
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
}
