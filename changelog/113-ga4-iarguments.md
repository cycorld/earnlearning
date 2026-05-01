# 113. GA4 gtag shim IArguments 복원 (#112 후속 핫픽스)

**날짜**: 2026-05-01
**태그**: 핫픽스, GA4, dataLayer, 분석

## 증상
#112 (consent default 추가) 배포 후 Playwright stage 검증 → `g/collect` 비콘 **여전히 0건**. dataLayer 순서/내용은 정상 (entry 0 = consent default). 그런데:

```
google_tag_data.ics:
  usedDefault: false   ← consent default 가 무시됨
  usedImplicit: true
tidrState: 2            ← consent 차단으로 측정 비활성
```

## 원인 — IArguments vs Array
표준 Google gtag 스니펫:
```js
function gtag() { dataLayer.push(arguments) }   // arguments = IArguments (array-like, NOT Array)
```

#111 코드:
```ts
window.gtag = function (...args) { dataLayer.push(args) }  // args = real Array
```

gtag.js 는 dataLayer entry 가 **IArguments 일 때만** gtag 명령으로 인식. 진짜 `Array.isArray() === true` 면 generic "data layer push" 로 분류하고 명령 무시. 결과:
- `consent default` 명령 → 무시 → ICS implicit denied 유지
- `js`/`config`/`event page_view` 모두 무시 → collect 비콘 0건

Playwright 로 같은 페이지에서 표준 패턴 함수를 즉석 정의해서 `gtag('consent', 'update', { granted... })` 호출하니 **즉시** `ics.active=true`, `usedUpdate=true` 전환됨 → 확정.

## 수정
`frontend/src/lib/analytics.ts`:
```ts
window.gtag = function () {
  window.dataLayer.push(arguments as unknown as unknown[])
}
```

표준 Google 스니펫 패턴 그대로. 코드 상단 주석에 "변경 금지" 경고 박아둠.

## 회귀 테스트
`Array.isArray(dataLayer[0]) === false` 강제. Array 였으면 fail. 이 테스트가 깨지면 Realtime 0 회귀.

## 학습 포인트
- 표준 Google 스니펫의 `dataLayer.push(arguments)` 는 우연이 아님. **gtag.js 가 IArguments 객체를 sentinel 로 사용해서 gtag 명령과 일반 데이터 레이어 push 를 구별함**.
- TypeScript / 모던 JS 스타일로 `(...args)` 로 받으면 안 됨. `function() { ... arguments ... }` 로 유지 필수.
- ESLint `prefer-rest-params` 규칙은 이 경우 **반드시 disable**.

## 다음
- stage·prod 재배포 후 Playwright 로 `g/collect` 비콘 발사 재검증.
- GA Realtime dashboard 활성 사용자 표시 확인.
