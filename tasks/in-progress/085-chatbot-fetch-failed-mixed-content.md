---
id: 085
title: 챗봇 failed to fetch 핫픽스 — host nginx trailing slash 301 → Mixed Content
priority: high
type: fix
branch: fix/chatbot-nginx-trailing-slash
created: 2026-04-19
---

## 증상
프로덕션 챗봇이 "failed to fetch" 로 실패. 콘솔 에러:
```
Mixed Content: The page at 'https://earnlearning.com/...' was loaded over HTTPS,
but requested an insecure resource 'http://earnlearning.com/api/chat/sessions/'.
```

## 원인
#072 SSE PR 에서 host nginx (`deploy/nginx-host.conf`) 에 추가한:
```
location /api/chat/sessions/ {
    ...
}
```
trailing slash 가 있는 location prefix 때문에 nginx 기본 동작으로 `/api/chat/sessions`
(no trailing) 요청에 대해 301 redirect → `http://earnlearning.com/api/chat/sessions/`
를 발행함. CF/origin 모두 plain HTTP scheme 으로 redirect 하기 때문에 브라우저가
Mixed Content 로 차단.

## 수정
- prefix location 대신 정규식 location 으로 SSE 엔드포인트만 정확히 매칭:
  ```
  location ~ ^/api/chat/sessions/[^/]+/ask/stream$ {
      proxy_buffering off;
      ...
  }
  ```
- 이렇게 하면 `/api/chat/sessions` (no trailing) 요청은 redirect 없이 `location /` 로 처리됨
- nginx-app.conf (docker compose) 는 이미 `location /api/` 로 안전 — 변경 불필요

## 후속
- 호스트 nginx 수동 sync 후 검증
