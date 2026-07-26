package company

import "time"

// #181: 회사가 등록한 개별 서비스.
// 레거시 companies.service_url (쉼표 구분 다중 URL, #115) 과 별개의 테이블이며,
// 서비스마다 정규화된 HTTPS URL + 도달성 검증 상태 + Rybbit(웹 애널리틱스) 연동 상태를 가진다.
type CompanyService struct {
	ID        int    `json:"id"`
	CompanyID int    `json:"company_id"`
	Name      string `json:"name"`
	// URL 은 학생이 입력한 원문, NormalizedURL 은 중복 판정 / 외부 호출용 정규 형태.
	URL           string `json:"url"`
	NormalizedURL string `json:"normalized_url"`

	ValidationStatus    string     `json:"validation_status"`
	ValidationCheckedAt *time.Time `json:"validation_checked_at"`
	// ValidationDetail 은 실패 사유 머신 토큰 (예: private_ip, http_status_404).
	ValidationDetail string `json:"validation_detail"`

	RybbitStatus      string     `json:"rybbit_status"`
	RybbitSiteID      string     `json:"rybbit_site_id"`
	RybbitConnectedAt *time.Time `json:"rybbit_connected_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// #181: 검증 상태 — 프론트엔드와 동결된 문자열 계약.
const (
	ValidationUnvalidated = "unvalidated"
	ValidationValid       = "valid"
	ValidationInvalid     = "invalid"
)

// #181: Rybbit 연동 상태. URL 이 바뀌면 connected → needs_reconnect (드리프트).
const (
	RybbitNotConnected   = "not_connected"
	RybbitConnected      = "connected"
	RybbitNeedsReconnect = "needs_reconnect"
)

// MaxServiceNameLength — 서비스 표시 이름 최대 길이 (빈 이름은 허용).
const MaxServiceNameLength = 100
