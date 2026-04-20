---
id: 105
title: 챗봇 input/전송 버튼 폴리시 + iOS auto-zoom 방지
priority: medium
type: feat
branch: feat/chatbot-input-polish
created: 2026-04-20
---

## 변경
- textarea: `text-sm` (14px) → `text-base` (16px) — iOS focus auto-zoom 차단
- placeholder 짧게 ("질문을 입력하세요…"). desktop hint 는 작은 글씨로 옆에
- rows=1 시작 + auto-grow (max 5 lines)
- 전송 버튼: input 내부 우하단 absolute 배치 (당근/토스 패턴)
- 활성 시 brand green, disabled 시 회색 — 대비 명확화
- 입력 컨테이너에 `pb-[env(safe-area-inset-bottom)]` 추가

## 미포함 (의도)
- viewport meta `maximum-scale=1, user-scalable=no` — WCAG 1.4.4 위반, 추가 안 함
