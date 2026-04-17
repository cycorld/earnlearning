package handler

import (
	"net/http"
	"time"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/wallet"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
	"github.com/labstack/echo/v4"
)

type WalletHandler struct {
	walletUC *application.WalletUseCase
}

func NewWalletHandler(uc *application.WalletUseCase) *WalletHandler {
	return &WalletHandler{walletUC: uc}
}

// GetWallet godoc
//
//	@Summary		지갑 조회
//	@Description	내 지갑 잔액 및 정보 조회
//	@Tags			Wallet
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Failure		404	{object}	APIResponse
//	@Router			/wallet [get]
func (h *WalletHandler) GetWallet(c echo.Context) error {
	userID := middleware.GetUserID(c)
	resp, err := h.walletUC.GetWallet(userID)
	if err != nil {
		if err == wallet.ErrNotFound {
			return errorResponse(c, http.StatusNotFound, "WALLET_NOT_FOUND", err.Error())
		}
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}

	return successResponse(c, http.StatusOK, resp)
}

// GetTransactions godoc
//
//	@Summary		거래 내역 조회
//	@Description	내 지갑 거래 내역 조회 (필터, 페이지네이션)
//	@Tags			Wallet
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"페이지 번호"	default(1)
//	@Param			limit		query		int		false	"페이지 크기"	default(20)
//	@Param			tx_type		query		string	false	"거래 유형 필터"
//	@Param			start_date	query		string	false	"시작일 (YYYY-MM-DD)"
//	@Param			end_date	query		string	false	"종료일 (YYYY-MM-DD)"
//	@Success		200			{object}	APIResponse
//	@Router			/wallet/transactions [get]
func (h *WalletHandler) GetTransactions(c echo.Context) error {
	userID := middleware.GetUserID(c)
	page := intQuery(c, "page", 1)
	limit := intQuery(c, "limit", 20)
	txType := c.QueryParam("tx_type")

	var startDate, endDate *time.Time
	if sd := c.QueryParam("start_date"); sd != "" {
		t, err := time.Parse("2006-01-02", sd)
		if err == nil {
			startDate = &t
		}
	}
	if ed := c.QueryParam("end_date"); ed != "" {
		t, err := time.Parse("2006-01-02", ed)
		if err == nil {
			end := t.Add(24*time.Hour - time.Second)
			endDate = &end
		}
	}

	result, err := h.walletUC.GetTransactions(userID, txType, startDate, endDate, page, limit)
	if err != nil {
		if err == wallet.ErrNotFound {
			return errorResponse(c, http.StatusNotFound, "WALLET_NOT_FOUND", err.Error())
		}
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}

	return successResponse(c, http.StatusOK, result)
}

// SearchRecipients godoc
//
//	@Summary		송금 대상 검색
//	@Description	이름/이메일로 송금 대상 사용자 검색
//	@Tags			Wallet
//	@Produce		json
//	@Security		BearerAuth
//	@Param			q	query		string	false	"검색어"
//	@Success		200	{object}	APIResponse
//	@Router			/wallet/recipients [get]
func (h *WalletHandler) SearchRecipients(c echo.Context) error {
	userID := middleware.GetUserID(c)
	q := c.QueryParam("q")

	recipients, err := h.walletUC.SearchRecipients(userID, q)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "검색에 실패했습니다")
	}

	return successResponse(c, http.StatusOK, recipients)
}

// Transfer godoc
//
//	@Summary		송금
//	@Description	다른 사용자에게 송금
//	@Tags			Wallet
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		TransferRequest	true	"송금 정보"
//	@Success		200		{object}	APIResponse
//	@Failure		400		{object}	APIResponse
//	@Router			/wallet/transfer [post]
func (h *WalletHandler) Transfer(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var input application.TransferInput
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}

	// 자기 자신에게 송금 방지는 user 대상 때만 의미 있음 (company 는 별도 엔티티)
	if input.TargetType != "company" && input.TargetUserID == userID {
		return errorResponse(c, http.StatusBadRequest, "SELF_TRANSFER", "자기 자신에게 송금할 수 없습니다")
	}

	if err := h.walletUC.Transfer(userID, input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "TRANSFER_FAILED", err.Error())
	}

	return successResponse(c, http.StatusOK, map[string]string{"message": "송금이 완료되었습니다"})
}

// AdminTransfer godoc
//
//	@Summary		관리자 일괄 송금
//	@Description	관리자용: 여러 사용자에게 일괄 송금
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		AdminTransferRequest	true	"일괄 송금 정보"
//	@Success		200		{object}	APIResponse
//	@Router			/admin/wallet/transfer [post]
func (h *WalletHandler) AdminTransfer(c echo.Context) error {
	var input application.AdminTransferInput
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}

	count, err := h.walletUC.AdminTransfer(input)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "TRANSFER_FAILED", err.Error())
	}

	return successResponse(c, http.StatusOK, map[string]interface{}{
		"message":       "송금이 완료되었습니다",
		"success_count": count,
	})
}

// GetRanking godoc
//
//	@Summary		자산 랭킹
//	@Description	전체 사용자 자산 랭킹 조회
//	@Tags			Wallet
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int	false	"조회 수"	default(20)
//	@Success		200		{object}	APIResponse
//	@Router			/wallet/ranking [get]
func (h *WalletHandler) GetRanking(c echo.Context) error {
	limit := intQuery(c, "limit", 20)

	entries, err := h.walletUC.GetRanking(limit)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}

	if entries == nil {
		entries = []*wallet.RankEntry{}
	}

	return successResponse(c, http.StatusOK, entries)
}
