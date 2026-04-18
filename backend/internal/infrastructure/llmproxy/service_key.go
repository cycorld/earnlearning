package llmproxy

import (
	"context"
	"fmt"
	"log"
)

// ProvisionServiceKey 는 챗봇 서비스 전용 sk-stu-* 키를 llm-proxy 에 프로비저닝한다.
// 먼저 캐시 (config.Get) 를 확인하고, 없으면:
//  1. email 로 학생을 찾거나 없으면 생성
//  2. 새 API 키 발급 (admin API)
//  3. 평문 키를 config 에 저장
//
// 반환된 키는 호출 측에서 `Client.SetUserKey()` 로 주입.
//
// config 는 Get/Set 을 제공하는 어떤 store 든 상관없음 (DB 기반 추천).
type ConfigStore interface {
	Get(key string) (string, error)
	Set(key, value string) error
}

const (
	configKeyServiceKey  = "chatbot_service_key"
	serviceStudentEmail  = "chatbot-svc@earnlearning.com"
	serviceStudentName   = "EarnLearning Chatbot"
	serviceStudentAffil  = "EarnLearning LMS"
	serviceStudentNote   = "Server-side chatbot TA (#071). Auto-provisioned."
	serviceKeyLabel      = "chatbot-backend"
)

func ProvisionServiceKey(ctx context.Context, c *Client, cfg ConfigStore) (string, error) {
	// 1. 캐시된 키 재사용
	if cached, err := cfg.Get(configKeyServiceKey); err != nil {
		return "", fmt.Errorf("config.Get: %w", err)
	} else if cached != "" {
		return cached, nil
	}

	log.Printf("[llmproxy] chatbot service key not cached — provisioning...")

	// 2. 서비스 학생이 이미 있는지 확인, 없으면 생성
	student, err := c.FindStudentByEmail(ctx, serviceStudentEmail)
	if err != nil {
		return "", fmt.Errorf("find service student: %w", err)
	}
	var studentID int
	if student == nil {
		s, err := c.CreateStudent(ctx, serviceStudentName, serviceStudentAffil, serviceStudentEmail, serviceStudentNote)
		if err != nil {
			return "", fmt.Errorf("create service student: %w", err)
		}
		studentID = s.ID
		log.Printf("[llmproxy] created service student id=%d", studentID)
	} else {
		studentID = student.ID
	}

	// 3. 기존 활성 키가 있으면 revoke (신선한 키로 교체)
	keys, err := c.ListKeys(ctx, studentID)
	if err != nil {
		return "", fmt.Errorf("list service keys: %w", err)
	}
	for _, k := range keys {
		if k.RevokedAt == "" {
			if err := c.RevokeKey(ctx, k.ID); err != nil {
				log.Printf("[llmproxy] revoke stale service key %d: %v", k.ID, err)
			}
		}
	}

	// 4. 새 키 발급
	issued, err := c.IssueKey(ctx, studentID, serviceKeyLabel)
	if err != nil {
		return "", fmt.Errorf("issue service key: %w", err)
	}
	if issued.Key == "" {
		return "", fmt.Errorf("issued key is empty")
	}

	// 5. 캐시에 저장
	if err := cfg.Set(configKeyServiceKey, issued.Key); err != nil {
		return "", fmt.Errorf("config.Set: %w", err)
	}
	log.Printf("[llmproxy] chatbot service key provisioned (prefix=%s)", issued.Prefix)
	return issued.Key, nil
}
