# EarnLearning Technical Specification

## 1. 프로젝트 구조

```
earnlearning/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go                 # 엔트리포인트
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go              # 환경변수, 설정
│   │   ├── database/
│   │   │   ├── sqlite.go              # SQLite 연결 (WAL 모드)
│   │   │   ├── seed.go                # Admin 시드 데이터
│   │   │   └── migrations/
│   │   │       └── 001_init.sql       # 초기 스키마
│   │   ├── middleware/
│   │   │   ├── auth.go                # JWT 인증
│   │   │   ├── cors.go                # CORS
│   │   │   └── approved.go            # 승인된 사용자만 통과
│   │   ├── models/
│   │   │   ├── user.go
│   │   │   ├── company.go
│   │   │   ├── classroom.go
│   │   │   ├── wallet.go
│   │   │   ├── post.go
│   │   │   ├── assignment.go
│   │   │   ├── freelance.go
│   │   │   ├── investment.go
│   │   │   ├── exchange.go
│   │   │   ├── loan.go
│   │   │   └── notification.go
│   │   ├── handlers/
│   │   │   ├── auth.go
│   │   │   ├── admin.go
│   │   │   ├── company.go
│   │   │   ├── classroom.go
│   │   │   ├── wallet.go
│   │   │   ├── post.go
│   │   │   ├── assignment.go
│   │   │   ├── freelance.go
│   │   │   ├── investment.go
│   │   │   ├── exchange.go
│   │   │   ├── loan.go
│   │   │   ├── notification.go
│   │   │   └── upload.go
│   │   ├── services/
│   │   │   ├── auth_service.go
│   │   │   ├── wallet_service.go      # 잔고 변동, 에스크로
│   │   │   ├── company_service.go     # 설립, 기업가치
│   │   │   ├── investment_service.go  # 신주 발행, 배당
│   │   │   ├── exchange_service.go    # 주문 매칭 엔진
│   │   │   ├── loan_service.go        # 이자 계산, 연체
│   │   │   └── valuation_service.go   # 자산가치 계산
│   │   ├── ws/
│   │   │   ├── hub.go                 # WebSocket 허브
│   │   │   └── client.go             # 클라이언트 연결
│   │   └── router/
│   │       └── router.go             # 라우터 설정
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
│
├── frontend/
│   ├── src/
│   │   ├── app/
│   │   │   ├── layout.tsx
│   │   │   ├── page.tsx               # 랜딩/로그인
│   │   │   ├── (auth)/
│   │   │   │   ├── login/page.tsx
│   │   │   │   ├── register/page.tsx
│   │   │   │   └── pending/page.tsx   # 승인 대기
│   │   │   └── (main)/               # 승인된 사용자 레이아웃
│   │   │       ├── layout.tsx         # 하단 네비게이션
│   │   │       ├── feed/page.tsx      # 홈 피드
│   │   │       ├── wallet/page.tsx
│   │   │       ├── market/page.tsx    # 외주 마켓
│   │   │       ├── company/
│   │   │       │   ├── page.tsx       # 내 회사 목록
│   │   │       │   ├── new/page.tsx   # 회사 설립
│   │   │       │   └── [id]/page.tsx  # 회사 상세
│   │   │       ├── invest/page.tsx
│   │   │       ├── exchange/page.tsx
│   │   │       ├── bank/page.tsx
│   │   │       ├── profile/page.tsx
│   │   │       ├── notifications/page.tsx
│   │   │       └── admin/            # Admin Only
│   │   │           ├── page.tsx
│   │   │           ├── users/page.tsx
│   │   │           └── ...
│   │   ├── components/
│   │   │   ├── ui/                    # shadcn/ui
│   │   │   ├── layout/
│   │   │   │   ├── bottom-nav.tsx
│   │   │   │   └── header.tsx
│   │   │   ├── feed/
│   │   │   ├── wallet/
│   │   │   ├── company/
│   │   │   ├── market/
│   │   │   ├── invest/
│   │   │   ├── exchange/
│   │   │   └── bank/
│   │   ├── lib/
│   │   │   ├── api.ts                 # API 클라이언트
│   │   │   ├── auth.ts                # JWT 관리
│   │   │   ├── ws.ts                  # WebSocket 클라이언트
│   │   │   └── utils.ts
│   │   ├── hooks/
│   │   │   ├── use-auth.ts
│   │   │   ├── use-wallet.ts
│   │   │   └── use-ws.ts
│   │   └── types/
│   │       └── index.ts               # 공유 타입
│   ├── public/
│   ├── next.config.js
│   ├── tailwind.config.ts
│   ├── tsconfig.json
│   ├── package.json
│   └── Dockerfile
│
├── docker-compose.yml
├── nginx.conf
├── PRD.md
├── CLAUDE.md
└── docs/
    ├── SPEC.md
    └── prompts/
```

---

## 2. 데이터베이스 스키마 (SQLite DDL)

