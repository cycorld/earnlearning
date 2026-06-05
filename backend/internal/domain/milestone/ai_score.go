package milestone

import (
	"math"
	"regexp"
	"strings"
	"unicode"
)

// HeuristicScore — 한국어 에세이 AI 작성 확률 점수 + 근거 시그널.
// Score 는 0~100 (높을수록 AI 일 가능성). 0 = 완전 사람, 100 = 완전 AI.
type HeuristicScore struct {
	Score   int      `json:"score"`   // 0~100
	Signals []Signal `json:"signals"` // 점수에 기여한 개별 시그널들
}

// Signal — 한 개의 휴리스틱 평가 항목.
type Signal struct {
	Key     string  `json:"key"`     // "sentence_length_variance" 등 안정 ID
	Label   string  `json:"label"`   // 한국어 사람 친화 표시명
	Value   float64 `json:"value"`   // 측정된 raw 값 (디버깅용)
	Weight  int     `json:"weight"`  // 이 시그널이 더한 AI 점수 0~25
	Hint    string  `json:"hint"`    // 학생 가이드 ("문장 길이가 너무 균질합니다")
}

// 한국어 essay 에서 AI 풍 출력이 자주 보이는 표현들.
// 학생이 사람 글을 쓸 때도 가끔 쓰는 단어들이라 한 표현당 가중치는 작게.
var aiPhrases = []string{
	"~을 통해", "을 통해", "를 통해", "~에 대해", "에 대해서",
	"~로 인해", "로 인해", "~에 의해", "에 의해",
	"결론적으로", "다음과 같다", "또한,", "그러나,", "더 나아가",
	"~할 수 있다", "할 수 있다는 점에서", "할 수 있다는 점", "할 수 있다.",
	"중요한 점은", "주목할 만한 점", "이러한 점에서", "이를 통해",
	"향상시킬 수 있", "활용할 수 있", "기여할 수 있", "발전시킬 수 있",
	"실제로 적용", "실제로 활용", "다양한 측면",
}

// 1인칭 + 자기경험을 시사하는 표현.
var personalMarkers = []string{
	"나는", "내가", "내", "저는", "제가", "우리", "우리는",
	"느꼈다", "느꼈어", "느꼈습니다", "생각했", "기억",
	"좋았", "힘들었", "어려웠", "재밌", "재미있", "신기했", "놀랐",
	"처음에는", "그때", "지금은", "예전에",
}

// 회고 essay 톤에서 사람 글이 자주 쓰는 구체 시간/장소 표현.
var concreteMarkers = []string{
	"주차", "월요일", "화요일", "수요일", "목요일", "금요일",
	"오전", "오후", "저녁", "새벽",
	"강의실", "과제", "팀원", "교수님", "친구",
}

var sentenceSplit = regexp.MustCompile(`[.!?。\n][\s\n]*`)
var wordSplit = regexp.MustCompile(`\s+`)
var emojiRegex = regexp.MustCompile(`[\x{1F300}-\x{1FAFF}\x{2600}-\x{27BF}\x{1F600}-\x{1F64F}]|[ㅋㅎㅠㅜ]{2,}|[!?]{2,}`)

