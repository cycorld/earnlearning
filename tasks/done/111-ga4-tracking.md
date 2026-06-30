---
id: 111
title: LMS 프로덕션에 GA4 적용 (선생님이 먼저 dogfood)
priority: medium
type: feat
branch: feat/ga4-tracking
created: 2026-05-01
---

## 배경
8주차 강의에서 학생들에게 "본인 서비스에 GA4 심으세요" 가이드 예정. 정작 LMS(earnlearning.com) 자체엔 GA 안 깔려있어서 dogfooding 안 됨. 강의 신뢰도 + 우리도 학생 트래픽 패턴 파악하기 위해 먼저 적용.

## 측정 ID
`G-T4KX9MKVL0` (이미 사용자가 GA4 속성 만들어서 발급받음)

## 작업 (frontend only)
- `frontend/src/lib/analytics.ts` — GA4 init + page_view + custom event helper. **production-only** (dev 에선 호출 자체 noop).
- `frontend/src/hooks/use-ga-pageview.ts` — react-router `useLocation` listener → SPA 이동마다 page_view 발사
- `frontend/src/App.tsx` — `BrowserRouter` 안에 hook 호출 컴포넌트 mount
- `frontend/src/env.d.ts` — `VITE_GA_ID` 타입 선언
- `frontend/.env.production` — `VITE_GA_ID=G-T4KX9MKVL0` (빌드 타임 주입)

## 테스트 (vitest)
- analytics 모듈: dev 모드에선 noop 보장 / prod 모드에선 dataLayer push
- use-ga-pageview: 라우트 이동마다 `page_view` 1회씩 발사

## 미포함 (의도)
- Custom event tracking (signup_completed 등) — 본 PR 은 page_view 만. 후속 티켓에서.
- backend usage logging 변경 — 별도 시스템 (#108).
- 학생 키 트래픽 분석 — GA 는 web 만. 학생 LLM 호출은 cycorld FastAPI 가 별도 집계.
