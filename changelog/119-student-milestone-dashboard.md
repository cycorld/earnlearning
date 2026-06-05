# #119 · 학생 4대 평가지표 대시보드 + 관리자 승인 매트릭스

**날짜:** 2026-06-05
**브랜치:** `feat/student-milestone-dashboard`
**티켓:** [tasks/in-progress/119-student-milestone-dashboard.md](../tasks/in-progress/119-student-milestone-dashboard.md)

---

## 무엇을 했나

이 강의의 **절대평가 4대 지표** — ① 1차 MVP, ② 2차 MVP, ③ 사업계획서, ④ 회고 발표 —
의 제출·승인을 한곳에서 관리하는 대시보드를 만들었습니다.

- **학생용 `/milestones`**: 본인 4개 진행률, MVP는 자동 집계, 사업계획서·회고는 직접 제출
- **관리자용 `/admin/milestones`**: 전체 학생 매트릭스 + 그룹(A/B/C/D) 분류 + 승인/반려

## 왜 필요했나

syllabus-actual.md 의 평가 기준이 4가지 절대평가 항목이고, 4개 모두 완료 → A그룹, 3개 → B,
2개 → C, 1개 → D 로 학점 그룹이 결정됩니다. 그동안 이걸 추적할 곳이 없어서 학기말에 일일이
세야 했는데, **회사 service_url 이나 정부과제 응모 본문에 이미 URL이 들어있으니** 자동으로
모아주는 게 합리적이었습니다.

## 어떻게 만들었나

### 1. 자동 집계 (1·2차 MVP)

학생이 명시적으로 "이건 1차 MVP 입니다" 라고 표시하지 않아도, 다음에서 자동 detect:

1. **회사 service_url** (`#115` 에서 쉼표 구분 다중 URL 지원해둠) — 첫 URL = 1차, 두 번째 = 2차
2. **정부과제 응모 본문** (`grant_applications.proposal` 텍스트) — 정규식으로 URL 추출

집계 시점: 학생이 `/milestones` 진입할 때마다, 관리자가 매트릭스 새로고침할 때마다.
**한번 admin이 승인한 row 는 회사 URL 이 바뀌어도 갱신 안 함** — 승인 결과 보호.

### 2. URL 필터 (Deny list)

`ai.studio`, `aistudio.google.com`, `claude.ai`, `chatgpt.com`, `gemini.google.com`,
`localhost`, `127.0.0.1` 같은 **연습용 도메인은 자동으로 제외**합니다.
서브도메인 매칭(`*.claude.ai` 등). 학생이 vercel.app 또는 자체 도메인에 진짜로 배포했을 때만 인정.

같은 규칙을 [백엔드 Go](../backend/internal/domain/milestone/url_filter.go) 와
[프론트엔드 TS](../frontend/src/lib/milestone.ts) 양쪽에 동일하게 두고 단위 테스트로 락인했습니다.

### 3. 수동 제출 (사업계획서 / 회고 발표)

자동 집계가 불가능한 두 항목은 학생이 `/milestones` 에서 직접 제출. 본문 텍스트 + 선택 URL.
MVP1/MVP2 도 fallback 으로 학생이 직접 URL을 입력할 수 있게 했지만, 같은 deny list가 적용됩니다.

### 4. 관리자 승인

매트릭스에서 학생의 셀을 클릭하면 다이얼로그가 떠서 코멘트 + 승인/반려.
승인 시 학생에게 알림 발송 (`reference_type=milestone`).
승인 개수를 기반으로 그룹(A/B/C/D)이 자동 계산되어 매트릭스 우측에 표시됩니다.

## 데이터 모델

새 테이블 **`student_milestones`** 추가:

```sql
CREATE TABLE student_milestones (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    student_id      INTEGER NOT NULL REFERENCES users(id),
    milestone_type  TEXT NOT NULL CHECK (milestone_type IN ('mvp1','mvp2','business_plan','retrospective')),
    source_type     TEXT NOT NULL DEFAULT 'manual' CHECK (source_type IN ('manual','company','grant')),
    source_ref_id   INTEGER,   -- company.id 또는 grant_application.id
    url             TEXT NOT NULL DEFAULT '',
    content         TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected')),
    admin_note      TEXT NOT NULL DEFAULT '',
    approved_by     INTEGER REFERENCES users(id),
    approved_at     DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(student_id, milestone_type)  -- 학생당 4개만
);
```

