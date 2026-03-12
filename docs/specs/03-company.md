# 03. Company Domain — 회사 설립 & 명함

## 1. 유저 스토리

| ID | 역할 | 스토리 | 우선순위 |
|----|------|--------|---------|
| CMP-01 | 학생 | 회사명, 설명, 자본금을 입력하여 회사를 설립할 수 있다 | P0 |
| CMP-02 | 학생 | 여러 개의 회사를 설립할 수 있다 (1회사 = 1프로젝트) | P0 |
| CMP-03 | 학생 | 내 회사 목록을 조회할 수 있다 | P0 |
| CMP-04 | 학생 | 회사 상세 정보 (지분 구조, 기업가치, 지갑 잔고)를 볼 수 있다 | P0 |
| CMP-05 | 설립자 | 회사 정보(설명, 로고)를 수정할 수 있다 | P1 |
| CMP-06 | 설립자 | 회사 명함을 템플릿으로 생성하고 PDF 다운로드할 수 있다 | P1 |
| CMP-07 | 학생 | 회사의 지분 구조를 파이차트로 확인할 수 있다 | P0 |

---

## 2. 도메인 모델

### Entity

```go
type Company struct {
    ID             int
    OwnerID        int       // 설립자
    Name           string    // unique
    Description    string
    LogoURL        string
    InitialCapital int       // 설립 시 납입 자본금 (≥ 1,000,000)
    TotalCapital   int       // 총 자본금 (initial_capital + 투자금 누적)
    TotalShares    int       // 총 발행 주식 수 (초기 10,000)
    Valuation      int       // 기업가치
    Listed         bool      // 거래소 상장 여부 (TotalCapital ≥ 5,000만원)
    BusinessCard   string    // JSON (명함 데이터)
    Status         string    // 'active', 'dissolved'
    CreatedAt      time.Time
}

type Shareholder struct {
    ID              int
    CompanyID       int
    UserID          int
    Shares          int
    AcquisitionType string  // 'founding', 'investment', 'trade'
    AcquiredAt      time.Time
}

// 도메인 규칙: 상장 조건
func (c *Company) CheckListing() {
    if c.TotalCapital >= 50_000_000 && !c.Listed {
        c.Listed = true   // 한번 상장되면 영구 유지
    }
}

// 도메인 규칙: 기업가치 계산
// - 설립 시: initial_capital
// - 투자 라운드 성공 시: post_money_valuation
// - 주식 거래 체결 시: 체결가 × total_shares
```

### Value Object: BusinessCard

```go
type BusinessCard struct {
    Template    string // 'modern', 'classic', 'minimal', 'bold'
    CompanyName string
    PersonName  string
    Title       string // 직함
    Email       string
    Phone       string
    LogoURL     string
    Color       string // 테마 색상
}
```

---

## 3. 데이터베이스 스키마

```sql
CREATE TABLE companies (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id         INTEGER NOT NULL REFERENCES users(id),
    name             TEXT    NOT NULL UNIQUE,
    description      TEXT    DEFAULT '',
    logo_url         TEXT    DEFAULT '',
    initial_capital  INTEGER NOT NULL CHECK (initial_capital >= 1000000),
    total_capital    INTEGER NOT NULL DEFAULT 0,       -- 총 자본금 (설립 + 투자 누적)
    total_shares     INTEGER NOT NULL DEFAULT 10000,
    valuation        INTEGER NOT NULL DEFAULT 0,
    listed           INTEGER NOT NULL DEFAULT 0,       -- 영구: 한번 1이 되면 유지
    business_card    TEXT    DEFAULT '{}',
    status           TEXT    NOT NULL DEFAULT 'active'
                     CHECK (status IN ('active', 'dissolved')),
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE company_wallets (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id INTEGER NOT NULL UNIQUE REFERENCES companies(id),
    balance    INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE company_transactions (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    company_wallet_id INTEGER NOT NULL REFERENCES company_wallets(id),
    amount            INTEGER NOT NULL,
    balance_after     INTEGER NOT NULL,
    tx_type           TEXT    NOT NULL,
    description       TEXT    DEFAULT '',
    reference_type    TEXT    DEFAULT '',
    reference_id      INTEGER DEFAULT 0,
    created_at        DATETIME DEFAULT CURRENT_TIMESTAMP
);
-- tx_type: 'founding', 'investment', 'kpi_revenue', 'dividend_out'

CREATE TABLE shareholders (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id       INTEGER NOT NULL REFERENCES companies(id),
    user_id          INTEGER NOT NULL REFERENCES users(id),
    shares           INTEGER NOT NULL DEFAULT 0,
    acquisition_type TEXT    NOT NULL
                     CHECK (acquisition_type IN ('founding', 'investment', 'trade')),
    acquired_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(company_id, user_id)
);
```

