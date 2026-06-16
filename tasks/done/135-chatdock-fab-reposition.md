---
id: 135
title: 챗봇 FAB 충돌 해결 — 컨텍스트 숨김 + 6모서리 드래그
priority: medium
type: feat
branch: feat/135-chatdock-fab-reposition
created: 2026-06-16
---

## 배경
챗봇 플로팅 버튼(FAB)이 전역 `fixed bottom-right z-50`이라, 페이지 자체 하단 입력창의 전송버튼을 가림.
대표 충돌: DM `ConversationPage`(`/messages/:userId`) 하단 폼 우측 Send 버튼. 댓글·기타 composer도 동일.

## A. 컨텍스트 기반 숨김 (근본 픽스)
- DM 대화 라우트(`/messages/:userId`)에서 FAB 미표시.
- 챗봇 외부 input/textarea/contenteditable 포커스 시 FAB fade + `pointer-events-none` (타이핑 중엔 안 가림).

## C. 6모서리 드래그 이동
- 앵커 6곳: 상좌/상우, 중좌/중우, 하좌/하우. 기본 하우(현행).
- 드래그로 자유 이동 → 놓으면 가장 가까운 앵커로 스냅. 위치 localStorage 영속.
- 탭 vs 드래그 임계값(8px)으로 구분 — 탭은 챗봇 열기, 드래그는 이동.
- safe-area / 상단바 / 바텀네브 회피.

## TDD
- `nearestAnchor(release, w, h)` 순수 함수: 릴리즈 좌표 → 6앵커 중 최근접 반환.
- 앵커 load/save (localStorage) 라운드트립.
- 라우트 숨김: `/messages/123`에서 FAB 미렌더 회귀 테스트.