```sql
-- ============================================================
-- SQLite 설정
-- ============================================================
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;

-- ============================================================
-- 사용자
-- ============================================================
CREATE TABLE users (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    email       TEXT    NOT NULL UNIQUE,
    password    TEXT    NOT NULL,          -- bcrypt hash
    name        TEXT    NOT NULL,
    department  TEXT    NOT NULL,          -- 학과
    student_id  TEXT    NOT NULL,          -- 전체 학번 (예: "2024123456")
    role        TEXT    NOT NULL DEFAULT 'student' CHECK (role IN ('admin', 'student')),
    status      TEXT    NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    bio         TEXT    DEFAULT '',
    avatar_url  TEXT    DEFAULT '',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 강의실
-- ============================================================
CREATE TABLE classrooms (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT    NOT NULL,
    code            TEXT    NOT NULL UNIQUE,    -- 참여 코드
    created_by      INTEGER NOT NULL REFERENCES users(id),
    initial_capital INTEGER NOT NULL DEFAULT 50000000,  -- 5,000만원
    settings        TEXT    DEFAULT '{}',       -- JSON
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE classroom_members (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    classroom_id INTEGER NOT NULL REFERENCES classrooms(id),
    user_id      INTEGER NOT NULL REFERENCES users(id),
    joined_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(classroom_id, user_id)
);

-- ============================================================
-- 개인 지갑
-- ============================================================
CREATE TABLE wallets (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL UNIQUE REFERENCES users(id),
    balance INTEGER NOT NULL DEFAULT 0   -- 원 단위 (정수)
);

CREATE TABLE transactions (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    wallet_id     INTEGER NOT NULL REFERENCES wallets(id),
    amount        INTEGER NOT NULL,        -- 양수: 입금, 음수: 출금
    balance_after INTEGER NOT NULL,        -- 거래 후 잔고
    tx_type       TEXT    NOT NULL,        -- 아래 참조
    description   TEXT    DEFAULT '',
    reference_type TEXT   DEFAULT '',      -- 'company', 'job', 'loan', 'investment', ...
    reference_id  INTEGER DEFAULT 0,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);
-- tx_type: 'initial_capital', 'company_founding', 'freelance_escrow',
--          'freelance_payment', 'investment', 'dividend', 'loan_disbursement',
--          'loan_repayment', 'loan_interest', 'admin_transfer', 'stock_buy',
--          'stock_sell', 'assignment_reward'

CREATE INDEX idx_transactions_wallet ON transactions(wallet_id);
CREATE INDEX idx_transactions_created ON transactions(created_at);

-- ============================================================
-- 회사
-- ============================================================
CREATE TABLE companies (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id         INTEGER NOT NULL REFERENCES users(id),
    name             TEXT    NOT NULL UNIQUE,
    description      TEXT    DEFAULT '',
    logo_url         TEXT    DEFAULT '',
    initial_capital  INTEGER NOT NULL CHECK (initial_capital >= 1000000), -- ≥ 100만원
    total_shares     INTEGER NOT NULL DEFAULT 10000,  -- 신주 발행 시 증가
    valuation        INTEGER NOT NULL DEFAULT 0,      -- 기업가치
    listed           INTEGER NOT NULL DEFAULT 0,      -- 0: 비상장, 1: 상장
    business_card    TEXT    DEFAULT '{}',             -- JSON (명함 데이터)
    status           TEXT    NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'dissolved')),
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE company_wallets (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id INTEGER NOT NULL UNIQUE REFERENCES companies(id),
    balance    INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE company_transactions (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    company_wallet_id INTEGER NOT NULL REFERENCES company_wallets(id),
    amount        INTEGER NOT NULL,
    balance_after INTEGER NOT NULL,
    tx_type       TEXT    NOT NULL,   -- 'founding', 'investment', 'kpi_revenue', 'dividend_out'
    description   TEXT    DEFAULT '',
    reference_type TEXT   DEFAULT '',
    reference_id  INTEGER DEFAULT 0,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE shareholders (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id       INTEGER NOT NULL REFERENCES companies(id),
    user_id          INTEGER NOT NULL REFERENCES users(id),
    shares           INTEGER NOT NULL DEFAULT 0,
    acquisition_type TEXT    NOT NULL CHECK (acquisition_type IN ('founding', 'investment', 'trade')),
    acquired_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(company_id, user_id)
);

-- ============================================================
-- 채널 & 게시글
-- ============================================================
CREATE TABLE channels (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    classroom_id INTEGER NOT NULL REFERENCES classrooms(id),
    name         TEXT    NOT NULL,          -- '#공지', '#자유', '#과제', ...
    slug         TEXT    NOT NULL,          -- 'notice', 'free', 'assignment', ...
    channel_type TEXT    NOT NULL,          -- 'notice', 'free', 'assignment', 'showcase', 'market', 'invest', 'exchange'
    write_role   TEXT    NOT NULL DEFAULT 'all', -- 'admin', 'all'
    sort_order   INTEGER NOT NULL DEFAULT 0,
    UNIQUE(classroom_id, slug)
);

CREATE TABLE posts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id  INTEGER NOT NULL REFERENCES channels(id),
    author_id   INTEGER NOT NULL REFERENCES users(id),
    content     TEXT    NOT NULL,
    post_type   TEXT    NOT NULL DEFAULT 'normal', -- 'normal', 'assignment', 'showcase', 'ir'
    media       TEXT    DEFAULT '[]',     -- JSON array: [{url, type, name}]
    tags        TEXT    DEFAULT '[]',     -- JSON array: ["tag1", "tag2"]
    like_count  INTEGER NOT NULL DEFAULT 0,
    comment_count INTEGER NOT NULL DEFAULT 0,
    pinned      INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_posts_channel ON posts(channel_id, created_at DESC);

CREATE TABLE post_likes (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(post_id, user_id)
);

CREATE TABLE comments (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id   INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id INTEGER NOT NULL REFERENCES users(id),
    content   TEXT    NOT NULL,
    media     TEXT    DEFAULT '[]',   -- JSON (제출물 첨부용)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_comments_post ON comments(post_id, created_at);

-- ============================================================
-- 과제
-- ============================================================
CREATE TABLE assignments (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id       INTEGER NOT NULL UNIQUE REFERENCES posts(id),
    deadline      DATETIME NOT NULL,
    reward_amount INTEGER NOT NULL DEFAULT 0, -- 보상 금액
    max_score     INTEGER NOT NULL DEFAULT 100
);

CREATE TABLE submissions (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    assignment_id INTEGER NOT NULL REFERENCES assignments(id),
    student_id    INTEGER NOT NULL REFERENCES users(id),
    comment_id    INTEGER REFERENCES comments(id),  -- 제출 댓글 연결
    content       TEXT    DEFAULT '',
    files         TEXT    DEFAULT '[]',     -- JSON
    grade         INTEGER DEFAULT NULL,     -- 0~max_score
    rewarded      INTEGER NOT NULL DEFAULT 0, -- 0: 미지급, 1: 지급완료
    submitted_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(assignment_id, student_id)
);

-- ============================================================
-- 외주 마켓
-- ============================================================
CREATE TABLE freelance_jobs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id       INTEGER NOT NULL REFERENCES users(id),  -- 의뢰자
    title           TEXT    NOT NULL,
    description     TEXT    NOT NULL,
    budget          INTEGER NOT NULL,           -- 예산 (원)
    deadline        DATETIME,
    required_skills TEXT    DEFAULT '[]',        -- JSON array
    status          TEXT    NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open', 'in_progress', 'completed', 'disputed', 'cancelled')),
    freelancer_id   INTEGER REFERENCES users(id),  -- 수주자 (계약 후 설정)
    escrow_amount   INTEGER NOT NULL DEFAULT 0,    -- 에스크로 동결 금액
    agreed_price    INTEGER DEFAULT 0,             -- 합의 금액
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at    DATETIME
);

CREATE TABLE job_applications (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id     INTEGER NOT NULL REFERENCES freelance_jobs(id),
    user_id    INTEGER NOT NULL REFERENCES users(id),  -- 지원자
    proposal   TEXT    NOT NULL,
    price      INTEGER NOT NULL,              -- 견적
    status     TEXT    NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending', 'accepted', 'rejected')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(job_id, user_id)
);

CREATE TABLE freelance_reviews (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id     INTEGER NOT NULL REFERENCES freelance_jobs(id),
    reviewer_id INTEGER NOT NULL REFERENCES users(id),
    reviewee_id INTEGER NOT NULL REFERENCES users(id),
    rating     INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment    TEXT    DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(job_id, reviewer_id)
);

-- ============================================================
-- 투자
-- ============================================================
CREATE TABLE investment_rounds (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id     INTEGER NOT NULL REFERENCES companies(id),
    post_id        INTEGER REFERENCES posts(id),  -- IR 게시글 연결
    target_amount  INTEGER NOT NULL,       -- 모집 금액
    offered_percent REAL   NOT NULL,       -- 양도 지분율 (0.0~1.0)
    current_amount INTEGER NOT NULL DEFAULT 0, -- 현재 모집 금액
    price_per_share REAL   NOT NULL,       -- 주당 가격 (= target / 신주수)
    new_shares     INTEGER NOT NULL,       -- 발행 예정 신주 수
    status         TEXT    NOT NULL DEFAULT 'open'
                   CHECK (status IN ('open', 'funded', 'failed', 'cancelled')),
    allow_partial  INTEGER NOT NULL DEFAULT 0,  -- 부분 펀딩 허용
    expires_at     DATETIME,
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
    funded_at      DATETIME
);

CREATE TABLE investments (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    round_id  INTEGER NOT NULL REFERENCES investment_rounds(id),
    user_id   INTEGER NOT NULL REFERENCES users(id),
    amount    INTEGER NOT NULL,         -- 투자 금액
    shares    INTEGER NOT NULL,         -- 배정 주식 수
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 배당
-- ============================================================
CREATE TABLE dividends (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id     INTEGER NOT NULL REFERENCES companies(id),
    total_amount   INTEGER NOT NULL,       -- 총 배당금
    executed_by    INTEGER NOT NULL REFERENCES users(id),  -- 설립자
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE dividend_payments (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    dividend_id INTEGER NOT NULL REFERENCES dividends(id),
    user_id     INTEGER NOT NULL REFERENCES users(id),
    shares      INTEGER NOT NULL,          -- 배당 시점 보유 주식
    amount      INTEGER NOT NULL,          -- 수령 금액
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- KPI
-- ============================================================
CREATE TABLE kpi_rules (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id       INTEGER NOT NULL REFERENCES companies(id),
    rule_description TEXT    NOT NULL,      -- "일일 방문자 100명당 10만원"
    active           INTEGER NOT NULL DEFAULT 1,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE kpi_revenues (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id INTEGER NOT NULL REFERENCES companies(id),
    kpi_rule_id INTEGER REFERENCES kpi_rules(id),
    amount     INTEGER NOT NULL,            -- 부여 금액
    memo       TEXT    DEFAULT '',           -- Admin 메모
    created_by INTEGER NOT NULL REFERENCES users(id),  -- Admin
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 주식 거래소
-- ============================================================
CREATE TABLE stock_orders (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id      INTEGER NOT NULL REFERENCES companies(id),
    user_id         INTEGER NOT NULL REFERENCES users(id),
    order_type      TEXT    NOT NULL CHECK (order_type IN ('buy', 'sell')),
    price_type      TEXT    NOT NULL CHECK (price_type IN ('limit', 'market')),
    shares          INTEGER NOT NULL,
    remaining_shares INTEGER NOT NULL,     -- 미체결 수량
    price_per_share INTEGER,               -- 지정가 (market이면 NULL)
    status          TEXT    NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open', 'partial', 'filled', 'cancelled')),
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_orders_company ON stock_orders(company_id, status, price_per_share);

CREATE TABLE stock_trades (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id     INTEGER NOT NULL REFERENCES companies(id),
    buy_order_id   INTEGER NOT NULL REFERENCES stock_orders(id),
    sell_order_id  INTEGER NOT NULL REFERENCES stock_orders(id),
    buyer_id       INTEGER NOT NULL REFERENCES users(id),
    seller_id      INTEGER NOT NULL REFERENCES users(id),
    shares         INTEGER NOT NULL,
    price_per_share INTEGER NOT NULL,
    total_amount   INTEGER NOT NULL,       -- shares × price_per_share
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_trades_company ON stock_trades(company_id, created_at DESC);

-- ============================================================
-- 은행 (대출)
-- ============================================================
CREATE TABLE loans (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    borrower_id    INTEGER NOT NULL REFERENCES users(id),
    amount         INTEGER NOT NULL,         -- 대출 원금
    remaining      INTEGER NOT NULL,         -- 잔여 원금
    interest_rate  REAL    NOT NULL,          -- 주당 이자율 (예: 0.05 = 5%)
    penalty_rate   REAL    NOT NULL DEFAULT 0.10, -- 연체 이자율 (기본 2배)
    purpose        TEXT    DEFAULT '',
    status         TEXT    NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending', 'approved', 'rejected', 'active', 'paid', 'overdue')),
    approved_by    INTEGER REFERENCES users(id),
    approved_at    DATETIME,
    next_payment   DATETIME,                 -- 다음 이자 납부일
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE loan_payments (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    loan_id    INTEGER NOT NULL REFERENCES loans(id),
    amount     INTEGER NOT NULL,             -- 납부 금액
    principal  INTEGER NOT NULL DEFAULT 0,   -- 원금 상환분
    interest   INTEGER NOT NULL DEFAULT 0,   -- 이자분
    penalty    INTEGER NOT NULL DEFAULT 0,   -- 연체 이자분
    pay_type   TEXT    NOT NULL CHECK (pay_type IN ('interest', 'repayment', 'penalty', 'auto')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 알림
-- ============================================================
CREATE TABLE notifications (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id        INTEGER NOT NULL REFERENCES users(id),
    notif_type     TEXT    NOT NULL,
    title          TEXT    NOT NULL,
    body           TEXT    DEFAULT '',
    reference_type TEXT    DEFAULT '',       -- 'post', 'job', 'investment', 'loan', ...
    reference_id   INTEGER DEFAULT 0,
    is_read        INTEGER NOT NULL DEFAULT 0,
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_notifications_user ON notifications(user_id, is_read, created_at DESC);

-- ============================================================
-- 파일 업로드
-- ============================================================
CREATE TABLE uploads (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    filename    TEXT    NOT NULL,           -- 원본 파일명
    stored_name TEXT   NOT NULL,            -- 저장된 파일명 (UUID)
    mime_type   TEXT    NOT NULL,
    size        INTEGER NOT NULL,           -- bytes
    path        TEXT    NOT NULL,           -- /data/uploads/...
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 3. Admin 시드 데이터

```sql
-- 서버 최초 기동 시 실행 (이미 존재하면 스킵)
INSERT OR IGNORE INTO users (email, password, name, department, student_id, role, status)
VALUES (
    'cyc@snu.ac.kr',
    '$2a$10$...', -- bcrypt('test1234')
    '최용철',
    '관리자',
    '0000000000',
    'admin',
    'approved'
);