---

## 4. API 상세

### `POST /api/companies`
**미들웨어**: Approved

```json
// Request
{
  "name": "우리회사",
  "description": "바이브코딩 프로젝트",
  "initial_capital": 5000000
}

// Response 201
{
  "data": {
    "id": 1,
    "name": "우리회사",
    "initial_capital": 5000000,
    "total_capital": 5000000,
    "total_shares": 10000,
    "valuation": 5000000,
    "listed": false
  }
}
```

**비즈니스 로직** (트랜잭션):
1. 개인 지갑 잔고 확인 (잔고 ≥ initial_capital, initial_capital ≥ 1,000,000)
2. 개인 지갑에서 차감 (tx_type: `company_founding`)
3. Company 생성 (valuation = initial_capital, total_capital = initial_capital)
4. CompanyWallet 생성 (balance = initial_capital)
5. Shareholder 생성 (user_id = 설립자, shares = 10,000, type = `founding`)
6. total_capital ≥ 5,000만원이면 listed = true (영구 상장)

---

### `GET /api/companies/:id`
**미들웨어**: Approved

```json
// Response 200
{
  "data": {
    "id": 1,
    "owner": { "id": 2, "name": "김이화", "student_id_display": "24학번" },
    "name": "우리회사",
    "description": "바이브코딩 프로젝트",
    "logo_url": "/uploads/logo1.png",
    "initial_capital": 5000000,
    "total_capital": 15000000,
    "total_shares": 12500,
    "valuation": 50000000,
    "listed": true,
    "wallet_balance": 8500000,
    "shareholders": [
      { "user_id": 2, "name": "김이화", "shares": 10000, "percentage": 80.0 },
      { "user_id": 5, "name": "박투자", "shares": 2500, "percentage": 20.0 }
    ],
    "created_at": "2026-03-10T12:00:00Z"
  }
}
```

---

### `PUT /api/companies/:id`
**미들웨어**: Approved (설립자만)

```json
// Request
{
  "description": "바이브코딩으로 만든 커머스 플랫폼",
  "logo_url": "/uploads/new-logo.png"
}

// Response 200
{ "data": { "id": 1, "description": "...", "logo_url": "..." } }
```

---

### `GET /api/users/me/companies`
**미들웨어**: Approved

```json
// Response 200
{
  "data": [
    {
      "id": 1,
      "name": "우리회사",
      "valuation": 50000000,
      "listed": true,
      "my_shares": 10000,
      "my_percentage": 80.0,
      "wallet_balance": 8500000
    }
  ]
}
```

---

### `POST /api/companies/:id/business-card`
**미들웨어**: Approved (설립자만)

```json
// Request
{
  "template": "modern",
  "person_name": "김이화",
  "title": "CEO & Founder",
  "email": "ceo@ourcompany.com",
  "phone": "010-1234-5678",
  "color": "#4F46E5"
}

// Response 200
{
  "data": {
    "card_data": { ... },
    "pdf_url": "/api/companies/1/business-card/download"
  }
}
```

### `GET /api/companies/:id/business-card/download`
**미들웨어**: Approved

- Content-Type: `application/pdf`
- 명함 PDF 파일 다운로드

**명함 템플릿 종류**:
- `modern`: 깔끔한 모던 스타일
- `classic`: 전통적 명함 스타일
- `minimal`: 미니멀리즘
- `bold`: 굵은 타이포그래피

---

## 5. UI 스펙

### 5.1 내 회사 목록 (`/company`)

