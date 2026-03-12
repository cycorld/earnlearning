# 02. Classroom Domain — 강의실

## 1. 유저 스토리

| ID | 역할 | 스토리 | 우선순위 |
|----|------|--------|---------|
| CLS-01 | Admin | 강의실을 생성하고 참여 코드를 발급받을 수 있다 | P0 |
| CLS-02 | 학생 | 참여 코드를 입력하여 강의실에 참여할 수 있다 | P0 |
| CLS-03 | 학생 | 강의실 참여 시 초기 자본금(5,000만원)을 지급받는다 | P0 |
| CLS-04 | 학생 | 동일 강의실에 중복 참여할 수 없다 | P0 |

---

## 2. 도메인 모델

### Entity

```go
type Classroom struct {
    ID             int
    Name           string
    Code           string    // 6자리 랜덤 참여 코드
    CreatedBy      int       // Admin user_id
    InitialCapital int       // 기본 5,000만원
    Settings       string    // JSON (향후 확장)
    CreatedAt      time.Time
}

type ClassroomMember struct {
    ID          int
    ClassroomID int
    UserID      int
    JoinedAt    time.Time
}
```

### 도메인 규칙

- 강의실 코드: 서버에서 6자리 영문+숫자 랜덤 생성
- 참여 시 개인 지갑에 `initial_capital` 입금 (tx_type: `initial_capital`)
- 중복 참여 방지 (UNIQUE 제약)

---

## 3. 데이터베이스 스키마

```sql
CREATE TABLE classrooms (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT    NOT NULL,
    code            TEXT    NOT NULL UNIQUE,
    created_by      INTEGER NOT NULL REFERENCES users(id),
    initial_capital INTEGER NOT NULL DEFAULT 50000000,  -- 5,000만원
    settings        TEXT    DEFAULT '{}',
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE classroom_members (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    classroom_id INTEGER NOT NULL REFERENCES classrooms(id),
    user_id      INTEGER NOT NULL REFERENCES users(id),
    joined_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(classroom_id, user_id)
);
```

---

## 4. API 상세

### `POST /api/classrooms`
**미들웨어**: Admin

```json
// Request
{ "name": "2026 스타트업을위한코딩입문", "initial_capital": 50000000 }

// Response 201
{ "data": { "id": 1, "name": "2026 스타트업을위한코딩입문", "code": "ABC123" } }
```

- code는 서버에서 6자리 랜덤 생성 (중복 시 재생성)

---

### `POST /api/classrooms/join`
**미들웨어**: Approved

```json
// Request
{ "code": "ABC123" }

// Response 200
{ "data": { "classroom_id": 1, "initial_capital": 50000000 } }
```

**비즈니스 로직** (트랜잭션):
1. 코드로 강의실 조회
2. 중복 참여 확인 → 이미 참여했으면 `DUPLICATE` 에러
3. `classroom_members` 레코드 생성
4. 개인 지갑이 없으면 생성
5. 지갑에 `initial_capital` 입금
6. 트랜잭션 로그: `tx_type = 'initial_capital'`

---

### `GET /api/classrooms/:id`
**미들웨어**: Approved

```json
// Response 200
{
  "data": {
    "id": 1,
    "name": "2026 스타트업을위한코딩입문",
    "code": "ABC123",
    "initial_capital": 50000000,
    "member_count": 30,
    "created_at": "2026-03-01T00:00:00Z"
  }
}
```

---

## 5. UI 스펙

### 5.1 강의실 참여 (첫 로그인 시)

최초 승인된 사용자가 아직 강의실에 참여하지 않았다면 참여 화면으로 안내.

```
┌─────────────────────────────┐
│       강의실 참여             │
│                             │
│  강의실 코드를 입력하세요     │
│                             │
│  ┌───────────────────────┐  │
│  │  ABC123               │  │
│  └───────────────────────┘  │
│                             │
│  ┌───────────────────────┐  │
│  │        참여하기         │  │
│  └───────────────────────┘  │
│                             │
│  참여 시 초기 자본금          │
│  5,000만원이 지급됩니다.     │
└─────────────────────────────┘
```

- 참여 성공 시 축하 모달 → `/feed`로 이동
- 코드가 잘못되면 인라인 에러 메시지

### 5.2 Admin: 강의실 관리 (`/admin/classroom`)

```
┌─────────────────────────────────┐
│  강의실 관리                     │
│                                 │
│  ┌─────────────────────────┐   │
│  │ + 새 강의실 만들기        │   │
│  └─────────────────────────┘   │
│                                 │
│  ┌─────────────────────────┐   │
│  │ 2026 스타트업을위한코딩입문│   │
│  │ 코드: ABC123  (복사)     │   │
│  │ 참여: 28/30명            │   │
│  │ 초기자본: 5,000만원       │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
```
