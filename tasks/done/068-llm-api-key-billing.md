---
id: 068
title: LLM API 키 학생 프로비저닝 + 매일 자정 자동 과금
priority: high
type: feat
branch: feat/llm-api-key-billing
created: 2026-04-18
---

## 배경
`https://llm.cycorld.com` 을 학생들에게 제공. 학생들이 LMS 에서 DB 계정 생성하듯
LLM API 키를 발급받아 사용할 수 있어야 함.

## 스코프
1. **관리자 키 보관**: `admin-[REDACTED: stored in LLM_ADMIN_API_KEY env]` 를 서버
   env 로 보관 (절대 프론트에 노출 금지).
2. **학생용 API 키 발급 플로우**: LMS 에 "LLM 키" 페이지 신설 →
   - 아직 키 없으면 "발급받기" 버튼 → 서버가 admin API 로 student-scoped 키 생성 →
     DB 에 user_id ↔ llm_key_id 매핑 저장 → 학생에게 1회성으로 full secret 노출
   - 이미 있으면 키 메타(이름, 발급일, 누적 사용 토큰) + 재발급 버튼
3. **매일 자정 자동 과금** (KST 00:00 cron):
   - 전 학생 키의 전일 토큰 사용량을 admin API 로 수집
   - Opus 4.7 기준가로 원화 환산 → 학생 지갑에서 차감
   - 트랜잭션 원장(billing_ledger) 에 일자·토큰·금액 기록 → 중복 차감 방지
4. **프론트 UI**: 사용량 히스토리 + 일일 비용 추이 그래프 (최소 버전은 표 형태)

## 확인 필요 (사용자에게)
- **환율**: Opus 4.7 공식가 $15/MTok input, $75/MTok output → 원화 환산 레이트?
  (e.g. 1 USD = 1,400 원 고정? 일별 환율?) — 이 수업에선 **교육 목적의 가상 통화**
  이므로 단순 고정 환율 권장.
- **잔액 부족 시**: ① 마이너스 허용(부채) ② 키 비활성화 ③ 하루치만 가능한 만큼 차감
  후 나머지 기록만 남김
- **타임존**: "매일 자정" 은 KST 기준 00:00 확정?
- **현 시점 공개**: 재수강생/청강생 포함 모든 승인된 학생? 아니면 특정 클래스?

## 기술 스택
- Backend: Go + SQLite (기존 스택 유지)
- 새 테이블: `llm_api_keys` (user_id, provider_key_id, name, created_at, revoked_at)
           `llm_usage_daily` (user_id, date, input_tokens, output_tokens, cost_krw, billed_at)
- Cron: backend 내부 고루틴 (기존 스케줄러 있으면 재활용)
- Admin API 호출: llm.cycorld.com 의 Swagger 스펙 참조 (설계 전 탐색 필수)

## 상태
진행 예정 — 설계 질문 답변 후 구현 시작
