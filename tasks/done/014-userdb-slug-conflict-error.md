---
id: 014
title: UserDB 슬러그 충돌 에러 메시지 개선
priority: low
type: fix
branch: fix/userdb-slug-conflict-error
created: 2026-04-10
---

## 배경

#013 에서 학생 이메일 local-part 기반으로 PG 유저명 slug 를 만들었는데,
두 사용자가 서로 다른 도메인에서 같은 local-part 를 갖는 경우 (예:
`cyc@ewha.ac.kr` 와 `cyc@example.com`) PG 에서 `CREATE ROLE cyc_todoapp`
이 "role already exists" 에러로 실패한다.

현재는 이 에러가 `ErrProvisionFailed` 로 매핑돼서 HTTP 502 Bad Gateway 와
`pq: role "..." already exists` 같은 raw PG 에러 메시지가 노출된다.

## 목표

- PG 의 duplicate_object (42710) / duplicate_database (42P04) 에러를 감지
- `userdb.ErrSlugConflict` 도메인 에러로 매핑
- 핸들러에서 HTTP 409 Conflict + 사용자 친화 메시지 반환
  - "이 프로젝트 이름이 이미 사용 중입니다. 다른 이름을 시도해주세요."

## 작업

- [ ] `domain/userdb/errors.go`: `ErrSlugConflict` 추가
- [ ] `infrastructure/userdbadmin/provisioner.go`:
  - lib/pq 의 `*pq.Error` 타입 체크
  - Code `42710` (duplicate_object) / `42P04` (duplicate_database) 매핑
  - CREATE ROLE / CREATE DATABASE 각 단계에서 특정 에러 반환
- [ ] `application/userdb_usecase.go`: 에러 pass-through (이미 됨)
- [ ] `interfaces/http/handler/userdb_handler.go`: `ErrSlugConflict` → 409
- [ ] 회귀 테스트 추가 (Noop 으로는 테스트 불가 → provisioner 단위 테스트)

## 비-목표

- username 수집 UI 추가 (옵션 D) — 이번 작업 아님
- student_id 기반 slug (옵션 B) — 이번 작업 아님
- 슬러그 자동 고유화 (`_2` 접미사 등) — 이번 작업 아님

## 완료 기준

- 같은 slug 를 가진 두 사용자가 같은 project_name 을 만들려 하면:
  - HTTP 409
  - `{code: "SLUG_CONFLICT", message: "..."}`
  - `pq:` 로 시작하는 raw 에러 노출 없음
