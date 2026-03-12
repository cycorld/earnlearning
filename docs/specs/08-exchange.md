# 08. Exchange Domain — 주식 거래소

## 1. 유저 스토리

| ID | 역할 | 스토리 | 우선순위 |
|----|------|--------|---------|
| EXC-01 | 학생 | 상장된 회사 목록과 시세를 조회할 수 있다 | P0 |
| EXC-02 | 학생 | 특정 회사의 호가창(매수/매도 호가)을 볼 수 있다 | P0 |
| EXC-03 | 학생 | 지정가 매수 주문을 낼 수 있다 | P0 |
| EXC-04 | 학생 | 지정가 매도 주문을 낼 수 있다 | P0 |
| EXC-05 | 학생 | 미체결 주문을 취소할 수 있다 | P0 |
| EXC-06 | 학생 | 내 주문 내역(체결/미체결)을 조회할 수 있다 | P0 |
| EXC-07 | 학생 | 실시간으로 시세 변동과 체결 알림을 받을 수 있다 | P1 |

---

## 2. 도메인 모델

### Entity

```go
type OrderType string
const (
    OrderBuy  OrderType = "buy"
    OrderSell OrderType = "sell"
)

type OrderStatus string
const (
    OrderOpen      OrderStatus = "open"
    OrderPartial   OrderStatus = "partial"
    OrderFilled    OrderStatus = "filled"
    OrderCancelled OrderStatus = "cancelled"
)

type StockOrder struct {
    ID              int
    CompanyID       int
    UserID          int
    OrderType       OrderType     // 'buy', 'sell'
    Shares          int           // 주문 수량
    RemainingShares int           // 미체결 수량
    PricePerShare   int           // 지정가
    Status          OrderStatus
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type StockTrade struct {
    ID            int
    CompanyID     int
    BuyOrderID    int
    SellOrderID   int
    BuyerID       int
    SellerID      int
    Shares        int
    PricePerShare int
    TotalAmount   int      // shares × price_per_share
    CreatedAt     time.Time
}
```

### 주문 매칭 엔진 (도메인 로직)

```go
// matching.go
func MatchOrder(newOrder *StockOrder) []StockTrade {
    // 지정가 주문만 지원

    // 매수 주문 → 매도 주문 중 가격 ≤ 매수가인 것을 가격 ASC로 매칭
    // 매도 주문 → 매수 주문 중 가격 ≥ 매도가인 것을 가격 DESC로 매칭

    trades := []StockTrade{}

    oppositeOrders := findMatchableOrders(newOrder)
    for _, opposite := range oppositeOrders {
        if newOrder.RemainingShares == 0 {
            break
        }

        matchShares := min(newOrder.RemainingShares, opposite.RemainingShares)
        matchPrice := opposite.PricePerShare  // 먼저 걸린 주문의 가격으로 체결

        trade := StockTrade{
            CompanyID:     newOrder.CompanyID,
            Shares:        matchShares,
            PricePerShare: matchPrice,
            TotalAmount:   matchShares * matchPrice,
        }

        // 양쪽 주문 remaining_shares 차감
        newOrder.RemainingShares -= matchShares
        opposite.RemainingShares -= matchShares

        trades = append(trades, trade)
    }

    return trades
}
```

### 도메인 규칙

- **지정가 주문만** 지원 (시장가 미지원)
- **상장 회사만** 거래 가능 (company.listed == true)
- **매수 시**: 가용 잔고 ≥ shares × price 확인 (가용 잔고 = 잔고 - 미체결 매수 주문 총액)
- **매도 시**: 가용 주식 ≥ shares 확인 (가용 주식 = 보유 주식 - 미체결 매도 주문의 shares 합산)
- **체결가**: 먼저 걸린 주문(기존 주문)의 가격으로 체결
- **기업가치 갱신**: 체결 시 `company.valuation = 체결가 × total_shares`
- **자기 매매 금지**: 자신의 주문끼리 체결 불가

