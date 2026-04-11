package company

import "errors"

var (
	ErrNotFound           = errors.New("회사를 찾을 수 없습니다")
	ErrNotOwner           = errors.New("회사 소유자만 수행할 수 있습니다")
	ErrInsufficientFunds  = errors.New("잔액이 부족합니다")
	ErrMinCapital         = errors.New("초기 자본금은 최소 1,000,000원 이상이어야 합니다")
	ErrDuplicateName      = errors.New("이미 사용 중인 회사명입니다")
	ErrAlreadyShareholder = errors.New("이미 주주입니다")
	ErrWalletNotFound     = errors.New("회사 지갑을 찾을 수 없습니다")
	ErrDisclosureNotFound = errors.New("공시를 찾을 수 없습니다")
)
