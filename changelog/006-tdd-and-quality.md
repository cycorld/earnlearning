---
title: "TDD와 코드 품질: 테스트가 개발을 이끄는 방식"
date: "2026-03-15"
tags: ["TDD", "테스트", "회귀테스트", "빌드", "품질"]
---

## 무엇을 했나요?

개발 프로세스의 품질을 한 단계 높이기 위해 TDD(Test-Driven Development) 기반의 개발 방식을 도입했습니다:

- **TDD 기반 테스트 인프라 구축**: 테스트를 먼저 작성하고 코드를 구현하는 체계 확립
- **회귀 테스트 작성**: 기존 버그가 재발하지 않도록 자동 검증
- **피드 작성자 이름 표시 수정 (TDD 방식)**: 테스트 실패 → 코드 수정 → 테스트 통과 순서로 진행
- **개발 시드 데이터**: 테스트용 초기 데이터 자동 생성
- **프론트엔드 빌드 수정**: 테스트 파일이 프로덕션 빌드에 포함되지 않도록 분리

## 왜 필요했나요?

### 003에서 했던 테스트와 뭐가 다른가?

003에서는 "코드를 먼저 짜고, 테스트로 확인"하는 방식이었습니다. TDD는 정반대입니다:

```
기존 방식 (003):                TDD 방식 (006):
코드 작성 → 테스트 작성          테스트 작성 → 코드 작성
"이게 맞나?" 확인하는 테스트      "이렇게 동작해야 해" 정의하는 테스트
버그를 발견하는 도구              버그를 예방하는 도구
```

TDD의 핵심 사이클을 **Red-Green-Refactor**라고 합니다:

```
🔴 Red:    실패하는 테스트를 먼저 작성
🟢 Green:  테스트를 통과하는 최소한의 코드 작성
🔵 Refactor: 코드를 깔끔하게 정리 (테스트가 여전히 통과하는지 확인)
```

### 왜 TDD를 배워야 하는가?

**1. 요구사항을 코드로 표현**
"피드에 작성자 이름이 표시되어야 한다"는 요구사항을 테스트 코드로 먼저 작성합니다. 이렇게 하면 요구사항이 모호해질 수 없습니다.

**2. 자신감 있는 수정**
테스트가 있으면 코드를 수정한 후 "혹시 다른 곳이 깨지지 않았을까?" 걱정할 필요 없습니다. 테스트가 통과하면 안전합니다.

**3. 설계 개선**
테스트하기 어려운 코드는 대체로 설계가 나쁜 코드입니다. TDD를 하면 자연스럽게 테스트하기 쉬운, 즉 잘 설계된 코드를 작성하게 됩니다.

### 프론트엔드 빌드에서 테스트 파일을 왜 분리해야 하나?

```
문제 상황:
frontend/
├── src/
│   ├── pages/
│   │   ├── Dashboard.tsx         ← 프로덕션 코드
│   │   └── Dashboard.test.tsx    ← 테스트 코드 (프로덕션에 불필요!)
│   └── ...

테스트 파일이 프로덕션 빌드에 포함되면:
- 번들 크기 증가 (불필요한 코드)
- 테스트 라이브러리도 함께 번들됨
- 사용자가 다운로드해야 하는 파일이 커짐
- 빌드 자체가 실패할 수 있음 (테스트 의존성 누락)
```

## 어떻게 만들었나요?

### TDD 실습: 피드 작성자 이름 표시

실제로 TDD를 적용한 과정을 따라가 보겠습니다.

**버그 보고**: "피드에 글을 올리면 작성자 이름이 표시되지 않아요"

#### Step 1: 실패하는 테스트 작성 (Red)

```go
// tests/integration/feed_test.go

func TestFeedPostIncludesAuthorName(t *testing.T) {
    // 준비: 테스트 서버와 사용자 생성
    server := setupTestServer(t)
    defer server.Close()

    user := createAndLoginTestUser(t, server, "author@test.com", "테스트작성자")

    // 행동: 피드 게시글 작성
    post := createFeedPost(t, server, user.Token, "테스트 게시글입니다")

    // 검증: 게시글 조회 시 작성자 이름이 포함되어야 함
    feed := getFeed(t, server, user.Token)

    assert.Equal(t, "테스트작성자", feed[0].AuthorName,
        "피드 게시글에 작성자 이름이 표시되어야 합니다")
}
```

