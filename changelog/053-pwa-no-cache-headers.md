# 053. PWA 업데이트 확실하게 만들기 — Cache-Control 강화

> **날짜**: 2026-04-16
> **태그**: `fix`, `PWA`, `nginx`, `인프라`

## 무엇을 했나요?

PWA 엔트리 포인트(`index.html`)와 Service Worker 관련 파일들에 **브라우저 캐시가 절대 걸리지 않도록** nginx 응답 헤더를 강화했습니다. 새 배포를 했는데도 사용자 휴대폰에 오래된 빌드가 박혀 있는 문제를 구조적으로 방지합니다.

## 왜 필요했나요?

최근에 한 사용자의 PWA가 빌드 #206에 고정되어 있었습니다. 현재 운영 빌드는 #249였으니 43개 빌드 차이. 이런 상황을 분석해 보니 이미 있던 자동 업데이트 로직(`use-version-check.ts` 훅, `skipWaiting`, `forceRefresh`)은 **사용자가 실제로 새 `index.html`과 새 `sw.js`를 받아야** 동작할 수 있는데, 그 두 파일이 어딘가의 캐시 계층에 걸리면 로직이 돌기 전에 구버전 코드가 로드되는 악순환에 빠질 수 있다는 것이었습니다.

그래서 **전송 계층에서부터** 절대 캐시되지 않도록 못을 박는 게 근본 방어입니다.

## 어떻게 만들었나요?

### 1. nginx 응답 헤더 강화 (`frontend/nginx.conf`)

**Never cache** (배포할 때마다 바뀌거나, 바뀌면 즉시 전파되어야 하는 파일):
- `/`, `/index.html` — SPA 엔트리 포인트
- `/sw.js`, `/sw-push.js` — Service Worker
- `/workbox-*.js` — Workbox 런타임
- `/registerSW.js` — vite-plugin-pwa 등록 스텁
- `/manifest.webmanifest` — PWA 매니페스트

이 파일들은 모두 다음 헤더를 받습니다:
```
Cache-Control: no-cache, no-store, must-revalidate
CDN-Cache-Control: no-store
Pragma: no-cache
```

**Always cache** (해시가 파일명에 박혀 있어서 절대 충돌하지 않는 파일):
- `/assets/*.{js,css,png,...}` — Vite 해시 에셋
```
Cache-Control: public, max-age=31536000, immutable
```

### 2. `always` 플래그가 핵심

nginx `add_header` 디렉티브는 기본적으로 **성공 응답(2xx/3xx)에만** 적용됩니다. 그런데 PWA는 ETag 기반 조건부 요청을 자주 쓰고, 캐시 검증 결과는 대부분 **304 Not Modified**입니다. 기본 설정으로는 304 응답에 `Cache-Control` 헤더가 누락되어, 브라우저는 이전 응답의 Cache-Control을 그대로 재사용합니다. 구버전이 계속 박혀 있는 한 원인이 바로 이것이었을 가능성이 큽니다.

해결: 모든 `add_header`에 `always` 플래그를 추가:
```nginx
add_header Cache-Control "no-cache, no-store, must-revalidate" always;
```
이제 304 응답에도 최신 Cache-Control 지침이 함께 내려갑니다.

### 3. `CDN-Cache-Control: no-store` — Cloudflare 방어

Cloudflare는 origin의 `Cache-Control`을 존중하긴 하지만, `CDN-Cache-Control` 헤더를 명시하면 "이건 CDN 계층에서도 저장하지 말아라"는 지시를 더 명확하게 전달할 수 있습니다. 허리 벨트 + 멜빵 전략이에요.

### 4. 이중 `Cache-Control` 제거

기존 `expires 1y; add_header Cache-Control "public, immutable"` 조합은 nginx가 `expires`로부터 `Cache-Control: max-age=...` 를 자동 생성하고, `add_header`로 또 한 줄 추가해서 **응답에 Cache-Control 헤더가 두 번** 들어가는 부작용이 있었습니다. `add_header` 한 줄로 통일:
```nginx
add_header Cache-Control "public, max-age=31536000, immutable" always;
```

## 검증

로컬에서 실제 nginx 컨테이너를 띄우고 모든 경로의 응답 헤더를 curl로 확인했습니다:

| 경로 | Cache-Control |
|------|---------------|
| `/` | `no-cache, no-store, must-revalidate` |
| `/index.html` | `no-cache, no-store, must-revalidate` |
| `/sw.js` | `no-cache, no-store, must-revalidate` + `Service-Worker-Allowed: /` |
| `/workbox-b51dd497.js` | `no-cache, no-store, must-revalidate` |
| `/registerSW.js` | `no-cache, no-store, must-revalidate` |
| `/manifest.webmanifest` | `no-cache, no-store, must-revalidate` |
| `/favicon.png` | `no-cache` (revalidate만) |
| `/assets/index-XXXX.js` | `public, max-age=31536000, immutable` |

그리고 가장 중요한 **304 응답 테스트**:
```bash
$ curl -sI -H "If-None-Match: \"69db1ef1-312\"" http://localhost/index.html
HTTP/1.1 304 Not Modified
Cache-Control: no-cache, no-store, must-revalidate  ← 이전에는 여기에 없었음
CDN-Cache-Control: no-store
Pragma: no-cache
```

## 배운 점

- **`add_header`는 기본적으로 304에 적용되지 않는다** — PWA나 자주 revalidate하는 정적 사이트에서는 `always` 플래그가 실질적으로 필수입니다. 모르고 넘어가면 "왜 이게 캐시되지?"하고 하루 잃어버릴 수 있습니다.
- **자동 업데이트는 전송 계층에서 시작된다** — skipWaiting, forceRefresh, 폴링 훅 같은 애플리케이션 레이어 방어는 *엔트리 포인트가 항상 fresh*라는 전제 위에 돌아갑니다. 그 전제가 깨지면 모든 방어가 무력화됩니다.
- **구조적 방어 vs 상황적 방어** — #018에서 추가한 훅들은 "새 버전이 나왔어요, 새로고침하세요" 류의 상황적 방어였고, 이 티켓은 "그 훅이 항상 최신이도록" 보장하는 구조적 방어입니다. 상황적 방어는 자주 실패하니 구조적 방어가 밑에 깔려 있어야 합니다.

## 관련

- #018 PWA 자동 업데이트 (선행)
- #027 WebSocket force-reload broadcast (서버 push 방어, 상보적)
- #028 version-check 자동 refresh (클라이언트 로직 방어, 상보적)
