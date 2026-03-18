package email

import (
	"context"
	"fmt"
	"log"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

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

func strPtr(s string) *string {
	return &s
}
