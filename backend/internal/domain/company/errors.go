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
	ErrProposalNotFound   = errors.New("안건을 찾을 수 없습니다")
	ErrProposalClosed     = errors.New("이미 종료된 안건입니다")
	ErrNotShareholder     = errors.New("주주만 투표할 수 있습니다")
	ErrAlreadyVoted       = errors.New("이미 투표했습니다")
	ErrInvalidProposal    = errors.New("안건 정보가 올바르지 않습니다")
)
