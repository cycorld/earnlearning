package urlcheck

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// #181: URL 정규화 + SSRF 안전 검사 단위 테스트.

func TestNormalize(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantURL  string
		wantHost string
		wantErr  string // 실패 사유 토큰 ("" 이면 성공 기대)
	}{
		{name: "기본", raw: "https://app.example.com", wantURL: "https://app.example.com", wantHost: "app.example.com"},
		{name: "대소문자", raw: "https://App.EXAMPLE.com", wantURL: "https://app.example.com", wantHost: "app.example.com"},
		{name: "스킴 대문자", raw: "HTTPS://App.Example.com", wantURL: "https://app.example.com", wantHost: "app.example.com"},
		{name: "기본포트 443 제거", raw: "https://app.example.com:443/", wantURL: "https://app.example.com", wantHost: "app.example.com"},
		{name: "비표준 포트 유지", raw: "https://app.example.com:8443/x", wantURL: "https://app.example.com:8443/x", wantHost: "app.example.com"},
		{name: "끝 슬래시 제거", raw: "https://app.example.com/path/", wantURL: "https://app.example.com/path", wantHost: "app.example.com"},
		{name: "앞뒤 공백", raw: "  https://app.example.com/  ", wantURL: "https://app.example.com", wantHost: "app.example.com"},
		{name: "쿼리 유지", raw: "https://app.example.com/a?b=1", wantURL: "https://app.example.com/a?b=1", wantHost: "app.example.com"},
		{name: "IP 리터럴", raw: "https://127.0.0.1:8443/", wantURL: "https://127.0.0.1:8443", wantHost: "127.0.0.1"},
		{name: "http 거부", raw: "http://plain.com", wantErr: ReasonNotHTTPS},
		{name: "빈 문자열", raw: "", wantErr: ReasonBadSyntax},
		{name: "스킴 없음", raw: "not a url", wantErr: ReasonNotHTTPS},
		{name: "자격증명 포함", raw: "https://user:pw@h.com", wantErr: ReasonHasCredentials},
		{name: "프래그먼트 포함", raw: "https://h.com/#frag", wantErr: ReasonHasFragment},
		{name: "빈 프래그먼트도 거부", raw: "https://h.com/#", wantErr: ReasonHasFragment},
		{name: "호스트 없음", raw: "https:///path", wantErr: ReasonEmptyHost},
		{name: "너무 김", raw: "https://a.com/" + strings.Repeat("x", 2049), wantErr: ReasonTooLong},
		{name: "제어문자", raw: "https://exa\x7fmple.com", wantErr: ReasonBadSyntax},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotHost, err := Normalize(tt.raw)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Normalize(%q) = %q, 에러를 기대했음 (%s)", tt.raw, gotURL, tt.wantErr)
				}
				if got := ReasonOf(err); got != tt.wantErr {
					t.Fatalf("Normalize(%q) 사유 = %q, want %q", tt.raw, got, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Normalize(%q) 예상치 못한 에러: %v", tt.raw, err)
			}
			if gotURL != tt.wantURL {
				t.Errorf("Normalize(%q) = %q, want %q", tt.raw, gotURL, tt.wantURL)
			}
			if gotHost != tt.wantHost {
				t.Errorf("Normalize(%q) host = %q, want %q", tt.raw, gotHost, tt.wantHost)
			}
		})
	}
}

// 정규화는 멱등이어야 한다 (중복 검사 키로 쓰이므로).
func TestNormalize_Idempotent(t *testing.T) {
	for _, raw := range []string{
		"https://App.EXAMPLE.com:443/",
		"https://app.example.com/path/",
		"https://app.example.com/a?b=1",
	} {
		first, _, err := Normalize(raw)
		if err != nil {
			t.Fatalf("Normalize(%q): %v", raw, err)
		}
		second, _, err := Normalize(first)
		if err != nil {
			t.Fatalf("Normalize(%q): %v", first, err)
		}
		if first != second {
			t.Errorf("정규화가 멱등이 아님: %q → %q → %q", raw, first, second)
		}
	}
}

