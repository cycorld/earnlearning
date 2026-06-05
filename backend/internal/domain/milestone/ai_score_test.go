package milestone

import (
	"strings"
	"testing"
)

// 사람이 쓴 회고 같은 글 — 구체적 경험, 1인칭, 길이 다양, 감정 표현 포함.
const humanEssay = `이번 학기를 돌아보면 진짜 정신없었다 ㅋㅋ. 처음 강의실 들어갔을 때만 해도
이렇게까지 코딩에 빠질 줄은 몰랐다. 1주차에 교수님이 "여러분도 창업할 수 있어요" 하셨을 때
솔직히 흘려들었는데, 3주차쯤 우리 팀이랑 MVP 만들면서 생각이 바뀌었다.

내가 가장 힘들었던 건 8주차였다. 팀원이랑 디자인 갈등이 있었고, 결국 새벽까지 카톡으로
싸웠다. 다음날 강의실에서 어색하게 앉아있던 게 아직도 기억난다. 그래도 그 다음주에
교수님이 "팀워크는 갈등을 풀면서 늘어요" 하셨고, 우리는 결국 화해했다.

가장 좋았던 순간? 12주차 IR Day. 발표 끝나고 친구가 "너 진짜 잘하더라" 했을 때.
울 뻔했다 ㅠㅠ. 다음 학기도 듣고 싶다.`

// AI 가 쓴 풍의 회고 — 균질한 문장, "~을 통해" 남발, 1인칭 부족, 감정 표현 없음.
const aiEssay = `이번 학기는 매우 의미 있는 시간이었다. 다양한 활동을 통해 많은 것을 배울 수 있었다.
특히 팀 프로젝트를 통해 협업의 중요성을 깨달았다. 또한 다양한 기술을 활용할 수 있는 기회가 주어졌다.
결론적으로 이번 학기는 성장의 시간이었다고 할 수 있다.
이를 통해 더 나아가 새로운 도전을 이어갈 수 있는 발판을 마련하였다.
앞으로도 다양한 측면에서 발전시킬 수 있도록 노력할 것이다.
이러한 점에서 본 강의는 매우 가치 있는 경험이었다.
실제로 적용할 수 있는 다양한 지식을 습득하였다.
다음과 같은 점들이 향상되었다고 할 수 있다.
이번 학기를 통해 얻은 경험은 향후 진로에 큰 도움이 될 것으로 기대된다.`

func TestScoreHeuristic_HumanWritingLowScore(t *testing.T) {
	got := ScoreHeuristic(humanEssay)
	if got.Score > 30 {
		t.Errorf("human essay score = %d, want <= 30; signals: %+v", got.Score, signalKeys(got.Signals))
	}
}

func TestScoreHeuristic_AIWritingHighScore(t *testing.T) {
	got := ScoreHeuristic(aiEssay)
	if got.Score < 60 {
		t.Errorf("AI-like essay score = %d, want >= 60; signals: %+v", got.Score, signalKeys(got.Signals))
	}
}

func TestScoreHeuristic_AIScoresHigherThanHuman(t *testing.T) {
	h := ScoreHeuristic(humanEssay).Score
	a := ScoreHeuristic(aiEssay).Score
	if a <= h {
		t.Errorf("AI essay (%d) should score higher than human essay (%d)", a, h)
	}
}

func TestScoreHeuristic_TooShort(t *testing.T) {
	got := ScoreHeuristic("너무 짧은 글")
	if got.Score != 0 {
		t.Errorf("too-short essay score = %d, want 0", got.Score)
	}
	if len(got.Signals) != 1 || got.Signals[0].Key != "too_short" {
		t.Errorf("expected too_short signal, got %+v", got.Signals)
	}
}

func TestScoreHeuristic_AIPhraseDetection(t *testing.T) {
	// "~을 통해" 와 "결론적으로" 가 많은 글 → ai_phrase_density 시그널 발화
	text := strings.Repeat("이번 학기를 통해 많은 것을 배웠다. 결론적으로 성장의 시간이었다. ", 30)
	got := ScoreHeuristic(text)
	found := false
	for _, s := range got.Signals {
		if s.Key == "ai_phrase_density" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected ai_phrase_density signal, got %+v", signalKeys(got.Signals))
	}
}

func TestScoreHeuristic_PersonalMarkersReduceScore(t *testing.T) {
	// 같은 길이의 글: 1인칭 포함 vs 안 포함
	withPersonal := strings.Repeat("내가 처음에는 이런 게 어려울 줄 몰랐다. 그때 친구가 도와줬다. ", 20)
	withoutPersonal := strings.Repeat("이런 것이 어려울 것이라고 예상하지 못하였다. 도움이 필요한 상황이었다. ", 20)

	with := ScoreHeuristic(withPersonal).Score
	without := ScoreHeuristic(withoutPersonal).Score
	if with >= without {
		t.Errorf("with personal markers should score lower: with=%d, without=%d", with, without)
	}
}

// #121 — 사람이 키보드로 입력 안 하는 유니코드 검출

