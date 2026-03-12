# 05. Feed Domain — SNS & 과제

## 1. 유저 스토리

| ID | 역할 | 스토리 | 우선순위 |
|----|------|--------|---------|
| FED-01 | 학생 | 채널별로 게시글 피드를 볼 수 있다 (무한 스크롤) | P0 |
| FED-02 | 학생 | 텍스트, 이미지, 링크를 포함한 게시글을 작성할 수 있다 | P0 |
| FED-03 | 학생 | 게시글에 좋아요를 누르거나 취소할 수 있다 | P0 |
| FED-04 | 학생 | 게시글에 댓글을 작성할 수 있다 | P0 |
| FED-05 | 학생 | 해시태그로 게시글을 검색할 수 있다 | P1 |
| FED-06 | Admin | 공지 채널에 게시글을 작성할 수 있다 | P0 |
| FED-07 | Admin | 과제를 출제할 수 있다 (마감일, 보상 금액) | P0 |
| FED-08 | 학생 | 과제에 댓글(+첨부)로 제출할 수 있다 | P0 |
| FED-09 | Admin | 과제 제출 현황을 확인하고 채점/보상 지급할 수 있다 | P0 |
| FED-10 | 학생 | #쇼케이스 채널에 바이브코딩 결과물을 공유할 수 있다 | P1 |

---

## 2. 도메인 모델

### Entity

```go
type Channel struct {
    ID          int
    ClassroomID int
    Name        string    // '#공지', '#자유', '#과제', ...
    Slug        string    // 'notice', 'free', 'assignment', ...
    ChannelType string    // 'notice', 'free', 'assignment', 'showcase',
                          // 'market', 'invest', 'exchange'
    WriteRole   string    // 'admin', 'all'
    SortOrder   int
}

type Post struct {
    ID           int
    ChannelID    int
    AuthorID     int
    Content      string
    PostType     string   // 'normal', 'assignment', 'showcase', 'ir'
    Media        string   // JSON array: [{url, type, name}]
    Tags         string   // JSON array: ["tag1", "tag2"]
    LikeCount    int
    CommentCount int
    Pinned       bool
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type Comment struct {
    ID        int
    PostID    int
    AuthorID  int
    Content   string
    Media     string   // JSON (제출물 첨부용)
    CreatedAt time.Time
}

type Assignment struct {
    ID           int
    PostID       int      // 1:1 관계
    Deadline     time.Time
    RewardAmount int      // 보상 금액
    MaxScore     int      // 기본 100점
}

type Submission struct {
    ID           int
    AssignmentID int
    StudentID    int
    CommentID    *int     // 제출 댓글 연결
    Content      string
    Files        string   // JSON
    Grade        *int     // 0~max_score (nil = 미채점)
    Rewarded     bool
    SubmittedAt  time.Time
}
```

### 도메인 규칙

