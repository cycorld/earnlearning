package llm

import "testing"

func TestCostKRW_FullPriceNoCache(t *testing.T) {
	// 1M prompt + 1M completion, no cache → (15 + 75) USD = 90 USD × 1400 = 126,000원
	got := CostKRW(1_000_000, 1_000_000, 0)
	want := 126_000
	if got != want {
		t.Fatalf("CostKRW full price: got %d, want %d", got, want)
	}
}

func TestCostKRW_TypicalSession(t *testing.T) {
	// 7768 prompt + 5988 completion, no cache
	// ≈ 7768*15/1M + 5988*75/1M = 0.1165 + 0.4491 = 0.5656 USD
	// × 1400 = 792원 (rounded)
	got := CostKRW(7768, 5988, 0)
	if got < 780 || got > 800 {
		t.Fatalf("typical session: got %d, expected around 792", got)
	}
}

func TestCostKRW_ZeroUsage(t *testing.T) {
	if got := CostKRW(0, 0, 0); got != 0 {
		t.Fatalf("zero usage: got %d", got)
	}
}

func TestCostKRW_AllPromptTokensCached(t *testing.T) {
	// 1M prompt, 1M cached (100% 캐시 적중) + 0 completion
	// 캐시 할인 적용: 1M × 1.50 / 1M × 1400 = 2,100원
	got := CostKRW(1_000_000, 0, 1_000_000)
	want := 2_100
	if got != want {
		t.Fatalf("all cached: got %d, want %d", got, want)
	}
}

func TestCostKRW_HalfCached(t *testing.T) {
	// 1M prompt, 500k cached
	// full input:   500k × 15/M = 7.5 USD
	// cached input: 500k × 1.5/M = 0.75 USD
	// output: 0
	// total: 8.25 USD × 1400 = 11,550원
	got := CostKRW(1_000_000, 0, 500_000)
	want := 11_550
	if got != want {
		t.Fatalf("half cache: got %d, want %d", got, want)
	}
}

func TestCostKRW_CacheTokensExceedingPromptClamped(t *testing.T) {
	// 방어 코드: cache_tokens > prompt_tokens 일 경우 prompt_tokens 로 clamp
	// → 전체가 캐시로 간주됨
	got := CostKRW(1_000_000, 0, 5_000_000)
	want := 2_100 // all cached price
	if got != want {
		t.Fatalf("clamped cache: got %d, want %d", got, want)
	}
}

func TestCostKRW_NegativeInputsTreatedAsZero(t *testing.T) {
	if got := CostKRW(-1000, -1000, -500); got != 0 {
		t.Fatalf("negative inputs: got %d, want 0", got)
	}
}

func TestCostKRW_NoCacheHitsWithRequests(t *testing.T) {
	// cache_tokens = 0 → full price
	got := CostKRW(1_000_000, 0, 0)
	want := 21_000 // 15 USD × 1400
	if got != want {
		t.Fatalf("no cache hits: got %d, want %d", got, want)
	}
}

// 실제 llama.cpp 캐시 적중 시나리오: 대부분의 이어지는 턴에서 앞쪽 프롬프트가
// 대부분 재사용됨 → 비용이 훨씬 낮아야 함.
func TestCostKRW_MostlyCached_ShouldBeCheaperThanNoCache(t *testing.T) {
	// 100k prompt, 5k 는 새로 추가, 95k 는 캐시 재사용
	withCache := CostKRW(100_000, 1_000, 95_000)
	noCache := CostKRW(100_000, 1_000, 0)
	if withCache >= noCache {
		t.Fatalf("cached should be cheaper: with=%d vs no-cache=%d", withCache, noCache)
	}
	// 95k * (15 - 1.5) / 1M * 1400 = 1,796 원 정도의 절약이 있어야 함
	savings := noCache - withCache
	if savings < 1_500 || savings > 2_100 {
		t.Fatalf("unexpected savings: %d (expected ~1,796)", savings)
	}
}
