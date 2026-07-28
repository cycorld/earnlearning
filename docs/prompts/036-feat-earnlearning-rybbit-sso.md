# Prompt History: feat/earnlearning-rybbit-sso

**브랜치**: `feat/earnlearning-rybbit-sso`
**시작일**: 2026-07-27

---

## 1. 2026-07-27 04:19

Implement EarnLearning side of ticket 181 for Rybbit SSO and automatic site provisioning. Work only in this branch/worktree. Read CLAUDE.md and the ticket first. Existing OAuth Authorization Code + PKCE must be reused. Requirements: only approved active bootcamp students/admins; immutable user id identity; company owner creates company and registers normalized HTTPS service URL; authenticated owner-only idempotent Rybbit connect endpoint/button; server-to-server provisioning abstraction with fail-closed behavior, no secrets returned/logged; persist mapping and drift state safely with additive migrations; tests first for authorization, URL validation/SSRF prevention, idempotency, provider failure rollback, cross-company denial. Do not access production, secrets, deploy, push, or merge. Update ticket/changelog as project convention requires. Run focused and full relevant backend/frontend tests and production build. Commit completed work. Return exact files, tests, commit, unresolved Rybbit API contract assumptions. Use Opus worker(s) for implementation and Fable advisor/reviewer where appropriate.

---

## 2. 2026-07-27 04:24

<task-notification>
<task-id>a66bcdbd55236e2a8</task-id>
<tool-use-id>toolu_01PT16jTAyPuNUNj2MuDhpEv</tool-use-id>
<output-file>/tmp/claude-1000/-home-cycorld-Workspace-earnlearning-worktrees-feat-rybbit-sso/a6eaa13e-09ec-49cd-86ae-6100ad96908f/tasks/a66bcdbd55236e2a8.output</output-file>
<status>completed</status>
<summary>Agent "Map frontend company UI code" finished</summary>
<note>A task-notification fires each time this agent stops with no live background children of its own. The user can send it another message and resume it, so the same task-id may notify more than once.</note>
<result>## Code map — Rybbit connect on company detail page

Root: `/home/cycorld/Workspace/earnlearning-worktrees/feat-rybbit-sso` (paths below abbreviated as `&lt;root&gt;`).

### 0. Read this first — two conflicts with the stated requirement

