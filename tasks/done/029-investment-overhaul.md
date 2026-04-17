---
id: 029
title: 투자유치 기능 전면 개선 (UI 필드 매핑 + 분할 투자 + 상세 라우트 + owner 검증 + 만료)
priority: high
type: fix
branch: feat/investment-overhaul
created: 2026-04-16
---

## 배경

스테이지 검증 결과 투자유치 기능에 여러 버그 발견 (see 이전 세션 기록).

## 수정할 버그

### Bug A: `/invest` 목록 항상 비어 보임 (critical)
- 프론트가 `GET /investment/rounds?status=active` 로 필터
- 백엔드 상태값은 `open|funded|failed|cancelled` → active는 0건
- **수정**: 프론트 필터를 `open` 으로 변경

### Bug B: `/invest/:id` 상세 페이지 영원히 로딩 (critical)
- 프론트가 `GET /investment/rounds/:id` 호출
- 백엔드에 이 라우트 없음 (list 만 있음)
- **수정**: 백엔드에 `GET /investment/rounds/:id` 추가 + 핸들러 + (repo FindRoundByID 이미 있음)

### Bug C: 포트폴리오/배당금 탭 전체 크래시 (critical)
- Backend PortfolioItem → flat `company_id, company_name, invested, user_shares` 
- Frontend → nested `company.id, company.name, company.valuation, shares, invested_amount, profit, dividends_received`
- ErrorBoundary: "Cannot read properties of undefined (reading 'name')"
- **수정**: 백엔드 PortfolioItem + DividendPayment 응답에 프론트가 기대하는 필드 추가
  - `company: { id, name, valuation }` 중첩 객체 추가
  - `shares` (alias for user_shares)
  - `invested_amount` (alias for invested)
  - `profit` = current_value - invested_amount
  - `dividends_received` = sum(dividend_payments.amount WHERE user_id = me AND dividend.company_id = companyID)

### Bug D: 분할 투자 불가 (critical) — 옵션 2로 구현
- 현재: 1라운드 = 1투자자 = 전액 일시불 (target_amount 통째로 debit)
- 문제: 프론트는 주식 수 입력받지만 백엔드가 무시 + 소액 분할 불가
- **수정**: 여러 명이 부분적으로 투자 가능하도록 변경
  - `Invest(roundID, userID, shares)` — shares 파라미터 추가
  - Validate: `1 <= shares <= remaining_shares`
  - `investAmount`: 
    - 마지막 구매자 (`remaining == shares`): `target_amount - current_amount` (exact close)
    - 그 외: `round(shares * price_per_share)`
  - round.current_amount 누적 증가
  - 최종 도달 시 status='funded', funded_at=NOW, valuation = target_amount / offered_percent
  - Multiple investments per round 지원 (investments table은 이미 FK만 있어서 가능)
  - Shareholders upsert는 이미 additive라 OK

### Bug E: CreateKpiRule에 owner 검증 없음 (medium)
- 아무나 남의 회사 KPI 규칙 생성 가능
- **수정**: `CreateKpiRule(input, userID)` 로 시그니처 변경, owner 체크

### Bug F: 라운드 만료 처리 (low)
- `expires_at` 필드 있지만 체크 안함
- **수정**: 라운드 조회 시 만료되었으면 `failed` 로 자동 마감

## 작업 범위

### Backend
- `backend/internal/application/investment_usecase.go`
  - `Invest(roundID, userID, shares)` 분할 투자 로직
  - `GetPortfolio` 반환 타입 개선 (profit, dividends_received, 중첩 company)
  - `GetMyDividends` 반환 타입 개선
  - `GetRound(id, userID)` 신규
  - `CreateKpiRule(input, userID)` owner 검증
  - 만료 자동 처리 (maybeAutoExpire)
- `backend/internal/domain/investment/entity.go`
  - `PortfolioItem` 필드 확장
  - `DividendPayment` JSON shape
  - 새 DTO들 필요 시
- `backend/internal/infrastructure/persistence/investment_repo.go`
  - `GetSoldShares(roundID)` or 기존 `ListByRound` 재사용
  - `UpdateRoundCurrentAmount(id, amount)` (partial update)
- `backend/internal/interfaces/http/handler/investment_handler.go`
  - `GetRound` handler
  - `Invest` 요청 body `{shares}` 파싱
- `backend/internal/interfaces/http/router/router.go`
  - `GET /investment/rounds/:id` 등록

### Frontend
- `frontend/src/routes/invest/InvestPage.tsx`
  - `?status=active` → `?status=open`
  - Portfolio / Dividend 렌더링 로직 새 shape에 맞춤
- `frontend/src/routes/invest/InvestDetailPage.tsx`
  - 실제 입력된 shares 반영
  - remaining shares 계산
  - 진행률 UI

### Tests (TDD)
- `backend/tests/integration/investment_test.go` 신규
  - 전체 투자 (기존 기능 유지 확인)
  - 부분 투자 + 다중 투자자
  - 마지막 구매로 정확히 target 도달
  - Overbuy 거부
  - Owner invest 거부
  - Non-shareholder (본인 회사) 거부
  - GetRound endpoint
  - Portfolio/Dividend 응답 shape
  - KPI owner 검증
  - 만료 처리

## 검증

- [ ] 백엔드 테스트 전체 통과
- [ ] 프론트 typecheck/build 통과  
- [ ] 스테이지 배포 후 agent-browser로 end-to-end 테스트
  - 라운드 생성 → 목록에 노출
  - 상세 페이지 로드
  - 부분 투자 (N주)
  - 다른 투자자 추가 투자
  - 마지막 구매로 target 도달 → 상태 funded 전환
  - 포트폴리오 정상 표시 (크래시 없음)
  - 배당 실행 → 배당금 탭 표시
