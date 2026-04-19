# 085. 챗봇 failed to fetch 핫픽스 — host nginx trailing slash 301 → Mixed Content

**날짜**: 2026-04-19
**태그**: 챗봇, 핫픽스, nginx, Mixed Content

## 증상
프로덕션 배포 직후 챗봇이 "failed to fetch" 로 실패.

브라우저 콘솔:
```
Mixed Content: The page at 'https://earnlearning.com/feed' was loaded over HTTPS,
but requested an insecure resource 'http://earnlearning.com/api/chat/sessions/'.
This request has been blocked.
```

## 원인 (#072 의 회귀)
SSE PR (#072) 에서 host nginx (`deploy/nginx-host.conf`) 에 추가했던:
```nginx
location /api/chat/sessions/ {  # ← 끝에 슬래시
    proxy_buffering off;
    ...
}
```

nginx 의 prefix location 끝에 `/` 가 붙으면 **자동으로 trailing-slash redirect** 가
켜짐. `/api/chat/sessions` (no trailing) 요청이 오면 nginx 가 알아서:
```
HTTP/1.1 301 Moved Permanently
Location: http://earnlearning.com/api/chat/sessions/
```
를 발행. **scheme 이 http** 로 떨어지는 이유는 nginx 가 origin 서버에서 받은
스킴 (HTTP) 으로 absolute URL 을 만들기 때문 (CF → host nginx 는 plain HTTP).

브라우저는 HTTPS 페이지에서 HTTP 리다이렉트를 보면 Mixed Content 로 차단.

## 수정
prefix location 대신 **정규식 location** 으로 SSE 엔드포인트만 정확히 매칭:
```nginx
location ~ ^/api/chat/sessions/[^/]+/ask/stream$ {
    proxy_buffering off;
    ...
}
```

이렇게 하면:
- `/api/chat/sessions` (no trailing) → location 매칭 안 됨 → `location /` 처리 → redirect 없음
- `/api/chat/sessions/14/ask/stream` → 정규식 매칭 → SSE 설정 적용

`nginx-app.conf` (docker compose 내부) 는 `location /api/` 라 영향 없음.

## 배운 점
1. **nginx prefix location 끝의 `/` 는 위험** — auto trailing-slash redirect 트리거.
   SSE/WebSocket 같은 특정 endpoint 는 정규식 location 으로 명확히.
2. **Mixed Content 디버깅** — 브라우저 console 의 "Mixed Content" 에러는 redirect
   의 결과인 경우가 많음. Network 탭에서 redirect chain 확인.
3. **HTTPS 종료가 CF/edge 에서** 일어나는 환경은 origin 이 plain HTTP 라는 사실을
   기억해야 함. Echo `c.Request().URL.Scheme` 같은 것도 마찬가지.

## 검증
브라우저 Playwright E2E:
- "안녕하세요" 질문 → 정상 응답 ✅
- console 에러 0건