- **채널 쓰기 권한**: `write_role = 'admin'`이면 Admin만 작성 가능 (#공지)
- **태그 자동 추출**: content에서 `#태그` 파싱하여 tags 필드에 저장
- **과제 제출**: assignment가 연결된 게시글에 댓글을 달면 자동으로 submission 생성
- **보상 지급**: Admin이 채점 후 개인 지갑에 `assignment_reward` 입금

---

## 3. 데이터베이스 스키마

```sql
CREATE TABLE channels (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    classroom_id INTEGER NOT NULL REFERENCES classrooms(id),
    name         TEXT    NOT NULL,
    slug         TEXT    NOT NULL,
    channel_type TEXT    NOT NULL,
    write_role   TEXT    NOT NULL DEFAULT 'all',
    sort_order   INTEGER NOT NULL DEFAULT 0,
    UNIQUE(classroom_id, slug)
);

CREATE TABLE posts (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id    INTEGER NOT NULL REFERENCES channels(id),
    author_id     INTEGER NOT NULL REFERENCES users(id),
    content       TEXT    NOT NULL,
    post_type     TEXT    NOT NULL DEFAULT 'normal',
    media         TEXT    DEFAULT '[]',
    tags          TEXT    DEFAULT '[]',
    like_count    INTEGER NOT NULL DEFAULT 0,
    comment_count INTEGER NOT NULL DEFAULT 0,
    pinned        INTEGER NOT NULL DEFAULT 0,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_posts_channel ON posts(channel_id, created_at DESC);

CREATE TABLE post_likes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id    INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(post_id, user_id)
);

CREATE TABLE comments (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id    INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id  INTEGER NOT NULL REFERENCES users(id),
    content    TEXT    NOT NULL,
    media      TEXT    DEFAULT '[]',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_comments_post ON comments(post_id, created_at);

CREATE TABLE assignments (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id       INTEGER NOT NULL UNIQUE REFERENCES posts(id),
    deadline      DATETIME NOT NULL,
    reward_amount INTEGER NOT NULL DEFAULT 0,
    max_score     INTEGER NOT NULL DEFAULT 100
);

CREATE TABLE submissions (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    assignment_id INTEGER NOT NULL REFERENCES assignments(id),
    student_id    INTEGER NOT NULL REFERENCES users(id),
    comment_id    INTEGER REFERENCES comments(id),
    content       TEXT    DEFAULT '',
    files         TEXT    DEFAULT '[]',
    grade         INTEGER DEFAULT NULL,
    rewarded      INTEGER NOT NULL DEFAULT 0,
    submitted_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(assignment_id, student_id)
);
```

### 강의실 생성 시 기본 채널 시드

```sql
-- 강의실 생성 시 자동 생성되는 채널들
INSERT INTO channels (classroom_id, name, slug, channel_type, write_role, sort_order)
VALUES
  (?, '#공지',      'notice',     'notice',     'admin', 1),
  (?, '#자유',      'free',       'free',       'all',   2),
  (?, '#과제',      'assignment', 'assignment', 'admin', 3),
  (?, '#쇼케이스',  'showcase',   'showcase',   'all',   4),
  (?, '#외주마켓',  'market',     'market',     'all',   5),
  (?, '#투자라운지', 'invest',     'invest',     'all',   6),
  (?, '#거래소',    'exchange',   'exchange',   'all',   7);
```

---

## 4. API 상세

### `GET /api/channels`
**미들웨어**: Approved

```json
// Response 200
{
  "data": [
    { "id": 1, "name": "#공지", "slug": "notice", "channel_type": "notice", "write_role": "admin" },
    { "id": 2, "name": "#자유", "slug": "free", "channel_type": "free", "write_role": "all" },
    { "id": 3, "name": "#과제", "slug": "assignment", "channel_type": "assignment", "write_role": "admin" }
  ]
}
```

---

### `GET /api/posts`
**미들웨어**: Approved

```
Query: ?channel_id=2&page=1&limit=20&tag=바이브코딩
```

```json
// Response 200
{
  "data": [
    {
      "id": 15,
      "channel": { "id": 2, "name": "#자유", "slug": "free" },
      "author": { "id": 5, "name": "박학생", "avatar_url": "...", "student_id_display": "26학번" },
      "content": "바이브코딩으로 첫 프로젝트 시작! #바이브코딩 #웹앱",
      "post_type": "normal",
      "media": [{ "url": "/uploads/abc.png", "type": "image", "name": "screenshot.png" }],
      "tags": ["바이브코딩", "웹앱"],
      "like_count": 12,
      "comment_count": 3,
      "is_liked": true,
      "pinned": false,
      "created_at": "2026-03-12T14:30:00Z"
    }
  ],
  "pagination": { "page": 1, "limit": 20, "total": 45, "total_pages": 3 }
}
```

---

### `POST /api/posts`
**미들웨어**: Approved

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

// Response 201
{ "data": { "id": 15, "tags": ["바이브코딩", "웹앱"], ... } }
```

- channel.write_role 검증 (#공지 → admin만)
- tags 자동 추출 (content에서 `#태그` 파싱)
- WebSocket: `new_post` 이벤트 발송

---

### `POST /api/posts/:id/like`
**미들웨어**: Approved

```json
// Response 200 (토글: 좋아요 추가 또는 취소)
{ "data": { "liked": true, "like_count": 13 } }
```

---

### `POST /api/posts/:id/comments`
**미들웨어**: Approved

```json
// Request
{
  "content": "멋진 프로젝트네요!",
  "media": []
}

// Response 201
{ "data": { "id": 30, "content": "멋진 프로젝트네요!", ... } }
```

- 과제 게시글에 댓글 시 자동으로 submission 생성/업데이트

---

### `POST /api/assignments`
**미들웨어**: Admin

```json
// Request
{
  "channel_id": 3,
  "content": "과제 1: 바이브코딩으로 랜딩 페이지 만들기\n\n요구사항: ...",
  "deadline": "2026-03-20T23:59:59Z",
  "reward_amount": 500000,
  "max_score": 100
}

// Response 201
{ "data": { "post_id": 20, "assignment_id": 1, ... } }
```

- #과제 채널에 게시글 + assignment 레코드 동시 생성

---

### `POST /api/assignments/:id/submit`
**미들웨어**: Approved

```json
// Request
{
  "content": "과제 제출합니다. 링크: https://...",
  "files": [{ "url": "/uploads/submit1.zip", "name": "project.zip" }]
}

// Response 201
{ "data": { "submission_id": 1, "comment_id": 35, "submitted_at": "..." } }
```

- 자동으로 과제 게시글에 댓글 생성 + submission 레코드 생성

---

### `PUT /api/assignments/:id/grade`
**미들웨어**: Admin

```json
// Request
{ "student_id": 5, "grade": 90 }

// Response 200
{
  "data": {
    "submission_id": 1,
    "grade": 90,
    "reward_amount": 450000,
    "rewarded": true
  }
}
```

**비즈니스 로직**:
1. 채점: `submission.grade = grade`
2. 보상 계산: `reward = reward_amount × grade / max_score`
3. 학생 지갑에 보상 입금 (tx_type: `assignment_reward`)
4. `submission.rewarded = true`

---

### `POST /api/uploads`
**미들웨어**: Approved

```
Content-Type: multipart/form-data
file: (binary)
```

```json
// Response 201
{
  "data": {
    "id": 1,
    "url": "/uploads/uuid-filename.png",
    "filename": "screenshot.png",
    "mime_type": "image/png",
    "size": 245760
  }
}
```

**제한사항**:
- 최대 파일 크기: 10MB
- 허용 MIME: image/*, application/pdf, application/zip, text/*

---

## 5. 파일 업로드 스키마

```sql
CREATE TABLE uploads (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    filename    TEXT    NOT NULL,
    stored_name TEXT    NOT NULL,
    mime_type   TEXT    NOT NULL,
    size        INTEGER NOT NULL,
    path        TEXT    NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 6. UI 스펙

### 6.1 피드 메인 (`/feed`)

```
┌─────────────────────────────────┐
│  EarnLearning              🔔 3 │
│                                 │
│  ┌─────────────────────────┐   │
│  │ [전체] [공지] [자유] [과제]│   │
│  │ [쇼케이스] [더보기 ▼]     │   │
│  └─────────────────────────┘   │
│                                 │
│  ┌─────────────────────────┐   │
│  │ 👤 박학생 · 26학번        │   │
│  │ 2시간 전 · #자유          │   │
│  │                          │   │
│  │ 바이브코딩으로 첫 프로젝트  │   │
│  │ 시작! #바이브코딩 #웹앱   │   │
│  │                          │   │
│  │ ┌──────────────────┐    │   │
│  │ │  [이미지 미리보기]  │    │   │
│  │ └──────────────────┘    │   │
│  │                          │   │
│  │ ❤️ 12  💬 3              │   │
│  └─────────────────────────┘   │
│                                 │
│  ┌─────────────────────────┐   │
│  │ 📌 관리자 · #공지         │   │
│  │ 이번 주 과제 안내...      │   │
│  └─────────────────────────┘   │
│                                 │
│  (무한 스크롤)                   │
│                                 │
│           [✏️ 글쓰기 FAB]       │
│                                 │
│ [홈] [자산] [마켓] [회사] [더보기] │
└─────────────────────────────────┘
```

### 6.2 게시글 작성 (모달 또는 페이지)

```
┌─────────────────────────────────┐
│  ← 글쓰기               [게시]  │
│                                 │
│  채널: [#자유 ▼]                │
│                                 │
│  ┌─────────────────────────┐   │
│  │                          │   │
│  │  내용을 입력하세요...     │   │
│  │  #해시태그를 입력하면      │   │
│  │  자동으로 태그됩니다      │   │
│  │                          │   │
│  └─────────────────────────┘   │
│                                 │
│  📎 이미지  📎 파일             │
│                                 │
│  첨부된 파일:                    │
│  [screenshot.png ×]             │
└─────────────────────────────────┘
```

### 6.3 과제 상세 & 제출

```
┌─────────────────────────────────┐
│  ← 과제 상세                     │
│                                 │
│  📋 과제 1: 랜딩 페이지 만들기   │
│  마감: 2026-03-20 23:59         │
│  보상: 50만원 (만점 기준)        │
│                                 │
│  요구사항: ...                   │
│                                 │
│  ── 내 제출 ────────────────     │
│  ┌─────────────────────────┐   │
│  │ 미제출                    │   │
│  │ [제출하기]                │   │
│  └─────────────────────────┘   │
│  또는                           │
│  ┌─────────────────────────┐   │
│  │ ✅ 제출 완료              │   │
│  │ 점수: 90/100              │   │
│  │ 보상: 450,000원 지급완료  │   │
│  └─────────────────────────┘   │
│                                 │
│  ── 제출 댓글 ──────────────     │
│  ...                            │
└─────────────────────────────────┘
```

### 6.4 하단 네비게이션

```
┌───────────────────────────────────┐
│ [🏠홈] [💰자산] [🏪마켓] [🏢회사] [⋯더보기] │
└───────────────────────────────────┘

더보기 메뉴:
├── 📈 투자
├── 📊 거래소
├── 🏦 은행
├── 👤 프로필
├── 🔔 알림
└── ⚙️ 관리자 (Admin만 표시)
```
