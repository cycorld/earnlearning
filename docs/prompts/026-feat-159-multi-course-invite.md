# Prompt History: feat/159-multi-course-invite

**브랜치**: `feat/159-multi-course-invite`
**시작일**: 2026-07-20

---

## 1. 2026-07-20 13:33

WORKLOG.md 를 읽고 #159 멀티 강의실 작업을 이어서 진행해줘. 현재 브랜치 feat/159-multi-course-invite (Phase 1~3 구현·테스트 완료). 남은 일: E2E 브라우저 검증(가상계정+두 강의실) → PR #163 머지 → 배포 → stage/prod에 2026 이화여자대학교 AI 바이브 코딩 창업부트캠프 강의실 생성. 테스트는 반드시 -tags fts5.

---

## 2. 2026-07-20 13:44

좋아. 다음 진행하고 있지?

---

## 3. 2026-07-20 13:44

최종적으로 프러덕션 새 강의실 초대코드 까지 알려줘.

---

## 4. 2026-07-20 13:45

새 강의실 이름은 알고 있어?

---

## 5. 2026-07-20 13:45

맞아. 쭉 진행해줘.

---

## 6. 2026-07-20 14:05

<task-notification>
<task-id>ad5c39cf5a4fff5b1</task-id>
<tool-use-id>toolu_01S8A31fRhwQdDkNVkAtgZto</tool-use-id>
<output-file>/tmp/claude-1000/-home-cycorld-Workspace-earnlearning/c4bd23fd-626a-4b72-b744-b4bf09b7d7a2/tasks/ad5c39cf5a4fff5b1.output</output-file>
<status>completed</status>
<summary>Agent "E2E multi-classroom browser QA" finished</summary>
<note>A task-notification fires each time this agent stops with no live background children of its own. The user can send it another message and resume it, so the same task-id may notify more than once.</note>
<result>Decisive: at a realistic mobile viewport (390x844), the submit button (bottom y=580) sits well above the bottom nav (top y=779) — no overlap, fully clickable. So the earlier submit failure was a headless-default-viewport artifact (the default was only 577px tall), NOT a product bug. That flips the finding cleanly.

I have everything needed. Here is the final report.

---

# E2E QA Report — Multi-Classroom Feature (#159)

## 1. PASS/FAIL per scenario step

1. **PASS** — Admin login → `/admin` renders the "관리 중인 강의실" classroom-context card (initially "강의실이 없습니다"). (shot 02)
2. **PASS** — Created 강의실 A (초기자본 1,000,000원, code BLBCYV) and 강의실 B (초기자본 500,000원, code 6SKWRS); both appear in the list. (shot 04)
3. **PASS** — Registered student1 (학생일) and student2 (학생이) via signup UI; both got the "관리자 승인을 기다리고 있습니다" pending dialog. (shots 06, 07) — *Deviation: literal "logout" was not exercised; I used isolated browser sessions per user (admin stays live for approvals). End-user perspective is identical.*
4. **PASS** — Admin approved both from 사용자 관리; both flipped 대기→승인됨. (shots 08, 09)
5. **PASS** — student1 login → JoinClassroomGate ("강의실 입장") appeared → entered BLBCYV → landed in 강의실 A; wallet showed 1,000,000원, matching A's capital. (shots 10, 12)
6. **PASS (wallet isolation) / see Finding B** — student1's B wallet is a separate wallet showing 500,000원 (vs A's 1,000,000). **But joining a 2nd classroom has NO UI path** — done via authenticated API call (NOT-UI-verified). (shot 13)
7. **PASS** — In A, created company "에이컴퍼니" (capital 1,000,000); it appears under 내 회사 (1). (shots 15, 18, 17)
8. **FAIL** — Switched to B; A's company **leaks** into B's 내 회사 tab and into B's wallet portfolio. (Only 전체 기업, and the empty invest/market/grant, were B-scoped.) (shots 19, 20, 21, 22)
9. **PASS** — Switched back to A; company visible again; A's cash correctly 0 (A-scoped, spent on founding). Note the portfolio total is inflated by the same leak. (shots 23, 24)
10. **PASS** — student2 joined only B via the real JoinGate UI; sees 내 회사 (0) / 전체 기업 (0) — never sees A's company; wallet a clean 500,000원. (shot 25)

## 2. Invite codes
- **강의실 A**: `BLBCYV` (initial capital 1,000,000원)
- **강의실 B**: `6SKWRS` (initial capital 500,000원)

