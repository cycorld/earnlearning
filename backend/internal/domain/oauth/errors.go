package oauth

import "errors"

var (
	ErrClientNotFound     = errors.New("OAuth 클라이언트를 찾을 수 없습니다")
	ErrInvalidRedirectURI = errors.New("유효하지 않은 redirect_uri입니다")
	ErrInvalidScope       = errors.New("유효하지 않은 스코프입니다")
	ErrInvalidSecret      = errors.New("client_secret이 올바르지 않습니다")
	ErrCodeNotFound       = errors.New("인가 코드를 찾을 수 없습니다")
	ErrCodeExpired        = errors.New("인가 코드가 만료되었습니다")
	ErrCodeUsed           = errors.New("이미 사용된 인가 코드입니다")
	ErrTokenNotFound      = errors.New("토큰을 찾을 수 없습니다")
	ErrTokenExpired       = errors.New("토큰이 만료되었습니다")
	ErrTokenRevoked       = errors.New("폐기된 토큰입니다")
	ErrInsufficientScope  = errors.New("권한이 부족합니다")
	ErrInvalidGrant       = errors.New("유효하지 않은 grant입니다")
	ErrInvalidPKCE        = errors.New("PKCE 검증에 실패했습니다")
	ErrNotOwner           = errors.New("클라이언트 소유자가 아닙니다")
)
