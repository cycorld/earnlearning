---
id: 113
title: GA4 gtag shim 이 IArguments 대신 Array push — collect 비콘 0건 (#112 후속)
priority: high
type: fix
branch: fix/ga4-iarguments
created: 2026-05-01
---

## 배경
#112 (consent default 추가) 배포 후에도 Playwright stage 검증에서 `g/collect` 비콘 여전히 0건. `dataLayer` 순서·내용 정상 (entry 0 = consent default). 그러나 `google_tag_data.ics`:
- `usedDefault: false` ← consent default 가 무시됨
- `usedImplicit: true`
- `tidrState: 2` (consent 차단)

## 원인 — IArguments vs Array
표준 Google gtag 스니펫:
```js
function gtag(){dataLayer.push(arguments);}  // arguments = IArguments (array-like)
```

#111 코드:
```ts
window.gtag = function (...args) { dataLayer.push(args); }  // args = real Array
```

gtag.js 는 dataLayer entry 가 IArguments 일 때만 gtag 명령으로 해석. 진짜 Array 면 generic "data layer push" 로 처리하고 명령은 무시. 결과적으로 `consent default` / `js` / `config` / `event page_view` 모두 dataLayer 에 들어가지만 gtag.js 가 명령으로 인식 못 함 → ICS 미적용 → collect 비콘 차단.

Playwright 로 직접 검증: 같은 페이지에서 `function gtag2(){dataLayer.push(arguments);}` 정의하고 `gtag2('consent', 'update', {...granted})` 호출하니 즉시 `ics.active=true`, `usedUpdate=true` 로 전환됨.

## 작업
- `frontend/src/lib/analytics.ts`: gtag shim 을 표준 패턴으로 복원 (IArguments push)
- 회귀 테스트: `Array.isArray(dataLayer[0]) === false` 강제 (Array 였으면 fail)

## 미포함
- 다른 변경 없음. consent · 측정 ID · 라우터 hook 그대로.
