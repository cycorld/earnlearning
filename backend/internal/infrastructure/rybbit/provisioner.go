// Package rybbit 은 자체 호스팅 Rybbit(웹 애널리틱스)의 EarnLearning 전용
// 프로비저닝 API 클라이언트다 (#181).
//
// 계약 문서: Rybbit 저장소 EARNLEARNING_SSO.md (커밋 6cc9ee30 기준).
//
//	POST {base}/api/earnlearning/provision
//	Content-Type: application/json
//	x-earnlearning-timestamp: <unix 초>
//	x-earnlearning-signature: hex( HMAC-SHA256( secret, "<ts>" + "." + <raw_body> ) )
//
// 계약 요점:
//   - 서명은 "전송한 바이트 그대로" 를 서명한다. 재직렬화 금지 → 본문은 한 번만
//     marshal 하고 그 바이트를 서명·전송에 함께 쓴다.
//   - Origin 헤더를 보내지 않는다 (Rybbit 은 신뢰하지 않는 Origin 의 unsafe 메서드를
//     라우트 진입 전에 403 으로 거른다; Origin 없음 = 정상 서버간 호출).
//   - 요청 스키마는 strict — 모르는 필드는 400. user.role 은 "member" 만 허용.
//   - grants.siteKeys 는 누적이 아니라 재조정(reconcile) 이다. 부분 집합을 보내면
//     빠진 키의 기존 접근이 회수되므로, 호출자는 항상 "회사의 전체 유효 집합" 을 보낸다.
//   - 사이트 키는 service:<id> 로 고정 (Rybbit earnlearning_links 에 영구 매핑됨),
//     user.id 는 OAuth userinfo 의 sub 와 동일해야 한다 → user.OAuthSubject 사용.
//   - site.name / site.domain 은 Rybbit 이 "사이트 생성 시에만" 반영한다. 같은 키로
//     재호출하면 기존 사이트를 재사용하므로, URL 드리프트 후 재연동해도 Rybbit 쪽
//     도메인 표시는 그대로다 (grants 재조정은 정상 동작).
//
// 보안: 시크릿/요청 본문(이메일·이름 포함)/응답 본문은 절대 로그·에러에 넣지 않는다.
// 실패 시 상태 코드와 고정 힌트 문구만 노출한다.
package rybbit

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/earnlearning/backend/internal/domain/user"
)

// ErrNotConfigured — 프로비저너가 설정되지 않음 (핸들러가 503 으로 매핑).
// 설정된 프로비저너의 일시적 장애는 절대 이 에러로 보고하지 않는다 (그건 502).
var ErrNotConfigured = errors.New("rybbit provisioner not configured")

// MinProvisionSecretLength — 공유 HMAC 시크릿 최소 길이 (Rybbit 쪽 요구와 동일).
const MinProvisionSecretLength = 32

// maxGrantKeys — Rybbit 요청 스키마의 grants.siteKeys 상한.
// 초과분을 조용히 잘라 보내면 잘린 키의 접근이 회수되므로, 초과는 에러다.
const maxGrantKeys = 100

// maxNameLength — company.name / site.name / user.name 의 계약 상한 (rune 기준).
const maxNameLength = 256

// SiteKey 는 회사 서비스의 안정 Rybbit 사이트 키다.
// company_services.id 는 AUTOINCREMENT 라 재사용되지 않으며, 이 키는 Rybbit
// earnlearning_links 에 영구 저장되므로 형식을 절대 바꾸지 않는다.
func SiteKey(serviceID int) string { return "service:" + strconv.Itoa(serviceID) }

// companyExternalID 는 Rybbit organization 에 1:1 매핑되는 회사 외부 식별자다.
// companies.id 역시 재사용되지 않는 정수 PK 이므로 십진 문자열로 고정한다.
func companyExternalID(companyID int) string { return strconv.Itoa(companyID) }

