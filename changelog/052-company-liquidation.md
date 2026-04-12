# 052. 회사 청산 기능

> **날짜**: 2026-04-12
> **태그**: `feat`, `회사`, `청산`, `세금`, `주주총회`, `백엔드`, `프론트엔드`

## 무엇을 했나요?

주주총회에서 청산 안건이 가결되면, 회사의 모든 자산을 **세금 20%** + **주주별 지분 분배**로 처리하고 회사를 영구 정지하는 **회사 청산 기능**을 만들었습니다.

## 왜 필요했나요?

스타트업 시뮬레이션에는 "회사를 접는" 경험이 필요합니다. 실패한 프로젝트를 정리하고, 남은 자본을 주주들에게 돌려주고, 세금을 내고, 새롭게 시작할 수 있어야 합니다. 실제 스타트업에서도 중요한 학습 포인트입니다.

## 청산 프로세스

### 1단계: 청산 안건 상정 (#022 시스템 활용)
1. 주주가 회사 상세 페이지에서 "안건 상정" → "회사 청산" 선택
2. 가결 기준 70% (기본값) 이상 설정
3. 다른 주주들이 찬성/반대 투표

### 2단계: 가결 확인
- 찬성 지분율 ≥ 70% 달성 시 자동 가결
- 또는 반대 지분율이 임계값을 초과하면 즉시 부결

### 3단계: 청산 집행
가결된 안건에 "청산 집행" 버튼이 표시됩니다 (주주만 클릭 가능):
1. 확인 다이얼로그 → 동의 시 실행
2. 회사 지갑 잔액의 **20% 세금** 계산
3. **나머지 80%를 주주별 지분율로 분배**
4. 각 주주의 개인 지갑에 분배금 입금
5. 회사 상태를 `dissolved`로 변경
6. 청산 공시 자동 생성 (투명성 확보)
7. 주주 전원에게 분배금 알림 전송

## API

### POST `/api/proposals/:pid/execute`
가결된 청산 안건을 집행합니다.

**권한**: 주주만 (403 `NOT_SHAREHOLDER`)
**전제 조건**:
- `proposal_type == 'liquidation'`
- `status == 'passed'`
- 회사 상태 `!= 'dissolved'`

**응답**:
```json
{
  "company_id": 1,
  "company_name": "acme",
  "total_balance": 10000000,
  "tax": 2000000,
  "distributable": 8000000,
  "payouts": [
    { "user_id": 1, "user_name": "Alice", "shares": 5000, "amount": 4000000 },
    { "user_id": 2, "user_name": "Bob",   "shares": 3000, "amount": 2400000 },
    { "user_id": 3, "user_name": "Carol", "shares": 2000, "amount": 1600000 }
  ],
  "residual_tax": 0,
  "executed_at": "2026-04-12T..."
}
```

## 분배 계산 예시

**시나리오**: 잔액 10,000,000원, 주주 3명 (50% / 30% / 20%)

| 항목 | 금액 |
|------|------|
| 총 자산 | 10,000,000원 |
| 세금 20% | 2,000,000원 |
| 분배 가능 | 8,000,000원 |
| Alice (50%) | 4,000,000원 |
| Bob (30%) | 2,400,000원 |
| Carol (20%) | 1,600,000원 |

### 나눗셈 반올림 처리
정수 나눗셈으로 발생하는 소수점 이하 잔여는 **추가 세금(residual_tax)** 으로 소멸됩니다. 예: 10,000,001원 → 세금 2,000,000, 분배 가능 8,000,001 → 분배 후 잔여 1원 → 추가 세금 1원.

## 청산 후 제약

회사가 `dissolved` 상태가 되면 다음 작업이 차단됩니다:
- 신규 공시 작성 (`POST /companies/:id/disclosures` → 400)
- 신규 주주총회 안건 상정 (`POST /companies/:id/proposals` → 400)
- (향후 확장) 신규 투자 라운드, 거래소 상장 등

## 어떻게 만들었나요?

### Backend

1. **도메인 확장**
   - `wallet.TxLiquidationPayout`, `wallet.TxLiquidationTax` TxType 추가
   - `notification.NotifLiquidationPayout` 알림 타입 추가
   - `company.CompanyRepository.UpdateStatus(companyID, status)` 추가

2. **유스케이스** (`application/proposal_usecase.go`)
   - `ExecuteLiquidation(proposalID, userID)` — 청산 집행 오케스트레이션
   - `LiquidationTaxPercent = 20` 상수
   - `LiquidationResult`, `LiquidationPayout` 반환 타입
   - 정수 나눗셈 반올림 잔여는 자동으로 세금에 추가

3. **HTTP 핸들러** (`interfaces/http/handler/proposal_handler.go`)
   - `ExecuteLiquidation` 핸들러 — 에러 코드 매핑

4. **라우터** — `POST /api/proposals/:pid/execute` 등록

5. **공시 가드** — 청산된 회사에 공시 작성 시도 시 400 에러

6. **TDD 통합 테스트** (`tests/integration/liquidation_test.go`) — 9개:
   - 단독 주주 전액 분배
   - 다수 주주 지분율 분배
   - 미가결 안건 집행 거부
   - 일반 안건 집행 거부
   - 비주주 집행 거부
   - 청산 후 신규 공시 차단
   - 청산 후 신규 안건 차단
   - 중복 집행 거부
   - 잔액 0원 청산도 정상 종료

### Frontend

1. **ProposalSection.tsx** — 가결된 청산 안건 카드에 "청산 집행" 버튼 추가
   - 확인 다이얼로그(`window.confirm`)로 실수 방지
   - 집행 후 토스트로 세금/분배 금액 표시
   - 회사 상태 리프레시 콜백

2. **CompanyDetailPage.tsx** — 청산 후 "청산됨" 뱃지 표시, `onCompanyChanged` 콜백

3. **NotificationsPage.tsx** — `liquidation_payout` 알림 지갑 아이콘

## 배운 점

- **범용 시스템 위에 특수 기능을 얹기**: #022 주주총회 투표 시스템이 범용이라 청산 기능은 추가 코드가 크지 않았습니다. 가결된 `liquidation` 타입 안건에 `execute` API 하나만 붙이면 됐습니다.
- **세금은 "소멸"로 구현**: 실제 시스템 지갑으로 송금하지 않고, 회사 지갑에서 차감만 하고 각 주주에게 분배하는 차액으로 세금을 자연스럽게 구현했습니다. 별도 treasury 계정이 필요 없습니다.
- **정수 연산의 잔여 처리**: 분배 시 `floor(distributable * shares / total)`을 사용하면 1~2원의 잔여가 생깁니다. 이를 마지막 주주에게 몰아주는 대신 "추가 세금"으로 처리하면 공정성과 단순함을 동시에 잡습니다.
- **돌이킬 수 없는 작업**: 청산은 복구가 불가능하므로 프론트에 `window.confirm`을 두고, 백엔드에 `double-execute` 차단 테스트를 꼭 추가했습니다.
