package wallet

import "errors"

var (
	ErrNotFound          = errors.New("지갑을 찾을 수 없습니다")
	ErrInsufficientFunds = errors.New("잔액이 부족합니다")
	ErrInvalidAmount     = errors.New("유효하지 않은 금액입니다")
)
