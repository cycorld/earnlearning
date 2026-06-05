package milestone

import "errors"

var (
	ErrInvalidType   = errors.New("유효하지 않은 평가지표 타입입니다")
	ErrInvalidStatus = errors.New("유효하지 않은 상태입니다")
	ErrNotFound      = errors.New("평가지표를 찾을 수 없습니다")
	ErrNotOwner      = errors.New("본인의 제출만 수정할 수 있습니다")
	ErrAlreadyApproved = errors.New("이미 승인된 제출은 수정할 수 없습니다")
)
