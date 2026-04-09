package userdb

import "errors"

var (
	ErrNotFound         = errors.New("데이터베이스를 찾을 수 없습니다")
	ErrForbidden        = errors.New("권한이 없습니다")
	ErrInvalidName      = errors.New("이름 형식이 올바르지 않습니다 (소문자/숫자/밑줄, 3~32자)")
	ErrNameTooLong      = errors.New("생성될 DB 이름이 63자를 초과합니다")
	ErrDuplicate        = errors.New("같은 이름의 프로젝트 DB가 이미 있습니다")
	ErrQuotaExceeded    = errors.New("DB 생성 가능 한도를 초과했습니다")
	ErrProvisionFailed  = errors.New("PG 서버에 DB 생성을 요청하지 못했습니다")
	ErrProvisionerDown  = errors.New("DB 프로비저너가 설정되지 않았습니다")
)
