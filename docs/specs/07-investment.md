# 07. Investment Domain — 투자, 배당, KPI

## 1. 유저 스토리

| ID | 역할 | 스토리 | 우선순위 |
|----|------|--------|---------|
| INV-01 | 설립자 | 투자 라운드를 생성할 수 있다 (모집 금액, 양도 지분율) | P0 |
| INV-02 | 학생 | IR 목록에서 투자 가능한 라운드를 조회할 수 있다 | P0 |
| INV-03 | 학생 | 투자 라운드에 전액 투자할 수 있다 (1라운드 = 1투자자) | P0 |
| INV-04 | 학생 | 내 투자 포트폴리오를 조회할 수 있다 | P1 |
| INV-05 | 설립자 | 배당을 실행하여 주주에게 수익을 분배할 수 있다 | P0 |
| INV-06 | 학생 | 배당 수령 내역을 조회할 수 있다 | P1 |
| INV-07 | Admin | 회사에 KPI 규칙을 설정할 수 있다 | P1 |
| INV-08 | Admin | KPI 실적에 따라 회사에 가상 소득을 부여할 수 있다 | P1 |

---

## 2. 도메인 모델

### Entity

```go
type RoundStatus string
const (
    RoundOpen      RoundStatus = "open"
    RoundFunded    RoundStatus = "funded"
    RoundFailed    RoundStatus = "failed"
    RoundCancelled RoundStatus = "cancelled"
)

type InvestmentRound struct {
    ID              int
    CompanyID       int
    PostID          *int          // IR 게시글 연결
    TargetAmount    int           // 모집 금액
    OfferedPercent  float64       // 양도 지분율 (0.01~0.99)
    CurrentAmount   int           // 현재 모집 금액 (0 또는 target_amount)
    PricePerShare   float64       // 주당 가격
    NewShares       int           // 발행 예정 신주 수
    Status          RoundStatus
    ExpiresAt       *time.Time
    CreatedAt       time.Time
    FundedAt        *time.Time
}

type Investment struct {
    ID        int
    RoundID   int
    UserID    int
    Amount    int           // 투자 금액 (= target_amount)
    Shares    int           // 배정 주식 수 (= new_shares)
    CreatedAt time.Time
}

type Dividend struct {
    ID          int
    CompanyID   int
    TotalAmount int
    ExecutedBy  int          // 설립자 user_id
    CreatedAt   time.Time
}

type DividendPayment struct {
    ID         int
    DividendID int
    UserID     int
    Shares     int           // 배당 시점 보유 주식
    Amount     int           // 수령 금액
    CreatedAt  time.Time
}

type KpiRule struct {
    ID              int
    CompanyID       int
    RuleDescription string    // "일일 방문자 100명당 10만원"
    Active          bool
    CreatedAt       time.Time
}

type KpiRevenue struct {
    ID        int
    CompanyID int
    KpiRuleID *int
    Amount    int
    Memo      string
    CreatedBy int            // Admin
    CreatedAt time.Time
}
```

### 핵심 규칙: 1라운드 = 1투자자

- 부분 펀딩 없음 → 한 투자자가 `target_amount` 전액 투자
- 투자 시 즉시 펀딩 확정
- 라운드 만료 시 자동 실패 (환불 없음 — 투자 전이므로)

### 신주 계산 공식

```
new_shares     = total_shares × offered_percent / (1 - offered_percent)
price_per_share = target_amount / new_shares
post_money     = target_amount / offered_percent
pre_money      = post_money - target_amount
```

**예시**: 설립자 10,000주(100%), 1,000만원 모집, 지분 20% 제안
- new_shares = 10,000 × 0.20 / 0.80 = 2,500주
- price_per_share = 10,000,000 / 2,500 = 4,000원
- post_money = 10,000,000 / 0.20 = 50,000,000원
- 설립자 10,000주(80%), 투자자 2,500주(20%)

---

## 3. 데이터베이스 스키마

```sql
CREATE TABLE investment_rounds (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id      INTEGER NOT NULL REFERENCES companies(id),
    post_id         INTEGER REFERENCES posts(id),
    target_amount   INTEGER NOT NULL,
    offered_percent REAL    NOT NULL CHECK (offered_percent > 0 AND offered_percent < 1),
    current_amount  INTEGER NOT NULL DEFAULT 0,
    price_per_share REAL    NOT NULL,
    new_shares      INTEGER NOT NULL,
    status          TEXT    NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open', 'funded', 'failed', 'cancelled')),
    expires_at      DATETIME,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    funded_at       DATETIME
);

CREATE TABLE investments (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    round_id   INTEGER NOT NULL REFERENCES investment_rounds(id),
    user_id    INTEGER NOT NULL REFERENCES users(id),
    amount     INTEGER NOT NULL,
    shares     INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE dividends (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id   INTEGER NOT NULL REFERENCES companies(id),
    total_amount INTEGER NOT NULL,
    executed_by  INTEGER NOT NULL REFERENCES users(id),
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE dividend_payments (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    dividend_id INTEGER NOT NULL REFERENCES dividends(id),
    user_id     INTEGER NOT NULL REFERENCES users(id),
    shares      INTEGER NOT NULL,
    amount      INTEGER NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE kpi_rules (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id       INTEGER NOT NULL REFERENCES companies(id),
    rule_description TEXT    NOT NULL,
    active           INTEGER NOT NULL DEFAULT 1,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE kpi_revenues (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id  INTEGER NOT NULL REFERENCES companies(id),
    kpi_rule_id INTEGER REFERENCES kpi_rules(id),
    amount      INTEGER NOT NULL,
    memo        TEXT    DEFAULT '',
    created_by  INTEGER NOT NULL REFERENCES users(id),
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 4. API 상세

### `POST /api/companies/:id/rounds`
**미들웨어**: Approved (설립자만)

```json
// Request
{
  "target_amount": 10000000,
  "offered_percent": 0.20,
  "expires_at": "2026-04-01T00:00:00Z"
}