```
┌─────────────────────────────────┐
│  내 회사                   [+ 설립] │
│                                 │
│  ┌─────────────────────────┐   │
│  │ 🏢 우리회사              │   │
│  │ 기업가치: 5,000만원       │   │
│  │ 내 지분: 80% (10,000주)  │   │
│  │ 회사 잔고: 850만원        │   │
│  │ ● 상장                   │   │
│  └─────────────────────────┘   │
│  ┌─────────────────────────┐   │
│  │ 🏢 사이드프로젝트         │   │
│  │ 기업가치: 300만원         │   │
│  │ 내 지분: 100% (10,000주) │   │
│  │ 회사 잔고: 300만원        │   │
│  │ ○ 비상장                 │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
```

### 5.2 회사 설립 (`/company/new`)

```
┌─────────────────────────────────┐
│  ← 회사 설립                     │
│                                 │
│  ┌─────────────────────────┐   │
│  │ 회사명                    │   │
│  └─────────────────────────┘   │
│  ┌─────────────────────────┐   │
│  │ 회사 설명                 │   │
│  │                          │   │
│  └─────────────────────────┘   │
│  ┌─────────────────────────┐   │
│  │ 자본금 (최소 100만원)     │   │
│  └─────────────────────────┘   │
│  ┌─────────────────────────┐   │
│  │ 로고 업로드 (선택)        │   │
│  └─────────────────────────┘   │
│                                 │
│  ℹ️ 자본금은 개인 지갑에서 차감됩니다 │
│  ℹ️ 5,000만원 이상 시 즉시 상장    │
│                                 │
│  내 잔고: 4,500만원              │
│                                 │
│  ┌─────────────────────────┐   │
│  │        설립하기           │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
```

### 5.3 회사 상세 (`/company/[id]`)

```
┌─────────────────────────────────┐
│  ← 우리회사              [편집]  │
│                                 │
│  🏢 로고                        │
│  기업가치: 5,000만원  ● 상장     │
│  회사 잔고: 850만원              │
│                                 │
│  ── 지분 구조 ──────────────     │
│  ┌─────────────────────────┐   │
│  │     [파이차트 시각화]      │   │
│  │   김이화 80% | 박투자 20% │   │
│  └─────────────────────────┘   │
│                                 │
│  ── 주주 목록 ──────────────     │
│  김이화 (설립자)  10,000주 80%  │
│  박투자 (투자)     2,500주 20%  │
│                                 │
│  ── 액션 ───────────────────    │
│  [투자 라운드 생성]  (설립자만)   │
│  [배당 실행]        (설립자만)   │
│  [명함 만들기]      (설립자만)   │
└─────────────────────────────────┘
```

### 5.4 명함 생성 (`/company/[id]/card`)

```
┌─────────────────────────────────┐
│  ← 명함 만들기                   │
│                                 │
│  ── 템플릿 선택 ────────────     │
│  [Modern] [Classic] [Minimal] [Bold] │
│                                 │
│  ── 미리보기 ───────────────     │
│  ┌─────────────────────────┐   │
│  │  ┌───────────────────┐  │   │
│  │  │   우리회사          │  │   │
│  │  │   김이화            │  │   │
│  │  │   CEO & Founder    │  │   │
│  │  │   ceo@example.com  │  │   │
│  │  │   010-1234-5678    │  │   │
│  │  └───────────────────┘  │   │
│  └─────────────────────────┘   │
│                                 │
│  ── 정보 입력 ──────────────     │
│  이름: [김이화              ]   │
│  직함: [CEO & Founder       ]   │
│  이메일: [ceo@example.com   ]   │
│  전화: [010-1234-5678       ]   │
│  테마색: [■ #4F46E5        ]   │
│                                 │
│  [저장]  [PDF 다운로드]          │
└─────────────────────────────────┘
```

---

## 6. 기업가치 산정 규칙

| 시점 | 기업가치 계산 |
|------|-------------|
| 설립 시 | `initial_capital` |
| 투자 라운드 성공 시 | `target_amount / offered_percent` (Post-money) |
| 주식 거래 체결 시 | `체결가 × total_shares` |
| 거래 이력 없을 때 | 마지막으로 설정된 valuation 유지 |

### 상장 조건

- **기준**: `total_capital` (설립 자본금 + 투자금 누적) ≥ 5,000만원
- **영구 상장**: 한번 상장되면 자본금이 줄어도 상장 유지
- **상장 트리거 시점**: 회사 설립 시, 투자 라운드 성공 시
