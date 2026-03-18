---
title: "토큰 자동 갱신과 앱 버전 관리: 사용자가 모르게 부드럽게"
date: "2026-03-15"
tags: ["JWT", "인증", "토큰갱신", "버전관리", "UX"]
---

## 무엇을 했나요?

사용자가 서비스를 이용하다가 갑자기 로그인 화면으로 튕기거나, 새 기능이 반영 안 되는 문제를 해결했습니다:

- **토큰 만료 감지 → 로그인 페이지 리다이렉트**: 만료된 토큰으로 무한 재시도하는 대신 로그인 유도
- **Silent Token Refresh**: 토큰 만료 전에 자동으로 갱신하여 재로그인 불필요
- **앱 버전 체크**: 새 배포 후 페이지 이동 시 자동으로 최신 버전 로드

## 왜 필요했나요?

### JWT 토큰의 생명주기 문제

우리 서비스의 JWT 토큰은 24시간 후 만료됩니다:

```
사용자 로그인 (3월 15일 오전 10시)
    ↓
JWT 토큰 발급 (24시간 유효)
    ↓
... 하루 동안 정상 사용 ...
    ↓
3월 16일 오전 10시: 토큰 만료!
    ↓
API 호출 → 401 Unauthorized
WebSocket → 연결 거부 → 무한 재시도 (쓸모없는 요청이 서버에 계속 날아감)
```

기존에는 토큰이 만료되면:
- API 에러가 화면에 뜨거나
- WebSocket이 무한 재연결을 시도하거나
- 사용자가 직접 새로고침 + 재로그인해야 했습니다

### 배포 후 캐시 문제

SPA(Single Page Application)의 특성상:

```
1. 사용자가 앱을 열면 index.html + JS 번들을 다운로드
2. 이후 페이지 이동은 JS가 처리 (서버 요청 없음)
3. 서버에 새 버전 배포
4. 사용자의 브라우저에는 여전히 옛날 JS가 실행 중!
5. 새로고침하기 전까지 새 기능을 볼 수 없음
```

## 어떻게 만들었나요?

### 1단계: 토큰 만료 시 로그인 페이지로 이동

가장 기본적인 처리 — API 응답이 401이면 로그인 페이지로 보냅니다:

```typescript
// lib/api.ts
async function request<T>(method: string, path: string, body?: unknown) {
  const res = await fetch(url, { method, headers, body })

  if (!res.ok) {
    if (res.status === 401) {
      // 토큰이 만료되었거나 유효하지 않음
      removeToken()                    // 저장된 토큰 삭제
      window.location.href = '/login'  // 로그인 페이지로 이동
      return
    }
    // ... 다른 에러 처리
  }
}
```

WebSocket에서도 같은 처리:

```typescript
// lib/ws.ts
this.ws.onclose = () => {
  if (isTokenExpired(this.token)) {
    // 만료된 토큰으로 재연결 시도해봤자 소용없음
    removeToken()
    window.location.href = '/login'
    return
  }
  // 토큰이 유효하면 재연결 시도 (네트워크 끊김 등)
  this.scheduleReconnect()
}
```

### 2단계: Silent Token Refresh — 만료 전에 자동 갱신

사용자가 재로그인하지 않아도 토큰을 자동으로 갱신하는 방법:

```
토큰 발급 ────────── 23시간 ──────────── 만료 1시간 전 ── 만료
                                              ↑
                                        여기서 자동 갱신!
                                              ↓
                                     POST /api/auth/refresh
                                              ↓
                                      새 토큰 발급 (24시간)
```

**백엔드: `/api/auth/refresh` 엔드포인트**

```go
// 기존 토큰(만료되었어도 7일 이내면)을 받아 새 토큰 발급
func (uc *AuthUseCase) RefreshToken(tokenStr string) (*AuthResponse, error) {
    // 7일의 유예기간을 두고 토큰 검증
    token, err := jwt.ParseWithClaims(tokenStr, claims,
        keyFunc, jwt.WithLeeway(7*24*time.Hour))

    if err != nil || !token.Valid {
        return nil, ErrInvalidCreds  // 7일도 지났으면 재로그인 필요
    }

    // DB에서 최신 사용자 정보 조회 (탈퇴/차단 확인)
    user, err := uc.userRepo.FindByID(claims.UserID)

    // 새 토큰 발급
    newToken, _ := uc.generateToken(user)
    return &AuthResponse{Token: newToken, User: user}, nil
}
```

핵심 설계 결정:
```
Q: 왜 별도의 Refresh Token을 안 쓰나요?
A: 프로젝트 규모에 맞는 단순한 방식을 선택했습니다.

Silent Refresh (우리 방식):
  - Access Token 하나로 운영
  - 만료 전에 같은 토큰으로 갱신 요청
  - 만료 후에도 7일 유예기간
  - 구현이 단순

Refresh Token (정석 방식):
  - Access Token (짧은 수명, 15분~1시간)
  - Refresh Token (긴 수명, 7~30일)
  - Refresh Token은 httpOnly 쿠키에 저장
  - 더 안전하지만 구현이 복잡

교육용 LMS에서는 Silent Refresh가 적절합니다.
금융 서비스라면 Refresh Token 방식을 써야 합니다.
```

**프론트엔드: 자동 갱신 로직**

