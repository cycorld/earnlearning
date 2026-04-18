---
slug: 061-disclosure-status-hardcoded
title: 공시 INSERT SQL의 status 하드코딩 제거 — 도메인 Status 필드 살리기
date: 2026-04-18
tags: [fix, 공시, 청산, SQL, TDD, 회귀테스트]
---

# 공시 INSERT SQL의 status 하드코딩 제거

## 왜 필요했는가

지난 PR #70 (청산 기능, #033) 에서 회사가 청산되면 **청산 내역을 자동으로 공시로 남기고, 그 공시는 심사 없이 바로 `approved` 상태로 표시**되도록 설계했습니다. 코드에도 이렇게 썼죠:

```go
// proposal_usecase.go:618
_, _ = uc.companyRepo.CreateDisclosure(&company.Disclosure{
    CompanyID: c.ID,
    Content:   disclosureContent,
    Status:    "approved",  // ← 자동 승인
})
```

그런데 스테이지에서 실제로 청산해봤더니, 자동 생성된 공시가 **"심사 중(pending)"** 으로 표시되었습니다. "왜 자동승인 안 되지?" 하고 코드를 따라가 봤더니, 레포지토리에 버그가 있었습니다.

## 무엇을 했는가

### 진짜 원인

`backend/internal/infrastructure/persistence/company_repo.go` 의 `CreateDisclosure` SQL이 `status` 컬럼을 **문자열 리터럴로 하드코딩**하고 있었습니다:

```go
// Before (버그)
INSERT INTO company_disclosures (..., status)
VALUES (?, ?, ?, ?, ?, 'pending')
//                       ^^^^^^^^^ 하드코딩
```

호출자가 `Disclosure.Status`에 뭐를 넣든 SQL이 무시하고 `'pending'`으로 덮어쓰고 있었어요. 청산 자동 공시뿐 아니라 **도메인 모델 전체에서 Status 필드가 사실상 작동하지 않는 상태**였던 셈.

### 고친 방법

SQL을 파라미터화해서 호출자 값을 쓰되, 빈 문자열이면 기존 기본값 `pending`을 유지:

```go
// After
func (r *CompanyRepo) CreateDisclosure(d *company.Disclosure) (int, error) {
    status := d.Status
    if status == "" {
        status = "pending"
    }
    res, err := r.db.Exec(`
        INSERT INTO company_disclosures (..., status)
        VALUES (?, ?, ?, ?, ?, ?)`,
        d.CompanyID, d.AuthorID, d.Content, d.PeriodFrom, d.PeriodTo, status,
    )
    // ...
}
```

기존 호출처인 `CompanyUsecase.CreateDisclosure` (사용자가 직접 공시 작성) 는 이미 `Status: "pending"` 을 명시적으로 넣고 있어서 동작 변화 없음. 청산 자동 공시만 의도대로 `approved` 로 저장됩니다.

### TDD로 방어

수정 전에 **실패하는 회귀 테스트를 먼저** 작성했습니다(Red → Green 순서):

```go
// #024 회귀: 청산 자동 공시가 approved 상태로 저장되는지
func TestLiquidation_AutoDisclosure_SavedAsApproved(t *testing.T) {
    // ... 청산 실행 후
    // admin이 전체 공시 조회 → cid에 해당하는 자동 공시가 approved 여야 함
    if found.Status != "approved" {
        t.Errorf("expected approved, got %q", found.Status)
    }
}

// #024 회귀: 일반 사용자 공시는 여전히 pending 기본값 유지
func TestDisclosure_UserCreated_DefaultsToPending(t *testing.T) {
    // POST /api/companies/:id/disclosures 로 작성 → status == "pending"
}
```

- 수정 전: 첫 번째 테스트 **FAIL**, 두 번째 **PASS** (버그 재현 + 기본값은 정상)
- 수정 후: **둘 다 PASS**

이 두 테스트가 레포에 남아 있어서 앞으로 누군가 또 SQL을 건드리다가 하드코딩으로 되돌려도 CI가 잡아줍니다.

## 사용한 프롬프트

```
백로그 중에 현재 반영 필요한거 하나씩 적용해줘.
```

AI가 backlog 5건 중 #024·#025 (둘 다 이전 청산 PR에서 파생된 버그 픽스) 를 우선순위로 골라 하나씩 처리하기로 판단했고, #024부터 TDD 순서(Red → Green → Refactor) 로 진행했습니다.

## 배운 점

- **ORM 없이 raw SQL 을 쓰면 리터럴 하드코딩이 조용히 "작동처럼 보이는" 버그를 만들 수 있다.** 엔티티 필드를 추가하거나 기본값을 바꿀 때 SQL 쪽을 반드시 같이 봐야 합니다.
- **도메인 필드와 SQL 컬럼의 1:1 매핑을 깨는 "편의 하드코딩"은 기술부채 원인 1순위.** 기본값이 필요하면 DB 스키마의 `DEFAULT` 에 맡기거나, 레포지토리에서 명시적으로 파라미터화하는 게 맞습니다.
- **PR 스테이지 검증에서 발견된 버그는 반드시 회귀 테스트로 남긴다.** 이번에도 스테이지에서만 잡혔던 이슈였는데, 테스트를 남겨 앞으로는 CI에서 잡도록 했습니다.

## 관련 티켓

- #024 (backlog → done) — 이 PR
- #025 — 함께 발견된 공시 기간 ISO 표시 버그 (다음 PR)
- #023 / #033 — 청산 기능 원본 PR들 (버그가 심어진 곳)
