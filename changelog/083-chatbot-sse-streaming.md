# 083. 챗봇 SSE 스트리밍 — 첫 글자 체감 시간 대폭 개선 [#072]

**날짜**: 2026-04-19
**태그**: 챗봇, SSE, UX, 스트리밍

## 배경
지금까지는 LLM 이 전체 응답을 끝낼 때까지 학생은 spinner 만 봤음. qwen-reasoning
+ effort=medium 기준 3~15초 대기. 긴 답변일수록 답답함이 컸다.

## 솔루션 — Server-Sent Events (SSE)
브라우저가 fetch 로 연결을 열어두고 서버가 응답 chunk 를 한 줄씩 흘려보냄. 토큰
하나가 생성되는 즉시 화면에 나타남.

## 구현

### Backend
1. **`llmproxy.ChatCompleteStream`** — `/v1/chat/completions` 를 `stream=true` 로
   호출하고 SSE 응답을 line-by-line 파싱해 `ChatStreamEvent` 채널로 흘림.
   `data: {...}\n\n` 형식, `data: [DONE]` 으로 종료.
2. **`ChatLLMClient` 인터페이스 확장** — `ChatCompleteStream(ctx, req) → chan
   LLMStreamEvent`.
3. **`ChatUseCase.AskStream`** — 기존 `Ask` 와 같은 도구 루프지만 결과를 채널로
   푸시. 도구 호출 hop 은 non-streaming, 최종 응답 turn 만 streaming.
   - `tool_call` event — 어시스턴트가 도구 호출 결정
   - `tool_result` event — 도구 실행 결과
   - `text_delta` event — 최종 응답 토큰 chunk
   - `done` event — 완료 (총 token 수 포함)
4. **HTTP**: `POST /api/chat/sessions/:id/ask/stream` — `text/event-stream` 응답.
   `X-Accel-Buffering: no`, `Cache-Control: no-cache, no-transform` 헤더 + Echo
   `c.Response().Flush()` 매 chunk.

### Frontend (`ChatDock`)
- `streamAsk()` 헬퍼: fetch + ReadableStream 으로 SSE 소비 (EventSource 는 GET
  만 지원하므로 직접 구현)
- 메시지 보낼 때 빈 assistant placeholder 를 미리 추가 → text_delta 마다
  content 누적 → 토큰 단위로 화면 업데이트
- tool_call / tool_result event 도 실시간 chip / details 로 표시

### Nginx
- `nginx-app.conf` (docker compose 내) — `/api/` 위치에 `proxy_buffering off`,
  `proxy_read_timeout 180s` 추가
- `nginx-host.conf` (EC2 호스트) — `/api/chat/sessions/` 별도 location 으로
  같은 설정. **수동 sync 필요** (`sudo cp deploy/nginx-host.conf
  /etc/nginx/conf.d/earnlearning.conf && sudo nginx -s reload`).

### 보존
- 기존 non-streaming `POST /chat/sessions/:id/ask` 는 그대로 유지 (호환성).
  프론트는 streaming 으로 전환했지만 외부 OAuth 클라이언트가 사용 가능.

## 트레이드오프
- **Cloudflare Free**: text/event-stream 은 자동 감지로 buffering 안 함. OK.
- **타임아웃**: stream 도중 cf 100s 제한 닿을 수 있음. 일반 응답은 60s 이내라
  대부분 OK. 매우 긴 답변 (>100s) 은 Cloudflare 가 끊을 수 있음 — 후속 개선 여지.
- **에러 핸들링**: stream 도중 끊기면 마지막 받은 chunk 까지만 표시. 사용자가
  retry 하면 됨.

## 배운 점
- **EventSource ≠ SSE** — EventSource API 는 GET 만 지원해서 POST 요청에는
  fetch + ReadableStream + TextDecoder 조합이 표준 패턴.
- **SSE event boundary** 는 `\n\n` (또는 일부 서버 `\r\n\r\n`). 두 케이스 모두
  처리해야 호환성 보장.
- **nginx proxy_buffering off** 가 SSE 통과의 핵심. 한 곳이라도 켜져 있으면
  chunk 가 모이고 "한꺼번에" 도착함 → 스트리밍 효과 사라짐.