-- Admin 지갑 생성 (잔고 무한 = 유동성 공급자)
INSERT OR IGNORE INTO wallets (user_id, balance)
VALUES (1, 0);  -- Admin은 transfer 시 잔고 체크 스킵
```

---

## 4. API 상세 스펙

### 4.1 공통

**Base URL**: `/api`

**인증**: `Authorization: Bearer <JWT>`

**JWT Payload**:
```json
{
  "user_id": 1,
  "email": "student@ewha.ac.kr",
  "role": "student",
  "status": "approved",
  "exp": 1234567890
}
```

**공통 응답 형식**:
```json
{
  "success": true,
  "data": { ... },
  "error": null
}
```
```json
{
  "success": false,
  "data": null,
  "error": { "code": "INSUFFICIENT_BALANCE", "message": "잔고가 부족합니다." }
}
```

**에러 코드**:
| 코드 | HTTP | 설명 |
|------|------|------|
| `UNAUTHORIZED` | 401 | 미인증 |
| `FORBIDDEN` | 403 | 권한 없음 |
| `NOT_APPROVED` | 403 | 승인 대기 중 |
| `NOT_FOUND` | 404 | 리소스 없음 |
| `DUPLICATE` | 409 | 중복 (회사명 등) |
| `INSUFFICIENT_BALANCE` | 400 | 잔고 부족 |
| `MIN_CAPITAL` | 400 | 최소 자본금 미달 |
| `NOT_LISTED` | 400 | 비상장 회사 |
| `ROUND_CLOSED` | 400 | 투자 라운드 마감 |
| `VALIDATION` | 422 | 입력값 오류 |

**미들웨어 체인**:
```
Public    → [CORS]
Auth      → [CORS] → [JWT 검증]
Approved  → [CORS] → [JWT 검증] → [status == 'approved' 확인]
Admin     → [CORS] → [JWT 검증] → [status == 'approved'] → [role == 'admin']
```

**페이지네이션** (목록 API 공통):
```
GET /api/posts?page=1&limit=20&channel_id=1
```
```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "total_pages": 8
  }
}
```

### 4.2 Auth

#### `POST /api/auth/register`
미들웨어: Public
```json
// Request
{
  "email": "student@ewha.ac.kr",
  "password": "mypassword123",
  "name": "김이화",
  "department": "컴퓨터공학과",
  "student_id": "2024123456"
}
// Response 201
{
  "data": {
    "id": 2,
    "email": "student@ewha.ac.kr",
    "name": "김이화",
    "status": "pending",
    "message": "관리자 승인을 기다리고 있습니다. 문의: cyc@snu.ac.kr"
  }
}
```
- 비밀번호: bcrypt 해싱, 최소 8자
- email unique 검증
- student_id 형식 검증 (숫자, 7~10자리)

#### `POST /api/auth/login`
미들웨어: Public
```json
// Request
{ "email": "student@ewha.ac.kr", "password": "mypassword123" }
// Response 200
{
  "data": {
    "token": "eyJhbG...",
    "user": {
      "id": 2,
      "email": "student@ewha.ac.kr",
      "name": "김이화",
      "role": "student",
      "status": "approved",  // 또는 "pending"
      "department": "컴퓨터공학과",
      "student_id_display": "24학번"   // 마스킹
    }
  }
}
```
- status가 "pending"이어도 로그인은 성공 (프론트에서 분기)
- status가 "rejected"이면 로그인 거부

#### `GET /api/auth/me`
미들웨어: Auth
```json
// Response 200
{
  "data": {
    "id": 2,
    "email": "student@ewha.ac.kr",
    "name": "김이화",
    "role": "student",
    "status": "approved",
    "department": "컴퓨터공학과",
    "student_id_display": "24학번",
    "bio": "",
    "avatar_url": "",
    "wallet_balance": 50000000,
    "total_asset_value": 52300000,
    "company_count": 2
  }
}
```

### 4.3 Admin

#### `GET /api/admin/users/pending`
미들웨어: Admin
```json
// Response 200
{
  "data": [
    {
      "id": 3,
      "email": "kim@ewha.ac.kr",
      "name": "김학생",
      "department": "경영학과",
      "student_id": "2026543210",   // Admin은 전체 학번
      "created_at": "2026-03-13T10:00:00Z"
    }
  ]
}
```

#### `PUT /api/admin/users/:id/approve`
미들웨어: Admin
```json
// Response 200
{ "data": { "id": 3, "status": "approved" } }
```

#### `PUT /api/admin/users/:id/reject`
미들웨어: Admin
```json
// Response 200
{ "data": { "id": 3, "status": "rejected" } }
```

#### `GET /api/admin/users`
미들웨어: Admin
```json
// Query: ?status=approved&page=1
// Response 200 - 전체 사용자 목록 (student_id 전체 노출)
```

### 4.4 Classroom

#### `POST /api/classrooms`
미들웨어: Admin
```json
// Request
{ "name": "2026 스타트업을위한코딩입문", "initial_capital": 50000000 }
// Response 201
{ "data": { "id": 1, "name": "...", "code": "ABC123" } }
```
- code는 서버에서 6자리 랜덤 생성

#### `POST /api/classrooms/join`
미들웨어: Approved
```json
// Request
{ "code": "ABC123" }
// Response 200
{ "data": { "classroom_id": 1, "initial_capital": 50000000 } }
```
- 참여 시 개인 지갑에 initial_capital 입금
- 중복 참여 방지
- 트랜잭션 로그: tx_type = 'initial_capital'

### 4.5 Company

#### `POST /api/companies`
미들웨어: Approved
```json
// Request
{
  "name": "우리회사",
  "description": "바이브코딩 프로젝트",
  "initial_capital": 5000000   // 500만원
}
// Response 201
{
  "data": {
    "id": 1,
    "name": "우리회사",
    "initial_capital": 5000000,
    "total_shares": 10000,
    "valuation": 5000000,
    "listed": false
  }
}
```
**비즈니스 로직** (트랜잭션):
1. 개인 지갑 잔고 확인 (≥ initial_capital, ≥ 1,000,000)
2. 개인 지갑에서 차감 (tx_type: 'company_founding')
3. Company 레코드 생성 (valuation = initial_capital)
4. CompanyWallet 생성 (balance = initial_capital)
5. ShareHolder 생성 (user_id = 설립자, shares = 10,000)
6. 자본금 ≥ 5,000만원이면 listed = 1

#### `GET /api/companies/:id`
미들웨어: Approved
```json
{
  "data": {
    "id": 1,
    "owner": { "id": 2, "name": "김이화", "student_id_display": "24학번" },
    "name": "우리회사",
    "description": "...",
    "logo_url": "...",
    "initial_capital": 5000000,
    "total_shares": 12500,
    "valuation": 50000000,
    "listed": true,
    "wallet_balance": 8500000,
    "shareholders": [
      { "user_id": 2, "name": "김이화", "shares": 10000, "percentage": 80.0 },
      { "user_id": 5, "name": "박투자", "shares": 2500, "percentage": 20.0 }
    ],
    "created_at": "..."
  }
}
```

#### `POST /api/companies/:id/dividend`
미들웨어: Approved (설립자만)
```json
// Request
{ "amount": 1000000 }   // 100만원 배당
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
3. 회사 지갑에서 차감
4. 각 주주에게 (amount × 보유주식 / 총주식) 개인 지갑에 입금
5. dividend, dividend_payments 레코드 생성
6. 각 주주에게 알림 발송