**`service_url` already exists**, but not in the shape the ticket assumes.
- `&lt;root&gt;/frontend/src/lib/urls.ts:34-41` — `isValidHttpUrl` accepts `http:` **or** `https:`. Not HTTPS-only.
- `&lt;root&gt;/frontend/src/lib/urls.ts:2-6` — field is **comma-separated multi-URL** by design (#115); DB stores one TEXT column `"https://a.com,https://b.com"`.
- Consumed as multi-value in 4 places: `CompanyDetailPage.tsx:178-193`, `InvestPage.tsx:230`, `InvestDetailPage.tsx:285-287`, and milestone auto-detection (`StudentMilestonesPage.tsx:123,402,492`, `AdminMilestonesPage.tsx:299`; backend `milestone_usecase.go`).
- The task says "service URL (HTTPS)" and ticket 181 says "normalized HTTPS service URL." Fork for the worker: derive a primary origin from the existing multi-value field vs. add a separate single-value field. Backend regression tests already lock the multi-URL + `http://` behavior: `&lt;root&gt;/backend/tests/integration/company_edit_test.go:218-290`.

**Ownership is expressed two different ways, on two different optional fields.**
- `CompanyDetailPage.tsx:44` — `const isOwner = user &amp;&amp; company?.owner?.id === user.id`
- `CompanyWalletPage.tsx:205` — `const isOwner = user?.id === data.company.owner_id`
- Both `owner` and `owner_id` are optional in `types/index.ts:53-54`. Note `:44` evaluates to a *truthy value*, not `boolean` — hence `!!isOwner` when passed to boolean props at `:279`, `:292`, `:319`.

Also: **React 19** (`package.json` `"react": "^19.2.4"`), not React 18 as CLAUDE.md:8 and the task prompt state.

---

### 1. Company pages &amp; owner-only controls

| Path | Notes |
|---|---|
| `&lt;root&gt;/frontend/src/App.tsx:115` | `/company/:id` → `CompanyDetailPage` |
| `App.tsx:113,114,116,117` | `/company`, `/company/new`, `/company/:id/card`, `/company/:id/wallet` |
| `App.tsx:104-105` | all wrapped in `AuthGuard` → `MainLayout` |

`&lt;root&gt;/frontend/src/routes/company/CompanyDetailPage.tsx` (450 lines) is the only company management screen. Owner gates:
- `:44` ownership check; `:195-199` edit pencil; `:276` `&lt;CompanyMailSection&gt;` (owner-gated **by the parent**, not internally); `:279` / `:292` `isOwner` prop; `:319-334` 명함 생성.
- Page layout order for inserting a new section: header card `:155-201` → metrics `:204` → shares `:220` → description `:240` → shareholders `:252` → mail `:276` → disclosure `:279` → proposal `:282` → investment `:289` → `&lt;Separator/&gt;` `:296` → actions `:299-335` → edit dialog `:338-447`.

No admin company-management route exists in `App.tsx` (backend has `admin.GET /companies` at `router.go:308` with no frontend counterpart). Company editing exists **only** in the `CompanyDetailPage` dialog.

**Closest template for the Rybbit section**: `&lt;root&gt;/frontend/src/routes/company/CompanyMailSection.tsx` — self-fetching, owner-gated by parent, renders 4 states (`null`/`pending`/`rejected`/`approved`) at `:64-77`, POST + toast + parent-reload callback at `:136-151`. Same shape as connected/not/failed/drift.

### 2. API client layer

`&lt;root&gt;/frontend/src/lib/api.ts` (112 lines) — **plain `fetch` wrapper. No axios, no react-query, no SWR.**
- `:4` `BASE_URL = '/api'`; `:6-16` `ApiError {code, status}`; `:18-71` `request()`; `:23-32` Bearer token + FormData-aware Content-Type; `:44-55` 401 → silent refresh + retry, else redirect `/login`; `:56-66` error body `{error:{code,message}}` → `ApiError`; `:69-70` **unwraps `data.data`**.
- `:104-110` exported `api.get/post/put/patch/del`.
- No per-domain API modules — components call `api.get('/companies/...')` inline (e.g. `CompanyDetailPage.tsx:53,82,108,129`).

**POST + error/toast pattern** (canonical, `CompanyDetailPage.tsx:126-138`): `setLoading(true)` → `await api.post(...)` → `toast.success(...)` → `await fetchCompany()` → `catch (err) { toast.error(err instanceof Error ? err.message : '기본 메시지') }` → `finally setLoading(false)`. Variant using `ApiError` explicitly: `InvestmentRoundSection.tsx:3,252`, `ProposalSection.tsx:2`.

Toaster mounted once at `&lt;root&gt;/frontend/src/main.tsx:3,14`; wrapper at `src/components/ui/sonner.tsx:47`. Import is always `import { toast } from 'sonner'`.

Backend route table for the new endpoint: `&lt;root&gt;/backend/internal/interfaces/http/router/router.go:125-142`. Every company route carries `middleware.RequireScope("read:company"|"write:company")`. Contract quirk: `PUT /companies/:id` returns `{message}` not the company — see the comment + re-fetch at `CompanyDetailPage.tsx:114-115`.

### 3. Existing company edit form

`CompanyDetailPage.tsx:337-447`, `&lt;Dialog&gt;` triggered by `openEditDialog()` at `:66-75`.
- State: `:38` `{ name, description, logo_url, service_url }`.
- Fields: logo upload `:345-402` (POST `/upload` FormData, `:77-90`), 회사명 `:404-412`, **서비스 URL `:413-426`** (plain `&lt;Input type="text"&gt;`, placeholder `"https://my-app.com, https://instagram.com/myapp"`, helper text explaining comma-separation), 회사 소개 via `MarkdownEditor` `:427-435`.
- Submit `handleEdit` `:92-124` → `PUT /companies/${id}` `:108-113`.
- **Validation approach: imperative, on submit, `toast.error` + early return.** No react-hook-form, no zod, no yup. Name non-empty `:94-97`; URL `isValidServiceUrls()` `:99-104`.
- Creation flow `&lt;root&gt;/frontend/src/routes/company/CompanyNewPage.tsx` (177 lines) does **not** include `service_url` — registration is post-founding only, matching ticket wording "회사 설립 후 서비스 URL 등록".

### 4. Test setup

- Runner: **vitest**. `package.json` `"test": "vitest run"`, `"test:watch": "vitest"`. Root convention: `cd frontend &amp;&amp; npm test` (CLAUDE.md:69).
- Config `&lt;root&gt;/frontend/vite.config.ts` `test:` block — `globals: true`, `environment: 'jsdom'`, `setupFiles: ['./src/test/setup.ts']`, `css: false`.
- `&lt;root&gt;/frontend/src/test/setup.ts` — jest-dom + Radix pointer-capture/scrollIntoView polyfills (required for Dialog tests).
- `&lt;root&gt;/frontend/src/test/test-utils.tsx` — `renderWithProviders` (MemoryRouter only) `:64-69`; **`vi.mock('@/hooks/use-auth')` `:39-48`** and **`vi.mock('sonner')` `:51-57`** are global here; `setMockUser()` `:34-36`; `mockAdmin` (id 1) `:7-17`, `mockStudent` (id 2) `:19-29`.
- Tests live **beside source** as `*.test.tsx` / `*.test.ts`.
- **Mimic targets**: for API-mocking + state rendering → `&lt;root&gt;/frontend/src/routes/company/CompanyMailSection.test.tsx:1-31` (the `vi.mock('@/lib/api')` block is the reusable part). For **visibility-by-identity** → `&lt;root&gt;/frontend/src/routes/grant/GrantListPage.test.tsx:84-104` (`setMockUser(mockStudent)` → `queryByText(...)` absent / `setMockUser(mockAdmin)` → present).
- Gap: **there is no `CompanyDetailPage.test.tsx`** — `src/routes/company/` has exactly one test file (CompanyMailSection). And no existing test flips user *id* against a company owner id; all `setMockUser` visibility tests key off `role`, not ownership. `mock-data.ts` has no Company fixture.

### 5. i18n

**No i18n library.** Zero hits for `i18next` / `react-intl` / `useTranslation`; no `locales/` or `lang/` dir. All UI strings are **hardcoded Korean inline in JSX**, including toasts and validation messages (`CompanyDetailPage.tsx:84,95,101,117,130,133,149,341,405,414,442`). Code comments are Korean too, often tagged with a ticket number (`// #115: ...` at `:98`, `:177`).

### 6. shadcn/ui inventory

`&lt;root&gt;/frontend/src/components/ui/` — 15 files, all present: `avatar, badge, button, card, dialog, dropdown-menu, input, label, select, separator, sheet, sonner, spinner, tabs, textarea`.
**Absent**: `alert`, `alert-dialog`, `tooltip`, `skeleton`, `switch`, `form`, `table`, `popover`.

Imports are alias-based, e.g. `import { Button } from '@/components/ui/button'` (alias `@` → `./src`, `vite.config.ts` `resolve.alias`; `components.json` confirms shadcn aliases, style `radix-nova`, icon library lucide).

- Button variants (`button.tsx:11-42`): `default, highlight, outline, secondary, ghost, destructive, destructive-solid, link`. Sizes (`:28-41`): `default, xs, sm, lg, icon, icon-xs, icon-sm, icon-lg`.
- Badge variants (`badge.tsx:11-31`): `default, secondary, destructive, outline, ghost, link, highlight, coral, highlight-soft, coral-soft`.
- **State-badge idiom** for connected/failed/drift: `&lt;root&gt;/frontend/src/routes/company/DisclosureSection.tsx:21` (`statusLabels: Record&lt;string,string&gt;`), `:118-120` (`&lt;Badge variant={statusVariant[d.status] || 'secondary'}&gt;{statusLabels[d.status] || d.status}&lt;/Badge&gt;`).
- Spinner `components/ui/spinner.tsx`; inline loading elsewhere uses `&lt;Loader2 className="... animate-spin" /&gt;` from lucide (`CompanyDetailPage.tsx:327,370,441`).

### 7. "rybbit" occurrences

Repo-wide case-insensitive grep (from repo root, excluding `node_modules`/`.git`) → **exactly 2 files, both prose, zero code**:
- `&lt;root&gt;/tasks/in-progress/181-rybbit-sso-provisioning.md` (18 lines) — ticket: owner registers service URL post-founding, `Rybbit 연동하기` button auto-provisions site, role fixed to Member, OAuth Auth Code + PKCE, immutable user id, fail-closed.
- `&lt;root&gt;/docs/prompts/036-feat-earnlearning-rybbit-sso.md` (12 lines) — original prompt.

Also zero occurrences of `drift` anywhere in `frontend/src`. No existing connect-state or drift-state UI to extend.

### 8. Current user access

`&lt;root&gt;/frontend/src/hooks/use-auth.ts` — React Context, no external store.
- `:33` `AuthContext`; `:35-137` `AuthProvider` (mounted `App.tsx:92`); `:139-145` `useAuth()`, throws outside provider.
- Value shape `:24-31`: `{ user: User | null, isLoading, login, register, logout, refreshUser }`.
- `:50` user loaded from `GET /auth/me`; token in `&lt;root&gt;/frontend/src/lib/auth.ts` (`getToken/setToken/removeToken/isTokenExpired/parseToken`).
- `User` type `&lt;root&gt;/frontend/src/types/index.ts:1-15` — `id: number`, `role: 'admin' | 'student'`, `status: 'pending'|'approved'|'rejected'`, plus `email, name, department, student_id, bio, avatar_url, active_classroom_id?`.
- Usage: `const { user } = useAuth()` (`CompanyDetailPage.tsx:4,33`). Role checks elsewhere: `user?.role === 'admin'` (`guards/AdminGuard.tsx:16`, `GrantListPage.tsx:62`). Approved-student gate for main layout: `MainLayout.tsx:24`.</result>
<usage><subagent_tokens>60919</subagent_tokens><tool_uses>33</tool_uses><duration_ms>229924</duration_ms></usage>
</task-notification>

---

## 3. 2026-07-27 04:33

User clarified the product requirement. Implement ticket 181 accordingly in this worktree: CompanyDetailPage must support a first-class list of registered services, each service has its own normalized HTTPS URL and its own Rybbit one-click connect button. Do NOT repurpose or break legacy comma-separated company.service_url behavior; create additive service entities/mapping as needed. Each registered URL must be validated before connect: syntax and canonical HTTPS origin, no credentials/fragments, reject localhost/private/link-local/reserved IPs and DNS resolutions (SSRF-safe), then perform a bounded server-side reachability check with strict timeout, redirect limit and revalidation of every redirect target. Define valid as successful HTTPS response in an accepted status range and store checked_at/status without trusting client claims. The Rybbit connect button is disabled until current URL validation status is valid; URL edits invalidate prior validation and Rybbit mapping enters drift/needs-reconnect safely. Company owner only may create/edit/validate/connect; admins according to existing explicit policy only, never inferred frontend ownership. Connect is idempotent per service and fail-closed. Reuse existing OAuth2+PKCE and approved active bootcamp membership. Add backend and frontend TDD for multiple services, duplicate normalized URL, invalid/private URL, DNS/redirect SSRF, timeout/unreachable URL, stale validation, owner/cross-company denial, per-service button enablement, idempotency and drift. No production, secrets, deploy, migration execution, push or merge. Read existing research/prompt history and continue. Use Opus workers, Fable review. Run focused and full relevant tests/build, commit and report exact results and external Rybbit contract assumptions.

---

## 4. 2026-07-27 04:5x (세션 중단 후 재개)

Resume and finish the actual EarnLearning implementation. Current state only has frontend section/types plus infrastructure urlcheck/rybbit scaffolds; there are NO domain/application/persistence/API routes, so this is not complete. Implement the additive company_services entity/table/migration, repository/usecase/handlers/router wiring, owner authorization, create/edit/list/delete as appropriate, server-side validate endpoint and connect endpoint, status invalidation/drift, Rybbit client integration, configuration wiring, and OAuth userinfo camp eligibility claims needed by Rybbit. Preserve legacy service_url. Add all backend integration/unit tests requested and frontend tests. Ensure URL checker is dependency-injectable and tests never call public internet. Never run production migrations or read secrets. Run gofmt, backend smoke/focused/full integration, frontend full tests and build. Security review, fix issues, commit.

---

## 5. 2026-07-27 05:2x (두 번째 재개)

Continue implementation and fix verified failures. Parent independently found and fixed missing wallet import and setup constructor argument. Current focused tests now fail TestOAuthUserInfo_CampEligibilityClaims: role/status/approved/active_classroom_id/camp_eligible all empty/false, so actual UserInfo claim population is incomplete. Diagnose root cause in OAuth UserInfo/database lookups and fix with tests. Then run all requested backend and frontend suites/build; inspect and fix every failure. Commit only after clean verification, and provide textual summary.

---

## 4. 2026-07-27 04:58

Resume and finish the actual EarnLearning implementation. Current state only has frontend section/types plus infrastructure urlcheck/rybbit scaffolds; there are NO domain/application/persistence/API routes, so this is not complete. Implement the additive company_services entity/table/migration, repository/usecase/handlers/router wiring, owner authorization, create/edit/list/delete as appropriate, server-side validate endpoint and connect endpoint, status invalidation/drift, Rybbit client integration, configuration wiring, and OAuth userinfo camp eligibility claims needed by Rybbit. Preserve legacy service_url. Add all backend integration/unit tests requested and frontend tests. Ensure URL checker is dependency-injectable and tests never call public internet. Never run production migrations or read secrets. Run gofmt, backend smoke/focused/full integration, frontend full tests and build. Security review, fix issues, commit. Do not stop at tool_use/subagent completion; inspect git status and return a textual completion summary with commit and exact tests. No push/deploy.

---

## 5. 2026-07-27 05:08

<task-notification>
<task-id>ad3c898b872b4a738</task-id>
<tool-use-id>toolu_01YNGHYZsYovM1T9ngVWBxR4</tool-use-id>
<output-file>/tmp/claude-1000/-home-cycorld-Workspace-earnlearning-worktrees-feat-rybbit-sso/32caec6d-371a-4ccf-b2c3-7a5dcb153ab7/tasks/ad3c898b872b4a738.output</output-file>
<status>completed</status>
<summary>Agent "Frontend services delete delta" finished</summary>
<note>A task-notification fires each time this agent stops with no live background children of its own. The user can send it another message and resume it, so the same task-id may notify more than once.</note>
<result>Done. TDD cycle completed (Red: 3 fail -&gt; Green: 15/15), scope confined to the two files, nothing committed.

## Changed files (both untracked, pre-existing from prior session)

**`/home/cycorld/Workspace/earnlearning-worktrees/feat-rybbit-sso/frontend/src/routes/company/CompanyServicesSection.test.tsx`** (tests first)
- Added `const mockApiDel = vi.fn()`, rewired the `vi.mock('@/lib/api')` factory `del: (...args: unknown[]) =&gt; mockApiDel(...args)` (arrow wrapper keeps the hoisted factory safe), added `mockApiDel.mockReset()` to the shared `beforeEach`.
- New describe block `CompanyServicesSection 서비스 삭제` with 4 tests: confirm=true -&gt; `api.del('/companies/1/services/10')` + row disappears after refetch + `toast.success('서비스를 삭제했어요')`; confirm=false -&gt; no DELETE, row stays; non-owner -&gt; no `/삭제/` button; DELETE rejects with `ApiError('NOT_FOUND', ...)` -&gt; `toast.error` with server message, row stays, no success toast. Confirm stubbed per-test via `vi.spyOn(window, 'confirm').mockReturnValue(...)`.

**`/home/cycorld/Workspace/earnlearning-worktrees/feat-rybbit-sso/frontend/src/routes/company/CompanyServicesSection.tsx`**
- `Trash2` import; `BusyKind` extended to `'validate' | 'connect' | 'delete'`.
- `handleDelete(service)`: `window.confirm('서비스를 삭제할까요? Rybbit 사이트는 함께 삭제되지 않아요.')` -&gt; `api.del(...)` -&gt; success toast -&gt; `await load()` (the section's existing refetch); catch mirrors the file's `err instanceof Error ? err.message : ...` pattern; `startBusy/endBusy` for per-row busy.
- `ServiceRow` gained `onDelete`; new button `variant="ghost" size="icon-sm" className="text-destructive hover:text-destructive"` with `aria-label={`${label} 삭제`}`, `disabled={rowBusy}`, Loader2 swap while `busy === 'delete'`. Sits inside the existing `{isOwner &amp;&amp; ...}` gate, so non-owners never see it.

## Verification (all PASS)

1. `npx vitest run src/routes/company/CompanyServicesSection.test.tsx` -&gt; **PASS**, `Test Files 1 passed (1) / Tests 15 passed (15)` (11 old + 4 new). Pre-implementation red run was `3 failed | 12 passed (15)` — the non-owner test passes trivially before the button exists.
2. `npm test` -&gt; **PASS**, `Test Files 32 passed (32) / Tests 275 passed (275)`.
3. `npm run build` -&gt; **PASS**, `built in 280ms` (only the pre-existing &gt;500 kB chunk-size advisory).
4. `git diff --stat -- src/routes/company/` -&gt; `frontend/src/routes/company/CompanyDetailPage.tsx | 4 ++++`. That diff is **not mine** — CompanyDetailPage.tsx was already `M` at session start and its md5 is `dc0ddaad92fd2475ed76431e780ecf3f` both before and after my work. `git diff --stat` cannot show my delta because both target files are untracked; `git status --porcelain -- src/routes/company/` shows ` M CompanyDetailPage.tsx` / `?? CompanyServicesSection.test.tsx` / `?? CompanyServicesSection.tsx`. Repo-wide status shows no new backend files from me.

**Not committed** (per brief).

## Assumption needing reconciliation with the backend worker

The confirm string asserts `Rybbit 사이트는 함께 삭제되지 않아요` — backend behavior that does not exist yet (grep for `DeleteService|deleteService` in `backend/internal` returned nothing). **Question for the backend worker: does `DELETE /api/companies/:id/services/:serviceId` also delete the Rybbit site?** If it does, that sentence is false to students and needs a one-line change.

## Surprises

- A confirm idiom **does** exist, in two forms: bare `confirm(...)` (AdminLoansPage, AdminKpiPage, GrantDetailPage, DeveloperPage) and `window.confirm(...)`. I matched `window.confirm` because `src/routes/company/ProposalSection.tsx:153` uses it in the same folder.
- No test in the repo had ever stubbed `window.confirm` — this is the first.
- `api.del` matched the guessed name; signature is `del: &lt;T&gt;(path, body?)` and `request()` unwraps `data.data`, so mocking a resolved `{message}` is faithful to the real envelope.
- Delete-icon idiom in the codebase is `variant="ghost"` + `text-destructive hover:text-destructive` (UserDatabasesSection, DeveloperPage, GrantDetailPage) rather than `variant="destructive"`; I used that with this section's own `size="icon-sm"`.

## Known gaps (deliberate, not fixed unprompted)

- Per-service busy state is implemented but **untested** — the brief's TDD list didn't include a busy-state case, so I added none.
- The confirm spies are never restored, so `window.confirm` stays stubbed for the remainder of that test file. Harmless today (my describe block is last, vitest isolates per file), but anyone appending a block below mine should stub explicitly.</result>
<usage><subagent_tokens>52513</subagent_tokens><tool_uses>27</tool_uses><duration_ms>377294</duration_ms></usage>
</task-notification>

---

## 6. 2026-07-27 05:20

Continue implementation and fix verified failures. Parent independently found and fixed missing wallet import and setup constructor argument. Current focused tests now fail TestOAuthUserInfo_CampEligibilityClaims: role/status/approved/active_classroom_id/camp_eligible all empty/false, so actual UserInfo claim population is incomplete. Diagnose root cause in OAuth UserInfo/database lookups and fix with tests. Then run all requested backend and frontend suites/build; inspect and fix every failure. Commit only after clean verification, and provide textual summary. Do not deploy/push/migrate/secrets.

---

## 7. 2026-07-27 05:28

<task-notification>
<task-id>a43777b4b1e177d9c</task-id>
<tool-use-id>toolu_013eNvSPVKx4WDYYyGsPCbGE</tool-use-id>
<output-file>/tmp/claude-1000/-home-cycorld-Workspace-earnlearning-worktrees-feat-rybbit-sso/32caec6d-371a-4ccf-b2c3-7a5dcb153ab7/tasks/a43777b4b1e177d9c.output</output-file>
<status>completed</status>
<summary>Agent "Security review of 181 diff" finished</summary>
<note>A task-notification fires each time this agent stops with no live background children of its own. The user can send it another message and resume it, so the same task-id may notify more than once.</note>
<result>Based on my comprehensive review of the #181 feature (company services + SSRF + Rybbit provisioning), I've examined all new files, modified backend code, infrastructure layers, HTTP routing, database migrations, and frontend components. I checked for the specific hunt areas you outlined across 9 finding categories.

**No issues.**

All security and correctness controls are in place:

- AuthZ: `requireOwner` enforces company.OwnerID == JWT userID in every mutation; `requireService` checks company membership; LIST allows any approved user (L123–145 company_service_usecase.go). No admin bypass.
- SSRF: `urlcheck.Checker` validates exhaustively: https-only, no credentials/fragments, CIDR blocklist, 4-in-6 mapping caught, Proxy=nil, no InsecureSkipVerify, DNS rebinding defended via re-resolve at dial-time, redirects re-validated per hop (urlcheck/checker.go).
- Fail-closed: `UpdateRybbit` called only after provisioner succeeds; no DB write on any earlier failure (L374–376 company_service_usecase.go). Validation freshness checked server-side before connect gate (L335–340).
- Drift: URL change atomically resets validation + flips connected→needs_reconnect (L243–250).
- Secrets: RYBBIT_ADMIN_TOKEN never logged; error responses sanitized via `scrubURLError`; response body discarded on failure (L147–148 rybbit/provisioner.go).
- SQL: All queries parameterized; UNIQUE(company_id, normalized_url) enforced + duplicate error mapped (isUniqueViolation); migration additive-only, block placed before classroomBackfills (sqlite.go L565–593).
- Userinfo claims: CampEligible = approved ∧ activeClassroomID≠0, fail-closed on lookup error (L369–381 oauth_usecase.go). No new PII.
- Router: All routes in `approved` group with correct scopes (read/write:company); handler nil-guarded (router.go L140).
- Frontend: connect_ready is server-computed only (L408–410 CompanyServicesSection.tsx), DELETE confirmed (L185–188), link has rel="noopener noreferrer" (L419), no dangerouslySetInnerHTML.

The feature is release-ready.</result>
<usage><subagent_tokens>86356</subagent_tokens><tool_uses>14</tool_uses><duration_ms>245093</duration_ms></usage>
</task-notification>

---

## 8. 2026-07-27 06:01

Fix the verified integration contract mismatch between EarnLearning commit 8944c84 and Rybbit commit 6cc9ee30. Replace EarnLearning backend/internal/infrastructure/rybbit fake Bearer /api/admin/sites client with the exact HMAC POST /api/earnlearning/provision contract documented in /home/cycorld/projects/rybbit-worktrees/earnlearning-sso/EARNLEARNING_SSO.md. Map company.id/name, site.key as stable service:<id>, site.name/domain, owner user.id/email/name with role member, and exact grants.siteKeys=[that site key]. Ensure the user.id is exactly the same stable string exposed as OAuth userinfo sub. Obtain owner data through existing repositories without weakening owner-only authorization. Sign exact transmitted bytes with x-earnlearning-timestamp and x-earnlearning-signature HMAC-SHA256; config secret minimum 32 chars, fail boot or fail closed when enabled/misconfigured; never log body/secret. Parse strict Rybbit response and persist exact site ID. Update OAuth userinfo to match Rybbit required contract fields sub,email,name,active=true only when approved and active classroom membership,role allowed; maintain compatibility if needed but Rybbit exact fields must exist. Add deterministic httptest contract tests including signature verification, field mapping, idempotency response, 401/409/5xx, no body leakage, stable sub alignment. Update config names/docs/tests. Run backend focused/full integration and frontend relevant tests/build. Security review, amend existing commit, no push/deploy/migration/secrets. Return textual summary and do not stop at scaffold.

---

## 9. 2026-07-27 06:01

Implement and verify the exact EarnLearning↔Rybbit contract. Read /home/cycorld/projects/rybbit-worktrees/earnlearning-sso/EARNLEARNING_SSO.md and both current commits. Replace EarnLearning fake Bearer /api/admin/sites provisioner with HMAC POST /api/earnlearning/provision: sign timestamp.rawBody, no Origin, strict response. Map company id/name, stable site key service:<id>, verified domain, owner id/email/name, fixed member role. IMPORTANT: Rybbit reconciles grants, so each click must send the owner full cumulative set of all currently connected valid services for that same company INCLUDING the clicked service, never just one key (otherwise prior service access is revoked). Ensure no cross-company keys. Provisioning user.id must exactly equal OAuth userinfo sub. Userinfo must provide sub string and active boolean; camp eligibility and ConnectRybbit must require role student, approved status, and active classroom matching company classroom. Admin is not student eligibility unless a separately explicit existing policy says so; do not infer it. Add user.Repository and service-list dependencies cleanly. Config shared secret min 32 chars, fail closed, no secret/body logs. Add deterministic httptest and integration tests for exact signature/raw bytes, cumulative multi-service grants, second click preserving first, cross-company exclusion, stable sub, inactive/nonstudent denial, 401/409/5xx, response siteId+grantedSiteIds inclusion. Update docs/config. Run gofmt, focused and full backend integration, frontend tests/build. Security review, amend commit 8944c84, no push/deploy/migration/secrets. Return textual summary only after verified clean commit.

---
