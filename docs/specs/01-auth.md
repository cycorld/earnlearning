# 01. Identity Domain — 인증 & 사용자 관리

## 1. 유저 스토리

| ID | 역할 | 스토리 | 우선순위 |
|----|------|--------|---------|
| AUTH-01 | 방문자 | 이메일, 이름, 학과, 학번, 비밀번호로 회원가입할 수 있다 | P0 |
| AUTH-02 | 방문자 | 이메일/비밀번호로 로그인할 수 있다 | P0 |
| AUTH-03 | Pending 사용자 | 승인 대기 안내 페이지를 볼 수 있다 | P0 |
| AUTH-04 | Admin | 가입 대기 중인 학생 목록을 볼 수 있다 | P0 |
| AUTH-05 | Admin | 학생을 승인/거절할 수 있다 | P0 |
| AUTH-06 | Admin | 전체 사용자 목록을 조회할 수 있다 (전체 학번 열람) | P0 |
| AUTH-07 | 학생 | 내 프로필 정보를 조회할 수 있다 | P0 |
| AUTH-08 | 학생 | 다른 사용자의 프로필을 조회할 수 있다 | P1 |

---

## 2. 도메인 모델

### Entity: User

```go
type Role string
const (
    RoleAdmin   Role = "admin"
    RoleStudent Role = "student"
)

type Status string
const (
    StatusPending  Status = "pending"
    StatusApproved Status = "approved"
    StatusRejected Status = "rejected"
)

type User struct {
    ID         int
    Email      string     // unique
    Password   string     // bcrypt hash
    Name       string
    Department string     // 학과
    StudentID  string     // 전체 학번 (예: "2024123456")
    Role       Role
    Status     Status
    Bio        string
    AvatarURL  string
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

// 학번 마스킹: 학생 간에는 앞 2자리만 표시 (예: "24학번")
func (u *User) StudentIDDisplay(viewerRole Role) string {
    if viewerRole == RoleAdmin {
        return u.StudentID
    }
    return u.StudentID[:2] + "학번"
}
```

### Repository Interface

```go
type UserRepository interface {
    Create(user *User) error
    FindByID(id int) (*User, error)
    FindByEmail(email string) (*User, error)
    FindByStatus(status Status, page, limit int) ([]*User, int, error)
    FindAll(page, limit int) ([]*User, int, error)
    UpdateStatus(id int, status Status) error
    Update(user *User) error
}
```

---

## 3. 데이터베이스 스키마

```sql
CREATE TABLE users (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    email       TEXT    NOT NULL UNIQUE,
    password    TEXT    NOT NULL,          -- bcrypt hash
    name        TEXT    NOT NULL,
    department  TEXT    NOT NULL,
    student_id  TEXT    NOT NULL,          -- 전체 학번 (예: "2024123456")
    role        TEXT    NOT NULL DEFAULT 'student'
                CHECK (role IN ('admin', 'student')),
    status      TEXT    NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'approved', 'rejected')),
    bio         TEXT    DEFAULT '',
    avatar_url  TEXT    DEFAULT '',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Admin 시드 데이터

```sql
-- 서버 최초 기동 시 실행 (이미 존재하면 스킵)
INSERT OR IGNORE INTO users (email, password, name, department, student_id, role, status)
VALUES (
    ${ADMIN_EMAIL},       -- .env 참조
    '$2a$10$...', -- bcrypt(${ADMIN_PASSWORD})
    '관리자',
    '관리자',
    '0000000000',
    'admin',
    'approved'
);
```

---

## 4. API 상세

### `POST /api/auth/register`
**미들웨어**: Public

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
    "message": "관리자 승인을 기다리고 있습니다. 문의: ${CONTACT_EMAIL}"
  }
}
```

**검증 규칙**:
- 비밀번호: bcrypt 해싱, 최소 8자
- email: unique 검증
- student_id: 숫자, 7~10자리

---

