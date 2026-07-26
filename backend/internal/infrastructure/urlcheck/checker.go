// Package urlcheck 는 학생이 등록한 서비스 URL 을 정규화하고, SSRF (Server-Side
// Request Forgery — 서버가 공격자가 지정한 내부 주소로 요청하게 만드는 공격) 에
// 안전하게 도달 가능성을 검사한다 (#181).
//
// 안전 규칙 (fail-closed — 확실히 안전할 때만 통과):
//   - https 만 허용. 자격증명(user:pw@) / 프래그먼트(#) 포함 URL 거부.
//   - DNS 해석 결과 IP 가 하나라도 사설/특수 대역이면 차단. 해석 실패도 차단.
//   - 실제 dial 시점에 같은 리졸버로 재해석 후 검증된 IP 로 직접 연결한다
//     (DNS 리바인딩 방어 — 사전 검사 후 DNS 응답을 사설 IP 로 바꿔치기하는 공격).
//   - 리다이렉트는 횟수 제한 + 매 홉마다 https / 정규화 규칙 / IP 검증을 다시 적용.
package urlcheck

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MaxURLLength — 등록 가능한 URL 최대 길이.
const MaxURLLength = 2048

// 정규화 실패 사유 토큰. 프론트엔드가 한국어 메시지로 매핑하는 고정 계약.
const (
	ReasonBadSyntax      = "bad_syntax"
	ReasonNotHTTPS       = "not_https"
	ReasonHasCredentials = "has_credentials"
	ReasonHasFragment    = "has_fragment"
	ReasonEmptyHost      = "empty_host"
	ReasonTooLong        = "too_long"
)

// 네트워크 검사 실패 사유 토큰.
const (
	ReasonPrivateIP        = "private_ip"
	ReasonDNSError         = "dns_error"
	ReasonTimeout          = "timeout"
	ReasonConnectError     = "connect_error"
	ReasonTooManyRedirects = "too_many_redirects"
	ReasonRedirectNotHTTPS = "redirect_not_https"
	ReasonRedirectBlocked  = "redirect_blocked"
)

// Error 는 실패 사유를 기계 판독 토큰으로 나르는 에러.
type Error struct{ Reason string }

func (e *Error) Error() string { return e.Reason }

// 사유별 센티널. errors.Is / errors.As 둘 다 쓸 수 있다.
var (
	ErrBadSyntax        = &Error{Reason: ReasonBadSyntax}
	ErrNotHTTPS         = &Error{Reason: ReasonNotHTTPS}
	ErrHasCredentials   = &Error{Reason: ReasonHasCredentials}
	ErrHasFragment      = &Error{Reason: ReasonHasFragment}
	ErrEmptyHost        = &Error{Reason: ReasonEmptyHost}
	ErrTooLong          = &Error{Reason: ReasonTooLong}
	ErrPrivateIP        = &Error{Reason: ReasonPrivateIP}
	ErrDNSError         = &Error{Reason: ReasonDNSError}
	ErrTooManyRedirects = &Error{Reason: ReasonTooManyRedirects}
	ErrRedirectNotHTTPS = &Error{Reason: ReasonRedirectNotHTTPS}
	ErrRedirectBlocked  = &Error{Reason: ReasonRedirectBlocked}
)

// ReasonOf 는 에러에서 사유 토큰을 뽑는다. 알 수 없는 에러는 bad_syntax 로 본다.
func ReasonOf(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Reason
	}
	return ReasonBadSyntax
}

// Normalize 는 URL 을 검증하고 중복 판정용 정규 형태로 바꾼다.
// 반환: (정규화 URL, 호스트(포트 제외, 소문자), 에러).
func Normalize(raw string) (string, string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", "", ErrBadSyntax
	}
	if len(s) > MaxURLLength {
		return "", "", ErrTooLong
	}

	u, err := url.Parse(s)
	if err != nil {
		return "", "", ErrBadSyntax
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return "", "", ErrNotHTTPS
	}
	if u.User != nil {
		return "", "", ErrHasCredentials
	}
	// 원문에 '#' 이 있으면 (빈 프래그먼트 포함) 거부 — 프래그먼트는 서버가 볼 수 없다.
	if u.Fragment != "" || strings.Contains(s, "#") {
		return "", "", ErrHasFragment
	}

	host := strings.ToLower(u.Hostname())
	if host == "" {
		return "", "", ErrEmptyHost
	}

	// 기본 포트(:443) 는 제거해 중복 URL 을 같은 키로 모은다.
	port := u.Port()
	hostport := host
	if strings.Contains(host, ":") { // IPv6 리터럴은 대괄호 복원
		hostport = "[" + host + "]"
	}
	if port != "" && port != "443" {
		hostport += ":" + port
	}

	// 끝 슬래시 제거 ("/a/" 와 "/a" 는 같은 서비스로 본다).
	path := u.EscapedPath()
	if path == "/" {
		path = ""
	} else {
		path = strings.TrimSuffix(path, "/")
	}

	normalized := "https://" + hostport + path
	if u.RawQuery != "" {
		normalized += "?" + u.RawQuery
	}
	return normalized, host, nil
}

