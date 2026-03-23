# 개발 규칙

## 테스트 규칙
- **TDD 방식 필수**: 버그 수정 및 새 기능 개발 시 반드시 TDD로 진행한다.
  1. 실패하는 테스트를 먼저 작성한다 (Red)
  2. 테스트를 통과시키는 최소한의 코드를 작성한다 (Green)
  3. 필요 시 리팩토링한다 (Refactor)
- **회귀 테스트 필수**: 버그 수정 시 반드시 해당 버그를 재현하는 회귀 테스트를 남겨 재발을 방지한다. 회귀 테스트는 삭제하지 않고 계속 축적한다.
- **스모크 테스트 필수**: 커밋 또는 다른 테스트 실행 전에 반드시 스모크 테스트를 통과해야 한다.
  ```bash
  cd backend && go test ./tests/integration/ -run TestSmoke -timeout 60s
  ```
- 스모크 테스트 실패 시 커밋하지 않고 원인을 먼저 수정한다.
- **Backend 테스트**: `go test ./tests/integration/ -timeout 60s`
- **Frontend 테스트**: `cd frontend && npm test`

## 개발 워크플로우 (PR 기반)
> 상세 브랜치 전략은 Claude memory `feedback_pr_workflow.md` 참조

- **main 직접 푸시 금지**: 모든 개발은 feature 브랜치에서 진행한다.
- **PR 생성 필수**: 작업 완료 후 PR을 생성하고 사용자가 리뷰 후 머지한다.
- **브랜치 네이밍**: `feat/기능명`, `fix/버그명`, `chore/작업명` 형식 사용.
- **배포**: 로컬에서 `./deploy-remote.sh` (빌드→GHCR→Stage) → 확인 → `./deploy-remote.sh promote` (Prod blue-green).
- **서버에서 빌드 금지**: t3.small 리소스 고갈 방지. 반드시 로컬에서 빌드 후 이미지를 push한다.

## 개발일지 (Changelog)
- **PR 생성 시 필수**: 모든 PR에 대해 `changelog/`에 교육용 개발일지 엔트리를 추가한다.
- **파일**: `changelog/NNN-slug.md` (기존 파일 다음 순번)
- **내용**: 무엇을 했는지, 왜 필요했는지, 어떻게 만들었는지, 사용한 프롬프트, 배운 점
- **학생 대상**: 친절한 교재처럼 작성. 기술 용어는 설명 포함.
- **index.json 업데이트**: 새 엔트리 추가 시 `changelog/index.json`에도 항목 추가

## DB 마이그레이션 규칙 (프로덕션 안전)
- **001_init.sql 수정 금지**: 이미 배포된 init 마이그레이션은 절대 수정하지 않는다.
- **ALTER TABLE 사용 필수**: 새 컬럼 추가는 반드시 `sqlite.go`의 `RunMigrations()` 내 `alterStatements` 배열에 `ALTER TABLE ... ADD COLUMN` 문을 추가한다.
- **DEFAULT 값 필수**: 새 컬럼에는 반드시 DEFAULT 값을 지정하여 기존 데이터와 호환되게 한다.
- **DROP/DELETE 절대 금지**: 테이블 삭제, 컬럼 삭제, 데이터 삭제 절대 금지. 프로덕션에 실제 학생/교수 데이터가 있다.
- **에러 무시 패턴**: `db.Exec(stmt)` — SQLite에서 "duplicate column" 에러를 무시하여 재실행에도 안전하게 동작한다.

## 알림 연동 체크리스트
새 기능에서 알림(`CreateNotification`)을 추가할 때 반드시 아래 두 곳을 함께 업데이트한다:
1. **프론트엔드 `getReferencePath()`** (`frontend/src/routes/notifications/NotificationsPage.tsx`) — `reference_type` → URL 매핑 추가
2. **프론트엔드 `getNotifIcon()`** (같은 파일) — `notif_type` → 아이콘 매핑 추가

현재 등록된 reference_type 매핑:
- `post`, `posts`, `assignment`, `submission` → `/feed`
- `company` → `/company/:id`
- `investment` → `/invest/:id`, `dividend` → `/invest`
- `transaction`, `wallet`, `admin_transfer` → `/wallet`
- `loan` → `/bank`
- `job`, `freelance_job` → `/market/:id`
- `grant` → `/grant/:id`
- `user` → `/profile/:id`

## 커밋 규칙
- 매 프롬프트 작업 완료 시 반드시 커밋한다.
- 커밋 전 반드시 스모크 테스트 통과 확인.
- 커밋 메시지 형식:
  ```
  [작업내용 요약 타이틀]

  prompt: [사용자가 입력한 원본 프롬프트]

  - [작업 내역 1]
  - [작업 내역 2]
  - ...
  ```
