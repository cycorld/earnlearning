# 051. 주주총회 투표 시스템

> **날짜**: 2026-04-12
> **태그**: `feat`, `주주총회`, `투표`, `회사`, `백엔드`, `프론트엔드`

## 무엇을 했나요?

회사의 중요한 의사결정을 주주들이 직접 결정할 수 있는 **주주총회 투표 시스템**을 만들었습니다. 주주가 안건을 상정하고, 다른 주주들이 찬성/반대로 투표하면, 지분율에 따라 자동으로 가결/부결이 결정됩니다.

## 왜 필요했나요?

회사 청산(#023) 같은 중대한 결정을 혼자 결정할 수 있다면 소수 주주의 권리가 무시됩니다. 그래서 먼저 범용적인 주주총회 투표 시스템을 만들고, 청산 기능은 이 시스템 위에서 동작하도록 설계했습니다.

## 주요 기능

### 1. 안건 상정 (`POST /companies/:id/proposals`)
- 주주(지분 > 0)만 상정 가능
- 안건 종류: `general`(일반) / `liquidation`(청산)
- 가결 기준(%) 커스텀 가능 (기본: 일반 50%, 청산 70%)
- 투표 기간(일) 설정
- 같은 종류의 활성 안건이 이미 있으면 중복 상정 불가

### 2. 투표 (`POST /proposals/:pid/vote`)
- 안건이 올라온 시점이 아닌 **투표 시점의 지분**을 스냅샷으로 기록
- 찬성/반대 중 택일, 1회만 가능
- 비주주는 투표 불가 (403 `NOT_SHAREHOLDER`)

### 3. 자동 집계 & 마감
투표마다 즉시 재집계하여 다음 조건에서 자동 마감합니다:
- **가결**: `찬성 지분율 >= 가결 기준` — 예: 70% 기준에서 75% 찬성 달성
- **부결 확정**: `반대 지분율 > (100 - 가결 기준)` — 남은 지분이 모두 찬성해도 가결 불가
- **기간 만료**: `end_date` 도달 시 자동 마감 (GET 요청 시에도 체크)

### 4. 알림 연동
- 안건 상정 → 모든 주주에게 `proposal_started` 알림
- 안건 마감 → 모든 주주에게 `proposal_closed` 알림 (결과 포함)

### 5. 프론트엔드 UI
회사 상세 페이지에 **주주총회 섹션** 추가:
- 안건 카드에 찬성/반대 비율을 3색 스택 바로 시각화
- 주주는 카드에서 바로 찬성/반대 버튼으로 투표
- 내 투표 기록, 마감 결과, 상정자 정보 표시

## 어떻게 만들었나요?

### 백엔드 (Go + SQLite)

1. **도메인 모델** (`domain/company/proposal.go`)
   - `Proposal`, `Vote`, `ProposalTally` 구조체
   - 타입/상태/선택지 상수

2. **마이그레이션** (`persistence/sqlite.go`)
   ```sql
   CREATE TABLE shareholder_proposals (...)
   CREATE TABLE shareholder_votes (..., UNIQUE(proposal_id, user_id))
   ```
   `shareholders` 테이블의 상태를 건드리지 않고 투표 시점 지분만 `shares_at_vote`로 스냅샷.

3. **유스케이스** (`application/proposal_usecase.go`)
   - `CreateProposal`, `CastVote`, `GetProposalsByCompanyID`, `GetProposal`, `CancelProposal`
   - `tallyProposal` — 투표 집계 + 프로젝티드 상태 계산
   - `maybeAutoClose` — 투표 후 즉시 재집계해서 마감 여부 결정
   - `closeProposal` — 결과 확정 + 주주 알림 전송

4. **HTTP 핸들러 & 라우터**
   ```
   POST   /api/companies/:id/proposals
   GET    /api/companies/:id/proposals
   GET    /api/proposals/:pid
   POST   /api/proposals/:pid/vote
   POST   /api/proposals/:pid/cancel
   ```

5. **TDD 통합 테스트** (`tests/integration/proposal_test.go`) — 9개
   - 단독 주주 가결
   - 청산 안건 기본 임계값(70%)
   - 비주주 상정/투표 차단
   - 중복 투표 차단
   - 임계값 도달 자동 가결
   - 수학적 부결 확정 (남은 지분 전부 찬성해도 불가능한 경우)
   - 목록 조회
   - 동일 종류 중복 활성 안건 차단

### 프론트엔드 (React + TypeScript)

1. **타입 정의** (`types/index.ts`) — `Proposal`, `ProposalTally`, `ProposalVote`, `VoteChoice` 등
2. **`ProposalSection.tsx`** — 안건 목록/상정 다이얼로그/투표 버튼/결과 시각화
3. **`CompanyDetailPage.tsx`** — 회사 상세 페이지에 섹션 추가, 주주 여부 계산 (`shareholders`에서 찾음)
4. **알림 매핑** — `proposal_started`/`proposal_closed` 아이콘 + `/proposal/:id` 라우트 (스켈레톤)

## 기술적 포인트

### 투표 시점 지분 스냅샷
투표 후 지분이 바뀌더라도(거래소 매매 등) **투표 당시의 지분**이 계속 유효합니다. 이를 위해 `shares_at_vote` 컬럼에 투표 시점의 지분을 기록하고, 집계 시 이 값을 사용합니다.

### 수학적 부결 감지
예: 70% 기준, 총 지분 10000. 반대가 3500주(35%) 찍히면 → 남은 6500주 전부 찬성해도 65% < 70% → **바로 부결 확정**. 쓸데없이 투표 기간을 끌지 않습니다.

### 자동 마감 (Write & Read 양쪽)
- `CastVote` 후: 즉시 `maybeAutoClose` 호출
- `GetProposal`/`GetProposalsByCompanyID` 조회 시: `end_date`가 지났으면 그 자리에서 마감
  → 크론 없이도 정확한 상태 유지

## 다음 단계

이제 #023 회사 청산 기능이 이 시스템을 활용합니다:
1. 청산 안건(`liquidation` type)을 상정
2. 주주 70% 이상 찬성 시 가결
3. 가결 상태에서 별도 API로 실제 청산 집행 (세금 20% + 지분별 분배)

## 배운 점

- **범용 시스템 먼저, 특수 케이스 나중에**: 청산 전용 시스템이 아니라 범용 투표 시스템을 만들고 청산은 그 위에서 동작하게 했습니다. 재사용성이 높아지고 테스트도 더 단순해집니다.
- **수학으로 기간 단축**: 모든 지분이 투표하기를 기다릴 필요가 없습니다. 가결 또는 부결이 "수학적으로 확정"되면 그 자리에서 마감하는 게 사용자에게 훨씬 직관적입니다.
- **투표 시점 스냅샷**: 지분이 유동적인 환경에서는 "당시 결정권"을 고정해두지 않으면 결과가 왜곡됩니다.
