---
id: 084
title: 챗봇 스트리밍 — 첫 hop 에서 도구 없이 답한 텍스트 누락 수정
priority: high
type: fix
branch: fix/chatbot-stream-text-emit
created: 2026-04-19
---

## 증상
"안녕" 같은 도구 불필요한 짧은 질문을 ask/stream 으로 보내면 응답에 `text_delta`
event 가 전혀 없고 `done` 만 옴. 프론트는 빈 메시지를 보게 됨.

## 원인
`AskUseCase.AskStream` 의 첫 hop 은 non-streaming 으로 LLM 호출 → 도구 호출 여부
판단. 도구 호출이 없고 content 가 있는 케이스에서 `finalizeStreamFromText` 만
호출하고 `text_delta` event 발행을 누락.

## 수정
첫 hop 에서 content 받았으면 → 전체 content 를 한 번의 `text_delta` 로 emit
한 후 finalize. UX 차원에서 "한 번에" 도착하지만 정확한 정보는 전달.

## 후속 (별도 티켓 가치 있음)
완전한 streaming 을 원하면 첫 hop 부터 streaming 으로 호출하고 tool_calls delta
를 파싱해야 함. 현재는 "도구 사용 후 답변" 케이스만 진짜 streaming, 나머지는
한 번에 emit. 트레이드오프 — 도구 미사용 응답은 보통 짧아 체감 차이 작음.