---

## 3. 데이터베이스 스키마

```sql
CREATE TABLE stock_orders (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id       INTEGER NOT NULL REFERENCES companies(id),
    user_id          INTEGER NOT NULL REFERENCES users(id),
    order_type       TEXT    NOT NULL CHECK (order_type IN ('buy', 'sell')),
    shares           INTEGER NOT NULL,
    remaining_shares INTEGER NOT NULL,
    price_per_share  INTEGER NOT NULL,
    status           TEXT    NOT NULL DEFAULT 'open'
                     CHECK (status IN ('open', 'partial', 'filled', 'cancelled')),
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_orders_company ON stock_orders(company_id, status, price_per_share);

CREATE TABLE stock_trades (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id      INTEGER NOT NULL REFERENCES companies(id),
    buy_order_id    INTEGER NOT NULL REFERENCES stock_orders(id),
    sell_order_id   INTEGER NOT NULL REFERENCES stock_orders(id),
    buyer_id        INTEGER NOT NULL REFERENCES users(id),
    seller_id       INTEGER NOT NULL REFERENCES users(id),
    shares          INTEGER NOT NULL,
    price_per_share INTEGER NOT NULL,
    total_amount    INTEGER NOT NULL,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_trades_company ON stock_trades(company_id, created_at DESC);
```

---

## 4. API 상세

### `GET /api/exchange/companies`
**미들웨어**: Approved

```json
// Response 200
{
  "data": [
    {
      "company_id": 1,
      "name": "우리회사",
      "logo_url": "/uploads/logo1.png",
      "last_price": 5000,
      "change_percent": 2.5,
      "volume_24h": 3500,
      "market_cap": 62500000,
      "total_shares": 12500
    }
  ]
}
```

---

### `GET /api/exchange/companies/:id/orderbook`
**미들웨어**: Approved

```json
// Response 200
{
  "data": {
    "company_id": 1,
    "company_name": "우리회사",
    "last_price": 5000,
    "change_percent": 2.5,
    "volume_24h": 3500,
    "market_cap": 62500000,
    "asks": [
      { "price": 5100, "shares": 200 },
      { "price": 5200, "shares": 500 }
    ],
    "bids": [
      { "price": 4900, "shares": 300 },
      { "price": 4800, "shares": 150 }
    ]
  }
}
```

- asks: 매도 호가 (가격 ASC)
- bids: 매수 호가 (가격 DESC)

---

### `POST /api/exchange/orders`
**미들웨어**: Approved

```json
// Request (매수)
{
  "company_id": 1,
  "order_type": "buy",
  "shares": 500,
  "price_per_share": 5000
}

// Response 201
{
  "data": {
    "order_id": 1,
    "status": "open",
    "remaining_shares": 300,
    "matched_trades": [
      {
        "trade_id": 1,
        "shares": 200,
        "price_per_share": 4900,
        "total_amount": 980000,
        "counterparty": "박학생"
      }
    ]
  }
}
```

**비즈니스 로직** (트랜잭션):
1. company.listed == true 확인
2. **매수**: 가용 잔고 확인 (잔고 - 미체결 매수 총액 ≥ shares × price_per_share)
3. **매도**: 가용 주식 확인 (shareholders.shares - 미체결 매도 주문 합 ≥ 주문 수량)
4. stock_orders 생성
5. 매칭 엔진 실행:
   ```
   WHILE 반대 주문 존재 AND 가격 조건 충족:
     → 체결 수량 = MIN(잔여 매수, 잔여 매도)
     → stock_trades 생성
     → 매수자 지갑 차감 (tx_type: stock_buy)
     → 매도자 지갑 입금 (tx_type: stock_sell)
     → shareholders 업데이트 (매수자 +shares, 매도자 -shares)
     → 양쪽 주문 remaining_shares 업데이트
     → 주문 상태 업데이트 (filled / partial)
     → company.valuation = 체결가 × total_shares
   ```
