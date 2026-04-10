# 041. 학생 DB 슬러그 충돌 에러 메시지 개선

> **날짜**: 2026-04-10
> **태그**: `fix`, `PostgreSQL`, `학생DB`, `UX`

## 무엇을 했나요?

#013 에서 학생 이메일 local-part 를 PG 사용자명 slug 로 쓰게 만들었는데
(`cyc@ewha.ac.kr` → `cyc`), 서로 다른 도메인에서 같은 local-part 를 가진
두 사용자가 같은 프로젝트명을 시도할 경우 PG 에서 "role already exists"
에러가 나면서 사용자에게는 502 Bad Gateway + `pq: role "..." already exists`
같은 raw 에러가 노출됐어요.

이걸 **HTTP 409 Conflict + "이 프로젝트 이름이 다른 사용자와 충돌합니다"**
로 매핑했어요.

## 왜 필요했나요?

이화여대 단일 도메인 내에서는 거의 발생하지 않지만:
- 조교/관리자 계정이 다른 도메인을 쓸 수 있음
- 미래에 외부 사용자(다른 학과, 외부 연사)가 추가될 여지
- 그리고 **raw PG 에러가 사용자에게 노출되는 것 자체가 보기 안 좋음**
  (내부 구조 노출 + 한국어 메시지 불일치)

#039 보안 평가에서 "raw error leakage" 는 언급하지 않았지만, 이 케이스는
502 로 분류되므로 운영자가 알림을 받는 등 오탐의 원인이 될 수도 있어요.

## 어떻게 만들었나요?

### 1. 도메인 에러 추가

`backend/internal/domain/userdb/errors.go`:
```go
ErrSlugConflict = errors.New("이 프로젝트 이름이 다른 사용자와 충돌합니다. 다른 이름을 시도해주세요")
```

### 2. PG 에러 코드 감지

`backend/internal/infrastructure/userdbadmin/provisioner.go`:
```go
func isDuplicateObjectError(err error) bool {
    var pqErr *pq.Error
    if errors.As(err, &pqErr) {
        return pqErr.Code == "42710"  // duplicate_object
    }
    return false
}

func isDuplicateDatabaseError(err error) bool {
    var pqErr *pq.Error
    if errors.As(err, &pqErr) {
        return pqErr.Code == "42P04"  // duplicate_database
    }
    return false
}
```

PostgreSQL 의 **SQLSTATE 코드** 는 표준화되어 있어요:
- `42710` = `duplicate_object` (CREATE ROLE 이 이미 존재)
- `42P04` = `duplicate_database` (CREATE DATABASE 가 이미 존재)

lib/pq 드라이버는 `*pq.Error` 를 반환하고 그 `Code` 필드에 SQLSTATE 가 들어 있어요.

### 3. Provisioner 에서 감지 + 매핑

`Create()` 안에서 두 단계 모두 에러 검사:
```go
// 1. CREATE ROLE
if err != nil {
    if isDuplicateObjectError(err) {
        return nil, userdb.ErrSlugConflict  // ← 직접 도메인 에러 반환
    }
    return nil, fmt.Errorf("create role: %w", err)
}

// 2. CREATE DATABASE (with ROLE rollback)
if err != nil {
    _, _ = p.db.Exec(fmt.Sprintf(`DROP ROLE IF EXISTS %s`, qUser))
    if isDuplicateDatabaseError(err) {
        return nil, userdb.ErrSlugConflict
    }
    return nil, fmt.Errorf("create database: %w", err)
}
```

### 4. Usecase 에서 pass-through

기존 usecase 는 모든 provisioner 에러를 `ErrProvisionFailed` 로 감쌌어요:
```go
if err != nil {
    return nil, fmt.Errorf("%w: %v", userdb.ErrProvisionFailed, err)
}
```

도메인 에러는 그대로 전파되도록 수정:
```go
if err != nil {
    if errors.Is(err, userdb.ErrSlugConflict) {
        return nil, err  // pass through
    }
    return nil, fmt.Errorf("%w: %v", userdb.ErrProvisionFailed, err)
}
```

### 5. 핸들러에서 HTTP 409 매핑

```go
case errors.Is(err, userdb.ErrSlugConflict):
    return c.JSON(http.StatusConflict, errorResp("SLUG_CONFLICT", err.Error()))
```

## 어떻게 테스트했나요?

### 회귀 테스트 (Noop)
기존 integration 테스트는 `NoopProvisioner` 를 써서 PG 에러 경로를 테스트할 수
없어요. 해당 경로는 E2E 로만 검증 가능.

### E2E (실제 PG)
SSH 터널로 EC2 PG 의 5432 에 붙어서:
1. 수동으로 `CREATE ROLE admin_conflicttest` 실행 (충돌 상황 시뮬레이션)
2. provisioner 로 `Create("admin", "conflicttest")` 호출
3. 반환된 에러가 `userdb.ErrSlugConflict` 인지 확인

결과:
```
=== RUN   TestPGProvisioner_SlugConflict
    conflict_e2e_test.go:33: got expected error: 이 프로젝트 이름이 다른 사용자와 충돌합니다. 다른 이름을 시도해주세요
--- PASS: TestPGProvisioner_SlugConflict (0.05s)
```

실제로 PG 가 42710 을 반환하고, 그게 `ErrSlugConflict` 로 잘 매핑되는 것 확인.

## 배운 점

### 1. PostgreSQL SQLSTATE 는 표준이라 믿을 수 있어
버전/플랫폼 무관하게 동일한 코드 → 드라이버 레벨에서 robust 한 에러 분류 가능.
문자열 매칭 ("already exists" 같은) 은 PG 에러 메시지가 번역되면 깨지지만,
SQLSTATE 는 숫자+영문 고정.

### 2. Clean Architecture 의 힘
PG 에러 → 도메인 에러 → HTTP 상태 변환이 각 레이어에서 한 줄씩만 바뀌어요.
변경 범위가 작고 테스트하기 쉬움.

### 3. raw 에러 누출은 버그
502 Bad Gateway + `pq:` 메시지는 **운영상 문제** (모니터링 오탐) +
**UX 문제** (한국어 일관성 붕괴) + **보안 약점** (내부 구조 노출).
도메인 에러로 매핑하는 것이 정석.

## 부작용 / 비-목표

이 작업으로 다음은 **해결되지 않아요**:
- 충돌이 발생하면 학생은 여전히 **다른 프로젝트명** 을 생각해야 함
  (자동 `_2`, `_3` 접미사 안 붙임)
- Username 수집 UI (옵션 D) — 충돌 자체를 줄이는 근본 해결은 추후 고민
- `student_id` 기반 slug (옵션 B) — 익명성 강화 방향

## 사용한 AI 프롬프트

```
응 너말대로 실패 에러 보여주기만 하자. 그리고 스테이지에서 어디서 디비 생성 가능해?
```
