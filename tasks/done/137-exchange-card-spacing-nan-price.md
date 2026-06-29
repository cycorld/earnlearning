---
id: 137
title: 거래소 카드 간격 + NaN원/주 가격 수정
priority: high
type: fix
branch: fix/137-exchange-card-spacing-nan-price
created: 2026-06-29
---

## 문제
거래소(`/exchange`) 페이지:
1. 상장 기업 카드들이 서로 붙어 있음 (세로 간격 없음).
2. 가격이 `NaN원/주`로 표시됨.

## 원인
1. `<Link>`가 `<a>` (display:inline)로 렌더 → `space-y-2`의 margin-top이 inline 요소에 적용 안 됨 → 간격 0.
2. `/exchange/companies` API는 `ListedCompany`(`last_price`, `market_cap` 등)를 반환하는데 프론트가 `Company` 타입으로 캐스팅 후 `valuation/total_shares` 계산 → `valuation` 필드 없음(undefined) → NaN.

## 수정
- 프론트: 응답 타입을 실제 API(`ListedCompany`)에 맞추고 `last_price`(시가) 직접 사용. `<Link>`에 `block` + 간격 부여.
- 백엔드: `GetListedCompanies`의 `last_price`가 거래 없을 때 마지막 funded 라운드 `price_per_share`로 폴백(시가 = 마지막 라운드 가격). 회귀 테스트 추가.
