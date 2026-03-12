package application

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/earnlearning/backend/internal/domain/loan"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

type LoanUseCase struct {
	db         *sql.DB
	repo       loan.Repository
	walletRepo wallet.Repository
}

func NewLoanUseCase(db *sql.DB, repo loan.Repository, wr wallet.Repository) *LoanUseCase {
	return &LoanUseCase{db: db, repo: repo, walletRepo: wr}
}

// --- Input types ---

type ApplyLoanInput struct {
	Amount  int    `json:"amount"`
	Purpose string `json:"purpose"`
}

type ApproveLoanInput struct {
	InterestRate float64 `json:"interest_rate"`
}

type RepayLoanInput struct {
	Amount int `json:"amount"`
}

// --- Use case methods ---

func (uc *LoanUseCase) ApplyLoan(input ApplyLoanInput, userID int) (*loan.Loan, error) {
	if input.Amount <= 0 {
		return nil, loan.ErrInvalidAmount
	}

	l := &loan.Loan{
		BorrowerID:   userID,
		Amount:       input.Amount,
		Remaining:    input.Amount,
		InterestRate: 0, // set by admin on approval
		PenaltyRate:  0,
		Purpose:      input.Purpose,
		Status:       loan.StatusPending,
	}

	id, err := uc.repo.Create(l)
	if err != nil {
		return nil, err
	}
	return uc.repo.FindByID(id)
}

func (uc *LoanUseCase) GetMyLoans(userID int) ([]*loan.Loan, error) {
	return uc.repo.ListByUser(userID)
}

func (uc *LoanUseCase) ApproveLoan(loanID, adminUserID int, input ApproveLoanInput) (*loan.Loan, error) {
	l, err := uc.repo.FindByID(loanID)
	if err != nil {
		return nil, err
	}
	if l.Status != loan.StatusPending {
		return nil, loan.ErrNotPending
	}

	interestRate := input.InterestRate
	if interestRate <= 0 {
		interestRate = 0.05 // default 5%
	}
	penaltyRate := interestRate * 2

	// Approve loan
	if err := uc.repo.Approve(loanID, adminUserID, interestRate, penaltyRate); err != nil {
		return nil, err
	}

	// Credit borrower wallet
	w, err := uc.walletRepo.FindByUserID(l.BorrowerID)
	if err != nil {
		return nil, err
	}
	err = uc.walletRepo.Credit(w.ID, l.Amount, wallet.TxLoanDisburse,
		fmt.Sprintf("대출 지급: %d원", l.Amount), "loan", loanID)
	if err != nil {
		return nil, err
	}

	// Notify borrower
	uc.createNotification(l.BorrowerID, "loan_approved",
		"대출이 승인되었습니다",
		fmt.Sprintf("%d원 대출이 승인되어 지급되었습니다. 이자율: %.1f%%", l.Amount, interestRate*100),
		"loan", loanID)

	return uc.repo.FindByID(loanID)
}

func (uc *LoanUseCase) RejectLoan(loanID int) error {
	l, err := uc.repo.FindByID(loanID)
	if err != nil {
		return err
	}
	if l.Status != loan.StatusPending {
		return loan.ErrNotPending
	}

	if err := uc.repo.UpdateStatus(loanID, loan.StatusRejected); err != nil {
		return err
	}

	// Notify borrower
	uc.createNotification(l.BorrowerID, "loan_rejected",
		"대출이 거절되었습니다",
		fmt.Sprintf("%d원 대출 신청이 거절되었습니다.", l.Amount),
		"loan", loanID)

	return nil
}