func TestPublicIPOnly(t *testing.T) {
	tests := []struct {
		ip    string
		allow bool
	}{
		{"127.0.0.1", false},
		{"10.0.0.1", false},
		{"172.16.0.1", false},
		{"192.168.1.1", false},
		{"169.254.169.254", false}, // 클라우드 메타데이터
		{"100.64.0.1", false},      // CGNAT
		{"192.0.0.1", false},
		{"192.0.2.1", false},
		{"198.51.100.1", false},
		{"203.0.113.1", false},
		{"198.18.0.1", false},
		{"240.0.0.1", false},
		{"255.255.255.255", false},
		{"0.0.0.0", false},
		{"224.0.0.1", false}, // 멀티캐스트
		{"::1", false},
		{"fe80::1", false},
		{"fc00::1", false},
		{"::", false},
		{"::ffff:10.0.0.1", false}, // 4-in-6 매핑
		{"8.8.8.8", true},
		{"1.1.1.1", true},
		{"2606:4700:4700::1111", true},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if ip == nil {
			t.Fatalf("테스트 IP 파싱 실패: %s", tt.ip)
		}
		if got := PublicIPOnly(ip); got != tt.allow {
			t.Errorf("PublicIPOnly(%s) = %v, want %v", tt.ip, got, tt.allow)
		}
	}
	if PublicIPOnly(nil) {
		t.Error("PublicIPOnly(nil) 은 false 여야 함")
	}
}

// --- 네트워크 검사 (httptest TLS 서버) ---

// allowLoopback — 테스트용: 루프백만 추가 허용, 나머지는 프로덕션과 동일 규칙.
func allowLoopback(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}
	return PublicIPOnly(ip)
}

// testChecker — httptest TLS 인증서를 신뢰하고 루프백을 허용하는 체커.
func testChecker(srv *httptest.Server, lookup func(context.Context, string) ([]net.IP, error)) *Checker {
	var tlsCfg *tls.Config
	if srv != nil {
		if tr, ok := srv.Client().Transport.(*http.Transport); ok {
			tlsCfg = tr.TLSClientConfig
		}
	}
	return &Checker{
		Timeout:         3 * time.Second,
		MaxRedirects:    3,
		AllowIP:         allowLoopback,
		LookupIP:        lookup,
		TLSClientConfig: tlsCfg,
	}
}

func TestCheck_Status(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(strings.Repeat("a", 5000))) // 1KB 초과 본문도 안전해야 함
		case "/missing":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	c := testChecker(srv, nil)

	if got := c.Check(context.Background(), srv.URL+"/ok"); !got.OK || got.Detail != "" {
		t.Errorf("200 → %+v, want OK", got)
	}
	if got := c.Check(context.Background(), srv.URL+"/missing"); got.OK || got.Detail != "http_status_404" {
		t.Errorf("404 → %+v, want http_status_404", got)
	}
	if got := c.Check(context.Background(), srv.URL+"/boom"); got.OK || got.Detail != "http_status_500" {
		t.Errorf("500 → %+v, want http_status_500", got)
	}
}

func TestCheck_Timeout(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(600 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := testChecker(srv, nil)
	c.Timeout = 120 * time.Millisecond

	if got := c.Check(context.Background(), srv.URL+"/slow"); got.OK || got.Detail != ReasonTimeout {
		t.Errorf("느린 응답 → %+v, want timeout", got)
	}
}

func TestCheck_PrivateIPFromResolver(t *testing.T) {
	var dialed int32
	lookup := func(ctx context.Context, host string) ([]net.IP, error) {
		atomic.AddInt32(&dialed, 0) // no-op, 가독성용
		return []net.IP{net.ParseIP("10.0.0.1")}, nil
	}
	c := testChecker(nil, lookup)
	c.AllowIP = PublicIPOnly

	got := c.Check(context.Background(), "https://internal.test")
	if got.OK || got.Detail != ReasonPrivateIP {
		t.Fatalf("사설 IP 해석 → %+v, want private_ip", got)
	}
}

// 여러 IP 중 하나라도 사설이면 차단 (fail-closed).
func TestCheck_MixedIPs_FailClosed(t *testing.T) {
	lookup := func(ctx context.Context, host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("8.8.8.8"), net.ParseIP("127.0.0.1")}, nil
	}
	c := testChecker(nil, lookup)
	c.AllowIP = PublicIPOnly

	if got := c.Check(context.Background(), "https://mixed.test"); got.OK || got.Detail != ReasonPrivateIP {
		t.Fatalf("혼합 IP → %+v, want private_ip", got)
	}
}

