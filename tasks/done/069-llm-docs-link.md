---
id: 069
title: /llm 페이지 "API 문서 보기" 링크 수정 (admin/docs → landing)
priority: low
type: fix
branch: fix/llm-docs-link
created: 2026-04-18
---

## 배경
`/llm` 페이지의 "API 문서 보기 ↗" 링크가 `https://llm.cycorld.com/admin/docs` 로
연결되어 있는데, 여기는 관리자 전용 Swagger UI 라서 학생이 admin key 없이 접속하면
401 이 난다.

## 수정
링크를 `https://llm.cycorld.com/` 로 변경 — 여기가 학생용 공개 랜딩/문서 페이지.

## 범위
- `frontend/src/routes/llm/LlmPage.tsx` 한 줄만 변경
- 테스트는 스모크만
