# 이메일 알림 시스템: AWS SES로 알림을 이메일로도 보내기

**날짜**: 2026-03-16
**태그**: `이메일`, `AWS SES`, `알림`, `IAM`, `DNS`, `DKIM`

## 무엇을 했나요?

기존에 웹 푸시(Push)로만 보내던 알림을 **이메일로도 동시에 발송**하도록 시스템을 확장했습니다. AWS SES(Simple Email Service)를 사용해서 `noreply@earnlearning.com` 주소로 예쁜 HTML 이메일을 보냅니다. 사용자는 프로필 페이지에서 이메일 알림을 켜고 끌 수 있습니다.

## 왜 필요했나요?

푸시 알림은 좋지만 한계가 있습니다:
- 앱을 설치하지 않은 학생은 알림을 받을 수 없음
- 푸시 알림을 거부한 학생도 있음
- 중요한 공지(과제, 투자, 대출 등)는 이메일로도 받고 싶을 수 있음

이메일은 가장 보편적인 알림 채널이라 추가하면 모든 학생에게 안정적으로 알림을 전달할 수 있습니다.

## 어떻게 만들었나요?

### 1단계: AWS SES 도메인 인증

이메일을 보내려면 "이 도메인의 이메일을 보낼 권한이 있다"는 걸 증명해야 합니다. 스팸 방지를 위해 이메일 서비스들이 요구하는 검증 절차입니다.

```bash
# 도메인 인증 요청
aws ses verify-domain-identity --domain earnlearning.com

# DKIM(이메일 위조 방지) 설정
aws ses verify-domain-dkim --domain earnlearning.com
```

이 명령을 실행하면 AWS가 "이 DNS 레코드를 추가하세요"라고 알려줍니다. Cloudflare DNS에 다음 레코드들을 추가했습니다:

| 종류 | 이름 | 용도 |
|------|------|------|
| TXT | `_amazonses.earnlearning.com` | 도메인 소유 증명 |
| CNAME × 3 | `xxx._domainkey.earnlearning.com` | DKIM (이메일 서명) |
| MX | `mail.earnlearning.com` | 반송 메일 처리 |
| TXT | `mail.earnlearning.com` | SPF (발신 서버 인증) |
| TXT | `_dmarc.earnlearning.com` | DMARC (이메일 정책) |

> **💡 왜 이렇게 복잡한가요?**
> 이메일 세계에서는 "보내는 사람이 진짜 그 도메인 소유자인가?"를 여러 단계로 검증합니다. DKIM은 이메일에 디지털 서명을 넣고, SPF는 허용된 서버 목록을 공개하고, DMARC는 검증 실패 시 정책을 정의합니다. 이 세 가지를 모두 설정해야 이메일이 스팸함에 안 빠집니다.

### 2단계: IAM 보안 설정

서버에서 이메일을 보내려면 AWS 자격증명이 필요합니다. 하지만 관리자 계정의 키를 그대로 넣으면 위험합니다. **최소 권한 원칙**에 따라 SES 전용 사용자를 만들었습니다:

```bash
# SES 전용 IAM 유저 생성
aws iam create-user --user-name earnlearning-ses

# 최소 권한 정책: noreply@earnlearning.com에서만 이메일 발송 가능
aws iam put-user-policy --user-name earnlearning-ses \
  --policy-name SES-SendOnly --policy-document '{
    "Statement": [{
      "Effect": "Allow",
      "Action": ["ses:SendEmail", "ses:SendRawEmail"],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "ses:FromAddress": "noreply@earnlearning.com"
        }
      }
    }]
  }'
```

> **💡 최소 권한 원칙이란?**
> 각 프로그램/사용자에게 꼭 필요한 권한만 부여하는 보안 원칙입니다. 만약 이 키가 유출되더라도 `noreply@earnlearning.com`에서 이메일을 보내는 것 외에는 아무것도 할 수 없습니다.

### 3단계: 백엔드 이메일 서비스

Go 코드에서 AWS SDK를 사용해 이메일을 보내는 서비스를 만들었습니다:

```go
// infrastructure/email/ses.go
func (s *SESService) SendEmail(to, subject, htmlBody, textBody string) error {
    input := &sesv2.SendEmailInput{
        FromEmailAddress: &s.fromEmail,  // noreply@earnlearning.com
        Destination: &types.Destination{
            ToAddresses: []string{to},
        },
        Content: &types.EmailContent{...},
    }
    _, err := s.client.SendEmail(ctx, input)
    return err
}
```

기존 `CreateNotification()` 함수에 이메일 발송을 추가했습니다. 알림이 생성되면 **WebSocket + Push + Email** 3채널로 동시 전송됩니다:

```go
func (uc *NotificationUseCase) CreateNotification(...) error {
    // 1. DB에 알림 저장
    // 2. WebSocket으로 실시간 전송
    // 3. Web Push로 푸시 전송
    // 4. Email로 이메일 전송 (새로 추가!)
    if uc.emailService != nil && uc.emailService.IsEnabled() {
        go uc.sendEmailNotification(userID, n)
    }
}
```

### 4단계: 이메일 알림 설정

사용자가 이메일 알림을 끄고 싶을 수 있으므로 설정 기능을 만들었습니다:

- **DB 테이블**: `email_preferences` (user_id, email_enabled)
- **기본값**: 모두 켜짐 (레코드가 없으면 켜짐으로 간주)
- **API**: `GET/PUT /notifications/email/preference`
- **프론트엔드**: 프로필 페이지에 토글 버튼

## 사용한 프롬프트

```
이메일 발송 시스템을 세팅할거야.
aws profile=k 에다가 ses 세팅해줘. 우리 도메인 earnlearning.com 을 세팅하도록 해줘.
푸시 노티와 함께 이메일로도 알림을 줘. 그리고 이메일 알림 켜고 끌 수 있도록 하돼, 기본값은 모두 켜줘.
```

## 배운 점

1. **이메일 인증은 복잡하다**: 도메인 하나로 이메일을 보내려면 TXT, CNAME, MX 등 여러 DNS 레코드가 필요합니다. 이걸 안 하면 이메일이 스팸함으로 갑니다.

2. **IAM 최소 권한 원칙**: AWS 키는 항상 필요한 최소 권한만 부여합니다. 만약 키가 유출되더라도 피해를 최소화할 수 있습니다.

3. **Go의 goroutine으로 비동기 처리**: `go uc.sendEmailNotification(...)` 한 줄로 이메일 발송을 백그라운드에서 처리합니다. 사용자는 알림 생성이 완료될 때까지 이메일이 다 보내질 때까지 기다릴 필요가 없습니다.

4. **기본값 설계의 중요성**: 이메일 알림은 기본 "켜짐"으로 설정했습니다. DB에 레코드가 없으면 켜짐으로 간주하는 방식으로, 기존 사용자들도 별도 마이그레이션 없이 바로 이메일을 받습니다.