## 3. Wallet balances observed
| User | Classroom | 보유 현금 (cash) | 주식 가치 | 회사 지분 | 총 자산 |
|---|---|---|---|---|---|
| student1 | A (before founding) | 1,000,000 | 0 | 0 | 1,000,000 |
| student1 | A (after founding) | 0 (correct, A-scoped) | 1,000,000 | 1,000,000 | 2,000,000 |
| student1 | B | 500,000 (correct, B-scoped) | **1,000,000 (leak)** | **1,000,000 (leak)** | **2,500,000** |
| student2 | B | 500,000 | 0 | 0 | 500,000 (clean) |

Cash is correctly per-classroom. Stock value / company equity are NOT — they are identical in A and B.

## 4. Bugs / anomalies

**Finding A — HIGH — Classroom-isolation leak in wallet portfolio + "내 회사" list.**
A company founded in 강의실 A appears in 강의실 B for its owner:
- Wallet 주식 가치 (1,000,000) and 회사 지분 (1,000,000) show in B, inflating 총 자산 to 2,500,000. (shot 19)
- 내 회사 tab in B lists 에이컴퍼니. (shot 20)
- API cross-check: `GET /api/companies` (전체 기업) correctly returns `[]` in B, but `GET /api/companies/mine` returns the company (classroom_id:1) regardless of active classroom.
- Root cause (source-confirmed):
- `internal/infrastructure/persistence/wallet_repo.go` `GetAssetBreakdown` — Cash query IS active-classroom-scoped (lines ~297-304, explicit #159 comment), but the `StockValue`, `CompanyEquity`, and `TotalDebt` queries filter only by `user_id`/`borrower_id` (lines ~309-337) with no classroom filter. #159 scoping was applied to cash only.
- `internal/interfaces/http/handler/company_handler.go` `GetMyCompanies` (line ~182) calls `h.uc.GetMyCompanies(userID)` — no classroom argument, unlike the scoped list endpoint.
- Leak is owner-scoped, not public: student2 (non-owner) in B sees nothing from A (shot 25).

**Finding B — HIGH — No UI path for a student to join a second classroom.** The headline multi-classroom capability is unreachable by real users. Both join-by-code UIs (`JoinClassroomGate` in `MainLayout.tsx` and the FeedPage join form) are gated on `classrooms.length === 0`, and `ClassroomSwitcher.tsx` only switches between already-joined classrooms — it has no "join another" affordance. This directly contradicts the task's hypothesis that the switcher exposes a join affordance. I had to join classroom B for student1 via a direct authenticated `POST /classrooms/join` (NOT-UI-verified) to continue.

**Finding C — MEDIUM (secondary / orthogonal to #159) — Founder equity double-counted.** For a brand-new 1,000,000 company, StockValue = 1,000,000 (shares × valuation / total_shares) AND CompanyEquity = 1,000,000 (company wallet balance × shares / total_shares) both count the same stake, so founding a company raised total assets from 1,000,000 → 2,000,000 out of nothing. Pre-existing valuation quirk, independent of multi-classroom; even without it, the 1,000,000 appearing in B would still be a leak.

**Finding D — LOW (code-level, NOT-UI-observed) — `TotalDebt` also not classroom-scoped.** Same `GetAssetBreakdown` — loan debt aggregates all classrooms (`WHERE borrower_id = ?`). No loans existed in this run, so not observed in UI; flagged as related risk.

**Finding E — LOW — Admin sees the JoinClassroomGate on `/feed`.** With no active classroom, the admin lands on the student onboarding "클래스룸 참여" gate (shot 01). May be by-design (admin has no classroom), but an admin arguably shouldn't be gated. Resolved itself once the admin had an active classroom.

**Non-bug (test-environment artifact, resolved):** On the first company-creation attempts the submit landed on `/market` and no company was created. Cause: the headless default viewport was only 577px tall, so the "회사 설립하기" button (bottom y=580) fell under the fixed bottom nav and clicks hit the 마켓 tab. At a realistic mobile viewport (390×844) the button (bottom y=580) clears the nav (top y=779) with ~200px to spare and clicks fine. **This is a test-harness viewport artifact, not a product bug.**

**Coverage caveat (important):** The invest/market/grant pages were empty in B — but they were *also* empty in A, because no jobs/grants/investment rounds were ever created. "Empty" is consistent with correct scoping but is NOT a positive isolation test (nothing existed to leak). Only the **company** surface was a discriminating isolation test, and it failed. I did not create jobs/grants because the company + wallet leak already proves the core defect.

## 5. Screenshot files (all under `/tmp/claude-1000/-home-cycorld-Workspace-earnlearning/c4bd23fd-626a-4b72-b744-b4bf09b7d7a2/scratchpad/e2e-shots/`)
- `01-admin-login.png` — admin on /feed, shown the JoinClassroomGate (Finding E)
- `02-admin-home-classroom-card.png` — admin `/admin` with "관리 중인 강의실" context card (step 1)
- `03-create-classroom-A-form.png` — new-classroom form
- `04-classroom-list-A-B.png` — A (BLBCYV/1,000,000) and B (6SKWRS/500,000) listed (step 2)
- `05-student1-signup-form.png` — student1 signup form filled
- `06-student1-signup-complete.png` — student1 pending dialog (step 3)
- `07-student2-signup-complete.png` — student2 pending dialog (step 3)
- `08-admin-pending-users.png` — both students in 대기 (step 4)
- `09-admin-users-approved.png` — both 승인됨 (step 4)
- `10-student1-joingate.png` — JoinClassroomGate for student1 (step 5)
- `11-student1-joined-A.png` — student1 in 강의실 A feed (step 5)
- `12-student1-wallet-A.png` — A wallet 1,000,000원 (step 5)
- `13-student1-wallet-B.png` — B wallet 500,000원, separate wallet (step 6)
- `14-switcher-dropdown-open.png` — ClassroomSwitcher dropdown (A/B)
- `15-company-A-form.png` — company creation form filled
- `16-company-A-listed.png` — pre-viewport-fix empty state (documents the harness artifact)
- `17-company-A-detail.png` — 에이컴퍼니 detail after 201 create
- `18-company-A-listed.png` — 에이컴퍼니 in A 내 회사 (1) (step 7)
- `19-student1-wallet-B-after-switch.png` — **LEAK**: B wallet total 2,500,000 with A's equity (Finding A)
- `20-company-B-empty.png` — **LEAK**: 에이컴퍼니 shows in B 내 회사 (Finding A)
- `21-grant-B-empty.png` — 지원금/정부과제 empty in B (see coverage caveat)
- `22-invest-B-empty.png` — 거래소/투자 empty in B (see coverage caveat)
- `23-student1-wallet-A-back.png` — back in A: cash 0 (A-scoped), total 2,000,000 (step 9)
- `24-company-A-back.png` — 에이컴퍼니 visible again in A (step 9)
- `25-student2-company-B.png` — student2 in B sees no companies (step 10, owner-scoped leak proof)</result>
<usage><subagent_tokens>145192</subagent_tokens><tool_uses>104</tool_uses><duration_ms>1522322</duration_ms></usage>
</task-notification>

---

## 6. 2026-07-20 14:26

<task-notification>
<task-id>a25cedb6ea66f2a6f</task-id>
<tool-use-id>toolu_01TyGVmFsWkcX7zUtjg52wpm</tool-use-id>
<output-file>/tmp/claude-1000/-home-cycorld-Workspace-earnlearning/c4bd23fd-626a-4b72-b744-b4bf09b7d7a2/tasks/a25cedb6ea66f2a6f.output</output-file>
<status>completed</status>
<summary>Agent "Fix classroom isolation leaks" finished</summary>
<note>A task-notification fires each time this agent stops with no live background children of its own. The user can send it another message and resume it, so the same task-id may notify more than once.</note>
<result>FeedPage's zero-classroom branch (lines 410-439) is a self-contained join-form Card with no dependency on an active classroom — it degrades gracefully (no crash) but is not role-gated, so a no-classroom admin landing on `/feed` sees the student "클래스룸 참여" form. That grounds my recommendation. Here is the final report.

---

## Summary

Three real defects fixed TDD-style; Defect 4 required no code change (diagnosis below). All required verifications pass.

### Files changed (exactly 5 — my edits)

- `/home/cycorld/Workspace/earnlearning/backend/internal/infrastructure/persistence/wallet_repo.go` — **Defect 1**. `GetAssetBreakdown` now resolves one "effective classroom" = the `classroom_id` of the same wallet row the Cash query picks (same `ORDER BY (classroom_id = active_classroom_id) DESC, classroom_id ASC LIMIT 1`), returns all-zeros early on `ErrNoRows` (no wallet / unassigned), and filters StockValue/CompanyEquity by `c.classroom_id = ?` and TotalDebt by `loans.classroom_id = ?` (column confirmed via `sqlite.go` alter statement).
- `/home/cycorld/Workspace/earnlearning/backend/internal/application/company_usecase.go` — **Defect 2**. `GetMyCompanies` now resolves the active classroom via `GetActiveClassroomID` and skips companies whose `ClassroomID != active` (mirrors `GetAllCompanies`). Signature unchanged, so the other `FindByOwnerID` caller (`milestone_usecase.go`) is untouched.
- `/home/cycorld/Workspace/earnlearning/backend/tests/integration/classroom_isolation_test.go` — two new regression tests + `activate`/`assetBreakdown` helpers.
- `/home/cycorld/Workspace/earnlearning/frontend/src/components/layout/ClassroomSwitcher.tsx` — **Defect 3**. Always renders the dropdown for ≥1 classroom (dropped the plain-label single-classroom case); added a separator + "초대 코드로 입장" item opening a controlled shadcn `Dialog` (code input → `api.post('/classrooms/join', {code})` → `window.location.reload()`, error display mirroring `JoinClassroomGate`).
- `/home/cycorld/Workspace/earnlearning/frontend/src/components/layout/ClassroomSwitcher.test.tsx` — new vitest file (jsdom pointer/ResizeObserver polyfills + `pointerEventsCheck: 0`).

### New tests — Red→Green

- `TestIsolation_AssetBreakdownScoping` — Red: active-B leaked `stock_value=50000000`, `company_equity=50000000`, `total_debt=1000000`; active-A non-zero passed. Green after Defect 1.
- `TestIsolation_MyCompaniesScoping` — Red: A-founded company appeared in `/companies/mine` when active B. Green after Defect 2.
- `ClassroomSwitcher › 강의실이 1개여도 드롭다운 버튼으로 렌더된다` and `› 초대 코드로 입장 항목으로 새 강의실에 참여한다` — Green. Red verified by inspection (old single-classroom branch rendered a plain `&lt;span&gt;`, so no `button` role and no join item existed); reverting to re-run would require a git state change the brief forbids.

### Verification (all pass)

- Backend smoke: `ok` (`-tags fts5 -run TestSmoke`).
- Backend full suite: `ok`, **371 PASS / 0 FAIL** (369 prior + 2 new; `-tags fts5 -timeout 600s`).
- Frontend `tsc -b --noEmit`: clean.
- Frontend `vitest run`: **26 files, 212 tests, all pass**.

### Deviation from brief — Defect 4 received zero code changes (intentional)

The brief's premise ("MainLayout shows JoinClassroomGate to a no-classroom admin") does not hold in the current code: `MainLayout.tsx` line 24 is `if (!user || user.role !== 'student' || user.status !== 'approved') { setNeedsJoin(false); return }`. Since `role` is `'admin' | 'student'`, `!== 'student'` already **is** the admin bypass — admins never trigger the gate. Adding `role === 'admin'` would be dead code, so I did not.

The actual "admin sees a student onboarding form" symptom lives in **`FeedPage.tsx` lines 410-439**: `if (classrooms.length === 0)` returns a "클래스룸 참여" join-form Card for *any* role (not role-gated), and the `*`→`/feed` catch-all (`App.tsx` line 158) makes `/feed` the default landing. I read the full branch: it is self-contained and degrades gracefully (no crash for a no-classroom admin) — it just shows a student-oriented form. Recommendation for the parent Advisor (a product decision, so not fixed here): either role-gate this branch or give admins a proper empty state. I did not edit FeedPage.

### Notes for the parent

- **Pre-existing external changes, not mine**: `frontend/vite.config.ts`, `changelog/159-classroom-wallets.md`, `changelog/index.json`, `WORKLOG.md`, `docs/prompts/026-*.md`, `tasks/backlog/164-*.md` show as modified/untracked but I never touched them (the start-of-session snapshot was stale). My edits are exactly the 5 files above. Per the hard constraints I left `vite.config.ts`, `changelog/`, `WORKLOG.md`, and `tasks/` alone.
- The founder's `StockValue` + `CompanyEquity` both counting ~50M (a double-count for a self-funded company) is **pre-existing valuation math I did not touch** — I only added classroom scoping. There's already a backlog ticket for it (`tasks/backlog/164-founder-equity-double-count.md`).
- No commits/pushes made; all changes left in the working tree as instructed.</result>
<usage><subagent_tokens>105694</subagent_tokens><tool_uses>59</tool_uses><duration_ms>1152192</duration_ms></usage>
</task-notification>

---

## 7. 2026-07-20 15:12

머지 후 계속 진행해줘

---