// Result 는 도달성 검사 결과. Detail 은 실패 사유 토큰 (성공 시 빈 문자열).
type Result struct {
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// Checker 는 SSRF 방어를 내장한 URL 도달성 검사기.
// 필드는 모두 테스트에서 교체 가능하지만, 기본값은 프로덕션 안전값이다.
type Checker struct {
	// Timeout 은 검사 1회 전체 예산 (기본 5s).
	Timeout time.Duration
	// MaxRedirects 는 허용 리다이렉트 횟수 (기본 3).
	MaxRedirects int
	// LookupIP 는 호스트 이름 해석기 (기본 net.DefaultResolver).
	LookupIP func(ctx context.Context, host string) ([]net.IP, error)
	// AllowIP 는 연결 허용 IP 판정기 (기본 PublicIPOnly).
	AllowIP func(net.IP) bool
	// TLSClientConfig 는 테스트가 httptest 자체 서명 인증서를 신뢰시키기 위한 훅.
	// nil 이면 Go 기본 검증 (프로덕션은 반드시 nil — InsecureSkipVerify 금지).
	TLSClientConfig *tls.Config
}

// NewChecker 는 프로덕션 기본값 체커를 만든다.
func NewChecker() *Checker {
	return &Checker{
		Timeout:      5 * time.Second,
		MaxRedirects: 3,
		LookupIP:     defaultLookupIP,
		AllowIP:      PublicIPOnly,
	}
}

func (c *Checker) timeout() time.Duration {
	if c.Timeout <= 0 {
		return 5 * time.Second
	}
	return c.Timeout
}

func (c *Checker) maxRedirects() int {
	if c.MaxRedirects <= 0 {
		return 3
	}
	return c.MaxRedirects
}

func (c *Checker) allowIP(ip net.IP) bool {
	if c.AllowIP == nil {
		return PublicIPOnly(ip)
	}
	return c.AllowIP(ip)
}

func (c *Checker) lookupIP(ctx context.Context, host string) ([]net.IP, error) {
	if c.LookupIP == nil {
		return defaultLookupIP(ctx, host)
	}
	return c.LookupIP(ctx, host)
}

func defaultLookupIP(ctx context.Context, host string) ([]net.IP, error) {
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, a := range addrs {
		ips = append(ips, a.IP)
	}
	return ips, nil
}

// Check 는 URL 을 정규화 → IP 검증 → GET 요청 순으로 검사한다.
func (c *Checker) Check(ctx context.Context, rawURL string) Result {
	normalized, host, err := Normalize(rawURL)
	if err != nil {
		return Result{OK: false, Detail: ReasonOf(err)}
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()

	// 사전 검증 — 요청 전에 목적지 IP 를 먼저 막는다.
	if err := c.vetHost(ctx, host); err != nil {
		return Result{OK: false, Detail: ReasonOf(err)}
	}

	client := &http.Client{
		Timeout:       c.timeout(),
		CheckRedirect: c.checkRedirect,
		Transport: &http.Transport{
			Proxy:                 nil, // 프록시 우회로 내부망에 닿는 경로 차단
			DialContext:           c.dialContext,
			TLSClientConfig:       c.TLSClientConfig,
			TLSHandshakeTimeout:   c.timeout(),
			ResponseHeaderTimeout: c.timeout(),
			DisableKeepAlives:     true,
			ForceAttemptHTTP2:     false,
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, normalized, nil)
	if err != nil {
		return Result{OK: false, Detail: ReasonBadSyntax}
	}
	req.Header.Set("User-Agent", "EarnLearning-URLCheck/1.0")
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		return Result{OK: false, Detail: classifyError(err)}
	}
	defer resp.Body.Close()
	// 본문은 최대 1KB 만 읽고 버린다 (커넥션 재사용/자원 보호).
	_, _ = io.CopyN(io.Discard, resp.Body, 1024)

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return Result{OK: true}
	}
	return Result{OK: false, Detail: fmt.Sprintf("http_status_%d", resp.StatusCode)}
}

// vetHost 는 호스트의 모든 목적지 IP 를 검사한다 (하나라도 막히면 전체 차단).
func (c *Checker) vetHost(ctx context.Context, host string) error {
	if ip := net.ParseIP(host); ip != nil {
		if !c.allowIP(ip) {
			return ErrPrivateIP
		}
		return nil
	}
	ips, err := c.lookupIP(ctx, host)
	if err != nil || len(ips) == 0 {
		return ErrDNSError
	}
	for _, ip := range ips {
		if !c.allowIP(ip) {
			return ErrPrivateIP
		}
	}
	return nil
}

// dialContext 는 dial 직전에 호스트를 재해석하고 검증된 IP 로 직접 연결한다.
// 주소만 IP 로 바꾸므로 TLS SNI / 인증서 검증은 원래 호스트 이름으로 유지된다.
func (c *Checker) dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, &Error{Reason: ReasonConnectError}
	}

	var candidates []net.IP
	if ip := net.ParseIP(host); ip != nil {
		candidates = []net.IP{ip}
	} else {
		ips, err := c.lookupIP(ctx, host)
		if err != nil || len(ips) == 0 {
			return nil, ErrDNSError
		}
		candidates = ips
	}
	for _, ip := range candidates {
		if !c.allowIP(ip) {
			return nil, ErrPrivateIP
		}
	}

	d := &net.Dialer{Timeout: c.timeout()}
	var lastErr error
	for _, ip := range candidates {
		conn, err := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = &Error{Reason: ReasonConnectError}
	}
	return nil, lastErr
}

