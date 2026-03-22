package handler

import (
	"net/http"
	"strconv"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/loan"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
	"github.com/labstack/echo/v4"
)

type LoanHandler struct {
	uc *application.LoanUseCase
}

func NewLoanHandler(uc *application.LoanUseCase) *LoanHandler {
	return &LoanHandler{uc: uc}
}

// ApplyLoan godoc
//
//	@Summary		대출 신청
//	@Description	대출 신청
//	@Tags			Loan
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		ApplyLoanRequest	true	"대출 정보"
//	@Success		201		{object}	APIResponse
//	@Router			/loans [post]
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

// GetMyLoans godoc
//
//	@Summary		내 대출 목록
//	@Description	내 대출 목록 조회
//	@Tags			Loan
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Router			/loans/mine [get]
func (h *LoanHandler) GetMyLoans(c echo.Context) error {
	userID := middleware.GetUserID(c)
	loans, err := h.uc.GetMyLoans(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	if loans == nil {
		loans = []*loan.Loan{}
	}
	return c.JSON(http.StatusOK, successResp(loans))
}

// ApproveLoan godoc
//
//	@Summary		대출 승인
//	@Description	관리자용: 대출 신청 승인
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int					true	"대출 ID"
//	@Param			body	body		ApproveLoanRequest	true	"승인 조건"
//	@Success		200		{object}	APIResponse
//	@Router			/admin/loans/{id}/approve [put]
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

// RejectLoan godoc
//
//	@Summary		대출 거절
//	@Description	관리자용: 대출 신청 거절
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"대출 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/admin/loans/{id}/reject [put]
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

// RepayLoan godoc
//
//	@Summary		대출 상환
//	@Description	대출금 일부 또는 전액 상환
//	@Tags			Loan
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int				true	"대출 ID"
//	@Param			body	body		RepayLoanRequest	true	"상환 금액"
//	@Success		200		{object}	APIResponse
//	@Router			/loans/{id}/repay [post]
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

// ProcessWeeklyInterest godoc
//
//	@Summary		주간 이자 처리
//	@Description	관리자용: 전체 대출 주간 이자 처리
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Router			/admin/loans/weekly-interest [post]
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

// AdminListLoans godoc
//
//	@Summary		전체 대출 목록
//	@Description	관리자용: 전체 대출 목록 조회
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status	query		string	false	"상태 필터"
//	@Param			page	query		int		false	"페이지"
//	@Param			limit	query		int		false	"크기"
//	@Success		200		{object}	APIResponse
//	@Router			/admin/loans [get]
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

// GetLoanPayments godoc
//
//	@Summary		상환 내역 조회
//	@Description	대출 상환 내역 조회
//	@Tags			Loan
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"대출 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/loans/{id}/payments [get]
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
