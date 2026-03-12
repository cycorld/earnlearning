# 04. Wallet Domain — 지갑 & 자산 관리

## 1. 유저 스토리

| ID | 역할 | 스토리 | 우선순위 |
|----|------|--------|---------|
| WLT-01 | 학생 | 내 현금 잔고와 총 자산가치를 확인할 수 있다 | P0 |
| WLT-02 | 학생 | 거래 내역(입출금)을 필터/검색할 수 있다 | P0 |
| WLT-03 | 학생 | 총 자산가치 기준 전체 랭킹을 볼 수 있다 | P1 |
| WLT-04 | Admin | 특정 학생 또는 전체에게 자금을 지급할 수 있다 (유동성 공급) | P0 |
| WLT-05 | 학생 | 자산 구성 비율(현금/주식/부채)을 시각적으로 확인할 수 있다 | P1 |

---

## 2. 도메인 모델

### Entity

```go
type Wallet struct {
    ID      int
    UserID  int
    Balance int  // 원 단위 (정수), 사용 가능한 잔고
}

type Transaction struct {
    ID            int
    WalletID      int
    Amount        int       // 양수: 입금, 음수: 출금
    BalanceAfter  int       // 거래 후 잔고
    TxType        TxType
    Description   string
    ReferenceType string    // 'company', 'job', 'loan', 'investment', ...
    ReferenceID   int
    CreatedAt     time.Time
}

type TxType string
const (
    TxInitialCapital   TxType = "initial_capital"
    TxCompanyFounding  TxType = "company_founding"
    TxFreelanceEscrow  TxType = "freelance_escrow"
    TxFreelancePayment TxType = "freelance_payment"
    TxInvestment       TxType = "investment"
    TxDividend         TxType = "dividend"
    TxLoanDisbursement TxType = "loan_disbursement"
    TxLoanRepayment    TxType = "loan_repayment"
    TxLoanInterest     TxType = "loan_interest"
    TxAdminTransfer    TxType = "admin_transfer"
    TxStockBuy         TxType = "stock_buy"
    TxStockSell        TxType = "stock_sell"
    TxAssignmentReward TxType = "assignment_reward"
    TxEscrowReturn     TxType = "escrow_return"
)
```

### Value Object: AssetValue

```go
type AssetValue struct {
    Cash          int  // 현금 잔고
    StockValue    int  // Σ(보유 주식 × 주당 가격)
    CompanyEquity int  // Σ(회사 지갑 잔고 × 내 지분율)
    TotalDebt     int  // Σ(미상환 원금 + 미납 이자)
    Total         int  // Cash + StockValue + CompanyEquity - TotalDebt
}
```

### 자산가치 계산 서비스

```go
func CalculateTotalAssetValue(userID int) AssetValue {
    cash := wallet.balance

    // 보유 주식 가치 (내 회사 + 타사 투자분)
    stockValue := 0
    for _, sh := range shareholders.FindByUser(userID) {
        company := companies.Find(sh.CompanyID)
        pricePerShare := company.Valuation / company.TotalShares
        stockValue += sh.Shares * pricePerShare
    }

    // 회사 지갑 잔고 중 내 지분
    companyEquity := 0
    for _, sh := range shareholders.FindByUser(userID) {
        cw := companyWallets.FindByCompany(sh.CompanyID)
        company := companies.Find(sh.CompanyID)
        companyEquity += cw.Balance * sh.Shares / company.TotalShares
    }

    // 부채
    totalDebt := 0
    for _, loan := range loans.FindActiveByUser(userID) {
        totalDebt += loan.Remaining + calculateAccruedInterest(loan)
    }

    return AssetValue{
        Cash:          cash,
        StockValue:    stockValue,
        CompanyEquity: companyEquity,
        TotalDebt:     totalDebt,
        Total:         cash + stockValue + companyEquity - totalDebt,
    }
}
```

### 도메인 규칙

- Admin 지갑은 transfer 시 **잔고 체크 스킵** (유동성 공급자)
- 모든 금액은 **원 단위 정수** (소수점 없음)
- 잔고 변동은 반드시 트랜잭션 로그와 함께 기록

---

## 3. 데이터베이스 스키마

```sql
CREATE TABLE wallets (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL UNIQUE REFERENCES users(id),
    balance INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE transactions (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    wallet_id      INTEGER NOT NULL REFERENCES wallets(id),
    amount         INTEGER NOT NULL,
    balance_after  INTEGER NOT NULL,
    tx_type        TEXT    NOT NULL,
    description    TEXT    DEFAULT '',
    reference_type TEXT    DEFAULT '',
    reference_id   INTEGER DEFAULT 0,
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_transactions_wallet ON transactions(wallet_id);
CREATE INDEX idx_transactions_created ON transactions(created_at);

-- Admin 지갑 시드 (잔고 체크 스킵 대상)
INSERT OR IGNORE INTO wallets (user_id, balance) VALUES (1, 0);
```

