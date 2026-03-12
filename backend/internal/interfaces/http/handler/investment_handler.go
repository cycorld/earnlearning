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

func (h *InvestmentHandler) Invest(c echo.Context) error {
	userID := middleware.GetUserID(c)
	roundID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 라운드 ID입니다"))
	}
	inv, err := h.uc.Invest(roundID, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusCreated, successResp(inv))
}

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

func (h *InvestmentHandler) GetPortfolio(c echo.Context) error {
	userID := middleware.GetUserID(c)
	portfolio, err := h.uc.GetPortfolio(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(portfolio))
}

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

func (h *InvestmentHandler) GetMyDividends(c echo.Context) error {
	userID := middleware.GetUserID(c)
	dividends, err := h.uc.GetMyDividends(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(dividends))
}

func (h *InvestmentHandler) CreateKpiRule(c echo.Context) error {
	var input application.CreateKpiRuleInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청입니다"))
	}
	rule, err := h.uc.CreateKpiRule(input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusCreated, successResp(rule))
}

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
