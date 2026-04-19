# 087. LLM proxy 동시 호출 cap + timeout 상향

**날짜**: 2026-04-19
**태그**: 챗봇, LLM, 부하, 안정성

## 배경
부하 테스트로 확인한 동시 사용자 한계:
- 10 동시: 100% 성공
- 15 동시: 47% 성공 (절반 15s timeout)
- 30 동시: 30% 성공

원인: `llmproxy.New()` 의 `http.Client.Timeout: 15s` hard cap. llm.cycorld.com 이
큐잉할 때 첫 바이트가 늦으면 우리 클라이언트가 포기.

## 수정 (2가지)

### 1) Timeout 15s → 60s
큐잉으로 흡수. 사용자는 좀 더 기다리지만 실패는 줄어듦.

### 2) 동시 호출 semaphore (default 8)
```go
chatSem chan struct{}  // make(chan struct{}, 8)

func (c *Client) ChatComplete(...) {
    c.acquireChat(ctx)  // cap 도달하면 큐잉
    defer c.releaseChat()
    ...
}
```
- env `LLM_PROXY_MAX_CONCURRENT` 로 조정 (default 8 — llm.cycorld.com 추정 슬롯 수)
- `ChatCompleteStream` 도 동일 (release 는 stream goroutine 종료 시)

## 효과
- llm.cycorld.com 에 한 번에 8 만 보냄 → 자체 과부하 방지
- 9번째부터는 우리 백엔드에서 큐잉 (블로킹). 60s timeout 안에 차례 오면 성공
- 실패가 큐잉 대기로 변환 → "느리지만 동작" UX

## 트레이드오프
- 큐잉 대기는 사용자에게 그냥 spinner 만 보임 (아직 "대기 중 N명" UI 없음 — 후속)
- llm.cycorld.com 이 슬롯을 늘리면 cap 도 같이 올려야 (env 로 조정)

## 후속 가치
- "현재 대기 N명" 표시 (#072 SSE event 추가)
- 자주 묻는 질문 캐싱
