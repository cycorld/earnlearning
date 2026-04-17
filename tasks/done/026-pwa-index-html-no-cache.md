---
id: 026
title: index.html을 Cache-Control no-cache로 강제하여 SW 업데이트 확실히 전파
priority: high
type: fix
branch: fix/pwa-index-html-no-cache
created: 2026-04-13
---

## 배경

PWA 자동 업데이트 시스템(#018)이 구축되어 있음:
- Vite `skipWaiting: true`, `clientsClaim: true`
- `use-version-check.ts` 훅에서 60초 폴링 + focus/visibility 체크 → 새 버전 감지 시 `forceRefresh()` (캐시 삭제 + SW unregister + hard reload)

그럼에도 실제 운영에서 구버전에 stuck되는 사례 발생 (예: #249 빌드 시점에 사용자 핸드폰에 #206 PWA 고정).

## 근본 원인 가설

이 모든 자동 업데이트 로직은 **브라우저가 새 `index.html`과 새 `sw.js`를 받아와야** 동작한다. 그런데 `index.html`이 중간 캐시 계층(브라우저 HTTP cache, Cloudflare, nginx sendfile cache 등)에서 캐시되면:
- 오래된 `index.html` → 오래된 `/assets/index-XXXX.js`를 로드 → 오래된 SW 등록 유지
- `use-version-check.ts` 자체가 구버전이면 `/api/version` 폴링도 안 함
- 결국 사용자가 사이트 데이터를 수동으로 지워야 복구됨

따라서 **`index.html`만큼은 어떤 layer에서도 캐시되면 안 된다**. HTML은 엔트리 포인트고 모든 해시 에셋 참조의 시작점이다.

## 현재 크기 (참고)

- `frontend/index.html` 소스: 567B
- `frontend/dist/index.html` 빌드 후: 786B

매우 작아서 `no-cache`로 강제해도 트래픽 부담 없음. 실제 재다운로드는 ETag/Last-Modified로 대부분 304 응답 처리됨.

## 작업 내용

### 1. Nginx 설정 (주된 변경)
`deploy/nginx/*.conf` 에서 `/index.html` 및 `/`에 대해 명시적으로:

```nginx
location = /index.html {
    add_header Cache-Control "no-cache, no-store, must-revalidate" always;
    add_header Pragma "no-cache" always;
    add_header Expires "0" always;
    try_files $uri =404;
}

location = / {
    add_header Cache-Control "no-cache, no-store, must-revalidate" always;
    try_files /index.html =404;
}
```

`/assets/*` 같은 해시 파일은 반대로 immutable long cache 유지 (이미 되어 있으면 확인).

### 2. `sw.js`, `workbox-*.js` 도 no-cache

```nginx
location ~* ^/(sw\.js|workbox-.*\.js|registerSW\.js)$ {
    add_header Cache-Control "no-cache, no-store, must-revalidate" always;
    add_header Service-Worker-Allowed "/" always;
    try_files $uri =404;
}
```

SW 파일도 해시가 붙지 않고 고정 경로라 중간 캐시에 걸리면 치명적.

### 3. Cloudflare Page Rule (있다면)
Cloudflare를 거치고 있으므로 동일하게 `/index.html`, `/sw.js`에 대해 "Cache Level: Bypass" 규칙 설정 필요. 관리자 대시보드 확인.

### 4. manifest.webmanifest도 동일 처리 검토
PWA manifest가 바뀌면 아이콘/이름/디스플레이 등도 업데이트되어야 하는데, 이 파일도 해시가 없음.

## 검증

- [ ] 스테이지에 배포 후 `curl -I https://stage.earnlearning.com/index.html` → `Cache-Control: no-cache, ...` 헤더 확인
- [ ] `curl -I https://stage.earnlearning.com/sw.js` → 동일 확인
- [ ] Chrome DevTools → Network → index.html 응답 헤더 확인
- [ ] 두 번째 로드 시 304 응답이 오는지 (ETag 동작 확인)
- [ ] 해시 에셋(`/assets/index-XXXX.js`)은 여전히 `Cache-Control: public, immutable, max-age=31536000` 유지

## 기대 효과

- 사용자가 PWA를 여는 순간 **반드시 최신 `index.html`** 을 받게 됨
- 최신 index.html → 최신 chunk → 최신 version-check 훅 → 최신 SW
- #206 같은 stuck 상황이 구조적으로 불가능해짐

## 관련

- #018 PWA 자동 업데이트 (선행 작업)
- 현재 `vite.config.ts`, `use-version-check.ts` 는 이미 정확히 구현돼 있음 — 이 티켓은 **전송 레이어**의 구멍을 막는 작업
