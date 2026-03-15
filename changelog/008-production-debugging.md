---
title: "운영 환경 트러블슈팅: 배포 후 서비스가 안 될 때"
date: "2026-03-15"
tags: ["트러블슈팅", "Docker", "Nginx", "WebSocket", "디버깅"]
---

## 무엇을 했나요?

배포 후 서비스가 완전히 먹통이 되는 두 가지 심각한 장애를 해결했습니다:

- **Bad Gateway (502 에러)**: 사이트 접속 자체가 불가능
- **WebSocket 401 에러**: 실시간 알림이 전혀 작동하지 않음

## 왜 이런 일이 발생했나요?

### 장애 1: Bad Gateway — "사이트가 안 열려요"

이전에 개발일지(changelog) 기능을 추가하면서 Docker 빌드 설정을 변경했습니다:

```dockerfile
# 변경 전: 프론트엔드 폴더가 빌드 컨텍스트
# context: ../frontend  →  nginx.conf가 프론트엔드 폴더에 있으니 OK

# 변경 후: 프로젝트 루트가 빌드 컨텍스트 (changelog 폴더 복사 필요)
# context: ..  →  nginx.conf를 어디서 찾지?
```

문제의 핵심:

```dockerfile
# Dockerfile 마지막 단계
FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf  # ← 이 줄!
```

빌드 컨텍스트가 프로젝트 루트로 바뀌면서, `COPY nginx.conf`가 **프론트엔드용 SPA 설정** 대신 **프로젝트 루트의 전체 Nginx 설정**을 복사했습니다:

```nginx
# 프로젝트 루트의 nginx.conf (전체 서버 설정)
events { worker_connections 1024; }  # ← 이게 문제!
http {
    upstream backend { server backend:8080; }
    ...
}

# 프론트엔드의 nginx.conf (SPA 라우팅용)
server {
    listen 80;
    root /usr/share/nginx/html;
    location / {
        try_files $uri $uri/ /index.html;  # SPA 핵심!
    }
}
```

`events` 디렉티브가 `conf.d/default.conf`에 들어가면 Nginx가 "이건 여기 올 수 없는 설정이야!"라며 크래시합니다:

```
nginx: [emerg] "events" directive is not allowed here
         in /etc/nginx/conf.d/default.conf:1
```

→ 컨테이너가 시작 → 크래시 → 재시작 → 크래시... 무한 반복

### 장애 2: WebSocket 401 — "알림이 안 와요"

프론트엔드는 WebSocket을 `/api/ws`로 연결합니다:

```typescript
// 프론트엔드 ws.ts
const wsUrl = `wss://${window.location.host}/api/ws?token=${token}`
this.ws = new WebSocket(wsUrl)
```

그런데 Go 백엔드의 라우터 설정을 보면:

```go
// router.go
api := e.Group("/api")          // /api 그룹
auth := api.Group("", JWTAuth)  // /api/* 에 JWT 미들웨어 적용

// WebSocket 핸들러는... 루트에 등록!
e.GET("/ws", ws.ServeWS)  // ← /ws (not /api/ws!)
```

Nginx가 `/api/ws` 요청을 백엔드로 전달하면:

```
브라우저 → /api/ws → Nginx → 백엔드의 /api/ws

백엔드 라우터 매칭:
  /api/ws → /api 그룹에 매칭 → JWTAuth 미들웨어 실행
         → Authorization 헤더 없음 (WS는 query param으로 토큰 전달)
         → 401 Unauthorized!

원래 의도:
  /ws → WebSocket 핸들러 → query param에서 토큰 직접 검증
