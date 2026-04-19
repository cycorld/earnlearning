# 092. LLM 동시성 메트릭 endpoint + 어드민 위젯

**날짜**: 2026-04-19
**태그**: 챗봇, LLM, 모니터링, 운영

## 배경
#087 + #088 에서 LLM cap (8) + 큐잉 도입. 운영 중에 "지금 몇 명 처리 중?",
"큐 막혀있나?" 를 외부에서 알 길이 없었음.

## 추가
### Backend
- 새 use case: `ChatUseCase.AdminLLMStats() LLMStats`
- 새 라우트: `GET /admin/chat/llm/stats` →
  ```json
  { "in_flight": 3, "waiting": 0, "cap": 8 }
  ```
- 이미 #088 에서 추가한 `llmproxy.ChatStats()` 를 재사용 — 추가 코드 거의 없음

### Frontend (AdminChatPage)
- 5초마다 polling 하는 `LLMStatsBadge` 컴포넌트
- 색상 분기:
  - `waiting > 0` → 빨강 ("막힘")
  - `in_flight ≥ 80%` → 주황 ("바쁨")
  - 기본 → 회색 ("여유")

## 표시 예시
```
⚡ LLM 동시성    처리 중: 3 / 8    대기: 0명         5초마다 갱신
```
부하 시:
```
⚡ LLM 동시성    처리 중: 8 / 8    대기: 12명        5초마다 갱신
                                                    (빨강 배경)
```

## 의도적으로 미포함 (#089 후속)
- 자동 cap 조정 — 신중한 설계 필요. 잘못 조정하면 부하 폭주 사이클.
- 누적 latency / 실패 카운트 — 별도 메트릭 store 필요. 일단 실시간만.
- Prometheus 노출 — 외부 의존성 추가. 수업 종료 후 분석 시점에 검토.

## 배운 점
관찰 가능성은 작은 비용으로 큰 운영 가치. 한 줄 endpoint + 작은 UI badge 만으로
"막혔는지/여유 있는지" 즉시 판단 가능.
