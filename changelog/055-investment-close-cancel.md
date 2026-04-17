# 055. 투자 라운드 조기 마감 + 수동 취소(환불) 기능

> **날짜**: 2026-04-16
> **태그**: `feat`, `투자`, `교육`, `백엔드`, `프론트엔드`

## 무엇을 했나요?

#029에서 투자유치 기능을 정비한 뒤, "100% 모금 못 하면 어떻게 돼?" 라는 질문이 나왔습니다. 그 시점 답은 "영원히 open 상태로 남아버리거나 expires_at 지나면 `failed`로 환불 없이 동결된다"였습니다. 학생 대상 시뮬레이션으로는 가혹해서, 대표가 라운드 운명을 직접 결정할 수 있는 두 가지 API를 추가했습니다.

### 1. 조기 마감 (early close)
- **API**: `POST /investment/rounds/:id/close` (owner)
- **쓰는 상황**: "60% 찼는데 더 안 들어오네. 여기서 접자."
- **동작**:
  - 투자자 최소 1명 이상 필요
  - status=open → status=funded
  - 이미 발행된 주식은 그대로 유지
  - **남은 미발행 주식은 발행되지 않음** (희석 덜 됨)
  - 회사 valuation = `price_per_share × new_total_shares` (투자자들이 실제로 합의한 단가 기준)
  - 주주 전원에게 "조기 마감됨" 알림

### 2. 수동 취소 + 환불 (cancel)
- **API**: `POST /investment/rounds/:id/cancel` (owner)
- **쓰는 상황**: "이번 라운드 접고 계획 다시 세울게. 돈 돌려줄게."
- **동작**:
  - status=open → status=cancelled
  - `investments` 테이블 순회:
    - 각 투자자에게 본인이 넣은 금액 환불 (company wallet → user wallet)
    - 주주 레코드에서 해당 주식 수 차감 (0 되면 삭제)
  - 회사 total_shares, total_capital 원복
  - 회사 valuation은 **원래 값 유지** (투자 전으로 되돌림)
  - 각 투자자에게 "투자금 환불" 알림 + 대표에게 요약 알림
- **안전 가드**: 회사 지갑 잔액이 환불할 돈보다 적으면 **취소 불가**. 대표가 돈을 먼저 채워야 함 → 거짓 환불 방지

### 3. 프론트 대표자 도구
`/invest/:id`에 라운드 상태가 open이고 현재 사용자가 해당 회사 대표면 **"대표자 도구"** 카드 노출:
- 교육용 HelpBox: "조기 마감 vs 취소 — 언제 뭘 써야 할까?"
- "조기 마감" 버튼 (변형: outline)
- "라운드 취소 (환불)" 버튼 (변형: destructive)
- 각 버튼은 `window.confirm`으로 내용 정확히 고지 후 실행

## 왜 이렇게 설계했나

### 조기 마감의 valuation 공식
조기 마감 시 회사 가치를 어떻게 정할지가 핵심 설계 포인트였습니다. 세 가지 옵션을 고려했어요:

1. **`target_amount ÷ offered_percent`**: 원래 설정한 포스트머니. 하지만 실제로는 지분 일부만 팔았으니 **과대평가**.
2. **`current_amount ÷ actual_equity_given`**: 수학적으로 정확하지만 공식이 복잡하고 투자자가 "내가 왜 이 가격으로 산 거지?" 혼란.
3. ✅ **`price_per_share × new_total_shares`**: 투자자들이 실제로 동의한 단가 × 현재 주식 총수. **가장 단순하고 투자자 관점과 일치**.

옵션 3을 채택했습니다. 예를 들어 price=400, 창업자 10000주 + 투자 1000주 = 총 11000주 → valuation = 4,400,000. 투자자가 400원/주에 샀으니 당연히 회사 전체는 400 × 전체 주식. 일관됨.

### 취소의 환불 가드
회사 지갑에 돈이 없으면 취소 불가로 막은 이유: 시뮬레이션이라도 "환불하겠다고 약속하고 못 돌려주는" 상황은 매우 혼란스럽습니다. 회사가 투자금을 이미 프리랜서에게 쓴 상태라면, 대표는:
1. 개인 지갑에서 회사로 자금 이동 (or admin transfer)
2. 취소 재시도

이 순서를 거쳐야 합니다. 학생한테도 "내 돈 쓴 뒤엔 환불 어려워진다"는 현실적 교훈.

### 왜 `cancelled` 상태에서 valuation을 원복 안 하나
이미 함. `company.Valuation`은 건드리지 않음(= 라운드 시작 전 가치 그대로). 이게 "라운드가 없었던 셈"의 의미.

## 검증

### TDD (backend/tests/integration/investment_test.go)
기존 7개 + **신규 6개**:
- `TestInvestment_EarlyClose_PartialFill_Revalues` — 40% 찬 라운드 조기 마감 후 valuation 4.4M 재계산 확인
- `TestInvestment_EarlyClose_ZeroInvestors_Rejected` — 투자자 0명이면 거부 (cancel 권장)
- `TestInvestment_EarlyClose_NonOwner_Forbidden` — 남이 마감 못 함
- `TestInvestment_Cancel_FullRefund` — 2명 투자 후 취소 → 각자 정확히 환불, 주주 정리, 회사 원복
- `TestInvestment_Cancel_InsufficientCompanyFunds_Rejected` — 회사가 돈 써버렸으면 취소 불가
- `TestInvestment_Cancel_NonOwner_Forbidden` — 남이 취소 못 함
- `TestInvestment_CloseOrCancel_AlreadyFundedRound_Rejected` — 이미 funded/cancelled/failed 라운드는 건드릴 수 없음

**통합 테스트 252개 pass** (이전 245 + 신규 7).

### 프론트엔드
- typecheck OK
- 기존 FeedPage 좋아요 카운트 테스트가 간헐적으로 flaky (1/3 fail, 기존에도 있던 경합) — 투자 관련 변경과 무관

## 배운 점

- **"동결 상태"는 시뮬레이션에서 최악**: 사용자가 "뭘 해도 안 바뀌는" 상태에 갇히면 학습 포기로 직결됩니다. 라운드가 미완 상태로 stuck되지 않도록 항상 owner가 결정 내릴 수 있는 탈출구를 열어둬야 합니다.
- **환불은 회사 지갑 잔액 가드 필수**: 회사가 이미 돈을 쓴 상황을 상정해야 합니다. "죄송, 환불 못 드려요" 가 나오면 믿음을 잃습니다. 프리컨디션으로 체크하고 에러로 막는 게 건강합니다.
- **조기 마감의 단가-기반 valuation**: 수학적으로 가장 단순한 게 교육적으로도 가장 이해하기 쉬웠습니다. "투자자가 400원에 샀으니 회사 전체 = 400 × 총 주식" 이 문장 하나로 설명 끝.
- **destructive vs outline 버튼**: 조기 마감(복구 가능성 없지만 돈 이동 없음)과 취소(돈 이동 + 지분 회수)는 위험도가 다릅니다. 단순히 스타일로만 구분해도 사용자가 덜 실수합니다.