이 테스트를 실행하면 **실패**합니다 (Red):
```bash
$ go test ./tests/integration/ -run TestFeedPostIncludesAuthorName
--- FAIL: TestFeedPostIncludesAuthorName
    Expected: "테스트작성자"
    Actual:   ""
```

작성자 이름이 빈 문자열로 반환되고 있습니다. 이제 원인을 찾아 수정합니다.

#### Step 2: 코드 수정 (Green)

원인 분석: 피드 조회 API의 SQL 쿼리에서 작성자 이름을 JOIN하지 않고 있었습니다.

```go
// 수정 전
func GetFeed(c echo.Context) error {
    rows, _ := db.Query("SELECT * FROM feed_posts ORDER BY created_at DESC")
    // → 작성자 이름이 없음!
}

// 수정 후
func GetFeed(c echo.Context) error {
    rows, _ := db.Query(`
        SELECT fp.*, u.name as author_name
        FROM feed_posts fp
        JOIN users u ON fp.user_id = u.id
        ORDER BY fp.created_at DESC
    `)
    // → JOIN으로 작성자 이름을 함께 가져옴
}
```

다시 테스트 실행:
```bash
$ go test ./tests/integration/ -run TestFeedPostIncludesAuthorName
--- PASS: TestFeedPostIncludesAuthorName
PASS
```

통과! (Green)

#### Step 3: 리팩토링 (Refactor)

코드가 동작하는 것을 확인한 후, 더 깔끔하게 정리합니다. 이때 테스트를 다시 실행하여 여전히 통과하는지 확인합니다.

### 테스트 인프라 구축

테스트를 효율적으로 작성하기 위한 도구들을 만들었습니다:

#### 테스트 헬퍼 함수

```go
// tests/helpers.go - 테스트에서 반복적으로 쓰는 함수들

// 테스트 서버 시작 (메모리 DB 사용)
func setupTestServer(t *testing.T) *httptest.Server {
    // 매 테스트마다 깨끗한 DB로 시작
    db := setupTestDB()
    e := setupEcho(db)
    return httptest.NewServer(e)
}

// 테스트 사용자 생성 + 로그인
func createAndLoginTestUser(t *testing.T, server *httptest.Server,
    email, name string) *TestUser {
    // 1. 회원가입
    register(server, email, name, "password123")
    // 2. 관리자 승인
    approveUser(server, email)
    // 3. 로그인
    token := login(server, email, "password123")
    return &TestUser{Email: email, Name: name, Token: token}
}

// 피드 게시글 작성
func createFeedPost(t *testing.T, server *httptest.Server,
    token, content string) *FeedPost {
    resp := postJSON(server, "/api/feed", token,
        map[string]string{"content": content})
    assert.Equal(t, 201, resp.StatusCode)
    var post FeedPost
    json.NewDecoder(resp.Body).Decode(&post)
    return &post
}
```

이런 헬퍼 함수가 있으면 새 테스트를 작성할 때 매우 간편합니다:

```go
// 새 테스트 작성이 이렇게 간단해집니다
func TestSomething(t *testing.T) {
    server := setupTestServer(t)
    user := createAndLoginTestUser(t, server, "test@test.com", "테스트")
    post := createFeedPost(t, server, user.Token, "내용")
    // ... 검증
}
```

#### 개발 시드 데이터

개발할 때 빈 화면만 보면 테스트하기 어렵습니다. 시드 데이터는 자동으로 테스트용 데이터를 생성합니다:

```go
// 개발 환경에서만 실행되는 시드 데이터 생성
func SeedDevData(db *sql.DB) {
    // 관리자 계정
    createUser(db, "admin@test.com", "관리자", "admin")

    // 학생 계정 5명
    for i := 1; i <= 5; i++ {
        createUser(db, fmt.Sprintf("student%d@test.com", i),
            fmt.Sprintf("학생%d", i), "student")
    }

    // 회사 3개
    createCompany(db, "스타트업A", 1)
    createCompany(db, "테크벤처B", 2)
    createCompany(db, "크리에이티브C", 3)

    // 과제 2개
    createAssignment(db, "비즈니스 모델 캔버스 작성", "...")
    createAssignment(db, "프로토타입 제작", "...")
}
```

