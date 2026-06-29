---
id: 142
title: 거래소 runMatching 전체를 DB 트랜잭션으로 래핑 (부분 커밋 차단)
priority: high
type: fix
branch: fix/142-exchange-matching-transaction
created: 2026-06-30
---

## 배경
#140 에서 체결 매칭 패닉(nil deref)을 고쳤다. 하지만 `runMatching` 은 여전히 트랜잭션 없이 여러 쓰기를 순차 실행한다: CreateTrade → 주문 상태 갱신 → 지갑 차감/입금 → 주주 이전 → 회사 valuation → 알림/오토포스트.

중간 어느 단계라도 실패하면(패닉이 아니어도 DB 에러·제약 위반 등) **앞 단계만 커밋된 부분 상태**가 남는다. #140 은 "패닉" 한 가지 경로만 막았을 뿐, 부분 커밋 가능성 자체는 그대로다.

## 목표
`runMatching`(또는 PlaceOrder 매칭 구간)을 **단일 `*sql.Tx`** 로 감싸 "전부 성공 아니면 전부 롤백" 보장.

## 작업 메모
- `walletRepo.Debit/Credit`, `companyRepo.Create/UpdateShareholder`, `exchangeRepo.CreateTrade/UpdateOrder`, `companyRepo.Update` 가 각자 `r.db` 직접 사용 → tx 를 받는 변형(또는 tx 핸들 주입)이 필요. 인터페이스 변경 범위 큼.
- 알림·오토포스트(`autoPoster`, `notify`)는 **트랜잭션 밖**(커밋 후)으로 빼야 외부 부수효과가 롤백과 엉키지 않음.
- TDD: 매칭 도중 한 단계가 실패하면 trade·지갑·주주가 모두 롤백되는지 검증하는 회귀 테스트.

## 참고
- #140 changelog `changelog/140-exchange-matching-panic.md` "배운 점" 참조.
