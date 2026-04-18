// Package llm contains domain entities and pure pricing logic for the student
// LLM API key / billing feature.
//
// 과금 기준 (티켓 #068 사용자 확정):
//   - 모델 가격 기준: Anthropic Claude Opus 4.7 공식가
//   - 환율: 1 USD = 1,400 KRW 고정
//   - 캐시 할인: Opus cached-input 는 $1.50/MTok (full 대비 90% off)
//     → llm-proxy 가 `cache_tokens` (llama.cpp timings.cache_n 합계) 를 반환하므로
//     정확한 캐시 적중 토큰 수를 그대로 사용.
package llm

import "math"

// USD per million tokens (Opus 4.7 공식 가격).
const (
	InputPricePerMTokUSD       = 15.0
	OutputPricePerMTokUSD      = 75.0
	CachedInputPricePerMTokUSD = 1.5
	USDToKRW                   = 1400.0
)

// CostKRW 는 하루치 사용량에서 원화 비용을 계산한다.
//
//   - promptTokens    : 전체 입력 토큰 수 (캐시 적중분 포함)
//   - completionTokens: 출력 토큰 수
//   - cacheTokens     : promptTokens 중 KV 캐시에서 재사용된 토큰 수
//     (llm-proxy UsageBucket.cache_tokens)
//
// 공식:
//
//	fullInput  = (promptTokens - cacheTokens) × $15  / 1M
//	cachedIn   = cacheTokens                  × $1.5 / 1M
//	output     = completionTokens             × $75  / 1M
//	cost_krw   = round((fullInput + cachedIn + output) × 1400)
//
// cacheTokens 가 promptTokens 를 넘으면 promptTokens 로 clamp. 음수는 0 으로 clamp.
func CostKRW(promptTokens, completionTokens, cacheTokens int) int {
	if promptTokens < 0 {
		promptTokens = 0
	}
	if completionTokens < 0 {
		completionTokens = 0
	}
	if cacheTokens < 0 {
		cacheTokens = 0
	}
	if cacheTokens > promptTokens {
		cacheTokens = promptTokens
	}
	fullInputTokens := promptTokens - cacheTokens

	usd := (float64(fullInputTokens)*InputPricePerMTokUSD +
		float64(cacheTokens)*CachedInputPricePerMTokUSD +
		float64(completionTokens)*OutputPricePerMTokUSD) / 1_000_000
	krw := usd * USDToKRW
	return int(math.Round(krw))
}