// Response 201
{
  "data": {
    "id": 1,
    "company_id": 1,
    "target_amount": 10000000,
    "offered_percent": 0.20,
    "new_shares": 2500,
    "price_per_share": 4000,
    "pre_money_valuation": 40000000,
    "post_money_valuation": 50000000,
    "status": "open"
  }
}
```

- IR 게시글을 #투자라운지 채널에 자동 생성
- 해당 회사에 open 라운드가 이미 있으면 거절

---

### `POST /api/rounds/:id/invest`
**미들웨어**: Approved

```json
// Request (금액 불필요 — target_amount 전액 투자)
{}

// Response 200
{
  "data": {
    "investment_id": 1,
    "amount": 10000000,
    "shares_acquired": 2500,
    "round_status": "funded",
    "company": {
      "total_shares": 12500,
      "valuation": 50000000,
      "listed": true
    }
  }
}
```

**비즈니스 로직** (트랜잭션):
1. 라운드 status == 'open' 확인
2. 투자자 ≠ 설립자 확인
3. 투자자 개인 지갑 잔고 확인 (잔고 ≥ target_amount)
4. 투자자 지갑에서 target_amount 차감 (tx_type: `investment`)
5. investments 레코드 생성 (amount = target_amount, shares = new_shares)
6. **즉시 펀딩 확정**:
   - round.current_amount = target_amount
   - round.status = 'funded'
   - round.funded_at = now
   - company.total_shares += new_shares
   - shareholders UPSERT (user_id = 투자자, shares = new_shares, type = 'investment')
   - company_wallet.balance += target_amount
   - company.total_capital += target_amount
   - company.valuation = target_amount / offered_percent (post_money)
   - company.CheckListing() → total_capital ≥ 5,000만원이면 listed = true (영구)
   - 설립자 & 투자자 알림

---

### `GET /api/rounds`
**미들웨어**: Approved

```
Query: ?status=open&page=1&limit=20
```

```json
// Response 200
{
  "data": [
    {
      "id": 1,
      "company": { "id": 1, "name": "우리회사", "valuation": 5000000, "logo_url": "..." },
      "owner": { "id": 2, "name": "김이화" },
      "target_amount": 10000000,
      "offered_percent": 0.20,
      "price_per_share": 4000,
      "new_shares": 2500,
      "status": "open",
      "expires_at": "2026-04-01T00:00:00Z",
      "created_at": "2026-03-10T12:00:00Z"
    }
  ],
  "pagination": { ... }
}
```

---

### `GET /api/portfolio`
**미들웨어**: Approved

```json
// Response 200
{
  "data": {
    "investments": [
      {
        "company": { "id": 1, "name": "우리회사", "valuation": 50000000 },
        "shares": 2500,
        "percentage": 20.0,
        "invested_amount": 10000000,
        "current_value": 10000000,
        "profit": 0,
        "dividends_received": 200000
      }
    ],
    "total_invested": 10000000,
    "total_current_value": 10000000,
    "total_dividends": 200000
  }
}
```

---

### `POST /api/companies/:id/dividend`
**미들웨어**: Approved (설립자만)

```json
// Request
{ "amount": 1000000 }

// Response 200
{
  "data": {
    "dividend_id": 1,
    "total_amount": 1000000,
    "payments": [
      { "user_id": 2, "name": "김이화", "shares": 10000, "percentage": 80.0, "amount": 800000 },
      { "user_id": 5, "name": "박투자", "shares": 2500, "percentage": 20.0, "amount": 200000 }
    ]
  }
}
```

**비즈니스 로직** (트랜잭션):
1. 요청자 == owner_id 확인
2. 회사 지갑 잔고 ≥ amount 확인
3. 회사 지갑에서 차감 (tx_type: `dividend_out`)
4. 각 주주에게 `floor(amount × 보유주식 / 총주식)` 개인 지갑 입금 (tx_type: `dividend`)
5. 단수 차이(나머지)는 회사 지갑에 잔류
6. dividend, dividend_payments 레코드 생성
7. 각 주주에게 알림 발송

---

### `GET /api/dividends/my`
**미들웨어**: Approved

```json
// Response 200
{
  "data": [
    {
      "id": 1,
      "company": { "id": 1, "name": "우리회사" },
      "shares": 2500,
      "amount": 200000,
      "total_dividend": 1000000,
      "created_at": "2026-03-12T14:00:00Z"
    }
  ]
}
```

---

### `POST /api/companies/:id/kpi-rules`
**미들웨어**: Admin

```json
// Request
{ "rule_description": "일일 방문자 100명당 10만원" }