### `POST /api/auth/login`
**미들웨어**: Public

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
      "status": "approved",
      "department": "컴퓨터공학과",
      "student_id_display": "24학번"
    }
  }
}
```

**규칙**:
- status가 "pending"이어도 로그인 성공 (프론트에서 /pending으로 분기)
- status가 "rejected"이면 로그인 거부

---

### `GET /api/auth/me`
**미들웨어**: Auth

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

---

### `GET /api/users/:id/profile`
**미들웨어**: Approved

```json
// Response 200
{
  "data": {
    "id": 5,
    "name": "박학생",
    "department": "경영학과",
    "student_id_display": "26학번",
    "bio": "바이브코딩 좋아하는 학생",
    "avatar_url": "/uploads/avatar5.png",
    "company_count": 1,
    "freelance_rating": 4.5,
    "freelance_completed": 3
  }
}
```

---

### `GET /api/admin/users/pending`
**미들웨어**: Admin

```json
// Response 200
{
  "data": [
    {
      "id": 3,
      "email": "kim@ewha.ac.kr",
      "name": "김학생",
      "department": "경영학과",
      "student_id": "2026543210",
      "created_at": "2026-03-13T10:00:00Z"
    }
  ]
}
```

---

### `PUT /api/admin/users/:id/approve`
**미들웨어**: Admin

```json
// Response 200
{ "data": { "id": 3, "status": "approved" } }
```

**부수 효과**: WebSocket `user_approved` 이벤트 발송

---

### `PUT /api/admin/users/:id/reject`
**미들웨어**: Admin

```json
// Response 200
{ "data": { "id": 3, "status": "rejected" } }
```

---

### `GET /api/admin/users`
**미들웨어**: Admin

```
Query: ?status=approved&page=1&limit=20
```
전체 사용자 목록 (student_id 전체 노출)

---

## 5. UI 스펙

### 5.1 회원가입 화면 (`/register`)

```
┌─────────────────────────────┐
│       EarnLearning          │
│                             │
│  ┌───────────────────────┐  │
│  │ 이메일                 │  │
│  └───────────────────────┘  │
│  ┌───────────────────────┐  │
│  │ 비밀번호 (8자 이상)     │  │
│  └───────────────────────┘  │
│  ┌───────────────────────┐  │
│  │ 이름                   │  │
│  └───────────────────────┘  │
│  ┌───────────────────────┐  │
│  │ 학과                   │  │
│  └───────────────────────┘  │
│  ┌───────────────────────┐  │
│  │ 학번 (숫자 7~10자리)    │  │
│  └───────────────────────┘  │
│                             │
│  ┌───────────────────────┐  │
│  │       회원가입          │  │
│  └───────────────────────┘  │
│                             │
│  이미 계정이 있나요? 로그인  │
└─────────────────────────────┘
```

- 실시간 입력 검증 (이메일 형식, 비밀번호 길이, 학번 형식)
- 가입 성공 시 "승인 대기" 안내 모달 후 `/pending`으로 이동

### 5.2 로그인 화면 (`/login`)

```
┌─────────────────────────────┐
│       EarnLearning          │
│                             │
│  ┌───────────────────────┐  │
│  │ 이메일                 │  │
│  └───────────────────────┘  │
│  ┌───────────────────────┐  │
│  │ 비밀번호               │  │
│  └───────────────────────┘  │
│                             │
│  ┌───────────────────────┐  │
│  │        로그인          │  │
│  └───────────────────────┘  │
│                             │
│  계정이 없나요? 회원가입     │
└─────────────────────────────┘
```

- 로그인 성공 후 status에 따라 라우팅:
  - `approved` → `/feed`
  - `pending` → `/pending`

### 5.3 승인 대기 화면 (`/pending`)

```
┌─────────────────────────────┐
│                             │
│          ⏳                 │
│                             │
│  관리자 승인을 기다리고       │
│  있습니다.                   │
│                             │
│  문의: ${CONTACT_EMAIL}        │
│                             │
│  ┌───────────────────────┐  │
│  │       로그아웃          │  │
│  └───────────────────────┘  │
└─────────────────────────────┘
```

- 자동 새로고침 또는 WebSocket으로 승인 시 자동 이동

### 5.4 Admin: 학생 승인 관리 (`/admin/users`)

```
┌─────────────────────────────────┐
│  학생 관리                       │
│  ┌──────┬──────┐               │
│  │ 대기중 │ 전체  │               │
│  └──────┴──────┘               │
│                                 │
│  ┌─────────────────────────┐   │
│  │ 김학생                    │   │
│  │ kim@ewha.ac.kr           │   │
│  │ 경영학과 · 2026543210     │   │
│  │ 가입일: 2026-03-13        │   │
│  │ [승인]  [거절]            │   │
│  └─────────────────────────┘   │
│  ┌─────────────────────────┐   │
│  │ 박학생                    │   │
│  │ ...                       │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
```

---

## 6. 인증 가드 (Frontend — React Router)

```typescript
// guards/AuthGuard.tsx
function AuthGuard({ children }: { children: React.ReactNode }) {
  const { user, isLoading } = useAuth()

  if (isLoading) return <LoadingSpinner />
  if (!user) return <Navigate to="/login" replace />
  if (user.status === 'pending') return <Navigate to="/pending" replace />
  if (user.status === 'rejected') return <Navigate to="/login" replace />

  return <>{children}</>
}

// guards/AdminGuard.tsx
function AdminGuard({ children }: { children: React.ReactNode }) {
  const { user } = useAuth()
  if (user?.role !== 'admin') return <Navigate to="/feed" replace />
  return <>{children}</>
}

// App.tsx (React Router)
function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Public */}
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/pending" element={<PendingPage />} />

        {/* Approved */}
        <Route element={<AuthGuard><MainLayout /></AuthGuard>}>
          <Route path="/feed" element={<FeedPage />} />
          <Route path="/wallet" element={<WalletPage />} />
          <Route path="/market" element={<MarketPage />} />
          <Route path="/company" element={<CompanyListPage />} />
          {/* ... */}

          {/* Admin */}
          <Route element={<AdminGuard><Outlet /></AdminGuard>}>
            <Route path="/admin" element={<AdminPage />} />
            <Route path="/admin/users" element={<AdminUsersPage />} />
            {/* ... */}
          </Route>
        </Route>

        <Route path="*" element={<Navigate to="/feed" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
```

**라우팅 규칙**:
```
미인증      → /login, /register
pending    → /pending (승인 대기 안내)
approved   → /* 모든 페이지 접근
admin      → /admin/* 추가 접근
```
