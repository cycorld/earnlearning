# 079. 챗봇 FAB 가 모바일 하단 메뉴에 가려지던 문제 수정

**날짜**: 2026-04-19
**태그**: 챗봇, UI, 모바일, 버그수정

## 증상
모바일에서 우하단 챗봇 말풍선 버튼(FAB) 이 하단 네비게이션 바에 일부 가려져
탭하기가 어려웠음. iPhone 처럼 화면 하단 홈바(safe area) 가 있는 기기에서 특히 심함.

## 원인
- BottomNav: `fixed bottom-0`, `h-16`(64px), `z-50`
- ChatDock FAB: `bottom-20`(80px), `z-40` — 80-64=**16px 만 떠 있던 셈**
- z-index 도 FAB(40) < Nav(50) 라 겹치면 Nav 가 위로 올라옴
- iOS safe-area-inset-bottom (홈바 약 34px) 까지 고려하면 사실상 가려짐

## 수정 (frontend/src/components/chat/ChatDock.tsx)
```tsx
// before
'fixed bottom-20 right-4 z-40 ... sm:bottom-6'

// after
'fixed bottom-[calc(5rem_+_env(safe-area-inset-bottom))] right-4 z-50 ... sm:bottom-6'
```

- `bottom-20`(80px) → `calc(80px + safe-area-inset-bottom)` — 홈바가 있는 기기에서도
  최소 80px 위에 위치
- `z-40` → `z-50` — BottomNav 와 동일 stacking, 시각적으로 항상 위
- 데스크탑(`sm:`) 은 기존 `bottom-6` 유지 — 데스크탑엔 BottomNav 자체가 없음

## 배운 점
1. **iOS safe-area 는 필수** — `env(safe-area-inset-bottom)` 으로 홈바 영역 자동 보정.
2. **z-index 는 절대값이 아니라 비교값** — 겹치는 요소들은 같거나 더 큰 z 가 위.
3. **Tailwind arbitrary 값에서 공백** — `_` 로 escape (`calc(5rem_+_env(...))`).
