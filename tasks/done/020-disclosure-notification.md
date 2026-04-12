---
id: 020
title: 공시 승인/거절 시 학생 알림 연동
priority: medium
type: feat
branch: feat/disclosure-notification
created: 2026-04-11
---

## 설명
회사 공시가 승인 또는 거절되었을 때 학생(대표)에게 알림을 보내는 기능.
현재 CompanyUsecase에 NotificationUseCase 의존성이 없어서 별도 작업이 필요하다.

## 작업 내용
- CompanyUsecase에 NotificationUseCase 의존성 주입 추가
- 공시 승인 시: "공시가 승인되었습니다. 수익금 XX원이 법인 계좌에 입금되었습니다." 알림
- 공시 거절 시: "공시가 거절되었습니다. 사유: {admin_note}" 알림
- notification entity에 NotifDisclosureApproved, NotifDisclosureRejected 타입 추가
- 프론트엔드 NotificationsPage에 reference_type "disclosure" → URL 매핑 추가
- 프론트엔드 getNotifIcon에 disclosure 관련 아이콘 매핑 추가
