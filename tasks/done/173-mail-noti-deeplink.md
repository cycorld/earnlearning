---
id: 173
title: 메일 수신 알림 클릭 시 해당 메일함·메일로 직행 (딥링크)
priority: high
type: fix
branch: fix/173-mail-noti-deeplink
created: 2026-07-21
---

## 증상
메일 수신 푸시/알림 클릭 → 홈(피드)으로 이동.

## 원인
- 푸시: `webpush.go`가 URL을 `/<reference_type>/<id>` 일반식으로 생성 → `/mail/123` 은 없는 라우트 → catch-all 로 /feed.
- 인앱 알림: `/mail` 매핑이라 메일함까지는 가지만 해당 메일함/메일 직행은 아님.

## 수정
- 푸시 URL·인앱 매핑: `reference_type=mail` → `/mail?open=<email_id>`
- MailboxPage: `?open` 처리 — 메일 상세 조회 → 소속 메일함 자동 선택 + 상세 즉시 표시
