package loan

import "time"

type LoanStatus string

const (
	StatusPending  LoanStatus = "pending"
	StatusRejected LoanStatus = "rejected"
	StatusActive   LoanStatus = "active"
	StatusPaid     LoanStatus = "paid"
	StatusOverdue  LoanStatus = "overdue"
)

type PayType string

const (
	PayInterest  PayType = "interest"
	PayRepayment PayType = "repayment"
	PayPenalty   PayType = "penalty"
	PayAuto      PayType = "auto"
)

type Loan struct {
	ID           int        `json:"id"`
	BorrowerID   int        `json:"borrower_id"`
	Amount       int        `json:"amount"`
	Remaining    int        `json:"remaining"`
	InterestRate float64    `json:"interest_rate"`
	PenaltyRate  float64    `json:"penalty_rate"`
	Purpose      string     `json:"purpose"`
	Status       LoanStatus `json:"status"`
	ApprovedBy   *int       `json:"approved_by"`
	ApprovedAt   *time.Time `json:"approved_at"`
	NextPayment  *time.Time `json:"next_payment"`
	CreatedAt    time.Time  `json:"created_at"`

	// Computed fields
	WeeklyInterest int    `json:"weekly_interest,omitempty"`
	BorrowerName   string `json:"borrower_name,omitempty"`
}

// CalcWeeklyInterest calculates weekly interest on the remaining balance.
func (l *Loan) CalcWeeklyInterest() int {
	return int(float64(l.Remaining) * l.InterestRate)
}

// CalcPenalty calculates penalty on the remaining balance.
func (l *Loan) CalcPenalty() int {
	return int(float64(l.Remaining) * l.PenaltyRate)
}

type LoanPayment struct {
	ID        int       `json:"id"`
	LoanID    int       `json:"loan_id"`
	Amount    int       `json:"amount"`
	Principal int       `json:"principal"`
	Interest  int       `json:"interest"`
	Penalty   int       `json:"penalty"`
	PayType   PayType   `json:"pay_type"`
	CreatedAt time.Time `json:"created_at"`
}
