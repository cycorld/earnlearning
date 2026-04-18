package llm

import (
	"testing"
	"time"
)

func TestNextBillingTime_BeforeTargetSameDay(t *testing.T) {
	// 2026-04-18 01:00 KST → 2026-04-18 03:33 KST
	now := time.Date(2026, 4, 18, 1, 0, 0, 0, KST)
	next := NextBillingTime(now)
	want := time.Date(2026, 4, 18, 3, 33, 0, 0, KST)
	if !next.Equal(want) {
		t.Fatalf("got %v, want %v", next, want)
	}
}

func TestNextBillingTime_ExactlyAtTargetMovesToNextDay(t *testing.T) {
	// 정확히 03:33 에 호출되면 다음날 03:33 을 돌려줘야 재발화 루프가 안 생김.
	now := time.Date(2026, 4, 18, 3, 33, 0, 0, KST)
	next := NextBillingTime(now)
	want := time.Date(2026, 4, 19, 3, 33, 0, 0, KST)
	if !next.Equal(want) {
		t.Fatalf("got %v, want %v", next, want)
	}
}

func TestNextBillingTime_AfterTargetSameDay(t *testing.T) {
	// 2026-04-18 10:00 KST → 2026-04-19 03:33 KST
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, KST)
	next := NextBillingTime(now)
	want := time.Date(2026, 4, 19, 3, 33, 0, 0, KST)
	if !next.Equal(want) {
		t.Fatalf("got %v, want %v", next, want)
	}
}

func TestNextBillingTime_AcceptsUTCInput(t *testing.T) {
	// UTC 기준의 timestamp 를 넘겨도 KST 기준으로 계산해야 함.
	// 2026-04-17 18:00 UTC == 2026-04-18 03:00 KST
	now := time.Date(2026, 4, 17, 18, 0, 0, 0, time.UTC)
	next := NextBillingTime(now)
	want := time.Date(2026, 4, 18, 3, 33, 0, 0, KST)
	if !next.Equal(want) {
		t.Fatalf("got %v, want %v", next, want)
	}
}

func TestBillingDate_IsYesterdayInKST(t *testing.T) {
	fireAt := time.Date(2026, 4, 18, 3, 33, 0, 0, KST)
	got := BillingDate(fireAt)
	want := time.Date(2026, 4, 17, 0, 0, 0, 0, KST)
	if !got.Equal(want) {
		t.Fatalf("billing date: got %v, want %v", got, want)
	}
}

func TestBillingDate_HandlesUTCInputCorrectly(t *testing.T) {
	// 2026-04-17 18:33 UTC == 2026-04-18 03:33 KST → billing date = 2026-04-17
	fireAt := time.Date(2026, 4, 17, 18, 33, 0, 0, time.UTC)
	got := BillingDate(fireAt)
	want := time.Date(2026, 4, 17, 0, 0, 0, 0, KST)
	if !got.Equal(want) {
		t.Fatalf("billing date: got %v, want %v", got, want)
	}
}
