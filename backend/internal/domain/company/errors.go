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
	// #159 멀티 강의실
	ErrNoClassroom      = errors.New("강의실에 소속되어야 수행할 수 있습니다")
	ErrWrongClassroom   = errors.New("다른 강의실의 항목입니다")
	ErrProposalNotFound = errors.New("안건을 찾을 수 없습니다")
	ErrProposalClosed   = errors.New("이미 종료된 안건입니다")
	ErrNotShareholder   = errors.New("주주만 투표할 수 있습니다")
	ErrAlreadyVoted     = errors.New("이미 투표했습니다")
	ErrInvalidProposal  = errors.New("안건 정보가 올바르지 않습니다")
	// #181 회사 등록 서비스 (URL 검증 + Rybbit 연동)
	ErrServiceNotFound     = errors.New("서비스를 찾을 수 없습니다")
	ErrDuplicateServiceURL = errors.New("이미 등록된 서비스 주소입니다")
	ErrInvalidServiceURL   = errors.New("서비스 주소가 올바르지 않습니다")
	ErrServiceNameTooLong  = errors.New("서비스 이름은 100자 이하여야 합니다")
	ErrValidationRequired  = errors.New("먼저 서비스 주소를 검증해야 합니다")
	ErrValidationStale     = errors.New("검증한 지 24시간이 지났습니다. 다시 검증해 주세요")
	ErrRybbitNotConfigured = errors.New("Rybbit 연동이 설정되지 않았습니다")
	ErrRybbitFailed        = errors.New("Rybbit 연동에 실패했습니다")
	// Rybbit 연동은 캠프(학생) 전용 — admin 은 자격이 없다 (#181, 명시 정책).
	ErrNotStudent  = errors.New("학생 계정만 Rybbit 연동을 할 수 있습니다")
	ErrNotApproved = errors.New("승인된 계정만 Rybbit 연동을 할 수 있습니다")
)

// InvalidServiceURLError 는 URL 정규화 실패를 사유 토큰과 함께 나른다 (#181).
// Reason 은 urlcheck 사유 토큰 (bad_syntax / not_https / has_credentials / ...),
// Message 는 학생에게 그대로 보여줄 한국어 설명.
// errors.Is(err, ErrInvalidServiceURL) 로 잡을 수 있다.
type InvalidServiceURLError struct {
	Reason  string
	Message string
}

func (e *InvalidServiceURLError) Error() string { return e.Message }

func (e *InvalidServiceURLError) Unwrap() error { return ErrInvalidServiceURL }