// ProvisionRequest — 한 번의 연동 클릭이 Rybbit 에 보낼 도메인 사실들.
type ProvisionRequest struct {
	CompanyID   int
	CompanyName string
	ServiceID   int    // 클릭한 서비스 (site.key = service:<id>)
	ServiceName string // 비면 CompanyName → Domain 순으로 대체
	Domain      string // 검증된 서비스 도메인 (urlcheck 정규화 결과의 호스트)
	OwnerUserID int    // 회사 소유자 — user.id 는 user.OAuthSubject(OwnerUserID)
	OwnerEmail  string
	OwnerName   string
	// GrantServiceIDs — 소유자가 접근을 유지해야 할 이 회사의 전체 서비스 ID 집합.
	// Rybbit 은 이 집합으로 "재조정" 하므로 여기서 빠진 키의 접근은 회수된다.
	// 클릭한 ServiceID 가 빠져 있어도 이 계층에서 반드시 합집합에 넣는다.
	GrantServiceIDs []int
}

// ProvisionResult — Rybbit 200 응답의 검증된 결과.
type ProvisionResult struct {
	SiteID         string // 응답 siteId (숫자) 의 십진 문자열
	OrganizationID string
	GrantedSiteIDs []int64 // 재조정 후 이 조직에서 보유한 전체 사이트 id (오름차순 계약)
}

// Provisioner 는 서비스 사이트 생성 + 소유자 접근 재조정을 한 번에 수행한다.
// 같은 요청을 다시 보내도 같은 결과가 나온다 (Rybbit 쪽 멱등 보장).
type Provisioner interface {
	Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error)
}

// New 는 설정에 따라 실제 HTTP 프로비저너 또는 Noop 을 반환한다.
//   - 둘 다 비면: 의도적 미설정 → (Noop, nil)
//   - 하나만 설정되거나 시크릿이 32자 미만이면: 잘못된 설정 → (Noop, 설명 에러)
//     — fail-closed: 약한 시크릿으로는 절대 호출하지 않는다. 에러 문자열에
//     시크릿 값은 포함하지 않는다.
func New(baseURL, secret string) (Provisioner, error) {
	base := strings.TrimSpace(baseURL)
	sec := strings.TrimSpace(secret)
	if base == "" && sec == "" {
		return NewNoop(), nil
	}
	if base == "" || sec == "" {
		return NewNoop(), errors.New("rybbit: RYBBIT_API_BASE_URL 과 RYBBIT_PROVISION_SECRET 는 함께 설정해야 합니다")
	}
	if len(sec) < MinProvisionSecretLength {
		return NewNoop(), fmt.Errorf("rybbit: RYBBIT_PROVISION_SECRET 는 최소 %d자 이상이어야 합니다", MinProvisionSecretLength)
	}
	return &httpProvisioner{
		baseURL: strings.TrimRight(base, "/"),
		secret:  sec,
		http:    &http.Client{Timeout: 10 * time.Second},
		now:     time.Now,
	}, nil
}

// --- NoopProvisioner (미설정 환경) ---

// NoopProvisioner 는 외부 호출 없이 항상 ErrNotConfigured 를 반환한다.
type NoopProvisioner struct{}

func NewNoop() *NoopProvisioner { return &NoopProvisioner{} }

func (n *NoopProvisioner) Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error) {
	return nil, ErrNotConfigured
}

// --- 페이로드 (strict 요청 스키마와 필드·순서 일치) ---

type payloadCompany struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type payloadSite struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type payloadUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
	Role  string `json:"role"`
}

type payloadGrants struct {
	SiteKeys []string `json:"siteKeys"`
}

type provisionPayload struct {
	Company payloadCompany `json:"company"`
	Site    payloadSite    `json:"site"`
	User    payloadUser    `json:"user"`
	Grants  payloadGrants  `json:"grants"`
}

// truncateRunes 는 계약 상한(256자)을 넘는 표시 이름을 rune 안전하게 자른다.
func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}

