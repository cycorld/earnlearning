---
id: 016
title: earnlearning-db CLI 삭제 시 LMS user_databases 고아 행 정리
priority: low
type: fix
branch: fix/userdb-cli-orphan-rows
created: 2026-04-10
---

## 배경

학생은 LMS 프로필 UI 에서 PG DB 를 만들고 (`POST /users/me/databases`),
그러면 두 곳에 데이터가 생긴다:

1. **PG 서버**: 실제 ROLE / DATABASE / 권한 (provisioner 가 생성)
2. **LMS SQLite**: `user_databases` 테이블에 메타데이터 1행

LMS UI 에서 삭제하면 (`DELETE /users/me/databases/:id`) 두 곳 다 정리되지만,
**운영자가 서버에서 직접** `sudo earnlearning-db delete <user> <proj>` 를
실행하면 PG 만 정리되고 SQLite 행이 남는다.

결과: 학생이 프로필을 다시 열면 카드가 보이지만 클릭하면 동작 안 함
(eye/rotate/delete 모두 PG 측 ROLE 이 없어서 실패).

실제로 #015 작업 중 stage 에서 발견. 수동으로
`sqlite3 ... DELETE FROM user_databases WHERE id=...` 로 정리.

## 해결 방향 (옵션)

### A. CLI 가 LMS DB 도 직접 정리
- `earnlearning-db delete` 가 SQLite 파일도 열어서 행 삭제
- 단점: CLI 가 LMS 내부 구조에 의존, 도커 볼륨 경로 하드코딩 필요
- 단점: stage / blue / green 어느 LMS DB 에 학생 데이터가 있는지 모름

### B. CLI 가 LMS API 호출
- 운영자용 admin 토큰 발급 → `DELETE /admin/user-databases/...` 호출
- 새 admin API 추가 필요
- 깔끔하지만 작업량 좀 있음

### C. LMS 가 시작 시 / 주기적 reconcile
- 백엔드가 부팅 시 또는 cron 으로 SQLite ↔ PG 정합성 검사
- 고아 SQLite 행 발견 → 자동 삭제 (또는 로그)
- 단점: 실시간성 부족, 학생이 카드 클릭 후에야 인지

### D. UI 가 lazy reconcile
- `GET /users/me/databases` 시 각 행을 PG 에 ping → 없으면 응답에서 제외 + SQLite 행 삭제
- 단점: 매 조회마다 PG round trip 늘어남

### E. 최소 변경: 운영 매뉴얼 변경
- `earnlearning-db delete` 사용 금지, 대신 admin 이 사용자 impersonate 후 UI 에서 삭제
- 또는 admin 이 직접 SQLite + PG 양쪽 SQL 실행
- 가장 빠르고 간단하지만 사용자 부주의 시 동일 문제 재발

## 권장

**B (admin API) + A (CLI 통합)** 조합:
1. 백엔드에 `DELETE /api/admin/user-databases/:db_name` 추가
   - admin only
   - LMS SQLite 행 + PG 양쪽 정리 (provisioner 재사용)
2. `earnlearning-db delete` 가 새 admin API 를 호출 (admin 토큰은 secrets 파일 또는 env)
3. 또는 CLI 는 그대로 두고 별도 `earnlearning-db sync` 명령으로 reconcile 만 하게 함

## 작업 (구현 시)

- [ ] backend: `DELETE /api/admin/user-databases/:db_name` 핸들러 + 라우터
- [ ] backend: usecase 메서드 `AdminDeleteByDBName(dbName string)` 추가
- [ ] backend: provisioner.Delete + repo.Delete 양쪽 호출
- [ ] CLI 통합: `earnlearning-db delete` 가 admin API 호출
- [ ] (선택) `earnlearning-db sync` 로 SQLite ↔ PG 정합성 검사 + 자동 정리
- [ ] 회귀 테스트
- [ ] 운영 매뉴얼 docs/POSTGRES_SETUP.md 업데이트

## 비-목표

- LMS 가 PG 의 single source of truth 이라는 가정을 깨지 말 것
- 학생 권한 변경 (admin 만 이 API 사용)
