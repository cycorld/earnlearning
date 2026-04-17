package application

import (
	"fmt"
	"strings"
	"time"

	"github.com/earnlearning/backend/internal/domain/company"
	"github.com/earnlearning/backend/internal/domain/user"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

type WalletUseCase struct {
	walletRepo  wallet.Repository
	userRepo    user.Repository
	companyRepo company.CompanyRepository
}

func NewWalletUseCase(wr wallet.Repository, ur user.Repository, cr company.CompanyRepository) *WalletUseCase {
	return &WalletUseCase{walletRepo: wr, userRepo: ur, companyRepo: cr}
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

type Recipient struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	StudentID  string `json:"student_id"`
	Department string `json:"department"`
	AvatarURL  string `json:"avatar_url"`
	Type       string `json:"type"` // "user" or "company"
}

func (uc *WalletUseCase) SearchRecipients(senderID int, query string) ([]*Recipient, error) {
	users, _, err := uc.userRepo.ListAll(1, 1000)
	if err != nil {
		return nil, err
	}

	var results []*Recipient
	q := query
	for _, u := range users {
		if u.ID == senderID {
			continue
		}
		if u.Status != "approved" {
			continue
		}
		if q != "" {
			match := contains(u.Name, q) || contains(u.StudentID, q) || contains(u.Department, q)
			if !match {
				continue
			}
		}
		results = append(results, &Recipient{
			ID:         u.ID,
			Name:       u.Name,
			StudentID:  u.StudentID,
			Department: u.Department,
			AvatarURL:  u.AvatarURL,
			Type:       "user",
		})
	}

	if uc.companyRepo != nil {
		companies, err := uc.companyRepo.FindAll()
		if err == nil {
			for _, c := range companies {
				if c.Status == "dissolved" {
					continue
				}
				if q != "" && !contains(c.Name, q) {
					continue
				}
				results = append(results, &Recipient{
					ID:         c.ID,
					Name:       c.Name,
					StudentID:  "", // 법인은 학번 없음
					Department: "법인",
					AvatarURL:  c.LogoURL,
					Type:       "company",
				})
			}
		}
	}

	return results, nil
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) > 0 && strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

type TransferInput struct {
	TargetUserID int    `json:"target_user_id"`
	TargetType   string `json:"target_type"` // "user" or "company"
	Amount       int    `json:"amount"`
	Description  string `json:"description"`
}

func (uc *WalletUseCase) Transfer(senderID int, input TransferInput) error {
	if input.Amount <= 0 {
		return wallet.ErrInvalidAmount
	}

	// Find sender wallet
	senderWallet, err := uc.walletRepo.FindByUserID(senderID)
	if err != nil {
		return fmt.Errorf("보내는 사람의 지갑을 찾을 수 없습니다")
	}

	if senderWallet.Balance < input.Amount {
		return wallet.ErrInsufficientFunds
	}

	sender, _ := uc.userRepo.FindByID(senderID)
	senderName := "알 수 없음"
	if sender != nil {
		senderName = sender.Name
	}

	desc := input.Description
	if desc == "" {
		desc = "개인 송금"
	}

	if input.TargetType == "company" {
		if uc.companyRepo == nil {
			return fmt.Errorf("법인 송금이 지원되지 않습니다")
		}
		// TargetUserID 는 여기서는 companyID 로 해석
		companyID := input.TargetUserID
		c, err := uc.companyRepo.FindByID(companyID)
		if err != nil {
			return fmt.Errorf("법인을 찾을 수 없습니다")
		}
		if c.Status == "dissolved" {
			return fmt.Errorf("청산된 법인에는 송금할 수 없습니다")
		}
		cw, err := uc.companyRepo.FindCompanyWallet(companyID)
		if err != nil {
			return fmt.Errorf("법인 지갑을 찾을 수 없습니다")
		}

		// Debit sender (개인 잔액 차감)
		if err := uc.walletRepo.Debit(senderWallet.ID, input.Amount, wallet.TxCompanyTransfer,
			fmt.Sprintf("[%s] 법인 송금: %s", c.Name, desc), "company", companyID); err != nil {
			return err
		}

		// Credit company wallet
		if err := uc.companyRepo.CreditCompanyWallet(cw.ID, input.Amount, string(wallet.TxCompanyTransfer),
			fmt.Sprintf("%s 개인 송금 입금: %s", senderName, desc), "user", senderID); err != nil {
			// Roll back the personal debit so funds are not lost.
			_ = uc.walletRepo.Credit(senderWallet.ID, input.Amount, wallet.TxCompanyTransfer,
				fmt.Sprintf("[%s] 법인 송금 실패 환불", c.Name), "company", companyID)
			return fmt.Errorf("법인 지갑 입금 실패: %w", err)
		}
		return nil
	}

	// Default: user → user
	receiverUserID := input.TargetUserID
	receiverWallet, err := uc.walletRepo.FindByUserID(receiverUserID)
	if err != nil {
		return fmt.Errorf("받는 사람의 지갑을 찾을 수 없습니다")
	}

	receiver, _ := uc.userRepo.FindByID(receiverUserID)
	receiverName := "알 수 없음"
	if receiver != nil {
		receiverName = receiver.Name
	}

	// Debit sender
	err = uc.walletRepo.Debit(senderWallet.ID, input.Amount, wallet.TxTransfer,
		fmt.Sprintf("%s에게 송금: %s", receiverName, desc), "user", receiverUserID)
	if err != nil {
		return err
	}

	// Credit receiver
	err = uc.walletRepo.Credit(receiverWallet.ID, input.Amount, wallet.TxTransfer,
		fmt.Sprintf("%s로부터 송금: %s", senderName, desc), "user", senderID)
	if err != nil {
		return err
	}

	return nil
}

func (uc *WalletUseCase) GetRanking(limit int) ([]*wallet.RankEntry, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return uc.walletRepo.GetRanking(limit)
}