### 4.6 Investment

#### `POST /api/companies/:id/rounds`
미들웨어: Approved (설립자만)
```json
// Request
{
  "target_amount": 10000000,    // 1,000만원 모집
  "offered_percent": 0.20,      // 20% 지분
  "allow_partial": false,
  "expires_at": "2026-04-01T00:00:00Z"
}
// Response 201
{
  "data": {
    "id": 1,
    "company_id": 1,
    "target_amount": 10000000,
    "offered_percent": 0.20,
    "new_shares": 2500,            // 자동 계산
    "price_per_share": 4000,       // 10,000,000 / 2,500
    "pre_money_valuation": 40000000,  // 기존 주식 × 주당가
    "post_money_valuation": 50000000
  }
}
```
**신주 계산 공식**:
```
new_shares = total_shares × offered_percent / (1 - offered_percent)
price_per_share = target_amount / new_shares
post_money = target_amount / offered_percent
pre_money = post_money - target_amount
```

#### `POST /api/rounds/:id/invest`
미들웨어: Approved
```json
// Request
{ "amount": 5000000 }   // 500만원 투자
// Response 200
{
  "data": {
    "investment_id": 1,
    "shares_acquired": 1250,
    "round_current_amount": 5000000,
    "round_status": "open"
  }
}
```
**비즈니스 로직** (트랜잭션):
1. 라운드 status == 'open' 확인
2. 투자자 개인 지갑 잔고 확인
3. 개인 지갑에서 차감 (tx_type: 'investment')
4. round.current_amount += amount
5. investments 레코드 생성
6. target 달성 시 → 펀딩 확정:
   - company.total_shares += new_shares
   - 각 투자자에게 shares 배정 → shareholders 추가/업데이트
   - 회사 지갑에 총 투자금 입금
   - company.valuation = post_money
   - 자본금 5,000만원 이상이면 company.listed = 1
   - 관련자 알림

