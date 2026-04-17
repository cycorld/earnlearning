---
id: 030
title: 투자 라운드 조기 마감 + 수동 취소(환불) 기능
priority: high
type: feat
branch: feat/investment-close-cancel
created: 2026-04-16
---

## 배경

#029 작업 후 "투자금 100% 유치 못하면 어떻게 돼?" 질문에서 나온 갭:
- 조기 마감 API 없음 → 90% 찼는데 더 안 들어오면 영원히 open
- 수동 취소 API 없음 → 대표가 라운드 접고 싶어도 방법 없음
- 부분 체결 환불 불가 → 투자자 보호 안 됨

## 요구사항

### 조기 마감 (early close)
- **권한**: owner
- **API**: `POST /investment/rounds/:id/close`
- **동작**:
  - status=open만 가능
  - 최소 1명 이상 투자했어야 함 (0건이면 cancel 권장)
  - status='funded'로 전환, funded_at=NOW
  - 회사 valuation 재계산: `price_per_share × new_total_shares` (남은 미발행 주식은 버려짐)
  - 이미 발행된 주식 수 = sold_shares, 나머지는 발행 안 함
  - 알림: 주주 전원에게 "라운드 조기 마감됨"

### 수동 취소 + 환불 (cancel + refund)
- **권한**: owner
- **API**: `POST /investment/rounds/:id/cancel`
- **동작**:
  - status=open만 가능
  - 회사 지갑 잔액 >= current_amount 전제 (아니면 에러)
  - 각 투자 건마다:
    - 회사 지갑 → 투자자 지갑 환불
    - 주주 주식 수 차감 (0이 되면 주주 레코드 삭제)
  - company.total_shares -= sum(sold_shares)
  - company.total_capital -= current_amount
  - status='cancelled'
  - 알림: 환불받은 투자자들에게 "투자금 환불 완료"

## TDD 범위

- `TestInvestment_EarlyClose_PartialFill`: 50% 찬 상태에서 조기 마감 → valuation 적절히 재계산
- `TestInvestment_EarlyClose_ZeroInvestors_Rejected`: 아무도 투자 안 한 라운드는 cancel 쓰도록 유도
- `TestInvestment_Cancel_FullRefund`: 여러 투자자 + 여러 투자 건 모두 환불, 주주 정리 확인
- `TestInvestment_Cancel_InsufficientCompanyWallet_Rejected`: 회사가 돈 써버렸으면 취소 불가
- `TestInvestment_NonOwnerClose_Forbidden`: owner 아닌 사람 차단
- `TestInvestment_NonOwnerCancel_Forbidden`: owner 아닌 사람 차단
- `TestInvestment_CloseFundedRound_Rejected`: 이미 funded/cancelled/failed 라운드는 건드릴 수 없음

## 프론트

- `InvestDetailPage.tsx`:
  - 라운드 status=open 이고 현재 사용자가 owner면 상단에 "조기 마감" / "취소" 버튼 2개
  - 각 버튼은 confirm dialog + 교육용 warning 띄우기
    - 조기 마감: "모집 금액만큼만 유치되고 나머지 주식은 발행되지 않습니다"
    - 취소: "투자자 전원에게 환불되고 지분이 회수됩니다"
- `InvestmentRoundSection.tsx`: 회사 상세 섹션에도 같은 버튼 노출 (status=open일 때)
