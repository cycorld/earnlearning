---
id: 164
title: 자산 계산에서 창업자 지분 이중계상 (주식 가치 + 회사 지분)
priority: medium
type: fix
branch: fix/164-founder-equity-double-count
created: 2026-07-20
---

## 증상
회사를 설립하면 총 자산이 무에서 증가한다. 예: 초기자본 1,000,000원으로 회사 설립 → 현금 0원이 되지만 총 자산은 2,000,000원.

## 원인
`wallet_repo.go` `GetAssetBreakdown`에서 같은 지분이 두 번 집계됨:
- **StockValue**: `shares × company.valuation / total_shares` = 1,000,000
- **CompanyEquity**: `company_wallet.balance × shares / total_shares` = 1,000,000

신설 법인은 valuation ≈ 회사 지갑 잔액이므로 사실상 동일한 가치를 이중계상.

## 발견 경위
#159 멀티 강의실 E2E 검증 중 발견. 강의실 격리와는 무관한 기존 밸류에이션 버그.

## 수정 방향(안)
- 상장 여부/밸류에이션 산정 방식에 따라 StockValue 와 CompanyEquity 중 하나만 집계하거나,
- CompanyEquity 를 "지분 장부가 − 주식 평가액" 차액 방식으로 재정의.
- 랭킹·자산 화면 모두 영향 — 회귀 테스트 필수.