```typescript
// hooks/use-auth.ts
useEffect(() => {
  const checkAndRefresh = () => {
    const token = getToken()
    const payload = parseToken(token)
    const msUntilExpiry = payload.exp * 1000 - Date.now()

    // 만료 1시간 전이면 갱신
    if (msUntilExpiry > 0 && msUntilExpiry < 60 * 60 * 1000) {
      api.post('/auth/refresh')
        .then(result => {
          setToken(result.token)      // 새 토큰 저장
          wsClient.connect(result.token)  // WS도 새 토큰으로 재연결
        })
    }
  }

  // 5분마다 체크
  const interval = setInterval(checkAndRefresh, 5 * 60 * 1000)
  return () => clearInterval(interval)
}, [])
```

**401 응답 시 자동 복구:**

```typescript
// API 호출 → 401 → refresh 시도 → 성공하면 원래 요청 재시도
if (res.status === 401) {
  const refreshed = await tryRefreshToken()
  if (refreshed) {
    return request(method, path, body)  // 원래 요청 재시도!
  }
  // refresh도 실패하면 로그인 페이지로
  window.location.href = '/login'
}
```

사용자 경험:
```
기존: 토큰 만료 → 에러 화면 → "뭐지?" → 새로고침 → 로그인 → 이전 작업 잃어버림
개선: 토큰 만료 → (자동 갱신) → 아무 일 없었다는 듯 계속 사용
```

### 3단계: 앱 버전 체크 — 배포 후 자동 새로고침

**백엔드: 버전 정보 API**

```go
// Go 빌드 시 ldflags로 주입
var (
    BuildNumber = "dev"    // -ldflags "-X main.BuildNumber=45"
    CommitSHA   = "local"  // -ldflags "-X main.CommitSHA=abc1234"
)

// GET /api/version
api.GET("/version", func(c echo.Context) error {
    return c.JSON(200, map[string]string{
        "build_number": BuildNumber,
        "commit_sha":   CommitSHA,
    })
})
```

**프론트엔드: 라우트 전환 시 버전 체크**

```typescript
// hooks/use-version-check.ts
export function useVersionCheck() {
  const location = useLocation()

  useEffect(() => {
    fetchVersion().then(serverVersion => {
      if (knownVersion && serverVersion !== knownVersion) {
        // 서버 버전이 바뀌었다! = 새로 배포되었다!
        window.location.reload()  // 새 JS 번들 로드
      }
      knownVersion = serverVersion
    })
  }, [location.pathname])  // 페이지 이동할 때마다 체크
}
```

전체 흐름:
```
1. 사용자가 앱 열기 → 서버 버전 "45-abc1234" 기록
2. 페이지 이동 (홈 → 마켓) → 버전 체크 → 같음 → 정상 진행
3. 개발자가 새 버전 배포 → 서버 버전 "46-def5678"로 변경
4. 사용자가 페이지 이동 → 버전 체크 → 다름! → 자동 새로고침
5. 새 JS 번들 로드 → 새 기능 사용 가능!
```

## 사용한 프롬프트

```
재로그인을 알아서들 할 수 있도록 로그인 페이지로 이동시켜줄 수 있어?
```

```
토큰 만료되면, 최대한 재 로그인 말고 토큰 리프레시를 해줄 방법도 있을까?
```

```
appversion 엔드포인트 등을 만들어서, 서비스 새로 배포 후,
프론트엔드에서 강제 리프레시 하게 하는 방법 있을까?
```

프롬프트의 핵심: **사용자 경험 관점에서 문제를 설명**하는 것입니다. "토큰 리프레시 구현해줘"보다 "재로그인 없이 계속 쓸 수 있게 해줘"가 더 좋은 결과를 만듭니다. AI가 사용자 경험을 고려한 설계를 해주기 때문입니다.

## 배운 점

### 1. 인증은 "보이지 않을수록" 좋다
사용자가 인증 과정을 인식하는 순간 UX가 나빠집니다. 토큰 갱신, 세션 유지는 모두 백그라운드에서 조용히 처리되어야 합니다.

### 2. 동시성 문제를 항상 생각하자
여러 API 호출이 동시에 401을 받으면 refresh 요청도 동시에 여러 번 날아갑니다. Promise를 공유하여 중복 요청을 방지했습니다:

```typescript
let refreshPromise: Promise<boolean> | null = null

async function tryRefreshToken() {
  if (refreshPromise) return refreshPromise  // 이미 진행 중이면 기다리기
  refreshPromise = doRefresh()               // 한 번만 실행
  return refreshPromise
}
```

### 3. 유예기간(Grace Period)의 중요성
토큰이 딱 만료되는 순간에 요청하면? 네트워크 지연 때문에 요청 시점에는 유효했지만 서버 도착 시에는 만료될 수 있습니다. 7일 유예기간으로 이런 경계 케이스를 처리합니다.

### 4. SPA의 캐시 문제는 모든 서비스가 겪는다
앱 버전 체크는 Netflix, YouTube 같은 대형 서비스도 사용하는 패턴입니다. 방법은 다양하지만 핵심은 같습니다: "서버 버전과 클라이언트 버전이 다르면 새로고침"

### 5. ldflags — Go의 빌드 타임 변수 주입
```bash
go build -ldflags "-X main.BuildNumber=45 -X main.CommitSHA=abc1234" -o server .
```
소스 코드를 수정하지 않고 빌드 시점에 변수 값을 주입하는 Go의 강력한 기능입니다. Docker, CI/CD와 함께 사용하면 빌드 메타데이터를 깔끔하게 관리할 수 있습니다.

---

## GitHub 참고 링크
- [커밋 945638d: 토큰 자동 갱신(silent refresh) 구현](https://github.com/cycorld/earnlearning/commit/945638d)
- [커밋 c9c7c87: 앱 버전 체크: 배포 후 자동 새로고침](https://github.com/cycorld/earnlearning/commit/c9c7c87)