`001_init.sql` 은 건드리지 않고 `RunMigrations()` 의 `CREATE TABLE IF NOT EXISTS` 로 idempotent 추가
(CLAUDE.md "DB 마이그레이션 규칙" 준수).

## 사용한 프롬프트

> 이제 학생들이 제출한 과제 대시보드를 새로 만들거야. 회사에 등록한 링크나 정부과제에 등록한 링크 기준으로 해서, vercelapp 링크 나 자체 도메인 붙인것만 인정해줘. ai studio 등은 연습용이야. 그리고 성적 평가용 지표 4가지가 뭔지 나에게 다시 알려줘. 그 4가지를 제출(이미 게시글, 회사 링크 등으로 추가한건 자동 집계)하고 내가 승인할 수 있는 기능도 만들어야해.

설계 결정은 사용자에게 4가지 선택지를 제시해서 답을 받음:
- **1차 vs 2차 MVP 구분**: "등록 순서 — 첫 URL = 1차, 두 번째 = 2차" 선택
- **정부과제 URL 출처**: "proposal 텍스트에서 정규식 추출" 선택
- **URL 인정 기준**: "Deny list 방식" 선택

## 배운 점

### 양쪽 deny list 동기화 = 문서로 못 막음

같은 deny list 규칙을 Go 와 TS 양쪽에 두면 한쪽만 업데이트하는 사고가 나기 쉽습니다.
대안은 (1) 서버에서 deny list를 API로 내려주고 클라이언트가 그걸 받는 패턴, 또는
(2) 단위 테스트로 "양쪽 deny list 항목이 같은지" 검증하는 패턴. 여기선 일단 양쪽에
**같은 코멘트**(`# 백엔드와 동기화`)를 달고 단위 테스트로 락인했습니다.

### "이미 승인된 row 는 보호" 의 필요성

처음에 자동 집계를 "매번 service_url 보고 덮어쓰기" 로 짰는데,
**학생이 회사 URL 을 바꾸면 이미 admin이 승인한 1차 MVP 결과가 사라지는 버그**가 나옵니다.
그래서 `SyncAuto()` 에서 `status='approved'` 인 row 는 건드리지 않게 하고
회귀 테스트 `TestMilestone_ApprovedNotOverwrittenOnResync` 로 락인했습니다.

### TDD 흐름

1. **Pure unit test 먼저**: `IsValidMilestoneURL` 같은 작은 함수는 29개 케이스로 deny/allow 검증 (Go 1회 + TS 1회)
2. **Integration test**: 회사 service_url → milestone 자동 detect 흐름을 endpoint 레벨로 (8개 시나리오)
3. 일부 테스트는 `-tags sqlite_fts5` 가 필요합니다 (Dockerfile 과 동일)

## 변경된 파일

### Backend
- `backend/internal/domain/milestone/{entity,errors,repository,url_filter,url_filter_test}.go` (신규)
- `backend/internal/infrastructure/persistence/milestone_repo.go` (신규)
- `backend/internal/infrastructure/persistence/sqlite.go` (마이그레이션 추가)
- `backend/internal/infrastructure/persistence/grant_repo.go` (`ListApplicationsByUserID` 추가)
- `backend/internal/domain/grant/repository.go` (인터페이스 확장)
- `backend/internal/application/milestone_usecase.go` (신규)
- `backend/internal/interfaces/http/handler/milestone_handler.go` (신규)
- `backend/internal/interfaces/http/router/router.go` (라우트 등록)
- `backend/cmd/server/main.go` + `backend/tests/integration/setup_test.go` (DI 와이어업)
- `backend/tests/integration/milestone_test.go` (신규 — 8개 통합 테스트)

### Frontend
- `frontend/src/lib/milestone.ts` (신규 — 도메인 타입 + URL 필터 + 그룹 분류)
- `frontend/src/lib/milestone.test.ts` (신규 — vitest)
- `frontend/src/routes/milestones/StudentMilestonesPage.tsx` (신규)
- `frontend/src/routes/admin/AdminMilestonesPage.tsx` (신규)
- `frontend/src/routes/admin/AdminPage.tsx` (메뉴 카드 추가)
- `frontend/src/routes/profile/ProfilePage.tsx` (네비 링크 추가)
- `frontend/src/routes/notifications/NotificationsPage.tsx` (`milestone` reference_type 매핑)
- `frontend/src/App.tsx` (라우트 등록 — `/milestones`, `/admin/milestones`)
