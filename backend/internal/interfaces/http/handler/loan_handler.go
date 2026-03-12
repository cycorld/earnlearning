package handler

import (
	"net/http"
	"strconv"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
	"github.com/labstack/echo/v4"
)

type LoanHandler struct {
	uc *application.LoanUseCase
}

func NewLoanHandler(uc *application.LoanUseCase) *LoanHandler {
	return &LoanHandler{uc: uc}
}

func (h *LoanHandler) ApplyLoan(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var input application.ApplyLoanInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청입니다"))
	}
	loan, err := h.uc.ApplyLoan(input, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusCreated, successResp(loan))
}

func (h *LoanHandler) GetMyLoans(c echo.Context) error {
	userID := middleware.GetUserID(c)
	loans, err := h.uc.GetMyLoans(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(loans))
}

func (h *LoanHandler) ApproveLoan(c echo.Context) error {
	adminUserID := middleware.GetUserID(c)
	loanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	var input application.ApproveLoanInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청입니다"))
	}
	loan, err := h.uc.ApproveLoan(loanID, adminUserID, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(loan))
}

func (h *LoanHandler) RejectLoan(c echo.Context) error {
	loanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	if err := h.uc.RejectLoan(loanID); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(map[string]string{"message": "대출이 거절되었습니다"}))
}

func (h *LoanHandler) RepayLoan(c echo.Context) error {
	userID := middleware.GetUserID(c)
	loanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	var input application.RepayLoanInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청입니다"))
	}
	loan, err := h.uc.RepayLoan(loanID, userID, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(loan))
}

func (h *LoanHandler) ProcessWeeklyInterest(c echo.Context) error {
	processed, err := h.uc.ProcessWeeklyInterest()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(map[string]interface{}{
		"processed": processed,
		"message":   "주간 이자 처리가 완료되었습니다",
	}))
}

func (h *LoanHandler) AdminListLoans(c echo.Context) error {
	status := c.QueryParam("status")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	loans, total, err := h.uc.AdminListLoans(status, page, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(map[string]interface{}{
		"loans": loans,
		"total": total,
	}))
}

func (h *LoanHandler) GetLoanPayments(c echo.Context) error {
	loanID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	payments, err := h.uc.GetLoanPayments(loanID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(payments))
}