func TestDetectSuspiciousChars_NormalKoreanZero(t *testing.T) {
	c := DetectSuspiciousChars(humanEssay)
	if c.ZeroWidth != 0 || c.MathAlpha != 0 || c.NBSP != 0 {
		t.Errorf("human essay should have 0 suspicious chars, got %+v", c)
	}
}

func TestDetectSuspiciousChars_ZeroWidth(t *testing.T) {
	// U+200B (ZWSP) 와 U+FEFF (BOM) — escape 로 명시 (소스에 직접 박으면 Go 컴파일러가 거부)
	text := "안녕하세요\u200B 저는 학생\uFEFF입니다."
	c := DetectSuspiciousChars(text)
	if c.ZeroWidth != 2 {
		t.Errorf("ZeroWidth = %d, want 2", c.ZeroWidth)
	}
}

func TestDetectSuspiciousChars_MathAlpha(t *testing.T) {
	text := "이번 학기를 통해 \U0001D400 학습했다." // 𝐀
	c := DetectSuspiciousChars(text)
	if c.MathAlpha != 1 {
		t.Errorf("MathAlpha = %d, want 1", c.MathAlpha)
	}
}

func TestDetectSuspiciousChars_NBSP(t *testing.T) {
	text := "프로젝트 관리 능력"
	c := DetectSuspiciousChars(text)
	if c.NBSP != 2 {
		t.Errorf("NBSP = %d, want 2", c.NBSP)
	}
}

func TestDetectSuspiciousChars_SmartQuotesAndDashes(t *testing.T) {
	text := `“이번 학기는 의미 있었다” — 결론적으로 ‘성장’ 했다 – 정말.`
	c := DetectSuspiciousChars(text)
	if c.SmartQuotes != 4 {
		t.Errorf("SmartQuotes = %d, want 4 (“”‘’)", c.SmartQuotes)
	}
	if c.EmEnDash != 2 {
		t.Errorf("EmEnDash = %d, want 2", c.EmEnDash)
	}
}

func TestScoreHeuristic_ZeroWidthTriggersSignal(t *testing.T) {
	// 사람 풍 글에 zero-width 1개만 박아도 시그널 발화
	text := humanEssay + "​"
	got := ScoreHeuristic(text)
	found := false
	for _, s := range got.Signals {
		if s.Key == "ai_zero_width" {
			found = true
			if s.Weight < 25 {
				t.Errorf("zero-width weight = %d, expected strong (>=25)", s.Weight)
			}
		}
	}
	if !found {
		t.Errorf("expected ai_zero_width signal, got %+v", signalKeys(got.Signals))
	}
}

func TestScoreHeuristic_MathAlphaTriggersSignal(t *testing.T) {
	text := humanEssay + "\U0001D400"
	got := ScoreHeuristic(text)
	found := false
	for _, s := range got.Signals {
		if s.Key == "ai_math_alpha" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ai_math_alpha signal, got %+v", signalKeys(got.Signals))
	}
}

func TestScoreHeuristic_NBSPTriggersSignal(t *testing.T) {
	text := humanEssay + "\u00A0"
	got := ScoreHeuristic(text)
	found := false
	for _, s := range got.Signals {
		if s.Key == "ai_nbsp" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ai_nbsp signal, got %+v", signalKeys(got.Signals))
	}
}

func TestScoreHeuristic_EmDashDensityTriggers(t *testing.T) {
	// 200자 이상 + em dash 밀도 높은 글 (1000자당 3개 이상)
	text := "한 학기를 마무리하면서 — 이번 학기는 정말 의미 있는 시간이었다. " +
		"팀워크 — 처음엔 갈등이 있었지만 — 결국 화해했다. " +
		"새로운 도전 — 실패도 했지만 — 그만큼 배움도 컸다. " +
		"다양한 경험을 했다 — 정말 좋았다 — 다음 학기도 듣고 싶다. " +
		"교수님 강의 — 매우 도움이 되었다 — 강력 추천한다. " +
		"동기들과의 협업 — 의외로 즐거웠다 — 친구도 많이 생겼다. " +
		"기술적인 도전 — 처음엔 막막했다 — 끝내 해냈다."
	got := ScoreHeuristic(text)
	found := false
	for _, s := range got.Signals {
		if s.Key == "ai_em_dash_density" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ai_em_dash_density signal, got %+v", signalKeys(got.Signals))
	}
}

func TestScoreHeuristic_AllAIEncodingFlagsPushScoreVeryHigh(t *testing.T) {
	// 사람 풍 글 + AI 흔적 다수 박음 → 점수가 사람 풍 글보다 훨씬 높아야 함
	base := ScoreHeuristic(humanEssay).Score
	tampered := humanEssay + "​ \U0001D400   “a” “b” “c” “d” “e”"
	got := ScoreHeuristic(tampered).Score
	if got <= base+30 {
		t.Errorf("tampered score (%d) should be much higher than base (%d) due to AI encoding flags", got, base)
	}
}

func signalKeys(ss []Signal) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = s.Key
	}
	return out
}
