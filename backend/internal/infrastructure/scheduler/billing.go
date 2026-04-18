// Package scheduler 는 배경 크론 고루틴을 관리한다.
// 현재는 LLM 일일 과금(#068) 한 개만 있음. 필요 시 다른 일일 작업도 여기에 추가.
package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/llm"
)

// StartLLMBilling 은 KST 03:33 마다 전체 학생의 LLM 사용료를 정산하는 고루틴을 띄운다.
// ctx 가 취소되면 다음 sleep 에서 빠져나와 종료한다.
//
// 실행 실패해도 다음날 다시 시도한다 (개별 학생 에러는 usecase 내부에서 집계).
func StartLLMBilling(ctx context.Context, uc *application.LLMUseCase) {
	go loop(ctx, uc)
}

func loop(ctx context.Context, uc *application.LLMUseCase) {
	for {
		now := time.Now()
		next := llm.NextBillingTime(now)
		wait := time.Until(next)
		log.Printf("[llm-billing] 다음 실행: %s (%.1f시간 후)", next.Format(time.RFC3339), wait.Hours())

		select {
		case <-ctx.Done():
			log.Printf("[llm-billing] 컨텍스트 취소로 종료")
			return
		case <-time.After(wait):
		}

		billDate := llm.BillingDate(time.Now())
		runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		processed, err := uc.BillAll(runCtx, billDate)
		cancel()
		if err != nil {
			log.Printf("[llm-billing] %s 정산 실패: %v (처리된 유저 %d명)", billDate.Format("2006-01-02"), err, processed)
		} else {
			log.Printf("[llm-billing] %s 정산 완료: %d명 차감", billDate.Format("2006-01-02"), processed)
		}
	}
}
