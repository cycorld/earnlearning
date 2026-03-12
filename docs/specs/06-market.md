# 06. Market Domain — 외주 마켓

## 1. 유저 스토리

| ID | 역할 | 스토리 | 우선순위 |
|----|------|--------|---------|
| MKT-01 | 학생(의뢰자) | 외주 일거리를 등록할 수 있다 (설명, 예산, 마감, 스킬) | P0 |
| MKT-02 | 학생(수주자) | 일거리 목록을 필터하여 조회할 수 있다 | P0 |
| MKT-03 | 학생(수주자) | 일거리에 지원서(포트폴리오, 견적)를 제출할 수 있다 | P0 |
| MKT-04 | 의뢰자 | 지원자를 선택하여 계약할 수 있다 (에스크로 동결) | P0 |
| MKT-05 | 의뢰자 | 작업물을 승인하여 정산할 수 있다 | P0 |
| MKT-06 | 양쪽 | 완료 후 상호 평점(1~5)을 남길 수 있다 | P1 |
| MKT-07 | Admin | 분쟁 시 중재할 수 있다 | P1 |
| MKT-08 | Admin | 일거리를 등록하여 학생에게 수익 기회를 제공할 수 있다 | P1 |

---

## 2. 도메인 모델

### Entity

```go
type JobStatus string
const (
    JobOpen       JobStatus = "open"
    JobInProgress JobStatus = "in_progress"
    JobCompleted  JobStatus = "completed"
    JobDisputed   JobStatus = "disputed"
    JobCancelled  JobStatus = "cancelled"
)

type FreelanceJob struct {
    ID             int
    ClientID       int        // 의뢰자
    Title          string
    Description    string
    Budget         int        // 예산 (원)
    Deadline       *time.Time
    RequiredSkills string     // JSON array
    Status         JobStatus
    FreelancerID   *int       // 수주자 (계약 후 설정)
    EscrowAmount   int        // 에스크로 동결 금액
    AgreedPrice    int        // 합의 금액
    WorkCompleted  bool       // 수주자 작업 완료 표시
    CreatedAt      time.Time
    CompletedAt    *time.Time
}

type ApplicationStatus string
const (
    AppPending  ApplicationStatus = "pending"
    AppAccepted ApplicationStatus = "accepted"
    AppRejected ApplicationStatus = "rejected"
)

type JobApplication struct {
    ID        int
    JobID     int
    UserID    int        // 지원자
    Proposal  string
    Price     int        // 견적
    Status    ApplicationStatus
    CreatedAt time.Time
}

type FreelanceReview struct {
    ID         int
    JobID      int
    ReviewerID int
    RevieweeID int
    Rating     int        // 1~5
    Comment    string
    CreatedAt  time.Time
}
```

### 도메인 규칙

- **에스크로**: 계약 시 의뢰자 **개인 지갑**에서 합의 금액 차감 & 동결
- **정산**: 의뢰자 승인 시 에스크로 → 수주자 **개인 지갑** 입금
- **개인 간 거래**: 회사가 아닌 개인 지갑 간 거래 (회사 설립 전에도 이용 가능)
- **분쟁**: disputed 상태에서 Admin이 판정

---

## 3. 데이터베이스 스키마

```sql
CREATE TABLE freelance_jobs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id       INTEGER NOT NULL REFERENCES users(id),
    title           TEXT    NOT NULL,
    description     TEXT    NOT NULL,
    budget          INTEGER NOT NULL,
    deadline        DATETIME,
    required_skills TEXT    DEFAULT '[]',
    status          TEXT    NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open','in_progress','completed','disputed','cancelled')),
    freelancer_id   INTEGER REFERENCES users(id),
    escrow_amount   INTEGER NOT NULL DEFAULT 0,
    agreed_price    INTEGER DEFAULT 0,
    work_completed  INTEGER NOT NULL DEFAULT 0,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at    DATETIME
);

CREATE TABLE job_applications (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id     INTEGER NOT NULL REFERENCES freelance_jobs(id),
    user_id    INTEGER NOT NULL REFERENCES users(id),
    proposal   TEXT    NOT NULL,
    price      INTEGER NOT NULL,
    status     TEXT    NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending', 'accepted', 'rejected')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(job_id, user_id)
);

CREATE TABLE freelance_reviews (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id      INTEGER NOT NULL REFERENCES freelance_jobs(id),
    reviewer_id INTEGER NOT NULL REFERENCES users(id),
    reviewee_id INTEGER NOT NULL REFERENCES users(id),
    rating      INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment     TEXT    DEFAULT '',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(job_id, reviewer_id)
);
```

