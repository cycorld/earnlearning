---
id: 091
title: 챗봇 큐 진행률 panic 핫픽스 — startQueueProgress race
priority: high
type: fix
branch: fix/queue-progress-panic
created: 2026-04-19
---

## 증상
부하 테스트 도중 stage 백엔드 panic:
```
panic: send on closed channel
goroutine ... [running]:
... runAskStream.func1 (emit) ...
... startQueueProgress.func1 (ticker emit) ...
```

## 원인
`startQueueProgress` goroutine 이 LLM 호출 종료 후에도 잔존하다가
`emit(...)` 호출 → `out` 채널이 이미 close 됨 → panic.

`close(qDone)` 는 비동기 시그널이라 goroutine 이 그걸 받아서 종료하기 전에
runAskStream 이 return 하면 `defer close(out)` 가 먼저 닫음.

## 수정
`startQueueProgress` 가 `chan<- struct{}` 대신 `func()` (stop) 반환.
stop() 은 done close + goroutine exited 를 wait — race-free.
