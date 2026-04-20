---
id: 107
title: PWA 당겨 내려 리프레시 (pull-to-refresh)
priority: medium
type: feat
branch: feat/pwa-pull-to-refresh
created: 2026-04-20
---

## 배경
PWA standalone 모드에서는 브라우저 기본 pull-to-refresh 가 동작 안 함 → 학생이 앱처럼 설치해 쓸 때 새로고침이 어려움.

## 작업
- 재사용 가능한 `PullToRefresh` 컴포넌트 (touch events, 임계값, 스피너 표시)
- Layout 에 래핑 → 모든 페이지에 기본 활성
- Trigger: `window.location.reload()` (단순, 확실. react-query 안 씀)
- 조건: scrollY === 0 + 아래로 pull
- PWA 설치 여부 무관하게 동작
- 데스크톱에서는 touch 없으므로 noop
