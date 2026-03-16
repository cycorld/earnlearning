# 공지 알림 전송 옵션: 푸시/이메일을 보낼지 선택하기

**날짜**: 2026-03-16
**태그**: `알림`, `공지`, `관리자`, `UX`

## 무엇을 했나요?

관리자가 공지를 보낼 때 **푸시 알림과 이메일도 함께 보낼지** 선택할 수 있는 체크박스를 추가했습니다.

## 왜 필요했나요?

모든 공지가 푸시+이메일까지 갈 필요는 없습니다. 간단한 안내는 앱 내 알림만으로 충분하고, 중요한 공지(과제 마감, 시스템 점검 등)만 푸시+이메일로 보내고 싶을 수 있습니다.

## 어떻게 만들었나요?

### 백엔드

기존 `CreateNotification()`은 항상 Push+Email을 보냈습니다. 여기에 WebSocket만 보내는 `CreateNotificationQuiet()` 메서드를 추가하고, `SendAnnouncement()`에서 `sendNotify` 플래그에 따라 분기합니다.

```go
if sendNotify {
    err = uc.CreateNotification(uid, ...)    // Push + Email + WebSocket
} else {
    err = uc.CreateNotificationQuiet(uid, ...) // WebSocket만
}
```

API에는 `send_notify` 필드를 추가했고, 생략하면 기본값 `true`로 기존 동작과 하위호환됩니다.

### 프론트엔드

공지 작성 폼에 체크박스를 추가했습니다. 체크 상태에 따라 확인 팝업 문구도 달라집니다.

## 사용한 프롬프트

```
관리자가 공지글을 작성할 때, 이메일(+푸시) 알림을 보낼지 말지 결정할 수 있도록 해줘.
```

## 배운 점

1. **기본값으로 하위호환**: `send_notify` 필드를 포인터(`*bool`)로 받아서, 기존 API 호출(필드 없음)은 자동으로 `true`로 처리됩니다.
2. **코드 재사용**: 기존 `CreateNotification()`을 복제하지 않고, Push/Email 발송 부분만 빠진 `CreateNotificationQuiet()`를 별도로 만들었습니다.