// ScoreHeuristic — 텍스트를 받아 AI 작성 확률 점수 + 근거 산출.
// Score 가 높을수록 AI 가능성 ↑. 너무 짧은 글(<200자)은 분석 불가로 0.
func ScoreHeuristic(text string) HeuristicScore {
	clean := strings.TrimSpace(text)
	runeLen := utf8RuneCount(clean)
	if runeLen < 200 {
		// 너무 짧으면 신뢰 못함. 점수 0, 시그널은 안내만.
		return HeuristicScore{
			Score: 0,
			Signals: []Signal{{
				Key: "too_short", Label: "분량 부족",
				Value: float64(runeLen), Weight: 0,
				Hint: "200자 이상이어야 분석 가능합니다.",
			}},
		}
	}

	var signals []Signal
	score := 0

	// 1) 문장 길이 표준편차 — AI 는 균질
	sLenStdev, sLenMean := sentenceLengthStats(clean)
	cv := 0.0
	if sLenMean > 0 {
		cv = sLenStdev / sLenMean // 변동계수
	}
	// 사람 글: 보통 CV > 0.5. AI 글: 보통 0.2~0.4 근처.
	if cv < 0.3 {
		w := 20
		signals = append(signals, Signal{
			Key: "sentence_length_variance", Label: "문장 길이가 너무 균질",
			Value: round2(cv), Weight: w,
			Hint: "AI는 비슷한 길이의 문장을 늘어놓는 경향이 있어요. 짧은 문장과 긴 문장을 섞어주세요.",
		})
		score += w
	} else if cv < 0.45 {
		w := 10
		signals = append(signals, Signal{
			Key: "sentence_length_variance", Label: "문장 길이 분산이 다소 적음",
			Value: round2(cv), Weight: w,
			Hint: "문장 길이를 좀 더 다양하게 섞어주세요.",
		})
		score += w
	}

	// 2) 어휘 다양성 (TTR — Type/Token Ratio)
	ttr := typeTokenRatio(clean)
	// 한국어 essay 800자 기준: 사람 TTR > 0.5 흔함. AI 는 0.4 부근이 많음.
	if ttr < 0.35 {
		w := 15
		signals = append(signals, Signal{
			Key: "low_lexical_diversity", Label: "어휘가 단조로움",
			Value: round2(ttr), Weight: w,
			Hint: "비슷한 단어가 반복되고 있어요. 표현을 더 다양하게 써보세요.",
		})
		score += w
	}

	// 3) AI 특유 구문 빈도
	aiHits := countMatches(clean, aiPhrases)
	per1000 := float64(aiHits) * 1000.0 / float64(runeLen)
	if per1000 > 6 {
		w := 25
		signals = append(signals, Signal{
			Key: "ai_phrase_density", Label: "AI 특유 구문이 많음",
			Value: round2(per1000), Weight: w,
			Hint: `"~을 통해", "결론적으로" 같은 GPT 풍 표현이 자주 보입니다. 본인 말투로 다듬어보세요.`,
		})
		score += w
	} else if per1000 > 3 {
		w := 12
		signals = append(signals, Signal{
			Key: "ai_phrase_density", Label: "AI 특유 구문이 다소 보임",
			Value: round2(per1000), Weight: w,
			Hint: `"~을 통해", "결론적으로" 같은 표현을 줄여보세요.`,
		})
		score += w
	}

	// 4) 1인칭/구체적 경험 부재 (사람 글 시그널이 적으면 AI 의심 ↑)
	personalHits := countMatches(clean, personalMarkers)
	concreteHits := countMatches(clean, concreteMarkers)
	personalPer1000 := float64(personalHits) * 1000.0 / float64(runeLen)
	concretePer1000 := float64(concreteHits) * 1000.0 / float64(runeLen)
	if personalPer1000 < 1.0 {
		w := 20
		signals = append(signals, Signal{
			Key: "no_first_person", Label: "1인칭/본인 경험 표현 부족",
			Value: round2(personalPer1000), Weight: w,
			Hint: `"내가 ~했다", "처음에는 ~", "느꼈다" 처럼 본인 경험을 구체적으로 써주세요.`,
		})
		score += w
	}
	if concretePer1000 < 0.5 {
		w := 10
		signals = append(signals, Signal{
			Key: "no_concrete_context", Label: "구체적 맥락(시간·장소·사람) 부족",
			Value: round2(concretePer1000), Weight: w,
			Hint: "주차·교수님·팀원·강의실 같은 구체적인 상황을 곁들이면 진정성이 살아납니다.",
		})
		score += w
	}

	// 5) 이모지·감탄사·반복문자 (ㅋㅋ, ㅠㅠ, !!!) 부재
	if !emojiRegex.MatchString(clean) {
		w := 10
		signals = append(signals, Signal{
			Key: "no_emoji_or_emphasis", Label: "감정 표현(이모지·ㅋㅋ·!!)이 전혀 없음",
			Value: 0, Weight: w,
			Hint: "AI 글은 매우 단정합니다. 감정이 묻어나는 표현이 자연스러워요.",
		})
		score += w
	}

	if score > 100 {
		score = 100
	}
	return HeuristicScore{Score: score, Signals: signals}
}

func sentenceLengthStats(text string) (stdev, mean float64) {
	parts := sentenceSplit.Split(text, -1)
	lens := make([]float64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		lens = append(lens, float64(utf8RuneCount(p)))
	}
	if len(lens) < 2 {
		return 0, 0
	}
	sum := 0.0
	for _, v := range lens {
		sum += v
	}
	mean = sum / float64(len(lens))
	variance := 0.0
	for _, v := range lens {
		d := v - mean
		variance += d * d
	}
	variance /= float64(len(lens))
	return math.Sqrt(variance), mean
}

func typeTokenRatio(text string) float64 {
	// 한국어 기준 어절(공백 분리) 단위. 너무 짧은 어절(1자) 제외.
	words := wordSplit.Split(text, -1)
	seen := map[string]bool{}
	total := 0
	for _, w := range words {
		w = strings.TrimFunc(w, func(r rune) bool {
			return unicode.IsPunct(r) || unicode.IsSpace(r)
		})
		if utf8RuneCount(w) < 2 {
			continue
		}
		seen[w] = true
		total++
	}
	if total == 0 {
		return 1
	}
	return float64(len(seen)) / float64(total)
}

func countMatches(text string, phrases []string) int {
	count := 0
	for _, p := range phrases {
		count += strings.Count(text, p)
	}
	return count
}

func utf8RuneCount(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
