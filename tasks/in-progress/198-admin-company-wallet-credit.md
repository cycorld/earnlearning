---
id: 198
title: 관리자 지갑 차감형 회사 지갑 보상 credit API
priority: high
type: feat
branch: feat/admin-company-wallet-credit
created: 2026-08-06
---
관리자가 운영 지갑에서 차감하여 classroom 회사 지갑에 보상금을 credit하는 관리자 전용 API를 추가한다. idempotency key, classroom 범위, 양수 금액, 회사 상태 및 관리자 잔액을 검증하고 거래 결과를 반환한다. 기존 회사/사용자 송금 API와 분리한다.

사용자 요청: 회사 입금용 공식 API 제작 후 누적 Rybbit 보상 입금.

보안: 공개 저장소에 실명·credential·secret을 기록하지 않는다.
