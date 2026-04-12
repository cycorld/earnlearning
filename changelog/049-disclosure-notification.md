# 049. 공시 승인/거절 시 학생 알림 연동

> **날짜**: 2026-04-12
> **태그**: `feat`, `알림`, `공시`

## 무엇을 했나요?

관리자가 회사 공시를 승인하거나 거절하면, 회사 대표(학생)에게 자동으로 알림이 전송되도록 했습니다.

## 어떻게 만들었나요?

기존 다른 유스케이스(투자, 거래소, 게시글 등)에서 사용하는 `SetNotificationUseCase` 패턴을 그대로 따랐습니다.

1. `notification` 도메인에 `NotifDisclosureApproved`, `NotifDisclosureRejected` 타입 추가
2. `CompanyUsecase`에 `SetNotificationUseCase` 메서드 추가
3. `ApproveDisclosure`에서 승인 후 알림 전송 (수익금 금액 + 코멘트 포함)
4. `RejectDisclosure`에서 거절 후 알림 전송 (거절 사유 포함)
5. 프론트엔드 알림 아이콘에 `disclosure_approved/rejected` 매핑 추가

## 배운 점

- `SetNotificationUseCase` 패턴은 순환 의존성을 방지하면서 알림을 연동하는 깔끔한 방법입니다
- `reference_type`을 "company"로 설정하면 기존 알림 클릭 매핑(`/company/:id`)을 재사용할 수 있습니다
