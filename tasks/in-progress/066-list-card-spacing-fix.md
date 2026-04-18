---
id: 066
title: 리스트 카드 간격 추가 정리 (사용자 피드백)
priority: medium
type: fix
branch: fix/list-card-spacing
created: 2026-04-18
---

## 배경
#065 디자인 시스템 리프레시 배포 후 사용자 피드백:
> "카드들이 다닥다닥 붙은 페이지가 많이 보인다"

Stage에서 실제 브라우저 확인 시 알림/메시지/칸반 페이지의 카드 리스트 간격이
space-y-2 (8px)로 좁게 남아있었음 (#065 검수에서 놓침).

## 변경 사항
- `NotificationsPage.tsx:217` space-y-2 → space-y-4
- `MessagesPage.tsx:81` space-y-2 → space-y-4
- `AdminTasksPage.tsx:100` space-y-2 → space-y-4

리스트 카드 간격 기준을 **space-y-4 (16px)** 로 최종 통일.
