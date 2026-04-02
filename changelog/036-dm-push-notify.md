# 036. DM 수신 시 웹 푸시 + 이메일 알림

> **날짜**: 2026-04-03
> **태그**: `DM`, `푸시알림`, `이메일`

## 무엇을 했나요?

DM을 받으면 웹 푸시 알림과 이메일 알림이 전송되도록 했습니다. 알림을 클릭하면 해당 대화 페이지로 바로 이동합니다.

## 어떻게 만들었나요?

- `NotifNewDM` 타입을 `PushEligibleTypes`에 추가하여 기존 알림 인프라(웹 푸시 + 이메일) 재사용
- `DMUseCase.SendMessage`에서 `CreateNotification` 호출로 알림 생성
- 알림의 `reference_type: "dm"` → 클릭 시 `/messages/:senderID`로 이동