### 4.7 Stock Exchange

#### `POST /api/exchange/orders`
미들웨어: Approved
```json
// Request (매도)
{
  "company_id": 1,
  "order_type": "sell",
  "price_type": "limit",
  "shares": 500,
  "price_per_share": 5000
}
// Response 201
{
  "data": {
    "order_id": 1,
    "status": "open",
    "matched_trades": []   // 즉시 체결된 거래 (있으면)
  }
}
```

**주문 매칭 엔진** (exchange_service.go):
```
매수 주문 접수 시:
  1. company.listed == 1 확인
  2. 매수: 개인 지갑 잔고 확인 (shares × price ≥ 잔고)
     매도: 보유 주식 확인 (shareholders.shares ≥ 주문 수량)
  3. 반대 주문 큐에서 매칭 시도:
     - 매수 → 매도 주문 중 가격 ≤ 매수가인 것을 가격 ASC로 매칭
     - 매도 → 매수 주문 중 가격 ≥ 매도가인 것을 가격 DESC로 매칭
  4. 체결 시:
     - stock_trades 생성
     - 매수자 지갑 차감, 매도자 지갑 입금
     - shareholders 업데이트 (매수자 +shares, 매도자 -shares)
     - company.valuation = 체결가 × total_shares
     - 미체결분은 open 상태로 유지
  5. WebSocket 시세 업데이트 브로드캐스트
```