func TestCheck_DNSError(t *testing.T) {
	lookup := func(ctx context.Context, host string) ([]net.IP, error) {
		return nil, errors.New("no such host")
	}
	c := testChecker(nil, lookup)

	if got := c.Check(context.Background(), "https://nowhere.invalid"); got.OK || got.Detail != ReasonDNSError {
		t.Fatalf("DNS 실패 → %+v, want dns_error", got)
	}

	// 빈 결과도 fail-closed
	c.LookupIP = func(ctx context.Context, host string) ([]net.IP, error) { return nil, nil }
	if got := c.Check(context.Background(), "https://nowhere.invalid"); got.OK || got.Detail != ReasonDNSError {
		t.Fatalf("빈 DNS 결과 → %+v, want dns_error", got)
	}
}

func TestCheck_NormalizationRejected(t *testing.T) {
	c := testChecker(nil, nil)
	for raw, want := range map[string]string{
		"http://plain.com":  ReasonNotHTTPS,
		"https://u:p@h.com": ReasonHasCredentials,
		"https://h.com/#f":  ReasonHasFragment,
		"":                  ReasonBadSyntax,
		"https://a.com/" + strings.Repeat("x", 2049): ReasonTooLong,
	} {
		if got := c.Check(context.Background(), raw); got.OK || got.Detail != want {
			t.Errorf("Check(%.30q) → %+v, want %s", raw, got, want)
		}
	}
}

func TestCheck_Redirects(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/to-private":
			http.Redirect(w, r, "https://internal.test/x", http.StatusFound)
		case r.URL.Path == "/to-http":
			http.Redirect(w, r, "http://127.0.0.1:1/x", http.StatusFound)
		case r.URL.Path == "/to-creds":
			http.Redirect(w, r, "https://u:p@public.test/x", http.StatusFound)
		case strings.HasPrefix(r.URL.Path, "/hop"):
			// /hop0 → /hop1 → /hop2 → /hop3 → /hop4 (리다이렉트 4회)
			n := r.URL.Path[len("/hop"):]
			switch n {
			case "0":
				http.Redirect(w, r, srv.URL+"/hop1", http.StatusFound)
			case "1":
				http.Redirect(w, r, srv.URL+"/hop2", http.StatusFound)
			case "2":
				http.Redirect(w, r, srv.URL+"/hop3", http.StatusFound)
			case "3":
				http.Redirect(w, r, srv.URL+"/hop4", http.StatusFound)
			default:
				w.WriteHeader(http.StatusOK)
			}
		case r.URL.Path == "/short-hop":
			http.Redirect(w, r, srv.URL+"/hop4", http.StatusFound)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	lookup := func(ctx context.Context, host string) ([]net.IP, error) {
		switch host {
		case "internal.test":
			return []net.IP{net.ParseIP("10.0.0.1")}, nil
		case "public.test":
			return []net.IP{net.ParseIP("8.8.8.8")}, nil
		}
		return nil, errors.New("no such host")
	}
	c := testChecker(srv, lookup)

	// 사설 IP 로 가는 리다이렉트 → 차단 (private_ip 또는 redirect_blocked)
	got := c.Check(context.Background(), srv.URL+"/to-private")
	if got.OK || (got.Detail != ReasonPrivateIP && got.Detail != ReasonRedirectBlocked) {
		t.Errorf("사설 리다이렉트 → %+v, want private_ip|redirect_blocked", got)
	}

	// http 로 다운그레이드 → redirect_not_https
	if got := c.Check(context.Background(), srv.URL+"/to-http"); got.OK || got.Detail != ReasonRedirectNotHTTPS {
		t.Errorf("http 리다이렉트 → %+v, want redirect_not_https", got)
	}

	// 자격증명 포함 리다이렉트 → redirect_blocked
	if got := c.Check(context.Background(), srv.URL+"/to-creds"); got.OK || got.Detail != ReasonRedirectBlocked {
		t.Errorf("자격증명 리다이렉트 → %+v, want redirect_blocked", got)
	}

	// 4회 리다이렉트 (MaxRedirects=3) → too_many_redirects
	if got := c.Check(context.Background(), srv.URL+"/hop0"); got.OK || got.Detail != ReasonTooManyRedirects {
		t.Errorf("4회 리다이렉트 → %+v, want too_many_redirects", got)
	}

	// 1회 리다이렉트는 정상 통과
	if got := c.Check(context.Background(), srv.URL+"/short-hop"); !got.OK {
		t.Errorf("1회 리다이렉트 → %+v, want OK", got)
	}
}

