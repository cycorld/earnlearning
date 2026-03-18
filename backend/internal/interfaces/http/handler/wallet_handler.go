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

func (h *WalletHandler) SearchRecipients(c echo.Context) error {
	userID := middleware.GetUserID(c)
	q := c.QueryParam("q")

	recipients, err := h.walletUC.SearchRecipients(userID, q)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "검색에 실패했습니다")
	}

	return successResponse(c, http.StatusOK, recipients)
}

func (h *WalletHandler) Transfer(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var input application.TransferInput
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}

	if input.TargetUserID == userID {
		return errorResponse(c, http.StatusBadRequest, "SELF_TRANSFER", "자기 자신에게 송금할 수 없습니다")
	}

	if err := h.walletUC.Transfer(userID, input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "TRANSFER_FAILED", err.Error())
	}

	return successResponse(c, http.StatusOK, map[string]string{"message": "송금이 완료되었습니다"})
}

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
