# 09. Bank Domain — 대출

## 1. 유저 스토리

| ID | 역할 | 스토리 | 우선순위 |
|----|------|--------|---------|
| BNK-01 | 학생 | 대출을 신청할 수 있다 (금액, 용도) | P0 |
| BNK-02 | Admin | 대출 신청을 심사하여 승인/거절할 수 있다 (이자율 설정) | P0 |
| BNK-03 | 학생 | 내 대출 현황 (원금, 잔여, 이자율, 다음 납부일)을 볼 수 있다 | P0 |
| BNK-04 | 학생 | 대출 원금/이자를 상환할 수 있다 | P0 |
| BNK-05 | 시스템 | 매주 자동으로 이자를 차감한다 | P0 |
| BNK-06 | 시스템 | 미납 시 연체 상태로 전환하고 연체 이자를 부과한다 | P0 |

---

## 2. 도메인 모델

### Entity

```go
type LoanStatus string
const (
    LoanPending  LoanStatus = "pending"
    LoanRejected LoanStatus = "rejected"
    LoanActive   LoanStatus = "active"
    LoanPaid     LoanStatus = "paid"
    LoanOverdue  LoanStatus = "overdue"
)

type Loan struct {
    ID           int
    BorrowerID   int
    Amount       int           // 대출 원금
    Remaining    int           // 잔여 원금
    InterestRate float64       // 주당 이자율 (예: 0.05 = 5%)
    PenaltyRate  float64       // 연체 이자율 (기본: interest_rate × 2)
    Purpose      string
    Status       LoanStatus
    ApprovedBy   *int
    ApprovedAt   *time.Time
    NextPayment  *time.Time    // 다음 이자 납부일
    CreatedAt    time.Time
}

type PayType string
const (
    PayInterest  PayType = "interest"
    PayRepayment PayType = "repayment"
    PayPenalty   PayType = "penalty"
    PayAuto      PayType = "auto"
)

type LoanPayment struct {
    ID        int
    LoanID    int
    Amount    int             // 총 납부 금액
    Principal int             // 원금 상환분
    Interest  int             // 이자분
    Penalty   int             // 연체 이자분
    PayType   PayType
    CreatedAt time.Time
}
```

### 도메인 규칙

- **이자 계산**: `weekly_interest = remaining × interest_rate`
- **연체 이자**: `overdue_interest = remaining × penalty_rate`
- **penalty_rate**: 승인 시 `interest_rate × 2`로 설정
- **상환 우선순위**: 연체이자 → 이자 → 원금
- **완납 조건**: `remaining == 0` → status = 'paid'
- **시간 규칙**: 1주일 = 1년, 이자는 주당 적용

---

## 3. 데이터베이스 스키마

```sql
CREATE TABLE loans (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    borrower_id   INTEGER NOT NULL REFERENCES users(id),
    amount        INTEGER NOT NULL,
    remaining     INTEGER NOT NULL,
    interest_rate REAL    NOT NULL,
    penalty_rate  REAL    NOT NULL DEFAULT 0,       -- 승인 시 interest_rate × 2로 설정
    purpose       TEXT    DEFAULT '',
    status        TEXT    NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','rejected','active','paid','overdue')),
    approved_by   INTEGER REFERENCES users(id),
    approved_at   DATETIME,
    next_payment  DATETIME,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE loan_payments (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    loan_id   INTEGER NOT NULL REFERENCES loans(id),
    amount    INTEGER NOT NULL,
    principal INTEGER NOT NULL DEFAULT 0,
    interest  INTEGER NOT NULL DEFAULT 0,
    penalty   INTEGER NOT NULL DEFAULT 0,
    pay_type  TEXT    NOT NULL
              CHECK (pay_type IN ('interest', 'repayment', 'penalty', 'auto')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 4. API 상세

### `POST /api/bank/loans/apply`
**미들웨어**: Approved

```json
// Request
{ "amount": 10000000, "purpose": "프로젝트 투자 자금" }

// Response 201
{ "data": { "id": 1, "amount": 10000000, "status": "pending" } }
```

---

### `GET /api/bank/loans`
**미들웨어**: Approved

```json
// Response 200
{
  "data": [
    {
      "id": 1,
      "amount": 10000000,
      "remaining": 8500000,
      "interest_rate": 0.05,
      "penalty_rate": 0.10,
      "purpose": "프로젝트 투자 자금",
      "status": "active",
      "next_payment": "2026-03-20T00:00:00Z",
      "weekly_interest": 425000,
      "created_at": "2026-03-06T10:00:00Z"
    }
  ]
}
```

---

### `PUT /api/bank/loans/:id/approve`
**미들웨어**: Admin

```json
// Request
{ "interest_rate": 0.05 }

