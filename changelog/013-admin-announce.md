# 공지 알림 + iOS 웹 푸시: 삽질의 기록

**날짜**: 2026-03-15
**태그**: `알림`, `푸시`, `관리자`, `WebPush`, `iOS`, `VAPID`, `트러블슈팅`

## 무엇을 했나요?

관리자가 전체 유저에게 **공지 알림**을 보낼 수 있는 기능을 만들었습니다. 알림은 앱 내 알림 + WebSocket 실시간 알림 + **웹 푸시 알림**(PC Chrome + iOS Safari PWA)으로 동시에 전송됩니다.

## 왜 필요했나요?

기존에는 관리자가 학생들에게 공지를 보낼 방법이 없었습니다. 수업 안내, 과제 마감 알림, 시스템 점검 공지 등을 **실시간으로** 전달할 수 있어야 했습니다. 특히 학생들이 앱을 열지 않아도 핸드폰에서 푸시 알림을 받을 수 있어야 했습니다.

## 어떻게 만들었나요?

### 1. 백엔드 — 공지 알림 API

`POST /api/admin/notifications/announce` 엔드포인트를 추가했습니다.

```json
{
  "title": "공지 제목",
  "body": "공지 내용",
  "user_ids": []  // 비어있으면 전체 승인 유저에게 전송
}
```

내부적으로는 기존 `CreateNotification` 파이프라인을 재활용합니다:
1. DB에 알림 저장
2. WebSocket으로 실시간 전달
3. Web Push로 푸시 알림 전송

### 2. 프론트엔드 — 공지 페이지 + 푸시 구독 개선

- `/admin/announce` 공지 알림 관리 페이지 추가
- 푸시 구독 API 경로 수정 (`/push/...` → `/notifications/push/...`)
- 푸시 토글에 **로딩 스피너, 에러 메시지, 타임아웃(15초)** 추가

### 3. iOS 웹 푸시 — 삽질의 핵심

이 기능의 80%는 iOS에서 웹 푸시를 동작시키는 데 들어갔습니다. 겪은 문제와 해결 과정을 정리합니다.

#### 문제 1: 푸시 구독 API 경로 불일치

**증상**: 프로필에서 "푸시 알림 켜기" 버튼을 눌러도 아무 반응 없음

**원인**: 프론트엔드가 `/push/subscribe`로 호출하는데 실제 API는 `/notifications/push/subscribe`

**해결**: API 경로를 올바르게 수정. 에러가 `catch`에서 조용히 무시되고 있어서 발견이 늦었습니다. → silent fail 패턴 제거

**교훈**: `catch { return false }` 같은 코드는 디버깅을 불가능하게 만듭니다. 에러는 반드시 로깅하거나 사용자에게 보여줘야 합니다.

#### 문제 2: Apple Push 403 에러 — VAPID_SUBJECT

**증상**: 구독은 성공하는데 푸시 전송 시 `403 Forbidden`

**원인**: `VAPID_SUBJECT`가 `mailto:admin@example.com`(기본값)으로 설정되어 있었음. Apple은 유효하지 않은 이메일의 VAPID JWT를 거부합니다.

**해결**: `VAPID_SUBJECT`를 실제 이메일(`mailto:cyc@ewha.ac.kr`)로 변경

**교훈**: 환경변수 기본값(`default`)을 실제로 유효한 값으로 설정하는 것이 중요합니다.

#### 문제 3: Apple Push 403 → BadJwtToken

**증상**: VAPID_SUBJECT를 변경했는데도 여전히 403, Apple 응답: `{"reason":"BadJwtToken"}`

**원인**: webpush-go 라이브러리가 `Subscriber` 필드에 자동으로 `mailto:` 프리픽스를 붙이는데, 설정값에 이미 `mailto:`가 포함되어 있어서 **`mailto:mailto:cyc@ewha.ac.kr`**이 되어 JWT가 무효화됨

```go
// webpush-go 내부 코드 (vapid.go:76-78)
if !strings.HasPrefix(subscriber, "https:") {
    subscriber = "mailto:" + subscriber  // ← 이미 mailto:가 있으면 중복!
}
```

**해결**: 전송 시 `strings.TrimPrefix(s.vapidSubject, "mailto:")`로 프리픽스 제거

**교훈**: 라이브러리가 내부에서 뭘 하는지 소스 코드를 직접 확인해야 합니다. 403 에러의 응답 body를 로깅하도록 추가한 것이 원인 파악의 결정적 단서였습니다.

#### 문제 4: iOS에서 재구독 시 hang

**증상**: 푸시 끄기 → 켜기 시 "처리 중..."이 무한 로딩

**원인**: iOS Safari에서 `unsubscribe()` 직후 바로 `subscribe()`를 호출하면 hang 발생

**해결**: unsubscribe 후 500ms 딜레이 추가 + 15초 타임아웃으로 무한 로딩 방지

#### 문제 5: VAPID 키 변경 후 기존 구독 무효

**증상**: 새 VAPID 키로 변경했는데 Apple 응답: `{"reason":"VapidPkHashMismatch"}`

**원인**: 기존 구독은 이전 VAPID public key로 생성된 것이라, 새 키로 전송하면 해시 불일치

**해결**: subscribe 시 기존 구독을 `unsubscribe()`로 완전히 해제하고 새로 구독하도록 변경. stale 구독은 DB에서 정리.

## 사용한 프롬프트

> admin 기능에 공지 알림 보낼 수 있도록 만들어줘

> 나는 푸시를 받아보고 싶은건데

> 푸시알림 켜기가 안되는건 왜그럴까?

> 핸드폰에서는 pwa 알림이 안뜨는 이유가 뭘까?

> 알림 안옴. 켜고 끌 때 최소한 펜딩이라도 보여줘야

## 배운 점

- **iOS 웹 푸시는 까다롭습니다**: PWA 설치 필수, Safari에서만 동작, VAPID 규격이 엄격합니다. 특히 Apple은 JWT의 `sub` claim을 정확히 검증합니다.
- **Silent fail은 독입니다**: `catch { return false }` 패턴은 에러를 숨깁니다. 사용자에게 피드백을 주고, 개발자에게 로그를 남겨야 합니다.
- **에러 응답 body를 꼭 확인하세요**: HTTP 403이라는 상태 코드만으로는 원인을 알 수 없습니다. Apple이 보내주는 `BadJwtToken`, `VapidPkHashMismatch` 같은 메시지가 해결의 열쇠입니다.
- **라이브러리 소스를 읽으세요**: webpush-go가 `mailto:`를 자동으로 붙이는 동작은 문서에 나와있지 않았습니다. 소스 코드를 직접 읽어서 원인을 찾았습니다.
- **기존 코드 재활용**: 공지 알림 기능 자체는 기존 `CreateNotification` 파이프라인을 그대로 활용해서 빠르게 만들 수 있었습니다.