#### `GET /api/exchange/companies/:id/orderbook`
미들웨어: Approved
```json
{
  "data": {
    "company_id": 1,
    "company_name": "우리회사",
    "last_price": 5000,
    "change_percent": 2.5,
    "volume_24h": 3500,
    "market_cap": 62500000,
    "asks": [   // 매도 호가 (가격 ASC)
      { "price": 5100, "shares": 200 },
      { "price": 5200, "shares": 500 }
    ],
    "bids": [   // 매수 호가 (가격 DESC)
      { "price": 4900, "shares": 300 },
      { "price": 4800, "shares": 150 }
    ]
  }
}
```

### 4.8 Bank (Loans)

#### `POST /api/bank/loans/apply`
미들웨어: Approved
```json
// Request
{ "amount": 10000000, "purpose": "프로젝트 투자 자금" }
// Response 201
{ "data": { "id": 1, "amount": 10000000, "status": "pending" } }
```

#### `PUT /api/bank/loans/:id/approve`
미들웨어: Admin
```json
// Request
{ "interest_rate": 0.05 }   // 주당 5%
// Response 200
```
**비즈니스 로직**:
1. loan.status → 'active'
2. 대출금 → 학생 개인 지갑 입금 (tx_type: 'loan_disbursement')
3. next_payment = now + 7일
4. penalty_rate = interest_rate × 2

