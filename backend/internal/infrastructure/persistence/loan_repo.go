package persistence

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/earnlearning/backend/internal/domain/loan"
)

type LoanRepo struct {
	db *sql.DB
}

func NewLoanRepo(db *sql.DB) *LoanRepo {
	return &LoanRepo{db: db}
}

func (r *LoanRepo) Create(l *loan.Loan) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO loans (borrower_id, amount, remaining, interest_rate, penalty_rate, purpose, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		l.BorrowerID, l.Amount, l.Remaining, l.InterestRate, l.PenaltyRate, l.Purpose, l.Status,
	)
	if err != nil {
		return 0, fmt.Errorf("create loan: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *LoanRepo) FindByID(id int) (*loan.Loan, error) {
	l := &loan.Loan{}
	var approvedBy sql.NullInt64
	var approvedAt sql.NullTime
	var nextPayment sql.NullTime

	err := r.db.QueryRow(`
		SELECT l.id, l.borrower_id, l.amount, l.remaining, l.interest_rate, l.penalty_rate,
			   l.purpose, l.status, l.approved_by, l.approved_at, l.next_payment, l.created_at,
			   u.name AS borrower_name
		FROM loans l
		JOIN users u ON u.id = l.borrower_id
		WHERE l.id = ?`, id).Scan(
		&l.ID, &l.BorrowerID, &l.Amount, &l.Remaining, &l.InterestRate, &l.PenaltyRate,
		&l.Purpose, &l.Status, &approvedBy, &approvedAt, &nextPayment, &l.CreatedAt,
		&l.BorrowerName,
	)
	if err == sql.ErrNoRows {
		return nil, loan.ErrLoanNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find loan: %w", err)
	}
	if approvedBy.Valid {
		ab := int(approvedBy.Int64)
		l.ApprovedBy = &ab
	}
	if approvedAt.Valid {
		l.ApprovedAt = &approvedAt.Time
	}
	if nextPayment.Valid {
		l.NextPayment = &nextPayment.Time
	}
	l.WeeklyInterest = l.CalcWeeklyInterest()
	return l, nil
}

func (r *LoanRepo) ListByUser(userID int) ([]*loan.Loan, error) {
	rows, err := r.db.Query(`
		SELECT l.id, l.borrower_id, l.amount, l.remaining, l.interest_rate, l.penalty_rate,
			   l.purpose, l.status, l.approved_by, l.approved_at, l.next_payment, l.created_at,
			   u.name AS borrower_name
		FROM loans l
		JOIN users u ON u.id = l.borrower_id
		WHERE l.borrower_id = ?
		ORDER BY l.created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list loans by user: %w", err)
	}
	defer rows.Close()
	return r.scanLoans(rows)
}

func (r *LoanRepo) ListAll(filter loan.LoanFilter, page, limit int) ([]*loan.Loan, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}

	if filter.BorrowerID > 0 {
		where = append(where, "l.borrower_id = ?")
		args = append(args, filter.BorrowerID)
	}
	if filter.Status != "" {
		where = append(where, "l.status = ?")
		args = append(args, filter.Status)
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM loans l WHERE "+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count loans: %w", err)
	}

	offset := (page - 1) * limit
	queryArgs := append(args, limit, offset)

	rows, err := r.db.Query(`
		SELECT l.id, l.borrower_id, l.amount, l.remaining, l.interest_rate, l.penalty_rate,
			   l.purpose, l.status, l.approved_by, l.approved_at, l.next_payment, l.created_at,
			   u.name AS borrower_name
		FROM loans l
		JOIN users u ON u.id = l.borrower_id
		WHERE `+whereClause+`
		ORDER BY l.created_at DESC
		LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list all loans: %w", err)
	}
	defer rows.Close()

	loans, err := r.scanLoans(rows)
	if err != nil {
		return nil, 0, err
	}
	return loans, total, nil
}

func (r *LoanRepo) scanLoans(rows *sql.Rows) ([]*loan.Loan, error) {
	var loans []*loan.Loan
	for rows.Next() {
		l := &loan.Loan{}
		var approvedBy sql.NullInt64
		var approvedAt sql.NullTime
		var nextPayment sql.NullTime

		if err := rows.Scan(
			&l.ID, &l.BorrowerID, &l.Amount, &l.Remaining, &l.InterestRate, &l.PenaltyRate,
			&l.Purpose, &l.Status, &approvedBy, &approvedAt, &nextPayment, &l.CreatedAt,
			&l.BorrowerName,
		); err != nil {
			return nil, fmt.Errorf("scan loan: %w", err)
		}
		if approvedBy.Valid {
			ab := int(approvedBy.Int64)
			l.ApprovedBy = &ab
		}
		if approvedAt.Valid {
			l.ApprovedAt = &approvedAt.Time
		}
		if nextPayment.Valid {
			l.NextPayment = &nextPayment.Time
		}
		l.WeeklyInterest = l.CalcWeeklyInterest()
		loans = append(loans, l)
	}
	return loans, nil
}

func (r *LoanRepo) UpdateStatus(id int, status loan.LoanStatus) error {
	_, err := r.db.Exec("UPDATE loans SET status = ? WHERE id = ?", status, id)
	return err
}

func (r *LoanRepo) Approve(id, approvedBy int, interestRate, penaltyRate float64) error {
	nextPayment := time.Now().AddDate(0, 0, 7)
	_, err := r.db.Exec(`
		UPDATE loans SET status = 'active', interest_rate = ?, penalty_rate = ?,
			approved_by = ?, approved_at = CURRENT_TIMESTAMP, next_payment = ?
		WHERE id = ?`,
		interestRate, penaltyRate, approvedBy, nextPayment, id,
	)
	return err
}

func (r *LoanRepo) UpdateRemaining(id, remaining int) error {
	_, err := r.db.Exec("UPDATE loans SET remaining = ? WHERE id = ?", remaining, id)
	return err
}

func (r *LoanRepo) UpdateNextPayment(id int, next *time.Time) error {
	_, err := r.db.Exec("UPDATE loans SET next_payment = ? WHERE id = ?", next, id)
	return err
}

func (r *LoanRepo) ListActiveLoans() ([]*loan.Loan, error) {
	rows, err := r.db.Query(`
		SELECT l.id, l.borrower_id, l.amount, l.remaining, l.interest_rate, l.penalty_rate,
			   l.purpose, l.status, l.approved_by, l.approved_at, l.next_payment, l.created_at,
			   u.name AS borrower_name
		FROM loans l
		JOIN users u ON u.id = l.borrower_id
		WHERE l.status IN ('active', 'overdue')
		ORDER BY l.next_payment ASC`)
	if err != nil {
		return nil, fmt.Errorf("list active loans: %w", err)
	}
	defer rows.Close()
	return r.scanLoans(rows)
}

// --- Payments ---

func (r *LoanRepo) CreatePayment(p *loan.LoanPayment) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO loan_payments (loan_id, amount, principal, interest, penalty, pay_type)
		VALUES (?, ?, ?, ?, ?, ?)`,
		p.LoanID, p.Amount, p.Principal, p.Interest, p.Penalty, p.PayType,
	)
	if err != nil {
		return 0, fmt.Errorf("create loan payment: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *LoanRepo) ListPayments(loanID int) ([]*loan.LoanPayment, error) {
	rows, err := r.db.Query(`
		SELECT id, loan_id, amount, principal, interest, penalty, pay_type, created_at
		FROM loan_payments
		WHERE loan_id = ?
		ORDER BY created_at DESC`, loanID)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()

	var payments []*loan.LoanPayment
	for rows.Next() {
		p := &loan.LoanPayment{}
		if err := rows.Scan(
			&p.ID, &p.LoanID, &p.Amount, &p.Principal, &p.Interest, &p.Penalty,
			&p.PayType, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, nil
}
