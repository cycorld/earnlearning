package user

import "errors"

var (
	ErrNotFound       = errors.New("사용자를 찾을 수 없습니다")
	ErrDuplicateEmail = errors.New("이미 사용 중인 이메일입니다")
	ErrInvalidCreds   = errors.New("이메일 또는 비밀번호가 올바르지 않습니다")
	ErrRejected       = errors.New("가입이 거절된 계정입니다")
	ErrWeakPassword   = errors.New("비밀번호는 최소 8자 이상이어야 합니다")
	ErrInvalidStudent = errors.New("학번은 7~10자리 숫자여야 합니다")
)