이렇게 하면 서버를 시작할 때 자동으로 데이터가 채워져서 바로 테스트할 수 있습니다.

### 프론트엔드 빌드 수정

테스트 파일을 프로덕션 빌드에서 제외하는 방법:

```typescript
// vite.config.ts
export default defineConfig({
  build: {
    rollupOptions: {
      external: [],
    },
  },
  // 테스트 파일을 빌드에서 제외
  resolve: {
    alias: {
      // ...
    },
  },
});

// tsconfig.json에서 빌드 대상 제외
{
  "exclude": [
    "**/*.test.ts",
    "**/*.test.tsx",
    "**/*.spec.ts",
    "**/*.spec.tsx",
    "src/test/**"
  ]
}
```

빌드 전후 비교:

```
수정 전: 번들 크기 1.8MB (테스트 코드 포함)
수정 후: 번들 크기 1.2MB (테스트 코드 제외)
→ 약 33% 감소!
```

사용자가 다운로드해야 하는 파일이 작아지면 페이지 로딩 속도가 빨라집니다.

## 사용한 프롬프트

### TDD 인프라 프롬프트
```
TDD 기반 테스트 인프라를 구축해줘.
테스트 헬퍼 함수 (서버 셋업, 유저 생성, 로그인 등)를 만들고,
기존 테스트를 이 인프라로 마이그레이션해줘.
회귀 테스트도 추가해줘.
```

### 피드 버그 수정 (TDD) 프롬프트
```
피드 게시글에 작성자 이름이 표시되지 않는 버그를 TDD로 수정해줘.
1. 먼저 실패하는 테스트 작성
2. 코드 수정으로 테스트 통과
3. 리팩토링
순서로 진행해줘.
```

### 빌드 수정 프롬프트
```
프론트엔드 빌드에서 테스트 파일(*.test.ts, *.test.tsx)이 포함되지 않도록 수정해줘.
tsconfig.json과 vite.config.ts 모두 확인해줘.
```

TDD 프롬프트의 핵심은 **"Red-Green-Refactor 순서로 진행해줘"**라고 명시하는 것입니다. 이렇게 하면 AI도 TDD 방식을 따릅니다.

## 배운 점

### 1. TDD는 사고방식이다
TDD는 단순한 기법이 아니라 **"무엇이 올바른 동작인지 먼저 정의하자"**는 사고방식입니다. 테스트를 먼저 작성하면 요구사항을 명확히 이해하게 됩니다.

### 2. Red-Green-Refactor 사이클
```
🔴 Red:     "이게 동작해야 해" (테스트 = 명세서)
🟢 Green:   "일단 동작하게 만들자" (최소한의 구현)
🔵 Refactor: "깔끔하게 다듬자" (품질 향상)
```
이 사이클을 빠르게 반복하면 높은 품질의 코드를 안정적으로 만들 수 있습니다.

### 3. 테스트 헬퍼는 생산성을 높인다
테스트를 작성하는 것 자체가 부담이 되면 안 됩니다. 반복되는 설정(서버 시작, 로그인 등)을 헬퍼 함수로 만들면 테스트 작성이 간편해지고, 더 많은 테스트를 작성하게 됩니다.

### 4. 시드 데이터로 개발 속도 향상
매번 회원가입 → 승인 → 로그인 → 데이터 입력을 수동으로 하는 대신, 시드 데이터로 자동화하면 개발 속도가 크게 빨라집니다.

### 5. 프로덕션 빌드는 최적화해야 한다
개발에 필요한 파일(테스트, 개발 도구)이 사용자에게 전달되면 안 됩니다. 번들 크기가 커지면 로딩이 느려지고, 사용자 경험이 나빠집니다.

### 6. 테스트는 문서다
잘 작성된 테스트 코드는 그 자체로 "이 기능이 어떻게 동작해야 하는가"를 설명하는 문서입니다. 새 팀원이 코드를 이해하는 가장 빠른 방법은 테스트 코드를 읽는 것입니다.

---

## GitHub 참고 링크
- [커밋 191d481: TDD 기반 테스트 인프라 구축 + 회귀 테스트 작성](https://github.com/cycorld/earnlearning/commit/191d481)
- [커밋 197518f: 피드 작성자 이름 표시 수정 (TDD) + 개발 시드 데이터](https://github.com/cycorld/earnlearning/commit/197518f)