---

## 4. API 상세

### `GET /api/wallet`
**미들웨어**: Approved

```json
// Response 200
{
  "data": {
    "balance": 45000000,
    "total_asset_value": 72500000,
    "asset_breakdown": {
      "cash": 45000000,
      "stock_value": 32500000,
      "company_equity": 5000000,
      "total_debt": -10000000
    },
    "rank": 3,
    "total_students": 30
  }
}
```

---

### `GET /api/wallet/transactions`
**미들웨어**: Approved

```
Query: ?page=1&limit=20&tx_type=dividend&from=2026-03-01&to=2026-03-31
```

```json
// Response 200
{
  "data": [
    {
      "id": 45,
      "amount": 200000,
      "balance_after": 45200000,
      "tx_type": "dividend",
      "description": "우리회사 배당금",
      "reference_type": "company",
      "reference_id": 1,
      "created_at": "2026-03-12T14:30:00Z"
    }
  ],
  "pagination": { "page": 1, "limit": 20, "total": 45, "total_pages": 3 }
}
```

---

### `POST /api/wallet/transfer`
**미들웨어**: Admin

```json
// Request (특정 학생 지급)
{
  "target_user_ids": [2, 3, 5],
  "amount": 1000000,
  "description": "과제 1 보상"
}

// Request (전체 학생 지급)
{
  "target_all": true,
  "amount": 1000000,
  "description": "이벤트 보상"
}

// Response 200
{ "data": { "transferred_count": 3, "total_amount": 3000000 } }
```

- `target_all: true` → approved 전체 학생에게 지급 (`target_user_ids` 무시)
- `target_user_ids`와 `target_all` 중 하나만 사용
- Admin 지갑은 잔고 체크 스킵

---

### `GET /api/wallet/ranking`
**미들웨어**: Approved

```json
// Response 200
{
  "data": [
    {
      "rank": 1,
      "user_id": 5,
      "name": "박학생",
      "student_id_display": "26학번",
      "total_asset_value": 85000000,
      "change_24h": 5000000
    },
    {
      "rank": 2,
      "user_id": 2,
      "name": "김이화",
      "student_id_display": "24학번",
      "total_asset_value": 72500000,
      "change_24h": -1000000
    }
  ]
}
```

---

## 5. UI 스펙

### 5.1 자산 메인 (`/wallet`)

```
┌─────────────────────────────────┐
│  내 자산                         │
│                                 │
│  ┌─────────────────────────┐   │
│  │    💰 총 자산가치          │   │
│  │    7,250만원              │   │
│  │    ▲ 500만원 (오늘)       │   │
│  │                          │   │
│  │  현금      4,500만원      │   │
│  │  주식      3,250만원      │   │
│  │  회사지분     500만원     │   │
│  │  부채     -1,000만원      │   │
│  └─────────────────────────┘   │
│                                 │
│  ── 자산 구성 ──────────────     │
│  ┌─────────────────────────┐   │
│  │  [도넛차트: 현금62% /     │   │
│  │   주식45% / 지분7%        │   │
│  │   / 부채-14%]            │   │
│  └─────────────────────────┘   │
│                                 │
│  🏆 랭킹 3위 / 30명            │
│                                 │
│  ── 최근 거래 ──────────────     │
│  +200,000  배당금 (우리회사)     │
│  -500,000  외주 에스크로          │
│  +1,000,000 과제1 보상           │
│  ...                            │
│  [전체 거래 내역 보기 →]         │
└─────────────────────────────────┘
```

### 5.2 거래 내역 (`/wallet/transactions`)

```
┌─────────────────────────────────┐
│  ← 거래 내역                     │
│                                 │
│  ┌─────────────────────────┐   │
│  │ [전체] [입금] [출금]       │   │
│  │ 기간: [3월 1일] ~ [3월 31일]│   │
│  └─────────────────────────┘   │
│                                 │
│  3월 12일                        │
│  ┌─────────────────────────┐   │
│  │ + 200,000원  배당금       │   │
│  │   우리회사 · 14:30        │   │
│  ├─────────────────────────┤   │
│  │ - 5,000,000원 투자       │   │
│  │   멋진회사 투자 · 10:15   │   │
│  └─────────────────────────┘   │
│                                 │
│  3월 11일                        │
│  ...                            │
└─────────────────────────────────┘
```

### 5.3 랭킹 (탭 또는 별도 섹션)

```
┌─────────────────────────────────┐
│  🏆 자산 랭킹                    │
│                                 │
│  1  박학생  26학번  8,500만원 ▲  │
│  2  김이화  24학번  7,250만원 ▲  │
│  3  이학생  25학번  6,800만원 ▼  │
│  4  ...                         │
│  ─────────────────────          │
│  15 나 (현재)  5,000만원         │
└─────────────────────────────────┘
```

- 내 순위가 화면 밖이면 하단에 고정 표시
- 24시간 변동 표시 (▲▼)
