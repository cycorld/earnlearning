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

// ListCompanies godoc
//
//	@Summary		거래소 회사 목록
//	@Description	거래소에 상장된 회사 목록
//	@Tags			Exchange
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Router			/exchange/companies [get]
func (h *ExchangeHandler) ListCompanies(c echo.Context) error {
	companies, err := h.uc.ListCompanies()
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}
	return successResponse(c, http.StatusOK, companies)
}

// GetOrderbook godoc
//
//	@Summary		호가창 조회
//	@Description	회사의 매수/매도 호가창 조회
//	@Tags			Exchange
//	@Produce		json
//	@Security		BearerAuth
//	@Param			companyId	path		int	true	"회사 ID"
//	@Success		200			{object}	APIResponse
//	@Router			/exchange/orderbook/{companyId} [get]
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

// PlaceOrder godoc
//
//	@Summary		주문 제출
//	@Description	주식 매수/매도 주문 제출
//	@Tags			Exchange
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		PlaceOrderRequest	true	"주문 정보"
//	@Success		201		{object}	APIResponse
//	@Router			/exchange/orders [post]
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

// CancelOrder godoc
//
//	@Summary		주문 취소
//	@Description	미체결 주문 취소
//	@Tags			Exchange
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"주문 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/exchange/orders/{id} [delete]
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

// GetMyOrders godoc
//
//	@Summary		내 주문 목록
//	@Description	내 주식 주문 목록 조회
//	@Tags			Exchange
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status		query		string	false	"상태 필터"
//	@Param			company_id	query		int		false	"회사 ID"
//	@Param			page		query		int		false	"페이지"	default(1)
//	@Param			limit		query		int		false	"크기"	default(20)
//	@Success		200			{object}	APIResponse
//	@Router			/exchange/orders/mine [get]
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
