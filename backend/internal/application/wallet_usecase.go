package application

import (
	"fmt"
	"time"

	"github.com/earnlearning/backend/internal/domain/user"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

type WalletUseCase struct {
	walletRepo wallet.Repository
	userRepo   user.Repository
}

func NewWalletUseCase(wr wallet.Repository, ur user.Repository) *WalletUseCase {
	return &WalletUseCase{walletRepo: wr, userRepo: ur}
}

type WalletResponse struct {
	Wallet        *wallet.Wallet         `json:"wallet"`
	Assets        *wallet.AssetBreakdown `json:"assets"`
	Rank          int                    `json:"rank"`
	TotalStudents int                    `json:"total_students"`
}

func (uc *WalletUseCase) GetWallet(userID int) (*WalletResponse, error) {
	w, err := uc.walletRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	assets, err := uc.walletRepo.GetAssetBreakdown(userID)
	if err != nil {
		return nil, err
	}

	// Calculate rank
	rank := 0
	totalStudents := 0
	rankings, err := uc.walletRepo.GetRanking(1000) // get all rankings
	if err == nil {
		totalStudents = len(rankings)
		for _, r := range rankings {
			if r.UserID == userID {
				rank = r.Rank
				break
			}
		}
	}

	return &WalletResponse{Wallet: w, Assets: assets, Rank: rank, TotalStudents: totalStudents}, nil
}

type TransactionListResult struct {
	Data       []*wallet.Transaction `json:"data"`
	Pagination PaginationInfo        `json:"pagination"`
}

func (uc *WalletUseCase) GetTransactions(userID int, txType string, startDate, endDate *time.Time, page, limit int) (*TransactionListResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	w, err := uc.walletRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	filter := wallet.TransactionFilter{
		TxType:    txType,
		StartDate: startDate,
		EndDate:   endDate,
	}

	txs, total, err := uc.walletRepo.GetTransactions(w.ID, filter, page, limit)
	if err != nil {
		return nil, err
	}

	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}

	if txs == nil {
		txs = []*wallet.Transaction{}
	}

	return &TransactionListResult{
		Data: txs,
		Pagination: PaginationInfo{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

type AdminTransferInput struct {
	TargetUserIDs []int  `json:"target_user_ids"`
	TargetAll     bool   `json:"target_all"`
	Amount        int    `json:"amount"`
	Description   string `json:"description"`
}

func (uc *WalletUseCase) AdminTransfer(input AdminTransferInput) (int, error) {
	if input.Amount == 0 {
		return 0, wallet.ErrInvalidAmount
	}

	var userIDs []int

	if input.TargetAll {
		users, _, err := uc.userRepo.ListAll(1, 10000)
		if err != nil {
			return 0, err
		}
		for _, u := range users {
			userIDs = append(userIDs, u.ID)
		}
	} else {
		userIDs = input.TargetUserIDs
	}

	successCount := 0
	for _, uid := range userIDs {
		w, err := uc.walletRepo.FindByUserID(uid)
		if err != nil {
			// try creating wallet
			walletID, createErr := uc.walletRepo.CreateWallet(uid)
			if createErr != nil {
				continue
			}
			w = &wallet.Wallet{ID: walletID, UserID: uid, Balance: 0}
		}

		desc := input.Description
		if desc == "" {
			desc = "관리자 송금"
		}

		if input.Amount > 0 {
			// Credit (admin adding money) — no balance check needed
			err = uc.walletRepo.Credit(w.ID, input.Amount, wallet.TxAdminTransfer, desc, "", 0)
		} else {
			// Debit (admin deducting money) — admin skips balance check
			// Use Credit with negative conceptually, but implementation-wise we use Debit
			// For admin, we allow debit even if balance goes negative
			err = uc.walletRepo.Debit(w.ID, -input.Amount, wallet.TxAdminTransfer, desc, "", 0)
		}

		if err != nil {
			// For admin transfer, log but continue
			fmt.Printf("admin transfer failed for user %d: %v\n", uid, err)
			continue
		}
		successCount++
	}

	return successCount, nil
}

func (uc *WalletUseCase) GetRanking(limit int) ([]*wallet.RankEntry, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return uc.walletRepo.GetRanking(limit)
}
