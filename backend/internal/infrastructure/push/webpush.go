package push

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/earnlearning/backend/internal/domain/notification"
)

type PushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Icon  string `json:"icon,omitempty"`
	URL   string `json:"url,omitempty"`
}

type WebPushService struct {
	vapidPublicKey  string
	vapidPrivateKey string
	vapidSubject    string
	notifRepo       notification.Repository
}

func NewWebPushService(publicKey, privateKey, subject string, repo notification.Repository) *WebPushService {
	return &WebPushService{
		vapidPublicKey:  publicKey,
		vapidPrivateKey: privateKey,
		vapidSubject:    subject,
		notifRepo:       repo,
	}
}

func (s *WebPushService) SendToUser(userID int, payload PushPayload) {
	if s.vapidPublicKey == "" || s.vapidPrivateKey == "" {
		return // VAPID keys not configured, skip push
	}

	subs, err := s.notifRepo.GetSubscriptionsByUserID(userID)
	if err != nil {
		log.Printf("push: get subscriptions for user %d: %v", userID, err)
		return
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("push: marshal payload: %v", err)
		return
	}

	for _, sub := range subs {
		go s.sendToSubscription(sub, body)
	}
}

func (s *WebPushService) sendToSubscription(sub *notification.PushSubscription, body []byte) {
	wpSub := &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}

	// webpush-go adds "mailto:" prefix automatically, so strip it if present
	subscriber := strings.TrimPrefix(s.vapidSubject, "mailto:")
	resp, err := webpush.SendNotification(body, wpSub, &webpush.Options{
		Subscriber:      subscriber,
		VAPIDPublicKey:  s.vapidPublicKey,
		VAPIDPrivateKey: s.vapidPrivateKey,
		TTL:             60,
	})
	if err != nil {
		log.Printf("push: send to %s: %v", sub.Endpoint, err)
		return
	}
	defer resp.Body.Close()

	// Auto-delete subscription on 410 Gone (subscription expired/unsubscribed)
	if resp.StatusCode == http.StatusGone {
		log.Printf("push: subscription gone, deleting: %s", sub.Endpoint)
		if err := s.notifRepo.DeleteSubscriptionByEndpoint(sub.Endpoint); err != nil {
			log.Printf("push: delete expired subscription: %v", err)
		}
	} else if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("push: send failed with status %d for endpoint %s body=%s", resp.StatusCode, sub.Endpoint, string(body))
	}
}

func (s *WebPushService) GetVAPIDPublicKey() string {
	return s.vapidPublicKey
}

func (s *WebPushService) FormatPayload(n *notification.Notification) PushPayload {
	return PushPayload{
		Title: n.Title,
		Body:  n.Body,
		URL:   fmt.Sprintf("/%s/%d", n.ReferenceType, n.ReferenceID),
	}
}
