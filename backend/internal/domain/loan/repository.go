package loan

import "time"

type LoanFilter struct {
	BorrowerID int
	Status     string
}

type Repository interface {
	Create(loan *Loan) (int, error)
	FindByID(id int) (*Loan, error)
	ListByUser(userID int) ([]*Loan, error)
	ListAll(filter LoanFilter, page, limit int) ([]*Loan, int, error)
	UpdateStatus(id int, status LoanStatus) error
	Approve(id, approvedBy int, interestRate, penaltyRate float64) error
	UpdateRemaining(id, remaining int) error
	UpdateNextPayment(id int, next *time.Time) error
	ListActiveLoans() ([]*Loan, error)

	CreatePayment(p *LoanPayment) (int, error)
	ListPayments(loanID int) ([]*LoanPayment, error)
}