#### `POST /api/bank/loans/:id/repay`
미들웨어: Approved
```json
// Request
{ "amount": 2000000 }
// Response 200
{
  "data": {
    "principal_paid": 1500000,
    "interest_paid": 500000,
    "remaining": 8500000,
    "status": "active"
  }
}
```
**이자 계산**:
```
weekly_interest = remaining × interest_rate
overdue_interest = remaining × penalty_rate  (연체 시)
상환 시: 이자 먼저 차감 → 나머지 원금 상환
remaining == 0 → status = 'paid'
```

### 4.9 Wallet

#### `GET /api/wallet`
미들웨어: Approved
```json
{
  "data": {
    "balance": 45000000,
    "total_asset_value": 72500000,
    "asset_breakdown": {
      "cash": 45000000,
      "stock_value": 32500000,        // Σ(보유 주식 × 최종 주가)
      "company_equity": 5000000,      // Σ(회사 지갑 잔고 × 내 지분율)
      "total_debt": -10000000         // Σ(미상환 원금 + 미납 이자)
    },
    "rank": 3,
    "total_students": 30
  }
}
```

#### `POST /api/wallet/transfer`
미들웨어: Admin
```json
// Request
{
  "target_user_ids": [2, 3, 5],    // 또는 "all" 전체
  "amount": 1000000,
  "description": "과제 1 보상"
}
// Response 200
{ "data": { "transferred_count": 3, "total_amount": 3000000 } }
```

#### `GET /api/wallet/ranking`
미들웨어: Approved
```json
{
  "data": [
    { "rank": 1, "user_id": 5, "name": "박학생", "student_id_display": "26학번", "total_asset_value": 85000000 },
    { "rank": 2, "user_id": 2, "name": "김이화", "student_id_display": "24학번", "total_asset_value": 72500000 }
  ]
}
```

### 4.10 Posts & SNS

