package exchange

import "errors"

var (
	ErrCompanyNotListed     = errors.New("상장되지 않은 회사입니다")
	ErrInsufficientBalance  = errors.New("잔액이 부족합니다")
	ErrInsufficientShares   = errors.New("보유 주식이 부족합니다")
	ErrOrderNotFound        = errors.New("주문을 찾을 수 없습니다")
	ErrNotOrderOwner        = errors.New("본인의 주문만 취소할 수 있습니다")
	ErrOrderNotCancellable  = errors.New("취소할 수 없는 주문 상태입니다")
	ErrInvalidShares        = errors.New("주문 수량은 1 이상이어야 합니다")
	ErrInvalidPrice         = errors.New("주문 가격은 1 이상이어야 합니다")
	ErrSelfTrade            = errors.New("자기 자신과 거래할 수 없습니다")
	ErrCompanyNotFound      = errors.New("회사를 찾을 수 없습니다")
)
