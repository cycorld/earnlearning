package grant

import "errors"

var (
	ErrGrantNotFound       = errors.New("정부과제를 찾을 수 없습니다")
	ErrGrantNotOpen        = errors.New("모집 중인 과제가 아닙니다")
	ErrAlreadyApplied      = errors.New("이미 지원한 과제입니다")
	ErrCannotApplyOwnGrant = errors.New("자신이 등록한 과제에 지원할 수 없습니다")
	ErrApplicationNotFound = errors.New("지원을 찾을 수 없습니다")
	ErrNotAdmin            = errors.New("관리자만 수행할 수 있습니다")
	ErrNotApproved         = errors.New("승인 대기 중인 지원이 아닙니다")
)
