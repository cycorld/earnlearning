# 054. 투자유치 기능 전면 개선 — 분할 투자 + 버그 수정 + 교육용 도움말

> **날짜**: 2026-04-16
> **태그**: `fix`, `feat`, `투자`, `교육`, `백엔드`, `프론트엔드`

## 무엇을 했나요?

스테이지에서 투자 기능을 end-to-end로 실제 돌려보니 심각한 버그 4개(+마이너 2개)가 발견됐습니다. 그걸 한 번에 고치고, 이왕 손댄 김에 학생 교육용으로 **투자 도메인 지식 도움말 + 가치평가 실시간 계산기**까지 넣었습니다. 그리고 "1라운드 = 1투자자 = 전액"이었던 제약을 풀어 **여러 명이 나눠서 투자하는 분할 투자**도 가능하게 했습니다.

## 고친 버그

### Bug A (critical) — `/invest` 목록 항상 비어있음
프론트가 `?status=active` 필터로 조회하는데 백엔드 enum은 `open/funded/failed/cancelled`. 일치하는 게 없어서 open 라운드가 DB에 있어도 **한 번도 노출된 적이 없었음**. → `?status=open`으로 수정.

### Bug B (critical) — `/invest/:id` 영원히 로딩
프론트가 `GET /investment/rounds/:id`를 호출하는데 이 라우트가 백엔드에 **아예 없었음** (list만 있음). → `GetRound` 핸들러 + 라우터 추가. 응답에 `sold_shares`, `remaining_shares` 파생 필드도 포함.

### Bug C (critical) — 투자한 사용자의 `/invest` 전체 페이지 크래시
Backend `PortfolioItem`이 flat shape(`company_name`, `user_shares`, `invested`)인데 프론트는 nested shape(`company.name`, `shares`, `invested_amount`, `profit`, `dividends_received`)를 기대. ErrorBoundary가 `Cannot read properties of undefined (reading 'name')`로 잡혔음. → 백엔드 `PortfolioItem`을 프론트 기대 shape에 맞춰 재설계:
- `Company: { id, name, valuation, logo_url }` 중첩 객체
- `Shares`, `InvestedAmount`, `Profit`, `DividendsReceived`, `Percentage` 파생 필드 포함
- `dividends_received`는 `SumDividendsByUserAndCompany` 집계 쿼리로 계산

### Bug D (critical) — 분할 투자 불가 (옵션 2 채택)
기존 설계: 1라운드 = 1투자자 = `target_amount` 전액 debit. 프론트는 shares 입력을 받지만 백엔드가 **무시**하고 전액 처리. → 아예 다중 투자자 분할 투자를 지원하도록 재설계:

- `Invest(roundID, userID, shares)` — shares 파라미터 추가
- `1 ≤ shares ≤ remaining_shares` 검증 (`SumSharesByRound`로 집계)
- 투자 금액 계산:
  - 마지막 구매자(`shares == remaining`)는 **`target - current` 을 정확히** 부담 → target에 딱 맞게 클로즈
  - 그 외엔 `round(shares × price_per_share)`
- `UpdateRoundCurrentAmount(id, amount)` 새 repo 메서드로 부분 갱신
- 최종 클로즈에서만 `status='funded'` + `funded_at` + 포스트머니 valuation 세팅
- `UpsertShareholder`는 기존에 additive라 OK (여러 번 사도 누적)
- Company `total_shares`/`total_capital`은 매 투자마다 증분

### Bug E (medium) — KPI 규칙 owner 검증 없음
`CreateKpiRule`에 owner 체크가 없어서 아무나 남의 회사 KPI 생성 가능했음. → `(input, userID)` 시그니처로 변경하고 `c.OwnerID != userID`면 `ErrNotOwner`.

### Bug F (low) — 라운드 만료 enforcement
`expires_at` 필드 있지만 한 번도 체크 안 하던 로직 추가. `maybeAutoExpire(round)`를 `Invest`, `GetRound` 진입 시 호출해 만료된 open 라운드를 `failed`로 자동 전환.

## 학생 교육용 도움말 (추가 요청 반영)

"학생 교육용이니까 도움말 + 가치평가 계산 보여달라" 요청에 따라 세 곳에 collapsible `HelpBox` 추가:

### 1. 라운드 개설 다이얼로그 (`InvestmentRoundSection`)
회사 상세 페이지에 **"투자 유치" 섹션**이 새로 생기고, owner만 "라운드 개설" 버튼 노출. 다이얼로그에서:
- **투자 라운드 기초 지식** 설명: 목표 금액 / 제공 지분 / 가치평가 / 주당 가격 / 동시 라운드 제약
- **실시간 가치평가 미리보기**: 입력값이 바뀔 때마다 프리머니 / 포스트머니 / 모집 금액을 2×2 그리드로 표시
- **다운라운드 경고**: 프리머니가 현재 기업가치보다 낮으면 "기존 주주에게 불리할 수 있음" 경고 표시