// checkRedirect 는 매 리다이렉트 홉에 https / 정규화 규칙 / 횟수 제한을 적용한다.
// 목적지 IP 는 dialContext 가 다시 검증한다.
func (c *Checker) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) > c.maxRedirects() {
		return ErrTooManyRedirects
	}
	if !strings.EqualFold(req.URL.Scheme, "https") {
		return ErrRedirectNotHTTPS
	}
	if _, _, err := Normalize(req.URL.String()); err != nil {
		if errors.Is(err, ErrNotHTTPS) {
			return ErrRedirectNotHTTPS
		}
		return ErrRedirectBlocked
	}
	return nil
}

// classifyError 는 http.Client 에러를 사유 토큰으로 바꾼다.
func classifyError(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Reason
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ReasonTimeout
	}
	var nerr net.Error
	if errors.As(err, &nerr) && nerr.Timeout() {
		return ReasonTimeout
	}
	return ReasonConnectError
}

// blockedCIDRs — IsPrivate 등 표준 판정으로 안 걸리는 특수 대역.
var blockedCIDRs = func() []*net.IPNet {
	raw := []string{
		"100.64.0.0/10",      // CGNAT (RFC6598)
		"192.0.0.0/24",       // IETF 프로토콜 할당
		"192.0.2.0/24",       // 문서용 TEST-NET-1
		"198.51.100.0/24",    // 문서용 TEST-NET-2
		"203.0.113.0/24",     // 문서용 TEST-NET-3
		"198.18.0.0/15",      // 벤치마킹
		"240.0.0.0/4",        // 예약 (클래스 E)
		"255.255.255.255/32", // 브로드캐스트
	}
	nets := make([]*net.IPNet, 0, len(raw))
	for _, cidr := range raw {
		if _, n, err := net.ParseCIDR(cidr); err == nil {
			nets = append(nets, n)
		}
	}
	return nets
}()

// PublicIPOnly 는 공인 IP 만 허용한다 (내부망 / 메타데이터 / 특수 대역 차단).
func PublicIPOnly(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4 // ::ffff:10.0.0.1 같은 4-in-6 매핑을 IPv4 규칙으로 판정
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() {
		return false
	}
	for _, n := range blockedCIDRs {
		if n.Contains(ip) {
			return false
		}
	}
	return true
}
