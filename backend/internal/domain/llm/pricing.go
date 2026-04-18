// Package llm contains domain entities and pure pricing logic for the student
// LLM API key / billing feature.
//
// 과금 기준 (티켓 #068 사용자 확정):
//   - 모델 가격 기준: Anthropic Claude Opus 4.7 공식가
//   - 환율: 1 USD = 1,400 KRW 고정
//   - 캐시 할인: Opus cached-input 는 $1.50/MTok (full 대비 90% off)
//     → 일별 cache_hits / requests 비율을 cached token 비율로 근사.
package llm

import "math"

// USD per million tokens (Opus 4.7 공식 가격).
const (
	InputPricePerMTokUSD       = 15.0
	OutputPricePerMTokUSD      = 75.0
	CachedInputPricePerMTokUSD = 1.5
	USDToKRW                   = 1400.0
)

// CostKRW 는 하루치 사용량 버킷에서 원화 비용을 계산한다.
//
// cacheHits / requests 비율을 prompt_tokens 의 "캐시 히트 비율" 로 근사한다.
// LLM proxy 가 cached-token 수를 따로 주지 않기 때문에 (요청 수 기준) 근사치.
//
// requests 가 0 이면 cache_hits 도 무시하고 전액 full-price 로 계산.
// 모든 계산은 float 로 하고 마지막에 반올림해서 정수 원화를 반환.
func CostKRW(promptTokens, completionTokens, cacheHits, requests int) int {
	if promptTokens < 0 {
		promptTokens = 0
	}
	if completionTokens < 0 {
		completionTokens = 0
	}

	ratio := 0.0
	if requests > 0 && cacheHits > 0 {
		ratio = float64(cacheHits) / float64(requests)
		if ratio > 1 {
			ratio = 1
		}
	}
	prompt := float64(promptTokens)
	completion := float64(completionTokens)

	fullInput := prompt * (1 - ratio) * InputPricePerMTokUSD
	cachedInput := prompt * ratio * CachedInputPricePerMTokUSD
	outputCost := completion * OutputPricePerMTokUSD

	usd := (fullInput + cachedInput + outputCost) / 1_000_000
	krw := usd * USDToKRW
	return int(math.Round(krw))
}
