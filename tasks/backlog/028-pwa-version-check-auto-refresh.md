---
id: 028
title: version-check 훅에서 사용자 무개입 자동 refresh 옵션 추가
priority: low
type: feat
branch: feat/pwa-version-check-auto-refresh
created: 2026-04-13
---

## 배경

`use-version-check.ts` 는 새 버전을 감지하면 토스트를 띄우고 사용자가 "지금 새로고침" 버튼을 눌러야 `forceRefresh()` 가 실행된다.

이 설계는 작업 중 데이터 유실을 방지하지만 두 가지 문제가 있다:
1. 사용자가 토스트를 놓치면 구버전을 계속 사용
2. 토스트를 의도적으로 무시하면 수시간~수일간 구버전 사용
3. 특히 PWA를 백그라운드에 깔아두는 사용자는 포커스 얻을 때만 체크함

## 작업 내용

`use-version-check.ts` 에 다음 규칙 추가:

### 1. N회 연속 같은 신규 버전 감지 시 자동 refresh
```ts
const AUTO_REFRESH_AFTER_DETECTIONS = 3
let sameVersionCount = 0
let lastDetectedVersion: string | null = null

// checkAndNotify 내부에서
if (serverVersion !== embeddedVersion()) {
  if (serverVersion === lastDetectedVersion) {
    sameVersionCount++
  } else {
    lastDetectedVersion = serverVersion
    sameVersionCount = 1
  }
  
  if (sameVersionCount >= AUTO_REFRESH_AFTER_DETECTIONS) {
    // 사용자가 이미 3번(약 3분) 토스트를 봤는데도 안 눌렀음
    // 입력 중이 아닌지 확인 후 자동 refresh
    if (isSafeToReload()) {
      void forceRefresh()
      return
    }
  }
  // ...기존 토스트 로직
}

function isSafeToReload(): boolean {
  // 활성 textarea/input에 포커스 있으면 불안전
  const active = document.activeElement
  if (active && (active.tagName === 'INPUT' || active.tagName === 'TEXTAREA' 
                 || (active as HTMLElement).isContentEditable)) {
    return false
  }
  // 모달/다이얼로그가 열려있으면 불안전 (shadcn dialog의 data-state="open" 확인)
  if (document.querySelector('[role="dialog"][data-state="open"]')) {
    return false
  }
  return true
}
```

### 2. 사용자가 토스트를 명시적으로 dismiss하면 해당 버전에 대해서는 자동 refresh 안 함
"나중에 할게" 의사로 해석. 다음 새 버전이 나올 때까지 대기.

### 3. 옵션: env flag로 자동 refresh 활성/비활성
```ts
const AUTO_REFRESH_ENABLED = __AUTO_REFRESH__ ?? true
```
vite define으로 빌드 시 제어. 수업 중에는 끄는 옵션 등.

## UX 고려

- **카운트다운 UI**: 토스트에 "10초 후 자동으로 새로고침됩니다 [취소]" 표시
- **취소 시**: 해당 버전에 대해 자동 refresh 안 함, 다음 새 버전에서 다시 카운트 시작
- **로그**: "3회 연속 감지 → 자동 refresh 실행" 을 console.info 로 남겨 디버깅 용이

## 검증

- [ ] 스테이지에서 수동으로 version 숫자를 올리고 PWA 열어두고 3분 기다리기
- [ ] 3회 폴링 후 자동 refresh 확인
- [ ] Input 포커스 상태에서는 refresh 안 되는지 확인
- [ ] Dialog 열려있을 때 refresh 안 되는지 확인
- [ ] 토스트 dismiss 시 자동 refresh 취소되는지 확인
- [ ] 새 버전 다시 나오면 카운트 초기화되는지 확인

## 한계 및 트레이드오프

- 자동 refresh는 **구버전에 이 로직이 먼저 배포되어 있어야** 동작한다. 즉 이 기능은 미래 stuck 케이스에만 방어적으로 작동하고, 현재 stuck된 구버전을 살릴 수는 없다 (그건 #026이 담당).
- Refresh 타이밍이 애매하면 UX 저해 가능 → `isSafeToReload` 로직을 충분히 보수적으로.

## 관련

- #018 PWA 자동 업데이트 (선행, 이 훅의 원형)
- #026 index.html no-cache (전송 레이어 방어 — 이 티켓은 클라이언트 로직 방어)
- #027 WS force-reload broadcast (서버 push 방어 — 상보적)

세 티켓(#026, #027, #028)이 각각 다른 레이어에서 동일 문제를 방어하므로 모두 구현하는 게 이상적.
