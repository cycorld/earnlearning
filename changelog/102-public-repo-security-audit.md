# 102. 공개 저장소 보안 점검 + 학생 PII redact

**날짜**: 2026-04-20
**태그**: 보안, 운영, PII, 공개저장소

## 배경
이 저장소는 `github.com/cycorld/earnlearning` **public**. 학생 데이터·키·시크릿이
실수로 commit 되지 않았는지 정기 점검.

## 점검 결과 요약
| Severity | 항목 |
|---|---|
| 🔴 Critical | OAuth 버그바운티 changelog/tasks 에 **학생 실명 4명** + 보상금 + 앱명 노출 (이전 세션 본인 작성) |
| 🟠 Critical | `e2e-test*.ts` 에 hardcoded `admin@ewha.ac.kr / admin1234` (prod 비번은 다름이 확인되어 즉시 위험은 없음, 그래도 env 화) |
| 🟡 Medium | filter-repo 흔적 → mirror clone 으로 deep history 재스캔 권장 |
| 🟢 OK | API 키 (sk-*, AKIA, ghp_, xoxb-*, JWT, VAPID, OpenAI, Anthropic 등), `.env*`, `.db`, `node_modules`, `dist` — 모두 깨끗 |

## 수정
### 학생 PII redact (6 파일, 새 커밋 — history 는 별도 결정)
- `tasks/done/011-oauth-bugfix-2.md`
- `tasks/done/097-oauth-bounty-followup.md`
- `changelog/034-oauth-scope-bugfix.md`
- `changelog/038-oauth-bugfix-2.md`
- `changelog/097-oauth-spec-typed-responses.md`
- `changelog/index.json`

매핑:
- 임서원 → `Student-#266`, 엄마맘 → `App-#266`
- 김나연 → `Student-#267`, Swipe2Eat → `App-#267`
- 우해든 → `Student-#271`, 수능체험 → `App-#271`
- 이서현 → `Student-#276`, 디핑 → `App-#276`

(grant_application id 는 LMS 운영 컨텍스트에 필요해서 유지. 외부 노출 시 학생을 식별할 수 없음.)

### e2e creds env 화
- `e2e-test.ts`, `e2e-test-extended.ts`: `'admin@ewha.ac.kr'` / `'admin1234'` → `process.env.E2E_ADMIN_EMAIL` / `process.env.E2E_ADMIN_PASSWORD` (기본값 fallback 으로 dev 동작 유지)

### CLAUDE.md 신규 섹션
"공개 저장소 보안 규칙 (#102)" 섹션 추가:
- 학생 실명·학번·이메일 금지
- API 키·`.env*`·DB 덤프 금지
- 챗봇 캡처 로그 추가 시 직접 검토
- 사고 발견 시 redact 후 history rewrite 여부 사용자 확인

## 미포함 (사용자 결정 필요)
- **git history rewrite (filter-repo + force push)**: 학생 이름이 이전 커밋 history 에는 남아 있음. 이미 GitHub 에 노출되었기 때문에 cache·fork·archive 에 남아있을 수 있음. force push 는 사용자 명시적 승인 필요.
- **mirror clone audit**: filter-repo 가 로컬에 적용되어 deep deleted history 가 안 잡힐 수 있음. `git clone --mirror` 로 재스캔.
- **prod admin 비번 회전**: `admin1234` 가 prod 와 같은지 사용자 확인 (memory 상으로는 다름).
- **gitleaks / detect-secrets pre-commit hook**: CI 에 도입 권장.

## 통계
- `git rev-list --all`: 436 commits (worktree 로컬, filter-repo 적용 후)
- `gh pr list --state all`: 112 PRs
- `gh issue list --state all`: 0 issues
