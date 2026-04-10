# 043. 기업 정보 수정 (이름/이미지) + 다른 학생 기업 둘러보기

> **날짜**: 2026-04-10
> **태그**: `feat`, `fix`, `회사`, `백엔드`, `프론트엔드`

## 무엇을 했나요?

회사(기업) 메뉴 두 가지 개선:

1. **회사 정보 수정**: 이름/설명/로고를 모두 편집 가능. (전엔 이름 변경이 무시되고, 로고 편집 UI 가 없었어요.)
2. **다른 학생 회사 둘러보기**: 회사 메뉴에 "내 회사 / 전체 기업" 탭이 생겨서 다른 학생들이 만든 회사도 카드로 볼 수 있어요. 카드 클릭하면 그 회사 상세로 이동.

## 왜 필요했나요?

### 이름 변경 버그
프론트는 `name` 을 `PUT /companies/:id` 로 보내고 있었는데 백엔드 핸들러는
`description` 과 `logo_url` 만 받고 있었어요. 그래서 학생이 이름을 바꾸려고 해도
조용히 무시됐고, 화면도 그대로였어요. 사용자 신뢰가 깨지는 종류의 버그.

### 로고 편집 UI 부재
회사 설립 시에는 로고 업로드 UI 가 있었는데, 설립 후 detail page 의 편집 다이얼로그에는
로고 필드가 아예 없었어요. 한 번 잘못 올리면 영원히 못 바꿈.

### 다른 학생 회사 못 봄
회사 메뉴는 `/companies/mine` 만 호출 → 본인 회사만 노출. 다른 학생이 어떤 회사를
만들었는지 알 수 없으니 투자, 명함 교환, 협업 같은 학습 활동 자체가 시작되기 어려웠어요.
"전체 기업 목록" 엔드포인트는 admin 전용 (`/admin/companies`) 만 있었어요.

## 어떻게 만들었나요?

### 백엔드

#### 1. UpdateCompany 가 name 을 받도록

**Before** (`application/company_usecase.go`):
```go
func (uc *CompanyUsecase) UpdateCompany(companyID, userID int, description, logoURL string) error {
    c, _ := uc.companyRepo.FindByID(companyID)
    c.Description = description
    c.LogoURL = logoURL
    return uc.companyRepo.Update(c)
}
```

**After**:
```go
type UpdateCompanyInput struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    LogoURL     string `json:"logo_url"`
}

func (uc *CompanyUsecase) UpdateCompany(companyID, userID int, input UpdateCompanyInput) error {
    c, _ := uc.companyRepo.FindByID(companyID)
    if c.OwnerID != userID { return company.ErrNotOwner }
    if input.Name != "" {  // 빈 문자열은 "변경 안 함" 의미 (구 클라이언트 호환)
        c.Name = input.Name
    }
    c.Description = input.Description
    c.LogoURL = input.LogoURL
    return uc.companyRepo.Update(c)
}
```

#### 2. Repo 의 UPDATE 쿼리에 name 추가 + UNIQUE 충돌 매핑

```go
func (r *CompanyRepo) Update(c *company.Company) error {
    _, err := r.db.Exec(`
        UPDATE companies SET name = ?, description = ?, logo_url = ?, business_card = ?
        WHERE id = ?`,
        c.Name, c.Description, c.LogoURL, c.BusinessCard, c.ID,
    )
    if err != nil {
        // UNIQUE constraint failed: companies.name → ErrDuplicateName
        if strings.Contains(err.Error(), "UNIQUE constraint failed: companies.name") {
            return company.ErrDuplicateName
        }
        return fmt.Errorf("update company: %w", err)
    }
    return nil
}
```

같은 매핑을 `Create` 에도 적용 — 신규 가입자가 다른 사람과 같은 회사명으로
설립 시도해도 친절한 에러 받음.

#### 3. 핸들러: ErrDuplicateName → 409 Conflict

```go
if err == company.ErrDuplicateName {
    return c.JSON(http.StatusConflict, errorResp("DUPLICATE_NAME", err.Error()))
}
```

#### 4. 학생용 전체 기업 목록 API

**Usecase**:
```go
type PublicCompanyItem struct {
    *company.Company
    OwnerName     string `json:"owner_name"`
    OwnerStudent  string `json:"owner_student_id"`
    WalletBalance int    `json:"wallet_balance"`
}

func (uc *CompanyUsecase) GetAllCompaniesWithOwners() ([]*PublicCompanyItem, error) {
    companies, _ := uc.companyRepo.FindAll()
    items := make([]*PublicCompanyItem, 0, len(companies))
    for _, c := range companies {
        item := &PublicCompanyItem{Company: c}
        if owner, err := uc.userRepo.FindByID(c.OwnerID); err == nil {
            item.OwnerName = owner.Name
            item.OwnerStudent = owner.StudentID
        }
        if cw, err := uc.companyRepo.FindCompanyWallet(c.ID); err == nil {
            item.WalletBalance = cw.Balance
        }
        items = append(items, item)
    }
    return items, nil
}
```

**Handler + Router**:
```go
// GET /api/companies — 학생 누구나 (read:company OAuth scope)
approved.GET("/companies", h.Company.ListCompaniesPublic, middleware.RequireScope("read:company"))
```

