package email

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// ErrSenderDisabled — 발신기가 비활성(FROM_EMAIL 미설정 등)일 때 SendMailFrom 이 반환한다.
// 조용한 성공(nil) 대신 명시적 실패로 처리해 "보냈다"는 거짓 성공 + 보낸편지함 저장을 막는다.
// (주의: SendEmail 알림 경로는 기존대로 비활성 시 무시(nil) 유지 — 여기서 바꾸지 않는다.)
var ErrSenderDisabled = errors.New("메일 발신기가 비활성화되어 있습니다")

type SESService struct {
	client    *sesv2.Client
	fromEmail string
	enabled   bool
}

type Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	FromEmail       string
}

func NewSESService(cfg Config) *SESService {
	if cfg.FromEmail == "" {
		log.Println("email: SES disabled (no FROM_EMAIL configured)")
		return &SESService{enabled: false}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := []func(*awsconfig.LoadOptions) error{}

	if cfg.Region != "" {
		opts = append(opts, awsconfig.WithRegion(cfg.Region))
	}
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	sdkCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		log.Printf("email: failed to load AWS config: %v", err)
		return &SESService{enabled: false}
	}

	client := sesv2.NewFromConfig(sdkCfg)

	fromDisplay := fmt.Sprintf("언러닝 <%s>", cfg.FromEmail)

	log.Printf("email: SES enabled (from=%s, region=%s)", fromDisplay, cfg.Region)
	return &SESService{
		client:    client,
		fromEmail: fromDisplay,
		enabled:   true,
	}
}

func (s *SESService) IsEnabled() bool {
	return s.enabled
}

func (s *SESService) SendEmail(to, subject, htmlBody, textBody string) error {
	if !s.enabled {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	input := &sesv2.SendEmailInput{
		FromEmailAddress: &s.fromEmail,
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data:    &subject,
					Charset: strPtr("UTF-8"),
				},
				Body: &types.Body{},
			},
		},
	}

	if htmlBody != "" {
		input.Content.Simple.Body.Html = &types.Content{
			Data:    &htmlBody,
			Charset: strPtr("UTF-8"),
		}
	}
	if textBody != "" {
		input.Content.Simple.Body.Text = &types.Content{
			Data:    &textBody,
			Charset: strPtr("UTF-8"),
		}
	}

	_, err := s.client.SendEmail(ctx, input)
	if err != nil {
		return fmt.Errorf("ses send email to %s: %w", to, err)
	}

	log.Printf("email: sent to %s subject=%q", to, subject)
	return nil
}

// OutgoingMail — 임의 From 주소로 보내는 발신 메일 (#166 학생 메일함).
// SendEmail 과 달리 From 을 학생 개인 주소로 지정하고 스레딩 헤더를 붙일 수 있다.
type OutgoingMail struct {
	FromDisplay string // "이름 <local@earnlearning.com>"
	To          string
	Subject     string
	TextBody    string
	HTMLBody    string
	InReplyTo   string // 원본 Message-ID (답장 스레딩)
	References  string // 스레드 References 체인
	ReplyTo     string // 미인증 From 폴백 시 Reply-To 로 사용할 학생 주소
}

// SendMailFrom — 학생 개인 주소를 From 으로 메일을 보낸다.
// SES 가 미인증 발신자라 거부(MessageRejected / "Email address is not verified")하면
// 설정된 FromEmail 로 From 을 바꾸고 Reply-To 에 학생 주소를 넣어 1회 재시도한 뒤
// 성공으로 처리한다. 어느 경로로 보냈는지 로그로 남긴다.
func (s *SESService) SendMailFrom(m OutgoingMail) error {
	if !s.enabled {
		// 조용한 성공 금지: 미설정 프로덕션에서 학생에게 거짓 "발송 완료" + 보낸편지함 기록이 남는 것을 막는다.
		return ErrSenderDisabled
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	build := func(from string, replyTo string) *sesv2.SendEmailInput {
		msg := &types.Message{
			Subject: &types.Content{Data: &m.Subject, Charset: strPtr("UTF-8")},
			Body:    &types.Body{},
		}
		if m.HTMLBody != "" {
			msg.Body.Html = &types.Content{Data: &m.HTMLBody, Charset: strPtr("UTF-8")}
		}
		if m.TextBody != "" {
			msg.Body.Text = &types.Content{Data: &m.TextBody, Charset: strPtr("UTF-8")}
		}
		var headers []types.MessageHeader
		if m.InReplyTo != "" {
			headers = append(headers, types.MessageHeader{Name: strPtr("In-Reply-To"), Value: strPtr(m.InReplyTo)})
		}
		if m.References != "" {
			headers = append(headers, types.MessageHeader{Name: strPtr("References"), Value: strPtr(m.References)})
		}
		msg.Headers = headers

		in := &sesv2.SendEmailInput{
			FromEmailAddress: &from,
			Destination:      &types.Destination{ToAddresses: []string{m.To}},
			Content:          &types.EmailContent{Simple: msg},
		}
		if replyTo != "" {
			in.ReplyToAddresses = []string{replyTo}
		}
		return in
	}

	// 1차: 학생 개인 주소를 From 으로.
	_, err := s.client.SendEmail(ctx, build(m.FromDisplay, ""))
	if err == nil {
		log.Printf("email: sent from=%q to=%s subject=%q (student-from)", m.FromDisplay, m.To, m.Subject)
		return nil
	}

	// 미인증 발신자 거부면 설정 From 으로 폴백 재시도.
	if isUnverifiedIdentity(err) {
		replyTo := m.ReplyTo
		_, ferr := s.client.SendEmail(ctx, build(s.fromEmail, replyTo))
		if ferr != nil {
			return fmt.Errorf("ses send (fallback) to %s: %w", m.To, ferr)
		}
		log.Printf("email: sent from=%q to=%s subject=%q (fallback replyto=%s; student-from unverified)", s.fromEmail, m.To, m.Subject, replyTo)
		return nil
	}

	return fmt.Errorf("ses send from %q to %s: %w", m.FromDisplay, m.To, err)
}

// isUnverifiedIdentity — SES 가 미인증 발신자 신원으로 거부했는지 판별.
func isUnverifiedIdentity(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Email address is not verified") ||
		strings.Contains(msg, "MessageRejected")
}

func strPtr(s string) *string {
	return &s
}
