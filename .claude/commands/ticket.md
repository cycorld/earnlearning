# /ticket — 티켓 관리 명령어

이 프로젝트의 `tasks/` 폴더 기반 티켓 보드를 관리합니다.

## 사용법

사용자가 `/ticket` 뒤에 서브커맨드를 입력합니다. 인자가 `$ARGUMENTS`로 전달됩니다.

### 서브커맨드

**`/ticket`** (인자 없음) 또는 **`/ticket board`**
- `tasks/` 전체 보드 현황을 보기 좋게 출력
- backlog, todo, in-progress, done 각 상태별 티켓 수와 제목 리스트
- done은 최근 5개만 표시

**`/ticket add [제목]`**
- `tasks/backlog/NNN-slug.md` 티켓 생성
- NNN은 전체 티켓 중 최대 번호 + 1
- slug는 제목에서 자동 생성
- frontmatter 포함 (id, title, priority: medium, type: feat, created: 오늘날짜)
- 생성 후 파일 경로 출력

**`/ticket start [id 또는 파일명]`**
- 해당 티켓을 `tasks/in-progress/`로 이동
- frontmatter에 `branch:` 필드 추가 (type에 따라 feat/fix/chore + slug)
- `updated:` 필드 업데이트
- 이동 후 "이제 `git checkout -b {branch}` 로 브랜치를 생성하세요" 안내

**`/ticket done [id 또는 파일명]`**
- 해당 티켓을 `tasks/done/`으로 이동
- `updated:` 필드 업데이트
- "PR 생성 후 머지하세요" 안내

**`/ticket show [id 또는 파일명]`**
- 해당 티켓 내용 전체 출력

**`/ticket edit [id 또는 파일명]`**
- 해당 티켓을 편집 (사용자에게 수정할 내용을 물어본 후 반영)

**`/ticket plan`**
- backlog + todo의 모든 티켓을 읽고, 우선순위/의존성을 고려하여 추천 실행 순서를 제안
- "다음에 어떤 작업을 할까요?" 형태로 안내

## 동작 규칙

1. 티켓 ID 검색: `tasks/` 하위 모든 폴더에서 `NNN-*.md` 패턴으로 검색. 숫자만 입력하면 ID 매칭, 파일명 입력하면 파일명 매칭.
2. 티켓 이동 시 `git mv` 대신 `mv` 사용 (tasks/는 .gitignore에 없으므로 git이 자동 추적)
3. 보드 출력 시 각 티켓의 title, priority, type을 포함
4. 에러 시 친절한 한국어 메시지 출력

## 인자

$ARGUMENTS
