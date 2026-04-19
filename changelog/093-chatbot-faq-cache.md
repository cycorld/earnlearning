# 093. 챗봇 FAQ 캐싱 — 인사/감사 즉시 응답 (LLM 슬롯 절약)

**날짜**: 2026-04-19
**태그**: 챗봇, 캐싱, 성능, 비용

## 배경
"안녕", "고마워", "잘가" 같은 짧은 인사조차 system prompt 800 토큰 + 도구 spec
수백 토큰을 LLM 에 매번 보냄. LLM 슬롯 1개 5–10초 점유 + 비용 발생.

## 수정
`backend/internal/application/chat_faq.go` 신규:
- `chatFAQ` 매핑 (한국어 + 영문) — 인사/감사/작별/테스트
- `normalizeFAQ()` — 입력 정규화:
  - trailing 구두점/이모지/공백 제거 (`"안녕!! 😊"` → `"안녕"`)
  - 소문자 변환
  - **10자 이상은 매칭 거부** — 진짜 질문 가로채기 방지
- `lookupFAQ(message) → (response, hit)` — exact match 만 (부분 매칭 금지)
- `respondFAQ()` — 매칭 시 즉시 SSE `text_delta` + `done` event,
  DB 저장 (`model="faq"` 마킹)

`AskStream` 진입 시 user message 저장 직후 FAQ 체크.

## 효과 추정
- 인사/감사 비율 ~10% 가정 → 슬롯 점유 10% 감소
- 인사 응답 latency 5초 → 50ms (100배 빨라짐)
- LLM 비용 동일 비율 절약

## 매칭 예시
| 입력 | 매칭 |
|---|---|
| "안녕" | ✅ |
| "안녕?" | ✅ |
| "안녕!! 😊" | ✅ (이모지/구두점 제거) |
| "Hi" | ✅ |
| "감사합니다" | ✅ |
| "안녕하세요 회사가 뭐예요?" | ❌ (10자 초과 + 진짜 질문) |
| "지갑 잔액 알려줘" | ❌ |

## 테스트
`chat_faq_test.go` 16 케이스 (긍정 + 부정).

## 미포함 (후속)
- LLM-generated 캐시 (자주 받는 질문 자동 학습)
- 위키 가이드 일부 캐시