---

## 4. API 상세

### `GET /api/jobs`
**미들웨어**: Approved

```
Query: ?status=open&skills=React&min_budget=100000&page=1&limit=20
```

```json
// Response 200
{
  "data": [
    {
      "id": 1,
      "client": { "id": 2, "name": "김이화", "rating": 4.8 },
      "title": "랜딩 페이지 제작",
      "description": "React로 간단한 랜딩 페이지...",
      "budget": 500000,
      "deadline": "2026-04-01T00:00:00Z",
      "required_skills": ["React", "CSS"],
      "status": "open",
      "application_count": 3,
      "created_at": "2026-03-10T10:00:00Z"
    }
  ],
  "pagination": { ... }
}
```

---

### `POST /api/jobs`
**미들웨어**: Approved

```json
// Request
{
  "title": "랜딩 페이지 제작",
  "description": "React로 간단한 랜딩 페이지...",
  "budget": 500000,
  "deadline": "2026-04-01T00:00:00Z",
  "required_skills": ["React", "CSS"]
}

// Response 201
{ "data": { "id": 1, "status": "open", ... } }
```

---

### `GET /api/jobs/:id`
**미들웨어**: Approved

```json
// Response 200
{
  "data": {
    "id": 1,
    "client": { "id": 2, "name": "김이화" },
    "title": "랜딩 페이지 제작",
    "description": "...",
    "budget": 500000,
    "status": "open",
    "applications": [
      {
        "id": 1,
        "user": { "id": 5, "name": "박학생", "rating": 4.5 },
        "proposal": "경험 많습니다...",
        "price": 450000,
        "status": "pending"
      }
    ]
  }
}
```

- applications는 의뢰자에게만 노출

---

### `POST /api/jobs/:id/apply`
**미들웨어**: Approved

```json
// Request
{ "proposal": "React 경험 2개월, 포트폴리오: ...", "price": 450000 }

// Response 201
{ "data": { "id": 1, "status": "pending" } }
```

- 자기 자신의 일거리에 지원 불가

---

### `PUT /api/jobs/:id/accept/:appId`
**미들웨어**: Approved (의뢰자만)

```json
// Response 200
{
  "data": {
    "job_id": 1,
    "freelancer_id": 5,
    "agreed_price": 450000,
    "escrow_amount": 450000,
    "status": "in_progress"
  }
}
```

**비즈니스 로직** (트랜잭션):
1. 의뢰자 지갑 잔고 확인 (잔고 ≥ agreed_price)
2. 의뢰자 지갑에서 차감 (tx_type: `freelance_escrow`)
3. job.escrow_amount = agreed_price
4. job.freelancer_id = 지원자
5. job.agreed_price = application.price
6. job.status = 'in_progress'
7. 수주자에게 알림

---

### `PUT /api/jobs/:id/approve`
**미들웨어**: Approved (의뢰자만)

```json
// Response 200
{ "data": { "job_id": 1, "status": "completed", "paid_amount": 450000 } }
```

**비즈니스 로직** (트랜잭션):
1. escrow → 수주자 개인 지갑 입금 (tx_type: `freelance_payment`)
2. job.status = 'completed'
3. job.completed_at = now
4. 상호 리뷰 요청 알림

---

