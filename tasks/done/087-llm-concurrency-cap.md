---
id: 087
title: 챗봇 LLM 호출 동시성 제한 + timeout 상향 — 동시 사용자 처리력 개선
priority: medium
type: fix
branch: fix/llm-concurrency-cap
created: 2026-04-19
---

## 배경
부하 테스트 결과 (Stage, qwen-chat fast):
- 10 동시: 100% 성공
- 15 동시: 47% 성공 (절반이 15s timeout)
- 30 동시: 30% 성공

원인: `llmproxy.New()` 의 `http.Client.Timeout: 15s` hard cap. llm.cycorld.com 이
자체 큐잉할 때 첫 바이트가 늦게 와서 우리 클라이언트가 포기.

## 가설
1. **Timeout 60s 로 상향** → llm.cycorld.com 큐가 풀릴 시간 충분 → 실패 → 대기로 변환
2. **semaphore 로 in-flight LLM 호출을 8 로 cap** → llm.cycorld.com 에 동시 8 만 보내고 나머지는 우리 백엔드에서 큐잉 → llm.cycorld.com 자체 과부하 방지

## 수정
- `llmproxy.New()` 의 `http.Client.Timeout: 15s → 60s`
- `llmproxy.Client` 에 `sem chan struct{}` 필드 추가, `LLM_PROXY_MAX_CONCURRENT` env (default 8)
- `ChatComplete` / `ChatCompleteStream` 진입 시 semaphore acquire, 종료/실패 시 release

## 검증
부하 테스트 동일 시나리오 재실행. 성공률 + p95 first-token 비교.
