# 112. GA4 Consent Mode v2 default 누락 수정 (#111 핫픽스)

**날짜**: 2026-05-01
**태그**: 핫픽스, GA4, consent, 분석

## 증상
#111 배포 후 GA Realtime 이 계속 0. 외관상 모든 게 정상이었음:
- `gtag/js?id=G-T4KX9MKVL0` HTTP 200 로드
- `window.gtag` 함수 정상
- `dataLayer` 에 page_view 정상 push

그런데 `g/collect` 비콘은 1건도 안 가는 상태.

## 디버깅
Playwright 로 직접 production 사이트 로드 → `window.google_tag_data` 내부 검사:

```json
"ics": {
  "active": false,
  "usedDefault": false,   ← consent default 한 번도 호출 안 됨
  "usedImplicit": true     ← 묵시적 denied 적용됨
}
"tidr.container.G-T4KX9MKVL0.state": 2  ← consent 차단으로 측정 비활성
```

## 원인
**Consent Mode v2** (Google 이 2024-03 부터 글로벌 적용) 규칙:

1. `gtag('consent', 'default', { ... })` 를 **`js`/`config`/`event` 보다 먼저** 호출해야 함.
2. 안 하면 묵시적 denied 모드 → `g/collect` 비콘 차단.
3. 더 까다로운 부분: `default` 없이 `update` 만 호출하면 `update` 도 무시됨 (테스트로 확인).

#111 코드는 `consent default` 자체가 없었음.

## 수정
`frontend/src/lib/analytics.ts` `initAnalytics()`:

```typescript
// dataLayer + gtag shim 먼저
window.dataLayer = window.dataLayer ?? []
window.gtag = function (...args) { window.dataLayer.push(args) }

// ★ js/config 보다 먼저 — Consent Mode v2 default
window.gtag('consent', 'default', {
  ad_storage: 'denied',           // LMS 광고 없음
  ad_user_data: 'denied',
  ad_personalization: 'denied',
  analytics_storage: 'granted',   // 분석만 명시 허용
  functionality_storage: 'granted',
  security_storage: 'granted',
})

window.gtag('js', new Date())
window.gtag('config', GA_ID, { send_page_view: false })
```

부수적 변경:
- `anonymize_ip: true` 제거 — UA 시절 옵션, GA4 에선 무의미 (자동 익명화)
- gtag.js script 주입을 dataLayer 큐 정의 **이후** 로 이동 — 큐가 안전하게 처리되도록

## 회귀 테스트
`analytics.test.ts` 에 #112 회귀 테스트 추가:
- dataLayer 첫 entry 가 `['consent', 'default', { analytics_storage: 'granted', ... }]` 인지
- 두 번째가 `js`, 세 번째가 `config` 인지 (순서 강제)

이 테스트가 깨지면 GA Realtime 0 으로 복귀하니 절대 삭제 금지.

## 동의 정책 메모
- LMS 는 학생 ToS 에 분석 사용 명시 + 로그인 게이트 → analytics_storage 즉시 granted 가 정당.
- 광고는 미사용 → ad_* 는 denied. 만약 추후 광고 도입 시 **별도 동의 UI 필수**.

## 검증
- frontend 149 tests pass · backend smoke 24 tests pass
- Playwright 로 stage·prod 직접 로드 후 `g/collect` 비콘 발사 확인 (다음 배포 시)
