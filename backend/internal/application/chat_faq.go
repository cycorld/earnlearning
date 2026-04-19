// Package application — 챗봇 FAQ shortcut (#090).
//
// 짧은 인사/감사 류 메시지는 LLM 호출 없이 즉시 정해진 응답으로 답변.
// 효과: LLM 슬롯 점유 + 비용 절약, 응답 latency 5초 → 50ms.
//
// 주의: exact match 또는 매우 짧은 문장만 매칭. 절대 부분 매칭 금지
// ("안녕하세요 회사가 뭐예요?" 같은 진짜 질문 가로채면 안 됨).
package application

import (
	"strings"
	"time"

	"github.com/earnlearning/backend/internal/domain/chat"
)

// chatFAQ — 정규화된 입력 → 응답 본문.
// key 는 normalizeFAQ 로 거른 형태 ("안녕하세요!" → "안녕하세요").
var chatFAQ = map[string]string{
	"안녕":         "안녕하세요! 이화여대 창업 수업 LMS(EarnLearning) 조교입니다 😊\n궁금한 점이 있으시면 언제든 편하게 물어봐 주세요.",
	"안녕하세요":      "안녕하세요! 이화여대 창업 수업 LMS(EarnLearning) 조교입니다 😊\n궁금한 점이 있으시면 언제든 편하게 물어봐 주세요.",
	"하이":         "안녕하세요! 이화여대 LMS 조교입니다. 무엇을 도와드릴까요?",
	"hi":         "안녕하세요! 이화여대 LMS 조교입니다. 무엇을 도와드릴까요?",
	"hello":      "안녕하세요! 이화여대 LMS 조교입니다. 무엇을 도와드릴까요?",
	"고마워":        "도움이 됐다면 다행이에요! 또 궁금한 거 있으면 말씀해 주세요 😊",
	"고마워요":       "도움이 됐다면 다행이에요! 또 궁금한 거 있으면 말씀해 주세요 😊",
	"감사합니다":      "도움이 됐다면 다행이에요! 또 궁금한 거 있으면 말씀해 주세요 😊",
	"감사해요":       "도움이 됐다면 다행이에요! 또 궁금한 거 있으면 말씀해 주세요 😊",
	"thanks":     "도움이 됐다면 다행이에요! 또 궁금한 거 있으면 말씀해 주세요 😊",
	"thank you":  "도움이 됐다면 다행이에요! 또 궁금한 거 있으면 말씀해 주세요 😊",
	"잘가":         "다음에 또 봐요! 👋",
	"bye":        "다음에 또 봐요! 👋",
	"수고":         "수고하셨어요! 또 도움 필요하면 불러주세요 😊",
	"수고하세요":      "수고하셨어요! 또 도움 필요하면 불러주세요 😊",
	"테스트":        "테스트 메시지 잘 받았어요! 챗봇이 정상 작동 중입니다 ✅",
}

// normalizeFAQ — 매칭 전 입력 정규화.
// 1) 양쪽 공백 제거
// 2) 끝의 ?!.~ 같은 구두점 / 이모지 (모든 비문자 trailing) 제거
// 3) 소문자
// 4) 너무 길면 (10 자 초과) 매칭 안 함 — 진짜 질문일 가능성 높음
func normalizeFAQ(s string) string {
	s = strings.TrimSpace(s)
	// trailing 비문자 / 구두점 / 이모지 / 공백 모두 제거.
	// 진짜 문자 (한글/영문/숫자) 또는 단어 사이 공백 만나면 멈춤.
	rs := []rune(s)
	for len(rs) > 0 {
		last := rs[len(rs)-1]
		if isAlnum(last) {
			break
		}
		// 공백이면 trim 으로 정리해보고 그 앞이 alnum 이면 멈춤
		if last == ' ' {
			// 만약 공백 앞에 alnum 있으면 단어 사이 공백 — 보존
			if len(rs) >= 2 && isAlnum(rs[len(rs)-2]) {
				break
			}
		}
		rs = rs[:len(rs)-1]
	}
	s = strings.ToLower(strings.TrimSpace(string(rs)))
	if len([]rune(s)) > 10 {
		return ""
	}
	return s
}

func isAlnum(r rune) bool {
	if r >= '0' && r <= '9' {
		return true
	}
	if r >= 'a' && r <= 'z' {
		return true
	}
	if r >= 'A' && r <= 'Z' {
		return true
	}
	// 한글 음절 (가–힣)
	if r >= 0xAC00 && r <= 0xD7A3 {
		return true
	}
	// 한글 자모
	if r >= 0x3131 && r <= 0x318E {
		return true
	}
	return false
}

// lookupFAQ — 입력이 FAQ 에 매칭되면 응답 + true 반환.
func lookupFAQ(message string) (string, bool) {
	key := normalizeFAQ(message)
	if key == "" {
		return "", false
	}
	resp, ok := chatFAQ[key]
	return resp, ok
}

// respondFAQ — FAQ 매칭 시 호출. 즉시 text_delta + done event 발행 + DB 저장.
func (uc *ChatUseCase) respondFAQ(sess *chat.Session, in AskInput, response string, out chan<- AskStreamEvent) {
	emit := func(ev AskStreamEvent) {
		select {
		case out <- ev:
		default: // ctx 취소 등 — 그냥 drop
		}
	}
	// text 즉시 발행
	emit(AskStreamEvent{Type: StreamEventTextDelta, Delta: response})

	// DB 저장 (assistant message, model="faq")
	now := time.Now()
	assistantMsg := &chat.Message{
		SessionID: sess.ID,
		Role:      chat.RoleAssistant,
		Content:   response,
		Model:     "faq",
		CreatedAt: now,
	}
	if _, err := uc.messageRepo.Create(assistantMsg); err == nil {
		_ = uc.sessionRepo.UpdateLastMessageAt(sess.ID, now, 0)
		if strings.TrimSpace(sess.Title) == "" || sess.Title == "새 대화" {
			_ = uc.sessionRepo.UpdateTitle(sess.ID, truncateForTitle(in.Message))
		}
	}
	emit(AskStreamEvent{Type: StreamEventDone, MessageID: assistantMsg.ID, Tokens: 0})
}
