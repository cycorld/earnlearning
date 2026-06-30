---
id: 112
title: GA4 Consent Mode v2 default 누락 — Realtime 0 원인 (#111 후속)
priority: high
type: fix
branch: fix/ga4-consent-mode
created: 2026-05-01
---

## 배경
#111 배포 후 GA Realtime 이 0. Playwright 로 직접 검증한 결과:
- `gtag/js` 200 로드 ✅
- `window.gtag` 함수 존재 ✅
- `dataLayer` 에 page_view 정상 push ✅
- **그러나 `g/collect` 비콘 호출은 0건** ❌

`google_tag_data.ics` 상태:
```
active: false
usedDefault: false  ← consent default 한 번도 호출 안 됨
usedImplicit: true  ← 묵시적 denied 모드
```

`tidr.container.G-T4KX9MKVL0.state: 2` (consent 차단으로 측정 비활성).

## 원인
**Consent Mode v2 (Google 2024-03+ 글로벌 적용)** 규칙에 따라 `gtag('consent', 'default', ...)` 호출을 `js`/`config`/`event` **이전** 에 반드시 선언해야 함. 안 하면 implicit denied 적용 → 비콘 차단.

`update` 만 호출해도 `default` 가 선행되지 않으면 무시됨 (테스트로 확인).

## 작업
- `frontend/src/lib/analytics.ts` `initAnalytics()` 에 consent default 선행 호출:
  - `analytics_storage: 'granted'` (LMS 는 광고 없음 — 분석만 명시 허용)
  - `ad_*: 'denied'` (필요 시 후속 티켓에서 변경)
  - `functionality_storage: 'granted'`, `security_storage: 'granted'`
- `anonymize_ip: true` 제거 — GA4 에선 무의미한 UA 시절 옵션
- 테스트 추가: dataLayer 첫 entry 가 `consent default` 인지 검증
- 배포 후 Playwright 로 stage·prod 둘 다 `g/collect` 비콘 발사 확인

## 미포함 (의도)
- 사용자 동의 UI 배너: LMS 는 로그인 게이트 + 학생 ToS 에 분석 사용 명시. 별도 배너 안 만듦.
- ad_storage 등 광고 관련 grant: 현재 LMS 광고 없음. 필요 시 후속.
