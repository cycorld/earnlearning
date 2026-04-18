package llm

import "testing"

func TestCostKRW_FullPriceNoCache(t *testing.T) {
	// 1M prompt + 1M completion, no cache → (15 + 75) USD = 90 USD × 1400 = 126,000원
	got := CostKRW(1_000_000, 1_000_000, 0, 1)
	want := 126_000
	if got != want {
		t.Fatalf("CostKRW full price: got %d, want %d", got, want)
	}
}

func TestCostKRW_TypicalSession(t *testing.T) {
	// 7768 prompt + 5988 completion, no cache
	// ≈ 7768*15/1M + 5988*75/1M = 0.1165 + 0.4491 = 0.5656 USD
	// × 1400 = 792원 (rounded)
	got := CostKRW(7768, 5988, 0, 1)
	if got < 780 || got > 800 {
		t.Fatalf("typical session: got %d, expected around 792", got)
	}
}

func TestCostKRW_ZeroUsage(t *testing.T) {
	if got := CostKRW(0, 0, 0, 0); got != 0 {
		t.Fatalf("zero usage: got %d", got)
	}
}

func TestCostKRW_AllCacheHitsDiscounted(t *testing.T) {
	// 1M prompt all cached (cache_hits == requests) + 0 completion
	// 캐시 할인 적용: 1M × 1.50 / 1M × 1400 = 2,100원
	got := CostKRW(1_000_000, 0, 10, 10)
	want := 2_100
	if got != want {
		t.Fatalf("all cached: got %d, want %d", got, want)
	}
}

func TestCostKRW_HalfCacheRatio(t *testing.T) {
	// 1M prompt, half requests cached
	// full input:   500k × 15/M = 7.5 USD
	// cached input: 500k × 1.5/M = 0.75 USD
	// output: 0
	// total: 8.25 USD × 1400 = 11,550원
	got := CostKRW(1_000_000, 0, 5, 10)
	want := 11_550
	if got != want {
		t.Fatalf("half cache: got %d, want %d", got, want)
	}
}

func TestCostKRW_CacheHitsExceedingRequestsClamped(t *testing.T) {
	// 방어 코드: cache_hits > requests 일 경우 ratio 는 1로 clamp
	got := CostKRW(1_000_000, 0, 100, 10)
	want := 2_100 // all cached
	if got != want {
		t.Fatalf("clamped cache: got %d, want %d", got, want)
	}
}

func TestCostKRW_NegativeInputsTreatedAsZero(t *testing.T) {
	if got := CostKRW(-1000, -1000, 0, 0); got != 0 {
		t.Fatalf("negative inputs: got %d, want 0", got)
	}
}

func TestCostKRW_NoCacheHitsWithRequests(t *testing.T) {
	// requests > 0 이지만 cache_hits == 0 → full price
	got := CostKRW(1_000_000, 0, 0, 10)
	want := 21_000 // 15 USD × 1400
	if got != want {
		t.Fatalf("no cache hits: got %d, want %d", got, want)
	}
}