func (uc *LoanUseCase) RepayLoan(loanID, userID int, input RepayLoanInput) (*loan.Loan, error) {
	l, err := uc.repo.FindByID(loanID)
	if err != nil {
		return nil, err
	}
	if l.BorrowerID != userID {
		return nil, fmt.Errorf("본인의 대출만 상환할 수 있습니다")
	}
	if l.Status != loan.StatusActive && l.Status != loan.StatusOverdue {
		return nil, loan.ErrNotActive
	}
	if input.Amount <= 0 {
		return nil, loan.ErrInvalidAmount
	}

	// Check balance
	w, err := uc.walletRepo.FindByUserID(userID)
	if err != nil {
		return nil, loan.ErrInsufficientFunds
	}
	if w.Balance < input.Amount {
		return nil, loan.ErrInsufficientFunds
	}

	// Distribute payment: penalty first, then interest, then principal
	remaining := input.Amount
	var penaltyPaid, interestPaid, principalPaid int

	// 1. Penalty (if overdue)
	if l.Status == loan.StatusOverdue {
		penalty := l.CalcPenalty()
		if penalty > 0 {
			if remaining >= penalty {
				penaltyPaid = penalty
				remaining -= penalty
			} else {
				penaltyPaid = remaining
				remaining = 0
			}
		}
	}

	// 2. Interest
	if remaining > 0 {
		interest := l.CalcWeeklyInterest()
		if interest > 0 {
			if remaining >= interest {
				interestPaid = interest
				remaining -= interest
			} else {
				interestPaid = remaining
				remaining = 0
			}
		}
	}

	// 3. Principal
	if remaining > 0 {
		if remaining > l.Remaining {
			principalPaid = l.Remaining
		} else {
			principalPaid = remaining
		}
	}

	totalPaid := penaltyPaid + interestPaid + principalPaid

	// Debit wallet
	err = uc.walletRepo.Debit(w.ID, totalPaid, wallet.TxLoanRepay,
		fmt.Sprintf("대출 상환: %d원", totalPaid), "loan", loanID)
	if err != nil {
		return nil, err
	}

	// Update remaining
	newRemaining := l.Remaining - principalPaid
	if err := uc.repo.UpdateRemaining(loanID, newRemaining); err != nil {
		return nil, err
	}

	// Create payment record
	payType := loan.PayRepayment
	if penaltyPaid > 0 {
		payType = loan.PayPenalty
	}
	payment := &loan.LoanPayment{
		LoanID:    loanID,
		Amount:    totalPaid,
		Principal: principalPaid,
		Interest:  interestPaid,
		Penalty:   penaltyPaid,
		PayType:   payType,
	}
	if _, err := uc.repo.CreatePayment(payment); err != nil {
		return nil, err
	}

	// Check if fully paid
	if newRemaining <= 0 {
		_ = uc.repo.UpdateStatus(loanID, loan.StatusPaid)
	} else {
		// Reset next payment date
		nextPayment := time.Now().AddDate(0, 0, 7)
		_ = uc.repo.UpdateNextPayment(loanID, &nextPayment)
		// If was overdue, set back to active
		if l.Status == loan.StatusOverdue {
			_ = uc.repo.UpdateStatus(loanID, loan.StatusActive)
		}
	}

	return uc.repo.FindByID(loanID)
}

func (uc *LoanUseCase) ProcessWeeklyInterest() (int, error) {
	loans, err := uc.repo.ListActiveLoans()
	if err != nil {
		return 0, err
	}

	processed := 0
	now := time.Now()

	for _, l := range loans {
		// Only process if next_payment is due
		if l.NextPayment != nil && l.NextPayment.After(now) {
			continue
		}

		interest := l.CalcWeeklyInterest()
		if interest <= 0 {
			continue
		}

		// Try to auto-debit interest from borrower wallet
		w, err := uc.walletRepo.FindByUserID(l.BorrowerID)
		if err != nil {
			// No wallet, set overdue
			_ = uc.repo.UpdateStatus(l.ID, loan.StatusOverdue)
			processed++
			continue
		}

		if w.Balance >= interest {
			// Debit interest
			err = uc.walletRepo.Debit(w.ID, interest, wallet.TxLoanInterest,
				fmt.Sprintf("대출 이자 자동 납부: %d원", interest), "loan", l.ID)
			if err != nil {
				_ = uc.repo.UpdateStatus(l.ID, loan.StatusOverdue)
				processed++
				continue
			}

			// Create auto payment record
			payment := &loan.LoanPayment{
				LoanID:   l.ID,
				Amount:   interest,
				Interest: interest,
				PayType:  loan.PayAuto,
			}
			_, _ = uc.repo.CreatePayment(payment)

			// Set next payment date
			nextPayment := now.AddDate(0, 0, 7)
			_ = uc.repo.UpdateNextPayment(l.ID, &nextPayment)
		} else {
			// Insufficient balance, set overdue
			_ = uc.repo.UpdateStatus(l.ID, loan.StatusOverdue)

			uc.createNotification(l.BorrowerID, "loan_overdue",
				"대출 연체 알림",
				fmt.Sprintf("대출 이자 %d원 납부에 실패하여 연체 상태가 되었습니다.", interest),
				"loan", l.ID)
		}
		processed++
	}
	return processed, nil
}

func (uc *LoanUseCase) AdminListLoans(status string, page, limit int) ([]*loan.Loan, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	filter := loan.LoanFilter{
		Status: status,
	}
	return uc.repo.ListAll(filter, page, limit)
}

func (uc *LoanUseCase) GetLoanPayments(loanID int) ([]*loan.LoanPayment, error) {
	return uc.repo.ListPayments(loanID)
}

func (uc *LoanUseCase) createNotification(userID int, notifType, title, body, refType string, refID int) {
	_, _ = uc.db.Exec(`
		INSERT INTO notifications (user_id, notif_type, title, body, reference_type, reference_id)
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID, notifType, title, body, refType, refID)
}