// Response 200
{
  "data": {
    "id": 1,
    "status": "active",
    "interest_rate": 0.05,
    "penalty_rate": 0.10,
    "next_payment": "2026-03-20T00:00:00Z"
  }
}
```

**비즈니스 로직**:
1. loan.status → 'active'
2. loan.interest_rate = 요청값
3. loan.penalty_rate = interest_rate × 2
4. 대출금 → 학생 개인 지갑 입금 (tx_type: `loan_disbursement`)
5. loan.next_payment = now + 7일
6. loan.approved_by, approved_at 설정
7. 학생에게 알림

---

### `PUT /api/bank/loans/:id/reject`
**미들웨어**: Admin

```json
// Response 200
{ "data": { "id": 1, "status": "rejected" } }
```

---

### `POST /api/bank/loans/:id/repay`
**미들웨어**: Approved

```json
// Request
{ "amount": 2000000 }

// Response 200
{
  "data": {
    "loan_id": 1,
    "paid_amount": 2000000,
    "penalty_paid": 0,
    "interest_paid": 425000,
    "principal_paid": 1575000,
    "remaining": 6925000,
    "status": "active"
  }
}
```

**비즈니스 로직** (트랜잭션):
1. 학생 지갑 잔고 확인 (잔고 ≥ amount)
2. 지갑에서 차감 (tx_type: `loan_repayment`)
3. 상환 분배 (우선순위):
   - 연체 이자 (penalty) 먼저 차감
   - 미납 이자 (interest) 차감
   - 나머지 → 원금 상환
4. loan.remaining 업데이트
5. loan_payments 레코드 생성
6. remaining == 0 → loan.status = 'paid'

---

## 5. 이자 자동 차감 (주간 배치)

```
매주 월요일 00:00 실행 (또는 Admin API 호출):
  → active + overdue 상태 대출 전체 조회
  → FOR EACH loan:
      → accrued_interest = remaining × interest_rate
      → IF status == 'overdue':
          → accrued_interest = remaining × penalty_rate
      → IF 지갑 잔고 >= accrued_interest:
          → 지갑 차감 (tx_type: 'loan_interest')
          → loan_payments 생성 (pay_type: 'auto')
          → loan.next_payment += 7일
          → IF status == 'overdue': status = 'active' (정상화)
      → ELSE:
          → loan.status = 'overdue'
          → 연체 알림 발송
```

---

## 6. UI 스펙

### 6.1 은행 메인 (`/bank`)

```
┌─────────────────────────────────┐
│  은행                            │
│                                 │
│  ── 내 대출 ────────────────     │
│  ┌─────────────────────────┐   │
│  │ 대출 #1                   │   │
│  │ 원금: 1,000만원           │   │
│  │ 잔여: 850만원             │   │
│  │ 이자율: 주당 5%           │   │
│  │ 다음 이자: 42.5만원       │   │
│  │ 납부일: 2026-03-20        │   │
│  │ 상태: ● 정상              │   │
│  │ [상환하기]                │   │
│  └─────────────────────────┘   │
│                                 │
│  ┌─────────────────────────┐   │
│  │ 대출 #2                   │   │
│  │ 잔여: 500만원             │   │
│  │ 상태: 🔴 연체             │   │
│  │ 연체이자율: 10%           │   │
│  │ [상환하기]                │   │
│  └─────────────────────────┘   │
│                                 │
│  ┌─────────────────────────┐   │
│  │     + 대출 신청하기        │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
```

### 6.2 대출 신청 (`/bank/apply`)

```
┌─────────────────────────────────┐
│  ← 대출 신청                     │
│                                 │
│  대출 금액: [          ] 원     │
│                                 │
│  용도:                           │
│  ┌─────────────────────────┐   │
│  │                          │   │
│  │ (예: 프로젝트 투자 자금)  │   │
│  └─────────────────────────┘   │
│                                 │
│  ⚠️ 이자율은 관리자가 설정합니다  │
│  ⚠️ 매주 이자가 자동 차감됩니다  │
│  ⚠️ 미납 시 연체이자(2배)가      │
│     부과됩니다                   │
│                                 │
│  ┌─────────────────────────┐   │
│  │       신청하기            │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
```

### 6.3 상환 (모달)

```
┌─────────────────────────────────┐
│  대출 상환                       │
│                                 │
│  잔여 원금: 850만원              │
│  미납 이자: 42.5만원             │
│                                 │
│  상환 금액: [          ] 원     │
│                                 │
│  [전액 상환: 8,925,000원]       │
│                                 │
│  예상 결과:                      │
│  이자 상환: 425,000원            │
│  원금 상환: 1,575,000원          │
│  잔여 원금: 692.5만원            │
│                                 │
│  내 잔고: 4,500만원              │
│                                 │
│  [상환하기]  [취소]              │
└─────────────────────────────────┘
```

### 6.4 Admin: 대출 심사 (`/admin/loans`)

```
┌─────────────────────────────────┐
│  대출 심사                       │
│  ┌──────┬──────┬──────┐        │
│  │ 대기중 │ 진행중 │ 완료  │        │
│  └──────┴──────┴──────┘        │
│                                 │
│  ┌─────────────────────────┐   │
│  │ 김학생                    │   │
│  │ 신청: 1,000만원           │   │
│  │ 용도: 프로젝트 투자 자금   │   │
│  │ 현재 자산: 4,500만원       │   │
│  │ 기존 대출: 없음            │   │
│  │                          │   │
│  │ 이자율: [5  ] %/주        │   │
│  │ [승인]  [거절]            │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
```
