# 10. Notification Domain — 알림 & WebSocket

## 1. 유저 스토리

| ID | 역할 | 스토리 | 우선순위 |
|----|------|--------|---------|
| NTF-01 | 학생 | 실시간으로 알림을 받을 수 있다 (투자, 배당, 외주, 이자 등) | P0 |
| NTF-02 | 학생 | 알림 목록을 조회할 수 있다 (읽음/안읽음) | P0 |
| NTF-03 | 학생 | 알림을 읽음 처리할 수 있다 | P0 |
| NTF-04 | 학생 | 실시간으로 주식 시세 변동을 볼 수 있다 | P1 |
| NTF-05 | 학생 | 실시간으로 지갑 잔고 변동을 볼 수 있다 | P1 |

---

## 2. 도메인 모델

### Entity

```go
type Notification struct {
    ID            int
    UserID        int
    NotifType     string     // 알림 유형
    Title         string
    Body          string
    ReferenceType string     // 'post', 'job', 'investment', 'loan', 'trade', ...
    ReferenceID   int
    IsRead        bool
    CreatedAt     time.Time
}
```

### 알림 유형

| NotifType | 설명 | 트리거 |
|-----------|------|--------|
| `user_approved` | 회원 승인 완료 | Admin 승인 시 |
| `new_assignment` | 새 과제 등록 | Admin 과제 생성 시 |
| `assignment_graded` | 과제 채점 완료 | Admin 채점 시 |
| `job_application` | 외주 지원 접수 | 지원서 제출 시 |
| `job_accepted` | 외주 수주 확정 | 의뢰자 수락 시 |
| `job_work_done` | 작업 완료 알림 | 수주자 완료 표시 시 (의뢰자에게) |
| `job_completed` | 외주 정산 완료 | 의뢰자 승인 시 |
| `job_review` | 리뷰 요청 | 정산 완료 시 |
| `investment_funded` | 투자 라운드 성공 | 펀딩 확정 시 |
| `investment_received` | 투자 유치 완료 | 펀딩 확정 시 (설립자) |
| `dividend_received` | 배당금 수령 | 배당 실행 시 |
| `kpi_revenue` | KPI 소득 부여 | Admin 소득 부여 시 |
| `loan_approved` | 대출 승인 | Admin 승인 시 |
| `loan_rejected` | 대출 거절 | Admin 거절 시 |
| `loan_interest` | 이자 차감 | 주간 배치 시 |
| `loan_overdue` | 연체 경고 | 이자 미납 시 |
| `stock_trade` | 주식 체결 | 주문 체결 시 |
| `admin_transfer` | 유동성 지급 | Admin 이체 시 |

---

## 3. 데이터베이스 스키마

```sql
CREATE TABLE notifications (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id        INTEGER NOT NULL REFERENCES users(id),
    notif_type     TEXT    NOT NULL,
    title          TEXT    NOT NULL,
    body           TEXT    DEFAULT '',
    reference_type TEXT    DEFAULT '',
    reference_id   INTEGER DEFAULT 0,
    is_read        INTEGER NOT NULL DEFAULT 0,
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_notifications_user ON notifications(user_id, is_read, created_at DESC);
```

---

## 4. API 상세

### `GET /api/notifications`
**미들웨어**: Approved

```
Query: ?is_read=false&page=1&limit=20
```

```json
// Response 200
{
  "data": [
    {
      "id": 1,
      "notif_type": "dividend_received",
      "title": "배당금 수령",
      "body": "우리회사에서 200,000원의 배당금을 수령했습니다.",
      "reference_type": "company",
      "reference_id": 1,
      "is_read": false,
      "created_at": "2026-03-12T14:30:00Z"
    }
  ],
  "pagination": { ... },
  "unread_count": 5
}
```

---

### `PUT /api/notifications/:id/read`
**미들웨어**: Approved

```json
// Response 200
{ "data": { "id": 1, "is_read": true } }
```

---

### `PUT /api/notifications/read-all`
**미들웨어**: Approved

```json
// Response 200
{ "data": { "updated_count": 5 } }
```

---

## 5. WebSocket 스펙

### 연결

```
ws://host/ws?token=<JWT>
```

- JWT 검증 후 연결 수립
- 인증 실패 시 연결 거부

### Hub 구조

```go
type Hub struct {
    clients    map[int]*Client     // user_id → Client
    broadcast  chan Message         // 전체 브로드캐스트
    personal   chan PersonalMessage // 개인 메시지
    register   chan *Client
    unregister chan *Client
}

type Message struct {
    Event string      `json:"event"`
    Data  interface{} `json:"data"`
}

type PersonalMessage struct {
    UserID  int
    Message Message
}
```

### 서버 → 클라이언트 이벤트

```json
// 지갑 잔고 변동 (개인)
{ "event": "wallet_update", "data": { "balance": 45000000, "total_asset_value": 72500000 } }

// 알림 수신 (개인)
{ "event": "notification", "data": { "id": 1, "type": "dividend_received", "title": "배당금 수령", "body": "..." } }

// 주식 시세 업데이트 (전체)
{ "event": "stock_price_update", "data": { "company_id": 1, "price": 5000, "change_percent": 2.5 } }

// 주식 체결 (전체)
{ "event": "stock_trade", "data": { "company_id": 1, "price": 5000, "shares": 100, "timestamp": "..." } }

// 호가창 업데이트 (전체)
{ "event": "orderbook_update", "data": { "company_id": 1, "asks": [...], "bids": [...] } }

// 새 게시글 (전체)
{ "event": "new_post", "data": { "channel_id": 2, "post_id": 15, "author": "박학생" } }

// 사용자 승인 (개인)
{ "event": "user_approved", "data": { "user_id": 3 } }
```

### 메시지 라우팅

| 이벤트 | 대상 | 트리거 |
|--------|------|--------|
| `wallet_update` | 해당 사용자 | 지갑 잔고 변동 시 |
| `notification` | 해당 사용자 | 알림 생성 시 |
| `stock_price_update` | 전체 | 주식 체결 시 |
| `stock_trade` | 전체 | 주식 체결 시 |
| `orderbook_update` | 전체 | 주문/체결/취소 시 |
| `new_post` | 전체 | 게시글 작성 시 |
| `user_approved` | 해당 사용자 | Admin 승인 시 |

---

## 6. UI 스펙

### 6.1 알림 목록 (`/notifications`)

```
┌─────────────────────────────────┐
│  알림                    [전체 읽음] │
│                                 │
│  오늘                            │
│  ┌─────────────────────────┐   │
│  │ 🔵 배당금 수령            │   │
│  │ 우리회사에서 200,000원의   │   │
│  │ 배당금을 수령했습니다.     │   │
│  │ 2시간 전                  │   │
│  └─────────────────────────┘   │
│  ┌─────────────────────────┐   │
│  │ 🔵 주식 체결              │   │
│  │ 우리회사 200주 × 4,900원  │   │
│  │ 체결되었습니다.            │   │
│  │ 3시간 전                  │   │
│  └─────────────────────────┘   │
│                                 │
│  어제                            │
│  ┌─────────────────────────┐   │
│  │ ○ 과제 채점 완료          │   │
│  │ 과제1: 90점 (45만원 지급) │   │
│  │ 어제 14:00                │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
```

- 🔵 = 안읽음, ○ = 읽음
- 탭하면 해당 리소스로 이동 (reference_type/reference_id 기반)
- 헤더에 안읽음 카운트 배지 표시
