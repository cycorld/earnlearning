# 111. LMS 프로덕션에 GA4 적용 — 선생님이 먼저 dogfood

**날짜**: 2026-05-01
**태그**: 분석, GA4, 트래킹, dogfooding, 강의

## 배경
8주차 강의(2026-04-24)에서 학생들에게 "본인 서비스에 GA4 심으세요" 가이드 + "여러분 트래픽을 매주 매출로 환산해 LMS 회사에 입금해줄게요" 정책 공지. 정작 LMS(earnlearning.com) 자체엔 GA 안 깔려있어서 dogfooding 자체가 안 됐던 상태. 강의 신뢰도 회복 + 학생 사용 패턴 파악을 위해 적용.

## 측정 ID
`G-T4KX9MKVL0` — earnlearning.com 전용 GA4 속성. **측정 ID 는 비밀이 아님** (브라우저 소스에 어차피 노출). API 키와 혼동 X.

## 추가 (frontend only)

### `frontend/src/lib/analytics.ts`
- `initAnalytics()` — gtag 스크립트 동적 주입 + dataLayer 초기화
- `trackPageView(path, title?)` — SPA 수동 page_view 발사
- `trackEvent(name, params?)` — 임의 이벤트 (signup_completed 등 후속 티켓에서 활용)
- **production-only 가드**: `import.meta.env.PROD` 체크. dev 에선 호출 자체 noop.
- `send_page_view: false` + `anonymize_ip: true` 설정

### `frontend/src/hooks/use-ga-pageview.ts`
- `useGAPageView()` — react-router `useLocation` 변화 감지 → `trackPageView` 호출
- `<GAPageViewTracker />` — `BrowserRouter` 안에 mount 하기 위한 무렌더 컴포넌트

### `frontend/src/main.tsx`
- 앱 시작 시 `initAnalytics()` 1회 호출

### `frontend/src/App.tsx`
- `<BrowserRouter>` 안 최상단에 `<GAPageViewTracker />` mount → 모든 라우트 변경 자동 추적

### `frontend/src/env.d.ts`
- `VITE_GA_ID` 타입 선언 (env override 지원, 기본값은 fallback 상수)

## 테스트 (TDD)
- `frontend/src/lib/analytics.test.ts` — 9 tests
  - dev 에선 noop · script 미주입 · throw X
  - prod + ID 있으면 script 1개만 주입 (idempotent) · dataLayer 활성 · page_view·custom event push 확인
- `frontend/src/hooks/use-ga-pageview.test.tsx` — 3 tests
  - mount 시 1회 발사 · 라우트 이동마다 추가 발사 · query string 포함

전체 frontend 148 tests pass · backend smoke 24 tests pass.

## 미포함 (의도)
- **Custom event** (`signup_completed`, `main_cta_clicked` 등): 본 PR 은 page_view 만. 후속 티켓에서 회원가입·게시 등 핵심 동작 추적.
- **학생 LLM 호출 분석**: GA 는 web 트래픽만. 학생 LLM API 키 사용량은 cycorld FastAPI llm-proxy DB (`/home/cycorld/llm-proxy/llm-proxy.db`) 에 별도 집계.
- **8주차 매출 연동 cron**: GA 데이터를 매주 학생 회사 wallet 으로 환산하는 batch job 은 별도 티켓.

## 운영 메모
- 측정 ID `G-T4KX9MKVL0` 의 GA 관리자 이메일은 `charleschoi87@gmail.com` + 본인 본 계정.
- staging 에선 같은 ID 로 들어가지만 DAU 가 거의 0 이라 noise 미미. 분리하고 싶으면 `VITE_GA_ID` env override 로 staging 전용 속성 발급 후 사용.
