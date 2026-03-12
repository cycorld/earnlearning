package loan

import "errors"

var (
	ErrLoanNotFound      = errors.New("대출을 찾을 수 없습니다")
	ErrNotPending        = errors.New("대기 중인 대출이 아닙니다")
	ErrNotActive         = errors.New("활성 대출이 아닙니다")
	ErrInsufficientFunds = errors.New("잔액이 부족합니다")
	ErrInvalidAmount     = errors.New("유효하지 않은 금액입니다")
	ErrAlreadyPaid       = errors.New("이미 상환 완료된 대출입니다")
)
