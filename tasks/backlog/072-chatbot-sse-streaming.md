---
id: 072
title: 챗봇 SSE 스트리밍 — 첫 글자 체감 시간 개선
priority: medium
type: feat
branch: feat/chatbot-streaming
created: 2026-04-19
---

## 배경
현재 챗봇 `Ask()` 는 non-streaming — LLM 이 전체 응답을 끝낼 때까지 클라이언트는
spinner 만 본다. qwen-reasoning + effort=medium 기준 3~15초 대기. UX 개선 필요.

## 스코프
1. `llmproxy.Client.ChatCompleteStream(ctx, req) → chan ChatEvent` 구현 (SSE)
2. `ChatUseCase.AskStream` 추가 — 기존 `Ask` 와 같은 도구 루프지만 delta 를 channel 로 흘림
3. HTTP: `POST /api/chat/sessions/:id/ask/stream` → SSE 응답 (text/event-stream)
4. Frontend `ChatDock` 을 EventSource 로 변경, 글자 단위 append
5. reasoning_content 는 별도 스트림(선택적 표시)

## 확인 필요
- Tool-call 중간에 나오는 delta 를 어떻게 그룹핑할지 (assistant content + tool_calls 분리)
- 도구 실행은 여전히 서버에서 합성 실행 → 클라에서 "도구 실행 중" 표시
- Cloudflare / Nginx SSE 타임아웃 설정 (기본 60s → 180s 연장 필요할 수 있음)