#### `POST /api/posts`
미들웨어: Approved
```json
// Request
{
  "channel_id": 2,
  "content": "바이브코딩으로 첫 프로젝트 시작! #바이브코딩 #웹앱",
  "media": [
    { "url": "/uploads/abc123.png", "type": "image", "name": "screenshot.png" }
  ],
  "post_type": "normal"
}
```
- channel.write_role 검증 (#공지 → admin만)
- tags 자동 추출 (content에서 #태그 파싱)

### 4.11 Freelance Market

#### `POST /api/jobs`
미들웨어: Approved
```json
// Request
{
  "title": "랜딩 페이지 제작",
  "description": "React로 간단한 랜딩 페이지...",
  "budget": 500000,
  "deadline": "2026-04-01T00:00:00Z",
  "required_skills": ["React", "CSS"]
}
```

#### `PUT /api/jobs/:id/accept/:appId`
미들웨어: Approved (의뢰자만)
```json
// Response 200
```
**비즈니스 로직**:
1. 의뢰자 지갑에서 agreed_price 차감 → escrow
2. job.status → 'in_progress'
3. job.freelancer_id 설정

#### `PUT /api/jobs/:id/approve`
미들웨어: Approved (의뢰자만)
```json
// Response 200 - 작업물 승인 & 정산
```
**비즈니스 로직**:
1. escrow → 수주자 지갑 입금
2. job.status → 'completed'
3. 상호 리뷰 요청 알림

---

## 5. WebSocket 이벤트

**연결**: `ws://host/ws?token=<JWT>`

**서버 → 클라이언트**:
```json
{ "event": "wallet_update",      "data": { "balance": 45000000, "total_asset_value": 72500000 } }
{ "event": "notification",       "data": { "id": 1, "type": "investment", "title": "..." } }
{ "event": "stock_price_update", "data": { "company_id": 1, "price": 5000, "volume": 100 } }
{ "event": "stock_trade",        "data": { "company_id": 1, "price": 5000, "shares": 100 } }
{ "event": "orderbook_update",   "data": { "company_id": 1, "asks": [...], "bids": [...] } }
{ "event": "new_post",           "data": { "channel_id": 2, "post_id": 15 } }
{ "event": "user_approved",      "data": { "user_id": 3 } }
```

**허브 구조**:
```go
type Hub struct {
    clients    map[int]*Client     // user_id → Client
    broadcast  chan Message
    register   chan *Client
    unregister chan *Client
}
```
- 전체 브로드캐스트 (시세), 개인 메시지 (알림, 지갑), 채널별 (게시글)

---

## 6. 자산가치 계산 서비스

```go
// valuation_service.go
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

---

## 7. Docker 구성

### docker-compose.yml
```yaml
version: '3.8'
services:
  backend:
    build: ./backend
    ports:
      - "8080:8080"
    volumes:
      - db_data:/data/db
      - upload_data:/data/uploads
    environment:
      - DB_PATH=/data/db/earnlearning.db
      - UPLOAD_PATH=/data/uploads
      - JWT_SECRET=${JWT_SECRET}
      - ADMIN_EMAIL=cyc@snu.ac.kr
      - ADMIN_PASSWORD=test1234

  frontend:
    build: ./frontend
    ports:
      - "3000:3000"
    environment:
      - NEXT_PUBLIC_API_URL=http://backend:8080
      - NEXT_PUBLIC_WS_URL=ws://backend:8080

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - backend
      - frontend

volumes:
  db_data:
  upload_data:
```

### nginx.conf
```nginx
events { worker_connections 1024; }
http {
    upstream backend  { server backend:8080; }
    upstream frontend { server frontend:3000; }

    server {
        listen 80;

        location /api/ {
            proxy_pass http://backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
        }

        location /ws {
            proxy_pass http://backend;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }

        location /uploads/ {
            proxy_pass http://backend;
        }

        location / {
            proxy_pass http://frontend;
        }
    }
}
```

---

## 8. Frontend 라우팅 & 인증 흐름

```
미인증 → /login, /register
pending → /pending (승인 대기 안내)
approved → /(main)/* 모든 페이지 접근
admin → /(main)/admin/* 추가 접근
```

### 인증 가드 (middleware.ts)
```typescript
// Next.js Middleware
export function middleware(request: NextRequest) {
  const token = request.cookies.get('token')
  const path = request.nextUrl.pathname

  // Public 경로
  if (['/login', '/register'].includes(path)) {
    return token ? redirect('/feed') : next()
  }

  // 미인증
  if (!token) return redirect('/login')

  // JWT 디코딩 → status 확인
  const payload = decodeJWT(token)
  if (payload.status === 'pending') return redirect('/pending')
  if (payload.status === 'rejected') return redirect('/login')

  // Admin 경로
  if (path.startsWith('/admin') && payload.role !== 'admin') {
    return redirect('/feed')
  }

  return next()
}
```

### 하단 네비게이션 (모바일 퍼스트)
```
[홈/피드] [자산] [마켓] [회사] [더보기]
                                  ├── 투자
                                  ├── 거래소
                                  ├── 은행
                                  ├── 프로필
                                  └── 관리자 (Admin)
```

---

## 9. 핵심 비즈니스 로직 흐름도

### 투자 라운드 → 신주 발행 전체 흐름
```
설립자: POST /companies/:id/rounds
  → 라운드 생성 (status: open)
  → IR 게시글 자동 생성 (#투자라운지)

투자자: POST /rounds/:id/invest
  → BEGIN TRANSACTION
  → 투자자 지갑 잔고 확인
  → 투자자 지갑 차감
  → round.current_amount += amount
  → investments 레코드 생성
  → IF current_amount >= target_amount:
      → round.status = 'funded'
      → company.total_shares += new_shares
      → 각 투자자 shares 계산 & shareholders upsert
      → company_wallet.balance += total_invested
      → company.valuation = post_money
      → IF company_wallet.balance >= 50,000,000:
          → company.listed = 1
      → 설립자 & 투자자 알림
  → COMMIT
```

### 주식 거래 체결 흐름
```
주문자: POST /exchange/orders
  → BEGIN TRANSACTION
  → company.listed == 1 확인
  → 매수: 지갑 잔고 확인 & 동결
     매도: 보유 주식 확인 & 동결
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

### 이자 자동 차감 (주간 배치)
```
매주 월요일 00:00 실행 (또는 API 호출):
  → active 상태 대출 전체 조회
  → FOR EACH loan:
      → weekly_interest = remaining × interest_rate
      → IF 지갑 잔고 >= weekly_interest:
          → 지갑 차감 (tx_type: 'loan_interest')
          → loan_payments 생성
          → loan.next_payment += 7일
      → ELSE:
          → loan.status = 'overdue'
          → overdue_interest = remaining × penalty_rate
          → 연체 알림 발송
```

---

## 10. 보안 체크리스트

- [ ] 비밀번호 bcrypt (cost 10+)
- [ ] JWT 만료 시간 (24시간) + Refresh Token 검토
- [ ] SQL Injection 방지: Prepared Statement 전용 (raw query 금지)
- [ ] XSS 방지: HTML sanitize (게시글 content)
- [ ] CORS: 프론트엔드 도메인만 허용
- [ ] Rate Limiting: 로그인 시도 제한 (5회/분)
- [ ] 파일 업로드: MIME 타입 검증, 크기 제한 (10MB)
- [ ] Admin API: role 검증 미들웨어 필수
- [ ] 금액 조작 방지: 서버 사이드 잔고 검증 (클라이언트 값 무시)
- [ ] 동시성: SQLite WAL + IMMEDIATE 트랜잭션 (주문 매칭)
