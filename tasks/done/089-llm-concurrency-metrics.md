---
id: 089
title: 챗봇 LLM 동시성 메트릭 노출 + 자동 조정
priority: low
type: feat
branch: feat/llm-concurrency-metrics
created: 2026-04-19
---

## 배경
#087 에서 `LLM_PROXY_MAX_CONCURRENT` (default 8) 도입. 현재는 env 로만 조정,
실제 in-flight / waiting / 평균 대기 시간을 외부에서 모를 수 있음.

## 스코프
1. **메트릭 endpoint** — `GET /api/admin/chat/llm/stats` 반환:
   - `in_flight` (현재 처리 중)
   - `waiting` (큐 대기)
   - `cap` (동시 cap)
   - `total_calls` (서버 시작 후 누적)
   - `avg_latency_ms` (최근 N분 이동 평균)
   - `p95_latency_ms`
   - `failed_count` (timeout/오류)
2. **AdminChatPage 에 위젯** — 실시간 표시 (5s polling)
3. **(선택) 자동 조정** — 평균 latency > 30s 면 cap -1, < 5s 면 cap +1
   (안전을 위해 기본 비활성, env `LLM_PROXY_AUTOSCALE=1` 로 켜기)

## 확인 필요
- llm.cycorld.com 의 슬롯 수 변동 가능성 — 자동 조정이 LLM 서버 상태와 미스매치 가능
- 자동 조정은 신중하게: 한 번 잘못 조정하면 부하 폭주 사이클 가능

## 우선순위
관찰 가능성 (#1, #2) 만 해도 운영 가치 큼. 자동 조정 (#3) 은 후속 후속.

## 후속
- Prometheus / OpenTelemetry 노출 (수업 종료 후 분석용)
