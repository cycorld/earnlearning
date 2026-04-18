---
id: 024
title: CreateDisclosure가 status='pending'을 하드코딩하여 도메인 Status 필드 무시
priority: medium
type: fix
branch: fix/disclosure-status-hardcoded
created: 2026-04-13
---

## 문제

`CompanyRepo.CreateDisclosure`가 INSERT SQL에서 `status`를 `'pending'`으로 하드코딩하고 있어, 호출자가 `Disclosure.Status`에 다른 값을 넣어도 무시된다.

## 재현 / 발견 경위

#023 회사 청산 기능(`ExecuteLiquidation`) 구현 시, 청산 완료 후 자동 생성되는 공시(청산 내역)를 `Status: "approved"`로 넣어 자동 승인 상태로 표시하려 했다. 하지만 스테이지 검증 결과 공시가 "심사 중"(pending)으로 떠서 관리자가 수동 승인해야 하는 상황.

## 원인 코드

`backend/internal/infrastructure/persistence/company_repo.go:292-297`

```go
func (r *CompanyRepo) CreateDisclosure(d *company.Disclosure) (int, error) {
    res, err := r.db.Exec(`
        INSERT INTO company_disclosures (company_id, author_id, content, period_from, period_to, status)
        VALUES (?, ?, ?, ?, ?, 'pending')`,  // ⬅ 'pending' 하드코딩
        d.CompanyID, d.AuthorID, d.Content, d.PeriodFrom, d.PeriodTo,
    )
```

`d.Status`가 완전히 무시되고 있다.

## 해결 방안

1. SQL을 파라미터화하여 `d.Status` 사용:
   ```go
   status := d.Status
   if status == "" {
       status = "pending"  // 기본값 유지
   }
   res, err := r.db.Exec(`
       INSERT INTO company_disclosures (company_id, author_id, content, period_from, period_to, status)
       VALUES (?, ?, ?, ?, ?, ?)`,
       d.CompanyID, d.AuthorID, d.Content, d.PeriodFrom, d.PeriodTo, status,
   )
   ```

2. 기본값이 `'pending'`으로 유지되도록 빈 문자열은 pending으로 변환 → 기존 `CreateDisclosure` 호출처(owner가 직접 공시 작성하는 플로우)는 영향 없음.

3. `ExecuteLiquidation`에서 이미 `Status: "approved"`로 넣고 있으므로 별도 변경 불필요.

4. 회귀 테스트:
   - 일반 공시 작성 → pending으로 남는지
   - 청산 후 자동 공시 → approved로 저장되는지

## 영향도

- **기능**: 큼 — 도메인 모델 Status 필드가 작동하지 않는 버그
- **사용자 체감**: 낮음 — 청산 공시가 "심사 중"으로 뜨는 것 외엔 일상적 영향 없음
- **작업 범위**: 작음 — 1 line SQL 변경 + 1 회귀 테스트
