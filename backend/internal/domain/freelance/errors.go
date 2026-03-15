package freelance

import "errors"

var (
	ErrJobNotFound        = errors.New("외주 작업을 찾을 수 없습니다")
	ErrNotClient          = errors.New("의뢰인만 수행할 수 있습니다")
	ErrNotFreelancer      = errors.New("프리랜서만 수행할 수 있습니다")
	ErrCannotApplyOwnJob  = errors.New("자신의 작업에 지원할 수 없습니다")
	ErrAlreadyApplied     = errors.New("이미 지원한 작업입니다")
	ErrJobNotOpen         = errors.New("모집 중인 작업이 아닙니다")
	ErrJobNotInProgress   = errors.New("진행 중인 작업이 아닙니다")
	ErrWorkNotCompleted   = errors.New("작업이 아직 완료되지 않았습니다")
	ErrJobNotCompleted    = errors.New("완료된 작업이 아닙니다")
	ErrAlreadyReviewed    = errors.New("이미 리뷰를 작성했습니다")
	ErrInvalidRating      = errors.New("평점은 1~5 사이여야 합니다")
	ErrNotParticipant     = errors.New("작업 참여자만 수행할 수 있습니다")
	ErrApplicationNotFound = errors.New("지원을 찾을 수 없습니다")
	ErrInsufficientFunds  = errors.New("잔액이 부족합니다")
	ErrMaxWorkersReached  = errors.New("최대 작업자 수에 도달했습니다")
	ErrFixedPriceMismatch = errors.New("고정 금액 의뢰는 예산과 동일한 금액으로만 지원할 수 있습니다")
	ErrInvalidPriceType   = errors.New("price_type은 'fixed' 또는 'negotiable'이어야 합니다")
)
