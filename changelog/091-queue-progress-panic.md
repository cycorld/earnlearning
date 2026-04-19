# 091. 챗봇 큐 진행률 panic 핫픽스 — startQueueProgress race

**날짜**: 2026-04-19
**태그**: 챗봇, 핫픽스, race-condition

## 증상
부하 테스트 도중 stage 백엔드 panic + 자동 재시작:
```
panic: send on closed channel
goroutine ... [running]:
... runAskStream.func1 (emit) ...
... startQueueProgress.func1 (ticker emit) ...
```

## 원인 (#088 회귀)
`startQueueProgress` goroutine 이 LLM 호출 종료 후에도 잔존하다가
`emit(...)` 호출 → `out` channel 이 이미 close 됨 → panic.

`close(qDone)` 는 비동기 시그널 — goroutine 이 받아서 종료하기 전에 runAskStream
이 return 하면 `defer close(out)` 가 먼저 닫음. **race condition.**

## 수정
`startQueueProgress` 가 `chan<- struct{}` 대신 `func()` (stop) 반환:
```go
func startQueueProgress(...) func() {
    done := make(chan struct{})
    exited := make(chan struct{})
    go func() {
        defer close(exited)
        ...
    }()
    return func() {
        close(done)
        <-exited  // 동기 대기 — race-free
    }
}
```
caller 는 `qStop()` 을 호출하면 goroutine 종료까지 블로킹.

## 배운 점
1. **채널 close 는 비동기 신호** — goroutine 이 그걸 받아 정리할 시간 보장 안 됨
2. **공유 자원 (out channel) 에 쓰는 goroutine 은 owner 가 close 하기 전에 반드시 종료** — sync 패턴 필수
3. **부하 테스트는 panic catcher** — 단일 사용자 흐름에선 timing 이 우호적이라 안 보였음