### `PUT /api/jobs/:id/complete`
**미들웨어**: Approved (수주자만, status = 'in_progress'일 때만)

```json
// Response 200
{ "data": { "job_id": 1, "status": "in_progress", "work_completed": true } }
```

- 수주자가 작업 완료를 의뢰자에게 알림
- job.status는 유지 (`in_progress`), 의뢰자 approve 대기
- 의뢰자에게 "작업 완료 확인 요청" 알림 발송

---

### `PUT /api/jobs/:id/cancel`
**미들웨어**: Approved (의뢰자만, status = 'open'일 때만)

```json
// Response 200
{ "data": { "job_id": 1, "status": "cancelled" } }
```

---

### `PUT /api/jobs/:id/dispute`
**미들웨어**: Approved (의뢰자 또는 수주자)

```json
// Response 200
{ "data": { "job_id": 1, "status": "disputed" } }
```

- Admin이 중재 후 에스크로 배분 결정

---

### `POST /api/jobs/:id/review`
**미들웨어**: Approved (완료된 job의 의뢰자/수주자)

```json
// Request
{ "rating": 5, "comment": "빠르고 정확한 작업이었습니다" }

// Response 201
{ "data": { "id": 1, "rating": 5 } }
```

---

## 5. UI 스펙

### 5.1 외주 마켓 목록 (`/market`)

```
┌─────────────────────────────────┐
│  외주 마켓                [+ 등록] │
│                                 │
│  ┌─────────────────────────┐   │
│  │ 상태: [전체▼]  예산: [전체▼]│   │
│  │ 스킬: [React] [×]       │   │
│  └─────────────────────────┘   │
│                                 │
│  ┌─────────────────────────┐   │
│  │ 랜딩 페이지 제작          │   │
│  │ 김이화 · ⭐ 4.8           │   │
│  │ 예산: 50만원              │   │
│  │ 마감: 2026-04-01          │   │
│  │ [React] [CSS]            │   │
│  │ 지원 3명 · 등록 3일 전     │   │
│  └─────────────────────────┘   │
│  ┌─────────────────────────┐   │
│  │ 데이터 시각화 대시보드     │   │
│  │ 관리자 · ⭐ -             │   │
│  │ 예산: 100만원             │   │
│  │ ...                       │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
```

### 5.2 일거리 상세 (`/market/[id]`)

```
┌─────────────────────────────────┐
│  ← 랜딩 페이지 제작              │
│                                 │
│  의뢰자: 김이화 ⭐ 4.8           │
│  예산: 50만원                    │
│  마감: 2026-04-01               │
│  상태: 모집중                    │
│                                 │
│  ── 상세 설명 ──────────────     │
│  React로 간단한 랜딩 페이지...   │
│                                 │
│  필요 스킬: [React] [CSS]        │
│                                 │
│  ── 지원하기 ───────────────     │  (본인 일거리가 아닐 때)
│  ┌─────────────────────────┐   │
│  │ 제안서                    │   │
│  │ [                        ]│   │
│  │ 견적: [        ] 원      │   │
│  │ [지원하기]                │   │
│  └─────────────────────────┘   │
│                                 │
│  ── 지원자 목록 ────────────     │  (의뢰자에게만)
│  박학생 ⭐4.5  견적: 45만원      │
│  [프로필] [수락] [거절]          │
└─────────────────────────────────┘
```

### 5.3 진행 중인 외주 관리

```
┌─────────────────────────────────┐
│  내 외주                         │
│  ┌──────┬──────┬──────┐        │
│  │ 의뢰 │ 수주  │ 완료  │        │
│  └──────┴──────┴──────┘        │
│                                 │
│  ┌─────────────────────────┐   │
│  │ 랜딩 페이지 제작          │   │
│  │ 수주자: 박학생            │   │
│  │ 금액: 45만원 (에스크로)   │   │
│  │ 상태: 진행중              │   │
│  │ [작업 승인]  [분쟁 신고]   │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
```