### 2. 라운드 상세 페이지 (`InvestDetailPage`)
- **가치평가 계산 보기** (기본 펼침): 현재 가치 / 프리머니 / 모집 금액 / 포스트머니 2×2 그리드 + 공식 설명
- **투자 전 주의사항**: 원금 손실 가능성 + 분할 투자 설명 + 매수 즉시 지갑 차감 고지
- **매수 예상 결과**: 주식 수 입력 즉시 매수 금액 / 취득 지분 % / 투자 후 총 주식 수 자동 계산해서 표시
- 마지막 주식 매수 시 **"🎯 라운드 마감 + 회사 가치 재평가 예정"** 알림

### 3. 투자 페이지 (`InvestPage`)
- **투자 라운드 탭**: "투자 라운드란?" 도움말
- **포트폴리오 탭**: "포트폴리오 읽는 법" (보유 주식 / 지분율 / 현재 가치 / 수익·손실 정의)
- **배당금 탭**: "배당금이란?" (지분율 비례 분배 예시)

## 검증

### 백엔드 통합 테스트 (TDD)
`backend/tests/integration/investment_test.go` 신규 — 7개 시나리오:
- 단일 투자자 전액 매수 → 라운드 즉시 `funded`
- **분할 매수 다중 투자자**: Alice 1000주(40만) + Bob 1500주(마지막 = 60만) → 합 100만 정확히, status=funded, 회사 valuation=500만
- Bob의 overbuy(2000주, 남은 건 1500주) 거부
- Portfolio 응답 shape 검증(중첩 `company`, `invested_amount`, `profit`, `dividends_received` 등)
- Owner invest 거부
- 유효하지 않은 shares(0, 음수) 거부
- KPI 규칙 owner 검증
- 배당 실행 후 `dividends_received` 포트폴리오에 롤업

전체 통합 테스트 **245개 pass** (이전 238 + 신규 7).

### 프론트엔드
- typecheck OK
- vitest 75 pass
- vite build 성공

## 주요 파일 변경

| 파일 | 변경 |
|------|------|
| `backend/internal/domain/investment/entity.go` | `PortfolioItem` shape 재설계, `RoundCompany`/`RoundOwner` 중첩, `SoldShares`/`RemainingShares` 추가 |
| `backend/internal/domain/investment/errors.go` | `ErrInvalidShares`, `ErrOverSubscribed`, `ErrRoundExpired` 추가 |
| `backend/internal/domain/investment/repository.go` | `UpdateRoundCurrentAmount`, `SumSharesByRound`, `SumDividendsByUserAndCompany` 추가 |
| `backend/internal/application/investment_usecase.go` | `Invest` 분할 투자 로직, `GetPortfolio` 새 shape, `GetRound`, `maybeAutoExpire`, `CreateKpiRule` owner 검증 |
| `backend/internal/infrastructure/persistence/investment_repo.go` | 신규 repo 메서드들, 중첩 company/owner 채우는 Scan 확장 |
| `backend/internal/interfaces/http/handler/investment_handler.go` | `Invest` body에서 shares 파싱, `GetRound` 핸들러 |
| `backend/internal/interfaces/http/router/router.go` | `GET /investment/rounds/:id` 등록 |
| `backend/tests/integration/investment_test.go` | 신규 (7 tests) |
| `frontend/src/types/index.ts` | `InvestmentRound` 타입에 `remaining_shares` 등 추가 |
| `frontend/src/routes/invest/InvestPage.tsx` | 필터 수정, Dividend shape 수정, HelpBox 3종 |
| `frontend/src/routes/invest/InvestDetailPage.tsx` | 가치평가 계산기, HelpBox 2종, 실시간 매수 예상 |
| `frontend/src/routes/company/InvestmentRoundSection.tsx` | 신규: 회사 상세 페이지의 투자 유치 섹션 + 개설 다이얼로그 |
| `frontend/src/routes/company/CompanyDetailPage.tsx` | `InvestmentRoundSection` 통합 |

## 배운 점

- **"백엔드는 정상이지만 프론트가 크래시"**: API 스모크 테스트만으로는 부족합니다. 실제 유저가 페이지에서 상호작용할 때 어떤 데이터가 렌더되는지까지 봐야 합니다. ErrorBoundary에 걸려서 사용자는 빈 화면만 보는 상황이 있었습니다.
- **필드명 미스매치는 반드시 테스트 대상**: 이번에 추가한 `TestInvestment_Portfolio_ResponseShape` 같은 "응답 shape 검증" 테스트가 다음번에 비슷한 백엔드-프론트 분기를 막아줍니다.
- **교육용 UX는 설명 + 계산기 + 경고 3종 세트**: 텍스트 설명만 있으면 안 읽히고, 계산기만 있으면 왜 이 값이 나오는지 모릅니다. 세 가지를 같이 붙여야 실제로 학습이 일어납니다.
- **"분할 투자"는 숫자 반올림에서 마지막 한 명이 희생**: 마지막 구매자에게 `target - current`를 그대로 부담시켜 target에 딱 맞게 마감시키는 방식이 가장 깔끔합니다.
