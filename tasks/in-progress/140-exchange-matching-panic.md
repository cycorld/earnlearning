---
id: 140
title: 거래소 체결 시 매칭 엔진 패닉(nil deref) → 500 + 데이터 손상 수정
priority: high
type: fix
branch: fix/140-exchange-matching-panic
created: 2026-06-30
---

## 증상
주문이 교차 체결되면 `POST /exchange/orders` 가 500. 체결(stock_trades)·지갑 이체는 커밋된 뒤 패닉으로 중단 → **돈은 이동했는데 주식 소유권 이전이 누락**되는 데이터 손상.

## 근본 원인
`exchange_usecase.go` `runMatching`:
- `companyRepo.FindShareholder()` 는 not-found 에 `(nil, nil)` 반환 (sql.ErrNoRows sentinel).
- 매수자 분기: `if err != nil { create } else { buyerSH.Shares ... }` — **처음 매수하는 사람**은 주주 레코드가 없어 `(nil,nil)` → `err==nil` → else 진입 → `buyerSH.Shares` nil deref panic (line 317).
- 매도자 분기: `if err == nil && updater != nil { sellerSH.Shares ... }` — 같은 잠재 버그.

## 수정
- 매수자: `if err != nil || buyerSH == nil { create } else { update }`.
- 매도자: 조건에 `sellerSH != nil` 추가.
- TDD: 첫 매수자 교차 체결 통합 테스트 (패닉 재현 → 수정 후 통과 + 주식 이전 검증).

## 후속(별도)
- runMatching 전체를 DB 트랜잭션으로 감싸 부분 커밋 원천 차단 (방어적 강화).
- 스테이지/프로덕션 기존 절반커밋 체결 데이터 정합성 점검.