// buildProvisionPayload 는 요청을 검증하고 "서명·전송에 함께 쓸" 바이트를 만든다.
// 반환 바이트가 곧 전송 본문이다 — 다시 marshal 하면 서명이 깨질 수 있다.
func buildProvisionPayload(req ProvisionRequest) ([]byte, int, error) {
	if req.CompanyID <= 0 || req.ServiceID <= 0 || req.OwnerUserID <= 0 {
		return nil, 0, errors.New("rybbit: 회사/서비스/소유자 식별자가 올바르지 않습니다")
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	if domain == "" {
		return nil, 0, errors.New("rybbit: empty domain")
	}
	companyName := truncateRunes(strings.TrimSpace(req.CompanyName), maxNameLength)
	if companyName == "" {
		return nil, 0, errors.New("rybbit: 회사 이름이 비어 있습니다")
	}
	email := strings.ToLower(strings.TrimSpace(req.OwnerEmail))
	if email == "" {
		return nil, 0, errors.New("rybbit: 소유자 이메일이 비어 있습니다")
	}

	siteName := truncateRunes(strings.TrimSpace(req.ServiceName), maxNameLength)
	if siteName == "" {
		siteName = companyName
	}
	if siteName == "" {
		siteName = domain
	}

	// 클릭한 서비스를 반드시 포함한 합집합 → 중복 제거 → id 오름차순 (결정적 본문).
	seen := map[int]bool{}
	ids := make([]int, 0, len(req.GrantServiceIDs)+1)
	for _, id := range append(append([]int{}, req.GrantServiceIDs...), req.ServiceID) {
		if id <= 0 {
			return nil, 0, errors.New("rybbit: 잘못된 grant 서비스 id 가 있습니다")
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	if len(ids) > maxGrantKeys {
		// 잘라 보내면 잘린 키의 접근이 회수된다 — 조용한 축소 금지.
		return nil, 0, fmt.Errorf("rybbit: grants.siteKeys 가 %d개를 초과합니다", maxGrantKeys)
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = SiteKey(id)
	}

	body, err := json.Marshal(provisionPayload{
		Company: payloadCompany{ID: companyExternalID(req.CompanyID), Name: companyName},
		Site:    payloadSite{Key: SiteKey(req.ServiceID), Name: siteName, Domain: domain},
		User: payloadUser{
			ID:    user.OAuthSubject(req.OwnerUserID),
			Email: email,
			Name:  truncateRunes(strings.TrimSpace(req.OwnerName), maxNameLength),
			Role:  "member",
		},
		Grants: payloadGrants{SiteKeys: keys},
	})
	if err != nil {
		return nil, 0, fmt.Errorf("rybbit: encode request: %w", err)
	}
	return body, len(keys), nil
}

// signProvision — hex( HMAC-SHA256( secret, "<ts>" + "." + body ) ), 소문자 64자.
func signProvision(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// --- httpProvisioner ---

type httpProvisioner struct {
	baseURL string
	secret  string
	http    *http.Client
	now     func() time.Time
}

func (p *httpProvisioner) Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error) {
	body, wantGrants, err := buildProvisionPayload(req)
	if err != nil {
		return nil, err
	}

	timestamp := strconv.FormatInt(p.now().Unix(), 10)
	signature := signProvision(p.secret, timestamp, body)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/api/earnlearning/provision", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("rybbit: build request: %w", err)
	}
	// Origin 은 절대 설정하지 않는다 (Rybbit 의 Origin 검사 훅이 403 으로 거른다).
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("x-earnlearning-timestamp", timestamp)
	httpReq.Header.Set("x-earnlearning-signature", signature)

	resp, err := p.http.Do(httpReq)
	if err != nil {
		// url.Error 는 요청 URL 을 포함하지만 시크릿은 헤더에만 있으므로 안전.
		return nil, fmt.Errorf("rybbit: provision request failed: %w", scrubURLError(err))
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	if resp.StatusCode == http.StatusOK {
		return parseProvisionResponse(raw, wantGrants)
	}
	// 응답 본문은 절대 노출하지 않는다 (상태 코드 + 고정 힌트만).
	return nil, provisionStatusError(resp.StatusCode)
}

// provisionStatusError — 상태 코드별 고정 진단 힌트. 본문/시크릿은 포함하지 않는다.
func provisionStatusError(status int) error {
	hint := ""
	switch status {
	case http.StatusBadRequest:
		hint = " (요청이 계약과 다릅니다 — EarnLearning 쪽 버그 가능성)"
	case http.StatusUnauthorized:
		hint = " (서명 거부 — 공유 시크릿과 서버 시계를 확인하세요)"
	case http.StatusForbidden:
		hint = " (사이트 키가 이 회사 소속이 아니라고 거부됨)"
	case http.StatusNotFound:
		hint = " (Rybbit 의 EARNLEARNING_SSO 가 꺼져 있습니다)"
	case http.StatusConflict:
		hint = " (이메일 또는 사이트 키 충돌 — 운영자 확인 필요)"
	}
	return fmt.Errorf("rybbit: provision failed with status %d%s", status, hint)
}

// provisionResponseDTO — 200 응답 계약. strict: 모르는 필드가 오면 계약 위반으로 본다.
type provisionResponseDTO struct {
	OrganizationID string  `json:"organizationId"`
	SiteID         int64   `json:"siteId"`
	UserID         string  `json:"userId"`
	MemberID       string  `json:"memberId"`
	GrantedSiteIDs []int64 `json:"grantedSiteIds"`
	Created        *struct {
		Organization bool `json:"organization"`
		Site         bool `json:"site"`
		User         bool `json:"user"`
		Member       bool `json:"member"`
	} `json:"created"`
}

// parseProvisionResponse 는 200 본문을 strict 하게 검증한다.
// 특히 "요청한 grant 개수 == 응답 grantedSiteIds 개수" 와 "본 요청의 siteId 포함" 을
// 확인한다 — 어느 하나라도 어긋나면 재조정 결과를 신뢰할 수 없으므로 실패 처리한다
// (usecase 는 실패 시 로컬 상태를 바꾸지 않는다).
func parseProvisionResponse(raw []byte, wantGrants int) (*ProvisionResult, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var dto provisionResponseDTO
	if err := dec.Decode(&dto); err != nil {
		return nil, errors.New("rybbit: unexpected provision response shape")
	}
	if dec.More() {
		return nil, errors.New("rybbit: unexpected provision response shape")
	}
	if dto.OrganizationID == "" || dto.UserID == "" || dto.MemberID == "" || dto.Created == nil {
		return nil, errors.New("rybbit: provision response missing required fields")
	}
	if dto.SiteID <= 0 {
		return nil, errors.New("rybbit: provision response has no site id")
	}
	if len(dto.GrantedSiteIDs) != wantGrants {
		return nil, fmt.Errorf("rybbit: granted site count %d, want %d", len(dto.GrantedSiteIDs), wantGrants)
	}
	found := false
	for _, id := range dto.GrantedSiteIDs {
		if id <= 0 {
			return nil, errors.New("rybbit: provision response has invalid granted site id")
		}
		if id == dto.SiteID {
			found = true
		}
	}
	if !found {
		return nil, errors.New("rybbit: provisioned site missing from grantedSiteIds")
	}
	return &ProvisionResult{
		SiteID:         strconv.FormatInt(dto.SiteID, 10),
		OrganizationID: dto.OrganizationID,
		GrantedSiteIDs: dto.GrantedSiteIDs,
	}, nil
}

// scrubURLError 는 전송 계층 에러에서 URL 을 떼어내 내부 주소 노출을 줄인다.
func scrubURLError(err error) error {
	var uerr *url.Error
	if errors.As(err, &uerr) {
		return errors.New(uerr.Op + " " + uerr.Err.Error())
	}
	return err
}