6. WebSocket: `stock_trade`, `orderbook_update`, `wallet_update`

---

### `DELETE /api/exchange/orders/:id`
**미들웨어**: Approved (주문자만)

```json
// Response 200
{ "data": { "order_id": 1, "status": "cancelled" } }
```

- status가 'open' 또는 'partial'인 주문만 취소 가능
- 부분 체결된 경우 미체결분만 취소

---

### `GET /api/exchange/my-orders`
**미들웨어**: Approved

```
Query: ?status=open&company_id=1&page=1&limit=20
```

```json
// Response 200
{
  "data": [
    {
      "id": 1,
      "company": { "id": 1, "name": "우리회사" },
      "order_type": "buy",
      "shares": 500,
      "remaining_shares": 300,
      "price_per_share": 5000,
      "status": "partial",
      "created_at": "2026-03-12T10:00:00Z"
    }
  ]
}
```

---

## 5. UI 스펙

### 5.1 상장 회사 목록 (`/exchange`)

```
┌─────────────────────────────────┐
│  거래소                          │
│                                 │
│  ┌─────────────────────────┐   │
│  │ 🏢 우리회사               │   │
│  │ 5,000원  ▲ +2.5%        │   │
│  │ 시총: 6,250만원           │   │
│  │ 거래량: 3,500주           │   │
│  └─────────────────────────┘   │
│  ┌─────────────────────────┐   │
│  │ 🏢 멋진프로젝트           │   │
│  │ 8,000원  ▼ -1.2%        │   │
│  │ 시총: 1억원               │   │
│  │ 거래량: 1,200주           │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
```

### 5.2 호가창 & 주문 (`/exchange/[companyId]`)

```
┌─────────────────────────────────┐
│  ← 우리회사                      │
│  5,000원  ▲ +2.5%  거래량 3,500 │
│                                 │
│  ── 호가창 ─────────────────     │
│  매도               매수         │
│  5,200  500주 ████              │
│  5,100  200주 ██                │
│  ─────── 5,000원 (현재가) ────── │
│                ██  300주  4,900 │
│                █   150주  4,800 │
│                                 │
│  ── 주문 ──────────────────     │
│  ┌──────┬──────┐               │
│  │  매수  │ 매도  │               │
│  └──────┴──────┘               │
│                                 │
│  가격: [5,000    ] 원           │
│  수량: [100      ] 주           │
│  총액: 500,000원                │
│                                 │
│  내 잔고: 4,500만원              │
│  (또는) 보유주식: 2,500주        │
│                                 │
│  ┌─────────────────────────┐   │
│  │       매수 주문           │   │
│  └─────────────────────────┘   │
│                                 │
│  ── 내 미체결 주문 ─────────     │
│  매수 300주 × 5,000원 [취소]    │
│                                 │
│  ── 최근 체결 ──────────────     │
│  200주 × 4,900원  12:30         │
│  100주 × 5,000원  12:15         │
└─────────────────────────────────┘
```

---

## 6. 주식 거래 체결 흐름도

```
주문자: POST /exchange/orders
  → BEGIN TRANSACTION
  → company.listed == true 확인
  → 매수: 가용잔고(잔고 - 미체결매수총액) ≥ shares × price 확인
     매도: 가용주식(보유주식 - 미체결매도합) ≥ shares 확인
  → stock_orders 생성
  → 매칭 엔진 실행:
      WHILE 반대 주문 존재 AND 가격 조건 충족:
        → 체결 수량 = MIN(잔여 매수, 잔여 매도)
        → stock_trades 생성
        → 매수자 지갑 차감, 매도자 지갑 입금
        → shareholders 업데이트
        → 양쪽 주문 remaining_shares 업데이트
        → company.valuation = 체결가 × total_shares
  → COMMIT
  → WebSocket: stock_trade, orderbook_update, wallet_update
```
