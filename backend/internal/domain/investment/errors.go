package investment

import "errors"

var (
	ErrRoundNotFound       = errors.New("투자 라운드를 찾을 수 없습니다")
	ErrRoundNotOpen        = errors.New("모집 중인 라운드가 아닙니다")
	ErrOpenRoundExists     = errors.New("이미 진행 중인 라운드가 있습니다")
	ErrNotOwner            = errors.New("회사 소유자만 수행할 수 있습니다")
	ErrCannotInvestOwnCompany = errors.New("자신의 회사에 투자할 수 없습니다")
	ErrInsufficientFunds   = errors.New("잔액이 부족합니다")
	ErrCompanyNotFound     = errors.New("회사를 찾을 수 없습니다")
	ErrInvalidPercent      = errors.New("제안 지분율이 유효하지 않습니다")
	ErrInvalidAmount       = errors.New("유효하지 않은 금액입니다")
	ErrCompanyWalletNotFound = errors.New("회사 지갑을 찾을 수 없습니다")
)
