---
id: 118
title: /grants (복수) 경로 silent fallback to /feed — TDD fix
priority: medium
type: fix
branch: fix/grants-route-redirect
created: 2026-06-05
---

## 증상
사용자가 `https://earnlearning.com/grants` 입력 → React Router 가 매칭 라우트 없음
→ catch-all 로 `/feed` 로 튕김 (URL 만 바뀌고 안내 없음).

## 진단
App.tsx 등록 라우트:
- `/grant` → GrantListPage
- `/grant/new`, `/grant/:id`
- `/grants/:id` → 레거시 redirect to /grant/:id (#NN)
- ❌ `/grants` (복수, 단독) 미등록

자연스러운 URL guess (영어 list = plural) 가 작동 안 함.

## TDD
1. 회귀 테스트: `<App>` MemoryRouter `/grants` initial → `/grant` 로 리다이렉트 확인
2. Fix: `<Route path="/grants" element={<Navigate to="/grant" replace />} />`
3. 회귀 통과
