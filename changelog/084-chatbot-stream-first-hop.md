# 084. 챗봇 스트리밍 — 첫 hop 도구 없는 응답 텍스트 누락 수정

**날짜**: 2026-04-19
**태그**: 챗봇, SSE, 버그수정

## 증상
"안녕" 같은 도구 불필요한 짧은 질문을 ask/stream 으로 보내면 응답에 `text_delta`
event 가 전혀 없이 `done` 만 도착. 프론트는 빈 메시지를 보게 됨.

```
data: {"type":"done","message_id":19,"tokens":1915}
data: {"type":"close"}
```

## 원인
`AskUseCase.AskStream` 의 첫 hop 은 non-streaming 으로 LLM 호출 (도구 호출 여부
판단용). 도구 호출이 없고 content 가 있는 케이스에서 `finalizeStreamFromText`
만 호출하고 `text_delta` event 발행을 누락했음.

## 수정
첫 hop 에서 content 받았으면 → 전체 content 를 한 번의 `text_delta` 로 emit.
UX 차원에서 "한 번에" 도착하지만 정확한 정보 전달.

```go
if choice.Message.Content != "" {
    emit(AskStreamEvent{Type: StreamEventTextDelta, Delta: choice.Message.Content})
    uc.finalizeStreamFromText(...)
    return
}
```

## 트레이드오프
완전한 streaming (첫 hop 부터 streaming + tool_calls delta 파싱) 은 후속 작업.
도구 미사용 응답은 보통 짧아 체감 차이가 작음.

## 배운 점
SSE 같은 스트리밍 인터페이스를 구현할 땐 모든 분기에서 적어도 하나의 텍스트
이벤트가 발행되는지 확인 필요. "done 만 와도 정상" 으로 가정하면 사용자에게
빈 메시지가 노출됨.
