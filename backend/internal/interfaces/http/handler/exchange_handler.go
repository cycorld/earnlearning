package handler

import (
	"net/http"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
	"github.com/labstack/echo/v4"
)

type ExchangeHandler struct {
	uc *application.ExchangeUseCase
}

func NewExchangeHandler(uc *application.ExchangeUseCase) *ExchangeHandler {
	return &ExchangeHandler{uc: uc}
}

func (h *ExchangeHandler) ListCompanies(c echo.Context) error {
	companies, err := h.uc.ListCompanies()
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}
	return successResponse(c, http.StatusOK, companies)
}

func (h *ExchangeHandler) GetOrderbook(c echo.Context) error {
	companyID, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 회사 ID입니다")
	}

	orderbook, err := h.uc.GetOrderbook(companyID)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "EXCHANGE_ERROR", err.Error())
	}
	return successResponse(c, http.StatusOK, orderbook)
}

func (h *ExchangeHandler) PlaceOrder(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var input application.PlaceOrderInput
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "입력값이 올바르지 않습니다")
	}

	result, err := h.uc.PlaceOrder(input, userID)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "ORDER_ERROR", err.Error())
	}
	return successResponse(c, http.StatusCreated, result)
}

func (h *ExchangeHandler) CancelOrder(c echo.Context) error {
	userID := middleware.GetUserID(c)
	orderID, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 주문 ID입니다")
	}

	if err := h.uc.CancelOrder(orderID, userID); err != nil {
		return errorResponse(c, http.StatusBadRequest, "CANCEL_ERROR", err.Error())
	}
	return successResponse(c, http.StatusOK, map[string]string{"message": "주문이 취소되었습니다"})
}

func (h *ExchangeHandler) GetMyOrders(c echo.Context) error {
	userID := middleware.GetUserID(c)
	status := c.QueryParam("status")
	companyID := intQuery(c, "company_id", 0)
	page := intQuery(c, "page", 1)
	limit := intQuery(c, "limit", 20)

	result, err := h.uc.GetMyOrders(userID, status, companyID, page, limit)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}
	return successResponse(c, http.StatusOK, result)
}
