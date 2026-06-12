---
id: 133
title: 멘션 자동완성 — 이메일 타이핑 간섭 회귀 테스트
priority: low
type: chore
branch: chore/133-mention-email-typing-test
created: 2026-06-13
---

#132 멘션 자동완성이 이메일 주소 타이핑을 방해하지 않는지 검증하는 프론트 회귀 테스트.

- 단어 중간 @(예: abc@gmail.com) → 드롭다운/검색 호출 없음
- 줄 시작 @도메인 → 검색 결과 없으면 드롭다운 없이 타이핑 유지
- 드롭다운 열린 상태에서 선택 없이 계속 타이핑 → 마크업 미삽입, Enter는 줄바꿈

stage 브라우저 검수로도 동일 시나리오 통과 확인 (build 352, 9c04375).
