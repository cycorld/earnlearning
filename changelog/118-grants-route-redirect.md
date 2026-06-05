# 118. /grants (복수) 경로 silent fallback to /feed 버그 — TDD fix

**날짜**: 2026-06-05
**태그**: 핫픽스, 라우팅, frontend, UX

## 증상
사용자가 `https://earnlearning.com/grants` 입력 → React Router 가 매칭 라우트 없음
→ catch-all 로 `/feed` 로 튕김 (URL 만 변하고 안내 없음).

자연스럽게 영어 list = 복수형 으로 추측해 입력한 학생들이 정부과제 목록 못 찾고 헤맴.

## 진단
App.tsx 라우트:
- ✅ `/grant` → GrantListPage
- ✅ `/grant/new`, `/grant/:id`
- ✅ `/grants/:id` → 레거시 redirect to `/grant/:id`
- ❌ `/grants` (복수, 단독) — 누락 → catch-all 발동

## TDD
1. 회귀 테스트 `App.routes.test.tsx` (3 tests):
   - `/grants` → `/grant` redirect
   - `/grants/14` → `/grant` redirect (레거시 호환)
   - `/grant` 직접 접근은 그대로 렌더 (no-op)
2. Fix: `<Route path="/grants" element={<Navigate to="/grant" replace />} />` 추가
3. 회귀 통과 ✅

## Impact
- 학생이 `/grants` 입력 → silent /feed 튕김 으로 인한 혼란 해소
- 모든 외부 링크 (블로그 공유 등) 호환성 ↑
- 한 줄 추가 + 회귀 테스트 3건. 다른 영향 0.

## 검증
- frontend 185 pass · vite build OK
- 다음 배포 후 prod `/grants` 직접 navigate → `/grant` 도달 확인
