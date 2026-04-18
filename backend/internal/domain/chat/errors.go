package chat

import "errors"

var (
	ErrSessionNotFound = errors.New("chat session not found")
	ErrSkillNotFound   = errors.New("chat skill not found")
	ErrForbidden       = errors.New("not authorized for this chat operation")
	ErrSkillDisabled   = errors.New("chat skill is disabled")
	ErrAdminOnly       = errors.New("chat skill is admin-only")
	ErrInvalidSlug     = errors.New("invalid skill slug")
)
