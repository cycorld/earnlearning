# 096. 학생 DB 고아 행 정리 admin API (#016)

**날짜**: 2026-04-19
**태그**: 운영, userdb, admin, 정합성

## 배경
학생은 LMS UI 에서 PG DB 생성/삭제 → SQLite + PG 양쪽 동기화. 그런데 운영자가
`sudo earnlearning-db delete` 로 PG 만 직접 지우면 SQLite 행 남음 → 학생 프로필
에 동작 안 하는 좀비 카드.

이전엔 `sqlite3 ... DELETE FROM user_databases WHERE id=...` 수동 정리.

## 추가
### Domain
- `userdb.Repository.FindByDBName(dbName)` — db_name 으로 조회
- `userdb.Repository.ListAll()` — 모든 메타 (admin reconcile 용)

### Provisioner
- `Provisioner.DBExists(dbName) (bool, error)` — pg_database 카탈로그 조회
- NoopProvisioner 는 항상 `true` (테스트가 reconcile 로 행 삭제하는 사고 방지)

### UseCase
- `AdminReconcile()` — 모든 SQLite 행 순회, PG 에 없으면 SQLite 행 삭제. 결과 리포트
- `AdminDeleteByDBName(dbName)` — PG + SQLite 양쪽 정리. PG 에 이미 없으면 SQLite 만

### HTTP
- `POST /api/admin/user-databases/reconcile` — 일괄 정합성 검사
- `DELETE /api/admin/user-databases/by-dbname/:db_name` — 특정 DB 정리

### Test
- `userdb_admin_test.go` 4 케이스

### Doc
- `docs/POSTGRES_SETUP.md` — 운영자 가이드 추가

## 미포함
- CLI 통합 (earnlearning-db delete 가 admin API 자동 호출) — admin 토큰 관리 부담
- 자동 cron — DDL 가까운 작업, 위험 가능성 있어 수동만

## 운영 가이드
```bash
ADMIN_TOKEN='Bearer ...'
# 특정 DB 정리
curl -X DELETE -H "Authorization: $ADMIN_TOKEN" \
  https://earnlearning.com/api/admin/user-databases/by-dbname/seowon_todoapp
# 일괄 reconcile
curl -X POST -H "Authorization: $ADMIN_TOKEN" \
  https://earnlearning.com/api/admin/user-databases/reconcile
```
