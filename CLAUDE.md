# EarnLearning LMS

## 프로젝트 개요
이화여자대학교 "스타트업을 위한 코딩입문" 강의용 게임화 창업 교육 LMS.

## 기술 스택
- **Backend**: Go (Echo) + SQLite (Docker volume persistent)
- **Frontend**: Vite + React 18 + TypeScript + Tailwind CSS + shadcn/ui
- **Realtime**: WebSocket + Web Push (VAPID)
- **PWA**: Vite PWA Plugin (홈 화면 설치, 오프라인 캐시, 웹 푸시)
- **Auth**: JWT (이메일 회원가입 + Admin 승인제)
- **Deploy**: AWS EC2 (t3.small) + Docker Compose + Nginx + Cloudflare (SSL/CDN)

## 배포
배포 관련 상세 가이드는 [docs/DEPLOY.md](docs/DEPLOY.md) 참조.
- **Production**: https://earnlearning.com
- **Staging**: https://stage.earnlearning.com
- **CI/CD**: main 머지 → Stage 자동 배포 (~33초) → 확인 → `promote` (~5초)

## 테스트 규칙
- **TDD 방식 필수**: 버그 수정 및 새 기능 개발 시 반드시 TDD로 진행한다.
  1. 실패하는 테스트를 먼저 작성한다 (Red)
  2. 테스트를 통과시키는 최소한의 코드를 작성한다 (Green)
  3. 필요 시 리팩토링한다 (Refactor)
- **회귀 테스트 필수**: 버그 수정 시 반드시 해당 버그를 재현하는 회귀 테스트를 남겨 재발을 방지한다.
- **스모크 테스트 필수**: 커밋 또는 다른 테스트 실행 전에 반드시 스모크 테스트를 통과해야 한다.
  ```bash
  cd backend && go test ./tests/integration/ -run TestSmoke -timeout 60s
  ```
- 스모크 테스트 실패 시 커밋하지 않고 원인을 먼저 수정한다.
- **Backend 테스트**: `go test ./tests/integration/ -timeout 60s`
- **Frontend 테스트**: `cd frontend && npm test`


## 개발 워크플로우 (PR 기반)
- **main 직접 푸시 금지**: 모든 개발은 feature 브랜치에서 진행한다.
- **PR 생성 필수**: 작업 완료 후 PR을 생성하고 사용자가 리뷰 후 머지한다.
- **CI/CD**: main에 머지되면 GitHub Actions가 자동으로 Stage에 배포한다.
- **브랜치 네이밍**: `feat/기능명`, `fix/버그명`, `chore/작업명` 형식 사용.
- **Production 배포**: Stage 확인 후 `./deploy.sh promote`로 수동 프로모트.

## 개발일지 (Changelog)
- **PR 생성 시 필수**: 모든 PR에 대해 `changelog/`에 교육용 개발일지 엔트리를 추가한다.
- **파일**: `changelog/NNN-slug.md` (기존 파일 다음 순번)
- **내용**: 무엇을 했는지, 왜 필요했는지, 어떻게 만들었는지, 사용한 프롬프트, 배운 점
- **학생 대상**: 친절한 교재처럼 작성. 기술 용어는 설명 포함.
- **index.json 업데이트**: 새 엔트리 추가 시 `changelog/index.json`에도 항목 추가

## DB 마이그레이션 규칙 (⚠️ 프로덕션 안전)
- **001_init.sql 수정 금지**: 이미 배포된 init 마이그레이션은 절대 수정하지 않는다.
- **ALTER TABLE 사용 필수**: 새 컬럼 추가는 반드시 `sqlite.go`의 `RunMigrations()` 내 `alterStatements` 배열에 `ALTER TABLE ... ADD COLUMN` 문을 추가한다.
- **DEFAULT 값 필수**: 새 컬럼에는 반드시 DEFAULT 값을 지정하여 기존 데이터와 호환되게 한다.
- **DROP/DELETE 절대 금지**: 테이블 삭제, 컬럼 삭제, 데이터 삭제 절대 금지. 프로덕션에 실제 학생/교수 데이터가 있다.
- **에러 무시 패턴**: `db.Exec(stmt)` — SQLite에서 "duplicate column" 에러를 무시하여 재실행에도 안전하게 동작한다.

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