// DNS 리바인딩 방어: 사전 검사는 통과하지만 dial 시점 재해석이 사설이면 차단.
func TestCheck_DNSRebinding(t *testing.T) {
	var calls int32
	lookup := func(ctx context.Context, host string) ([]net.IP, error) {
		if atomic.AddInt32(&calls, 1) == 1 {
			return []net.IP{net.ParseIP("8.8.8.8")}, nil // 사전 검사용 공인 IP
		}
		return []net.IP{net.ParseIP("192.168.0.5")}, nil // dial 시점에 사설로 전환
	}
	c := testChecker(nil, lookup)
	c.AllowIP = PublicIPOnly

	if got := c.Check(context.Background(), "https://rebind.test"); got.OK || got.Detail != ReasonPrivateIP {
		t.Fatalf("DNS 리바인딩 → %+v, want private_ip", got)
	}
}

// IP 리터럴 호스트는 리졸버 없이 AllowIP 로 직접 판단한다.
func TestCheck_IPLiteralBlocked(t *testing.T) {
	c := testChecker(nil, func(ctx context.Context, host string) ([]net.IP, error) {
		t.Fatalf("IP 리터럴은 DNS 조회를 하면 안 됨: %s", host)
		return nil, nil
	})
	c.AllowIP = PublicIPOnly

	if got := c.Check(context.Background(), "https://169.254.169.254/latest/meta-data"); got.OK || got.Detail != ReasonPrivateIP {
		t.Fatalf("메타데이터 IP → %+v, want private_ip", got)
	}
}

// 연결 실패 (닫힌 포트) → connect_error
func TestCheck_ConnectError(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := srv.Listener.Addr().String()
	srv.Close() // 즉시 닫아 포트를 죽인다

	c := testChecker(nil, nil)
	if got := c.Check(context.Background(), "https://"+addr+"/x"); got.OK || got.Detail != ReasonConnectError {
		t.Fatalf("닫힌 포트 → %+v, want connect_error", got)
	}
}

// 기본 생성자는 프로덕션 안전값 (엄격 TLS, 공인 IP만).
func TestNewChecker_Defaults(t *testing.T) {
	c := NewChecker()
	if c.Timeout != 5*time.Second {
		t.Errorf("기본 Timeout = %v, want 5s", c.Timeout)
	}
	if c.MaxRedirects != 3 {
		t.Errorf("기본 MaxRedirects = %d, want 3", c.MaxRedirects)
	}
	if c.TLSClientConfig != nil {
		t.Error("기본 TLSClientConfig 는 nil (Go 기본 검증) 이어야 함")
	}
	if c.AllowIP == nil || c.LookupIP == nil {
		t.Error("기본 AllowIP / LookupIP 가 설정되어야 함")
	}
	if c.AllowIP(net.ParseIP("127.0.0.1")) {
		t.Error("기본 AllowIP 는 루프백을 막아야 함")
	}
}
