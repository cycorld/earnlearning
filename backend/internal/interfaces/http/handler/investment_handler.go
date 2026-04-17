package handler

import (
	"net/http"
	"strconv"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
	"github.com/labstack/echo/v4"
)

type InvestmentHandler struct {
	uc *application.InvestmentUseCase
}

func NewInvestmentHandler(uc *application.InvestmentUseCase) *InvestmentHandler {
	return &InvestmentHandler{uc: uc}
}

// CreateRound godoc
//
//	@Summary		투자 라운드 생성
//	@Description	회사의 새 투자 라운드 생성
//	@Tags			Investment
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateRoundRequest	true	"라운드 정보"
//	@Success		201		{object}	APIResponse
//	@Router			/investment/rounds [post]
func (h *InvestmentHandler) CreateRound(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var input application.CreateRoundInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청입니다"))
	}
	round, err := h.uc.CreateRound(input, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusCreated, successResp(round))
}

// Invest godoc
//
//	@Summary		투자하기
//	@Description	투자 라운드에 투자
//	@Tags			Investment
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"라운드 ID"
//	@Success		201	{object}	APIResponse
//	@Router			/investment/rounds/{id}/invest [post]
func (h *InvestmentHandler) Invest(c echo.Context) error {
	userID := middleware.GetUserID(c)
	roundID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 라운드 ID입니다"))
	}
	var body struct {
		Shares int `json:"shares"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "요청 형식이 잘못되었습니다"))
	}
	inv, err := h.uc.Invest(roundID, userID, body.Shares)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusCreated, successResp(inv))
}

// CloseRoundEarly godoc — owner closes a partially-funded round.
//
//	@Summary		라운드 조기 마감 (owner)
//	@Description	목표 미달 상태에서 현재까지 유치된 금액으로 라운드 확정
//	@Tags			Investment
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"라운드 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/investment/rounds/{id}/close [post]
func (h *InvestmentHandler) CloseRoundEarly(c echo.Context) error {
	userID := middleware.GetUserID(c)
	roundID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 라운드 ID입니다"))
	}
	round, err := h.uc.CloseRoundEarly(roundID, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(round))
}

// CancelRound godoc — owner cancels a round and refunds all investors.
//
//	@Summary		라운드 취소 + 환불 (owner)
//	@Description	진행 중인 라운드를 취소하고 모든 투자자에게 환불
//	@Tags			Investment
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"라운드 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/investment/rounds/{id}/cancel [post]
func (h *InvestmentHandler) CancelRound(c echo.Context) error {
	userID := middleware.GetUserID(c)
	roundID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 라운드 ID입니다"))
	}
	round, err := h.uc.CancelRoundAndRefund(roundID, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(round))
}

// GetRound godoc — single round detail used by /invest/:id.
//
//	@Summary		투자 라운드 상세
//	@Description	특정 라운드의 상세 정보 (누적 금액, 남은 주식 등)
//	@Tags			Investment
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"라운드 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/investment/rounds/{id} [get]
func (h *InvestmentHandler) GetRound(c echo.Context) error {
	roundID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 라운드 ID입니다"))
	}
	round, err := h.uc.GetRound(roundID)
	if err != nil {
		return c.JSON(http.StatusNotFound, errorResp("NOT_FOUND", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(round))
}

// ListRounds godoc
//
//	@Summary		투자 라운드 목록
//	@Description	투자 라운드 목록 조회
//	@Tags			Investment
//	@Produce		json
//	@Security		BearerAuth
//	@Param			company_id	query		int		false	"회사 ID 필터"
//	@Param			status		query		string	false	"상태 필터"
//	@Param			page		query		int		false	"페이지"
//	@Param			limit		query		int		false	"크기"
//	@Success		200			{object}	APIResponse
//	@Router			/investment/rounds [get]
func (h *InvestmentHandler) ListRounds(c echo.Context) error {
	companyID, _ := strconv.Atoi(c.QueryParam("company_id"))
	status := c.QueryParam("status")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	rounds, total, err := h.uc.ListRounds(companyID, status, page, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(map[string]interface{}{
		"rounds": rounds,
		"total":  total,
	}))
}

// GetPortfolio godoc
//
//	@Summary		내 투자 포트폴리오
//	@Description	내 투자 내역 조회
//	@Tags			Investment
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Router			/investment/portfolio [get]
func (h *InvestmentHandler) GetPortfolio(c echo.Context) error {
	userID := middleware.GetUserID(c)
	portfolio, err := h.uc.GetPortfolio(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(portfolio))
}

// ExecuteDividend godoc
//
//	@Summary		배당 실행
//	@Description	회사 배당금 실행
//	@Tags			Investment
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		ExecuteDividendRequest	true	"배당 정보"
//	@Success		201		{object}	APIResponse
//	@Router			/investment/dividends [post]
func (h *InvestmentHandler) ExecuteDividend(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var input application.ExecuteDividendInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청입니다"))
	}
	dividend, err := h.uc.ExecuteDividend(input, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusCreated, successResp(dividend))
}

// GetMyDividends godoc
//
//	@Summary		내 배당 내역
//	@Description	내가 받은 배당 내역 조회
//	@Tags			Investment
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Router			/investment/dividends [get]
func (h *InvestmentHandler) GetMyDividends(c echo.Context) error {
	userID := middleware.GetUserID(c)
	dividends, err := h.uc.GetMyDividends(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(dividends))
}

// CreateKpiRule godoc
//
//	@Summary		KPI 규칙 생성
//	@Description	회사 KPI 규칙 생성
//	@Tags			Investment
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateKpiRuleRequest	true	"KPI 규칙"
//	@Success		201		{object}	APIResponse
//	@Router			/investment/kpi-rules [post]
func (h *InvestmentHandler) CreateKpiRule(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var input application.CreateKpiRuleInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청입니다"))
	}
	rule, err := h.uc.CreateKpiRule(input, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusCreated, successResp(rule))
}

// AddKpiRevenue godoc
//
//	@Summary		KPI 매출 등록
//	@Description	KPI 규칙에 매출 기록 추가
//	@Tags			Investment
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		AddKpiRevenueRequest	true	"매출 정보"
//	@Success		201		{object}	APIResponse
//	@Router			/investment/kpi-revenue [post]
func (h *InvestmentHandler) AddKpiRevenue(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var input application.AddKpiRevenueInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청입니다"))
	}
	revenue, err := h.uc.AddKpiRevenue(input, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusCreated, successResp(revenue))
}
