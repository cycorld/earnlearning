package application

import "testing"

func TestLookupFAQ(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantHit   bool
	}{
		{"안녕 정확", "안녕", true},
		{"안녕 + ?", "안녕?", true},
		{"안녕!! 이모지", "안녕!! 😊", true},
		{"안녕하세요", "안녕하세요", true},
		{"공백 trim", "  안녕   ", true},
		{"hi 영문", "hi", true},
		{"HI 대문자", "HI", true},
		{"thank you 공백 OK", "thank you", true},
		{"thanks!", "thanks!", true},
		{"고마워요", "고마워요", true},
		// 진짜 질문은 매칭 안 됨
		{"안녕 다음에 본문", "안녕하세요 회사가 뭐예요?", false},
		{"긴 인사", "안녕! 오늘 날씨 좋네요", false},
		{"질문은 매칭 X", "지갑 잔액 알려줘", false},
		{"LMS 관련 질문", "주주총회 가결 기준?", false},
		{"빈 문자열", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, ok := lookupFAQ(c.input)
			if ok != c.wantHit {
				t.Errorf("lookupFAQ(%q) hit = %v, want %v", c.input, ok, c.wantHit)
			}
		})
	}
}