기존 `/admin/companies` 는 그대로 두고 새 endpoint 를 추가했어요. 응답 구조가
다르고 (admin 은 raw company, 학생용은 owner 정보 포함), admin 권한 분리도 유지.

### 프론트엔드

#### CompanyDetailPage — 편집 다이얼로그에 로고 업로드 추가

`CompanyNewPage` 의 로고 업로드 패턴을 그대로 가져왔어요:
- 미리보기 박스 (16x16) + "이미지 선택" 버튼 + "제거" 버튼
- 업로드 중에는 spinner + 저장 버튼 disabled
- `POST /upload` → `{url}` 받아서 form 에 set
- 폼 제출 시 `logo_url` 도 함께 PUT

또 PUT 응답이 `{message: "..."}` 라서 `setCompany(updated)` 로는 화면이 갱신되지
않던 문제가 있었어요 → `await fetchCompany()` 로 다시 조회하도록 변경.

#### CompanyListPage — 탭 구조

```tsx
<Tabs defaultValue="mine">
  <TabsList>
    <TabsTrigger value="mine">내 회사 ({myCompanies.length})</TabsTrigger>
    <TabsTrigger value="all">전체 기업 ({otherCompanies.length})</TabsTrigger>
  </TabsList>
  <TabsContent value="mine"><CompanyGrid companies={myCompanies} /></TabsContent>
  <TabsContent value="all"><CompanyGrid companies={otherCompanies} showOwner /></TabsContent>
</Tabs>
```

- "전체 기업" 탭에는 본인 회사 제외하고 표시 (`otherCompanies = allCompanies.filter(c => c.owner_id !== user.id)`)
- 각 카드에 `대표 {owner_name} · 기업가치 {valuation}` 형태로 소유자 노출
- 카드 클릭 시 기존 detail page (`/company/:id`) 로 이동
- 두 API (`/companies/mine`, `/companies`) 를 `Promise.all` 로 병렬 호출

#### Type 추가
```ts
export interface Company {
  // ...기존 필드
  owner_id?: number
  owner_name?: string
  owner_student_id?: string
}
```

## TDD

회귀 테스트 6개 작성 (`backend/tests/integration/company_edit_test.go`):

| 테스트 | 검증 |
|---|---|
| `TestCompanyUpdate_NameChange_Success` | 이름 변경 후 다시 조회 시 새 이름 |
| `TestCompanyUpdate_LogoURLChange_Success` | 로고 URL 변경 후 GET 응답에 반영 |
| `TestCompanyUpdate_DuplicateName_Conflict` | 다른 회사 이름으로 변경 → 409 DUPLICATE_NAME |
| `TestCompanyUpdate_NotOwner_Forbidden` | 남의 회사 수정 시도 → 403 NOT_OWNER |
| `TestListAllCompanies_ReturnsAll` | 학생이 전체 목록 호출 시 본인+타인 모두 + owner_name 포함 |
| `TestListAllCompanies_NoAuth` | 토큰 없이 호출 → 401 |

7개 모두 통과 + 기존 통합 테스트 회귀 0건 (23초 내).

## 배운 점

### 1. 프론트가 보내는 필드와 백엔드가 받는 필드의 불일치는 조용한 버그
TypeScript / Go 양쪽 다 strict typing 인데도, 네트워크 경계에서는 어쨌든 JSON 이라
프론트가 `name` 을 보내도 백엔드 struct 가 안 받으면 그냥 무시되고 200 OK 가
떨어져요. **API 문서/swagger 최신화** + **e2e/integration test 로 round-trip 검증** 이
유일한 방어선.

### 2. SQLite UNIQUE 에러 매핑은 문자열 매칭
SQLite 의 lib(`mattn/go-sqlite3`) 는 PG `pq.Error.Code` 같은 SQLSTATE 를 노출 안 해요.
대신 에러 메시지에 `"UNIQUE constraint failed: <table>.<col>"` 가 포함돼요.
`strings.Contains` 로 검사하는 게 표준 패턴.

### 3. PUT 응답을 갱신용으로 쓰려면 핸들러도 객체를 돌려줘야 함
처음엔 핸들러가 `{message: ...}` 를 리턴해서 프론트의 `setCompany(updated)` 가 깨졌어요.
가장 깔끔한 수정은 핸들러도 업데이트된 회사를 응답에 넣는 것이지만, 일단 작은
범위로 막기 위해 프론트에서 `await fetchCompany()` 로 처리. 추후 핸들러 응답을
표준화할 때 정리할 것.

### 4. LSP 셋업하면 검증 빨라짐
이 작업 도중에 gopls + vtsls LSP 를 설정해서 `findReferences` / `documentSymbol` 로
시그니처 변경 시 호출자 추적이 한 번에 가능해졌어요. grep 으로 하던 것보다 훨씬 정확.

## 사용한 AI 프롬프트

```
수정사항:
1. 기업 정보 수정 가능하도록 (이름 변경시 에러 발생) + 이미지 수정 기능 추가
2. 학생들이 다른 학생들의 기업들도 볼 수 있도록 해줘 (회사 메뉴에서)
테스트 모두 작성 후 통과하면 스테이지 배포하고, 브라우저로 UI 검증도 모두 끝내줘.
```
