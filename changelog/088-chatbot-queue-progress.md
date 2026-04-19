# 088. 챗봇 대기 중 SSE event — "현재 N명 대기 중" 표시

**날짜**: 2026-04-19
**태그**: 챗봇, SSE, UX, 대기열

## 배경
#087 에서 LLM 동시 호출을 8 로 cap → 9번째 부터는 우리 백엔드에서 큐잉 (블로킹).
30 동시 시 p95 ~50초까지 대기. 사용자는 그냥 spinner 만 봐서 "고장 났나?" 의심.

## 수정

### Backend
- **`llmproxy.Client`**: `chatInFlight`, `chatWaiting` atomic 카운터 추가
  + `ChatStats() ChatStats { InFlight, Waiting, Cap }` 노출
- **`ChatLLMClient` 인터페이스**: `Stats() LLMStats` 추가 (adapter 가 forward)
- **`ChatUseCase.startQueueProgress(ctx, emit) chan<- struct{}`**:
  - LLM 호출 직전에 시작
  - 1.5s 안 끝나면 매 2s 마다 현재 `Waiting` 인원 push
  - LLM 호출 끝나면 `close(done)` 으로 종료, 이전에 push 한 적 있으면 0 으로 cleanup
- 모든 LLM 호출 (도구 hop + 최종 응답) 에 wrapping

### SSE event 추가
```
data: {"type":"queued","queue_waiting":3}
```
- `queue_waiting > 0` — 큐에 N명 대기 중
- `queue_waiting = 0` — 큐 빠져나옴

### Frontend (ChatDock)
- `queueWaiting` state 추가
- `onQueued` handler 가 카운트 갱신
- `text_delta` 첫 번째 받으면 카운트 0 으로 (UI 자연 정리)
- spinner 옆 메시지 분기:
  - waiting > 0: `"대기 중… 현재 N명이 함께 기다리고 있어요"`
  - 평소: `"답변 생성 중…"` / `"깊이 생각하는 중…"`

## 트레이드오프
- "현재 N명 대기" — 본인 위치가 아니라 전체 대기 인원 (간단 + 정직)
- 1.5s 미만의 짧은 대기엔 event 안 발행 — UI flicker 방지

## 검증
부하 테스트 30 동시 → 9번째 이후 client 가 `queued: N` event 수신 → UI "대기 중…" 표시.

## 후속
- 본인 위치 (1번째, 2번째…) 표시 — atomic counter 만으로는 정확히 못함, 큐 자체를 자료구조로 바꿔야
- 모니터링 endpoint (#089)
