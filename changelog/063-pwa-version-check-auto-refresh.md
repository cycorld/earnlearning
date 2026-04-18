---
slug: 063-pwa-version-check-auto-refresh
title: 같은 신버전 N회 연속 감지 시 자동 새로고침 (stuck 구버전 방어)
date: 2026-04-18
tags: [feat, PWA, 버전체크, 자동리프레시, TDD, 회귀테스트]
---

# 같은 신버전 N회 연속 감지 시 자동 새로고침

## 왜 필요했는가

`useVersionCheck` 훅은 60초마다 `/api/version` 을 폴링해서 새 버전을 감지하면 **토스트를 띄우고 사용자가 "지금 새로고침" 버튼을 눌러야** `forceRefresh()` 가 실행됩니다.

이 설계는 "입력 중 데이터 유실"을 막는 데 좋지만 두 가지 문제가 있었어요:

1. **토스트를 놓치면 구버전 계속 사용.** 모바일/PWA 환경에서 토스트는 쉽게 지나가고, 사용자가 백그라운드에 앱을 둔 채 며칠씩 구버전을 쓰는 케이스가 생김.
2. **토스트 중복 방지 플래그(`toastShown`) 가 사용자를 구버전에 가둠.** 한 번 토스트가 떴다가 자동 사라지면 다음에는 `toastShown=true` 때문에 재노출 안 됨 → 알림 없이 며칠.

실제로 #018 이전 빌드의 PWA 가 몇 주째 스테이지에 남아있던 사례(#026 티켓)를 보면, 서버 푸시(#027) 가 없는 한 **클라이언트가 스스로 빠져나올 수 있는 마지막 보루**가 필요합니다.

## 무엇을 했는가

`frontend/src/hooks/use-version-check.ts` 에 "연속 감지 카운트" 로직을 추가했습니다.

### 핵심 규칙

```ts
const AUTO_REFRESH_AFTER_DETECTIONS = 3
let sameVersionCount = 0
let lastDetectedVersion: string | null = null
let dismissedForVersion: string | null = null
```

폴링(또는 route 변경)마다 `checkAndNotify` 내부에서:

1. 서버 버전이 임베드와 같으면 **카운터 리셋** (롤백 대비).
2. 신버전이 이전 감지와 같으면 `sameVersionCount++`, 다르면 `sameVersionCount=1` + 과거 dismiss 기록 무효화.
3. `sameVersionCount >= 3` + 사용자가 명시적 dismiss 안 함 + `isSafeToReload()` 가 true → `forceRefresh()` 즉시 실행.
4. 그 외에는 기존 토스트 로직.

### `isSafeToReload()` — 데이터 유실 방지 가드

```ts
function isSafeToReload(): boolean {
  const active = document.activeElement as HTMLElement | null
  if (active) {
    const tag = active.tagName
    if (tag === 'INPUT' || tag === 'TEXTAREA' || active.isContentEditable) {
      return false
    }
  }
  if (document.querySelector('[role="dialog"][data-state="open"]')) {
    return false
  }
  return true
}
```

- 입력 필드 포커스 = 사용자가 글 쓰는 중 → 강제 리로드 보류
- shadcn/Radix Dialog 가 열림 (`data-state="open"`) = 결제/공시 작성 같은 워크플로 진행 중 → 보류

불안전 상태에서는 자동 refresh 가 "skip" 되고 기존 토스트만 뜹니다. 사용자가 입력을 끝내고 안전한 화면으로 돌아가면 다음 폴링에서 (카운트는 유지되어 있으므로) 자동 refresh 가 즉시 발동.

### Dismiss 기록 — 사용자 의사 존중

토스트의 `onDismiss` 콜백에서 `dismissedForVersion = serverVersion` 을 기록합니다. 이후 같은 버전에 대해서는 아무리 많이 감지되어도 자동 refresh 하지 않아요. "나중에 할게" 의사로 해석.

단, **다른 신버전이 등장하면** dismiss 기록은 무효화됩니다 (새 버전은 새 기회).

## TDD

단위 테스트를 먼저 작성했고 7건이 추가됐습니다 (`use-version-check.test.tsx`):

- ✅ 3회 연속 감지 시 `window.location.replace` 호출
- ✅ 2회까지는 자동 리프레시 안 함 (임계치 미만)
- ✅ input 포커스 상태에서는 3회여도 자동 리프레시 안 함
- ✅ dialog 열린 상태에서도 자동 리프레시 안 함
- ✅ 사용자 dismiss 후에는 같은 버전 자동 리프레시 제외
- ✅ 다른 신버전 등장 시 카운터 초기화
- ✅ 서버 롤백(버전 동일화) 시 카운터 리셋

`__testing` 네임스페이스를 export 해서 내부 상태 리셋 + `checkAndNotify` 직접 호출이 가능하도록 했습니다. 프로덕션 코드에서는 사용 금지(주석으로 명시). 테스트를 깔끔하게 만들기 위한 최소한의 트레이드오프.

## 사용한 프롬프트

```
#27, # 28 은 이미 반영된거 아닌가? 아직 안되었으면 반영해줘.
```

AI가 백엔드/프론트 모두 grep 으로 실제 반영 여부를 확인 → 둘 다 미반영을 확인 → 규모가 작은 #028 부터 진행. #027(WS force-reload) 은 이후 PR 로 분리.

## 배운 점

- **"토스트를 띄우기만 하는 알림"은 실제로는 놓치는 경우가 많다.** 특히 PWA 백그라운드 환경에서는 사용자가 포커스를 거의 안 둬서 체크 자체가 적게 발생. 마지막 보루로 자동 행동이 필요.
- **자동 행동은 반드시 "안전 가드"를 동반해야 한다.** 입력 중 리로드는 최악의 UX — `isSafeToReload` 같은 가드가 필수.
- **사용자 의사 존중 ≠ 영원히 무시.** Dismiss 는 "이 버전은 나중에"로 해석하고, 다음 신버전에서는 다시 기회를 준다. 영구 무시로 만들면 또 stuck 됨.
- **모듈 상태가 있는 훅은 테스트 시 리셋 메커니즘이 필요하다.** `__testing.resetState()` 패턴이 가장 간단. Pattern-level 정교함보다는 테스트 가독성 우선.

## 관련 티켓

- #028 (backlog → done) — 이 PR
- #027 — WS force-reload broadcast (다음 PR)
- #018 — PWA 자동 업데이트 원본 훅
- #026 — index.html no-cache (전송 레이어 방어, 상보적)
