---
id: 078
title: 챗봇 답변 품질 + 긴 context 타임아웃 완화 (통합)
priority: high
type: fix
branch: fix/chatbot-quality
created: 2026-04-19
---

## 배경
브라우저 E2E 테스트에서 발견된 2가지 이슈:

1. **환각** — "주주총회 가결 기준?" 질문에 search_wiki 결과(LMS 정책: 일반 50%, 청산 70%)
   를 받고도 일반 상법 "4분의 1 출석" 규정을 섞어 답변.
2. **긴 체인 타임아웃** — dev_helper 가 context7_search → context7_docs → 추가 쿼리 반복
   → 누적 context 5k+ tokens + reasoning 모드에서 ~24s 후 500.

## 수정

### 프롬프트 강화 (전 스킬)
시스템 프롬프트 공통 원칙 추가:
- "도구 결과에 명시된 사실만 인용. 없으면 '문서에 명시되지 않았습니다' 라고 답."
- "LMS 정책(지갑/회사/청산/주주총회 등)은 반드시 search_wiki 결과만 사용. 일반 상법/
  회사법과 LMS 규칙은 다를 수 있음."
- "확실하지 않으면 추측하지 말고 추가 질문으로 명확화를 요청."

### 응답 크기 축소
- `context7_docs` 기본 tokens: 3000 → 1500
- `fetch_url` 기본 max_chars: 6000 → 3500
- `maxToolHops`: 6 → 4

### dev_helper 프롬프트 개선
- "한 번에 가장 연관성 있는 쿼리 하나만 context7_search". 여러 번 호출 금지.
- "첫 context7_docs 결과로 답이 나오면 추가 도구 호출 없이 바로 응답 작성".

## 테스트
- "주주총회 가결 기준?" → "일반 안건 50%, 청산 70% (LMS 정책)" 정도로 답. 상법 용어 섞이면 fail.
- "TanStack Query v5 의 useSuspenseQuery" → 4-hop 내 완결. 500 이면 fail.
- Prod 승격은 퀄리티 검증 후.
