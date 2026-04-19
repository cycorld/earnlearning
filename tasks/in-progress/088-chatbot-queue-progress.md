---
id: 088
title: 챗봇 대기 중 SSE event — "현재 N명 대기 중" 표시
priority: medium
type: feat
branch: feat/chatbot-queue-progress
created: 2026-04-19
---

## 배경
#087 에서 LLM 동시 호출을 8 로 cap → 9번째 부터는 우리 백엔드에서 큐잉 (블로킹).
30 동시 시 p95 ~50초까지 대기. 사용자는 그냥 spinner 만 보다가 답답해함.

## 스코프
- **llmproxy**: in-flight + waiting 카운트 atomic 노출
- **AskStream**: LLM 호출 직전/도중에 ticker 로 "queued" SSE event 방출
  - 1.5s 내에 응답 시작되면 event 없음 (UX 안 흔들림)
  - 그 이후엔 매 2s 마다 현재 대기 인원 push
  - 풀려나면 0 으로 정리
- **Frontend ChatDock**: queued event 받으면 "대기 중… (현재 N명 대기)" 메시지

## SSE event 추가
```
data: {"type":"queued","queue_position":3}
```

## 검증
부하 테스트 30 동시 → 7번째 이후 사용자에게 queued event 가 흘러 들어감 (raw SSE 로 확인)