```

## 어떻게 해결했나요?

### Bad Gateway 수정

```dockerfile
# 수정: 빌드 컨텍스트가 루트이므로 경로를 명시
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
#     ^^^^^^^^ 프론트엔드 폴더의 nginx.conf를 명시적으로 지정
```

### WebSocket 401 수정

```go
// 수정: WS 핸들러를 /api 그룹 아래로 이동
api.GET("/ws", func(c echo.Context) error {
    return ws.ServeWS(hub, jwtSecret, c)
})
// 이제 /api/ws 요청이 여기로 매칭됨
// api 그룹에는 JWT 미들웨어가 없으므로 (auth 그룹에만 있음)
// WS 핸들러가 직접 토큰을 검증
```

### 디버깅 과정 — 이게 진짜 핵심!

장애를 만나면 당황하기 쉽습니다. 체계적으로 접근하는 방법:

```bash
# 1단계: 현재 상태 파악
docker ps  # 컨테이너 상태 확인
# → frontend가 "Restarting (1)" 상태! 크래시 루프 발견

# 2단계: 로그 확인
docker logs earnlearning-prod-frontend-1
# → "events directive is not allowed here" 에러 메시지 발견

# 3단계: 원인 추적
# "events가 conf.d에 있으면 안 된다"
# → conf.d/default.conf에 뭐가 들어갔지?
# → Dockerfile의 COPY 명령 확인
# → 빌드 컨텍스트 변경으로 잘못된 파일 복사!

# 4단계: 수정 후 검증
docker logs earnlearning-prod-frontend-1
# → 정상 시작 확인
curl http://localhost:8080/
# → 200 OK!
```

핵심 원칙: **로그를 읽어라.** 에러 메시지가 원인을 정확히 알려줍니다.

## 사용한 프롬프트

```
배포 후 bad gateway 뜨는데, 확인해줘.
```

```
여전히 ws 에러 나고 있는데 확인해줘.
```

디버깅 프롬프트는 짧아도 됩니다. 중요한 것은 **증상을 정확히 설명**하는 것입니다. "안 돼요" 대신 "bad gateway", "ws 에러"처럼 구체적인 증상을 말하면 AI가 정확한 방향으로 조사할 수 있습니다.

## 배운 점

### 1. Docker 빌드 컨텍스트를 이해하자

```
빌드 컨텍스트(context)가 바뀌면:
→ 모든 COPY 경로가 영향을 받음
→ 특히 멀티스테이지 빌드에서 주의!
→ 첫 번째 스테이지의 COPY와 두 번째 스테이지의 COPY는 다르게 동작

첫 번째 스테이지: COPY frontend/ .    → 빌드 컨텍스트에서 복사
두 번째 스테이지: COPY --from=builder  → 이전 스테이지에서 복사
두 번째 스테이지: COPY nginx.conf     → 빌드 컨텍스트에서 복사 ← 여기!
```

### 2. 경로 매칭은 예상과 다를 수 있다

```
프론트엔드: /api/ws 로 요청
Nginx: /api/* → 백엔드로 전달
백엔드: /ws 에 핸들러 등록
       /api/* 에 JWT 미들웨어 적용

→ /api/ws는 /api/* 에 먼저 매칭
→ JWT 미들웨어가 먼저 실행
→ 의도한 /ws 핸들러에 도달하지 못함

교훈: 요청이 어떤 경로로 매칭되는지 정확히 추적하자
```

### 3. "잘 되던 게 갑자기 안 되면" — 최근 변경 확인

```
git log --oneline -5  # 최근 커밋 확인
git diff HEAD~1       # 마지막 변경 내용 확인

대부분의 장애는 최근 변경에서 비롯됨
→ "뭘 바꿨지?"가 가장 강력한 디버깅 질문
```

### 4. 장애 대응 우선순위

```
1순위: 서비스 복구 (사용자가 먼저!)
2순위: 원인 분석
3순위: 재발 방지

이번 사례:
1순위: Dockerfile 수정 → 배포 → 서비스 복구 ✅
2순위: 빌드 컨텍스트 변경이 원인 ✅
3순위: PR 리뷰에서 Dockerfile 변경 시 주의 ✅
```
