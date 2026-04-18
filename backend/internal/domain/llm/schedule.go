package llm

import "time"

// KST 는 한국 표준시 (UTC+9).
var KST = time.FixedZone("KST", 9*3600)

// BillingHour / BillingMinute 는 일일 과금 크론 시각 (KST 기준).
// 자정이 아닌 03:33 으로 설정된 이유: LLM proxy 의 로그 flush / 타임존 skew 를
// 흡수하고, LMS 의 다른 자정 로직과 충돌하지 않게 하기 위함.
const (
	BillingHour   = 3
	BillingMinute = 33
)

// NextBillingTime 은 주어진 시점 이후 가장 가까운 KST 03:33 시각을 반환한다.
// 이미 같은 날 03:33 이 지났다면 다음날 03:33.
func NextBillingTime(now time.Time) time.Time {
	nowKST := now.In(KST)
	target := time.Date(nowKST.Year(), nowKST.Month(), nowKST.Day(),
		BillingHour, BillingMinute, 0, 0, KST)
	if !target.After(nowKST) {
		target = target.AddDate(0, 0, 1)
	}
	return target
}

// BillingDate 는 "과금 크론이 발화했을 때 대상으로 하는 달력 일자" 를 계산한다.
// 크론은 KST 03:33 에 발화하므로, 과금 대상 일자는 "전날(KST)" 이다.
// 예: 2026-04-18 03:33 KST 발화 → 과금 대상 일자는 2026-04-17.
func BillingDate(fireAt time.Time) time.Time {
	kst := fireAt.In(KST)
	yesterday := kst.AddDate(0, 0, -1)
	return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(),
		0, 0, 0, 0, KST)
}
