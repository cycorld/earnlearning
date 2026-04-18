package llm

import "errors"

var (
	ErrKeyNotFound      = errors.New("llm api key not found")
	ErrAlreadyHasKey    = errors.New("user already has an active key")
	ErrProxyUnavailable = errors.New("llm proxy unavailable")
	ErrNoEmail          = errors.New("user has no email — cannot provision llm key")
)