// Response 201
{ "data": { "id": 1, "rule_description": "일일 방문자 100명당 10만원", "active": true } }
```

---

### `POST /api/companies/:id/revenue`
**미들웨어**: Admin

```json
// Request
{ "kpi_rule_id": 1, "amount": 500000, "memo": "이번주 방문자 500명 달성" }

// Response 200
{ "data": { "id": 1, "amount": 500000, "company_wallet_balance": 9000000 } }
```

**비즈니스 로직**:
1. 회사 지갑에 입금 (tx_type: `kpi_revenue`)
2. kpi_revenues 레코드 생성
3. 설립자에게 알림

---

## 5. 투자 전체 흐름도

```
설립자: POST /companies/:id/rounds
  → 라운드 생성 (status: open)
  → IR 게시글 자동 생성 (#투자라운지)

투자자: POST /rounds/:id/invest
  → BEGIN TRANSACTION
  → 투자자 지갑 잔고 확인 (≥ target_amount)
  → 투자자 지갑 차감 (tx_type: investment)
  → investments 레코드 생성
  → 즉시 펀딩 확정:
      → round.status = 'funded'
      → company.total_shares += new_shares
      → shareholders UPSERT (투자자)
      → company_wallet.balance += target_amount
      → company.total_capital += target_amount
      → company.valuation = post_money
      → company.CheckListing()
      → 설립자 & 투자자 알림
  → COMMIT
```

---

## 6. UI 스펙

### 6.1 투자 라운드 목록 (`/invest`)

```
┌─────────────────────────────────┐
│  투자 라운드                     │
│  ┌──────┬──────┐               │
│  │ 모집중 │ 완료  │               │
│  └──────┴──────┘               │
│                                 │
│  ┌─────────────────────────┐   │
│  │ 🏢 우리회사               │   │
│  │ 김이화 · 기업가치 500만원  │   │
│  │                          │   │
│  │ 모집: 1,000만원           │   │
│  │ 지분: 20%                │   │
│  │ 주당: 4,000원             │   │
│  │ 마감: 2026-04-01          │   │
│  │                          │   │
│  │ [투자하기]                │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
```

### 6.2 투자 상세 (`/invest/[roundId]`)

```
┌─────────────────────────────────┐
│  ← 투자 라운드 상세              │
│                                 │
│  🏢 우리회사                     │
│  설립자: 김이화                  │
│                                 │
│  ── 라운드 정보 ────────────     │
│  모집 금액: 1,000만원            │
│  양도 지분: 20%                 │
│  신주 발행: 2,500주              │
│  주당 가격: 4,000원              │
│  Pre-money: 4,000만원           │
│  Post-money: 5,000만원          │
│  마감일: 2026-04-01             │
│                                 │
│  ── 현재 지분 구조 ─────────     │
│  김이화: 10,000주 (100%)        │
│                                 │
│  ── 투자 후 지분 구조 ──────     │
│  김이화: 10,000주 (80%)         │
│  나: 2,500주 (20%)              │
│                                 │
│  내 잔고: 4,500만원              │
│                                 │
│  ┌─────────────────────────┐   │
│  │   1,000만원 투자하기      │   │
│  └─────────────────────────┘   │
│  ⚠️ 전액 투자입니다. 취소 불가.  │
└─────────────────────────────────┘
```

### 6.3 내 포트폴리오 (탭 또는 별도 섹션)

```
┌─────────────────────────────────┐
│  내 투자 포트폴리오              │
│                                 │
│  총 투자금: 1,000만원            │
│  현재 가치: 1,050만원            │
│  누적 배당: 20만원               │
│                                 │
│  ┌─────────────────────────┐   │
│  │ 우리회사                  │   │
│  │ 2,500주 (20%)            │   │
│  │ 투자: 1,000만원           │   │
│  │ 현재: 1,050만원 (+5%)    │   │
│  │ 배당: 20만원              │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
```

### 6.4 배당 실행 (설립자 — 회사 상세 내)

```
┌─────────────────────────────────┐
│  ← 배당 실행                     │
│                                 │
│  회사 지갑 잔고: 850만원         │
│                                 │
│  배당 금액: [         ] 원      │
│                                 │
│  ── 배당 예상 ──────────────     │
│  김이화 (80%) → 800,000원       │
│  박투자 (20%) → 200,000원       │
│                                 │
│  ┌─────────────────────────┐   │
│  │       배당 실행           │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
```
