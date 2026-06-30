---
id: 144
title: 거래소 주문 UX 고도화 — 내 주문 상태 탭 + 보유/여유자금 표시·제약
priority: medium
type: feat
branch: feat/144-exchange-my-orders-tabs
created: 2026-06-30
---

## A. 메인 "내 주문" 상태 탭 (ExchangePage)
- 탭: [진행중][체결][취소][전체] + 개수 배지. 기본 진행중.
  - 진행중 = open+partial, 체결 = filled, 취소 = cancelled
- 행 보강: 실제 회사명·로고(companies 매핑), 매수/매도, 수량×가격, 총액, 상태 한글, 부분체결 잔여, 경과시간
- 진행중 주문은 행에서 바로 취소(DELETE /exchange/orders/:id)

## B. 주문 폼 보유/여유자금 표시 + 제약 (ExchangeDetailPage)
- 신규 백엔드 `GET /exchange/position/:companyId` → `{shares, available_shares, balance, available_cash}`
  - available_shares = 보유 − 미체결 매도(pendingSell), available_cash = 잔액 − 미체결 매수(pendingBuy)
  - PlaceOrder 가 검증에 쓰는 값과 동일하게 노출 (프론트 제약 = 백엔드 제약)
- 주문 폼에 "보유 N주 (매도가능 M)" + "여유자금 ₩X" 표시
- 매수: 수량×가격 ≤ available_cash, 매도: 수량 ≤ available_shares — 초과 시 제출 비활성 + 인라인 경고. "최대" 버튼.

## 범위/검증
- 백엔드: position 엔드포인트(read-only) + 통합 테스트(보유/pending 차감 정확성)
- 프론트: ExchangePage·ExchangeDetailPage. 스모크 + 빌드/타입 통과
