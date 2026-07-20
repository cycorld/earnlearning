# Prompt History: feat/166-mail-address-approval

**브랜치**: `feat/166-mail-address-approval`
**시작일**: 2026-07-21

---

## 1. 2026-07-21 00:27

언러닝 메일함에서 내가 접근 가능한 이메일 셀렉트로 선택할 수 있어야해 (개인 메일, 회사 메일, 접근이 허용된 공용 메일)

---

## 2. 2026-07-21 00:35

<task-notification>
<task-id>aec151677c735f2b9</task-id>
<tool-use-id>toolu_01DhsEUsBYFCvakUWcaDANxB</tool-use-id>
<output-file>/tmp/claude-1000/-home-cycorld-Workspace-earnlearning/c4bd23fd-626a-4b72-b744-b4bf09b7d7a2/tasks/aec151677c735f2b9.output</output-file>
<status>completed</status>
<summary>Agent "Frontend mail approval UI" finished</summary>
<note>A task-notification fires each time this agent stops with no live background children of its own. The user can send it another message and resume it, so the same task-id may notify more than once.</note>
<result>Advisor says ship it. Both eyeball checks hold: the `승인 후에는 변경할 수 없습니다` warning lives unconditionally in the shared `ClaimAddressView` card (present in new/rejected/change modes), and `CompanyMailSection` is gated by `{isOwner &amp;&amp; ...}` in CompanyDetailPage. Task complete.

## Summary

Added admin-approval UX plus the multi-mailbox and shared-mailbox model to the mail feature (#166), all within `frontend/src/**`. No new deps, no `dangerouslySetInnerHTML`, no commit/push.

### Files changed
- `frontend/src/routes/mail/MailboxPage.tsx` — Reworked to load `GET /mail/mailboxes`; status-driven gate (신청/승인대기/반려/승인); mailbox selector (Tabs) over personal/회사/공용 entries with 개인·회사·공용 + 대기중·반려됨 badges; all list/send calls now pass `address_id`; send body includes `address_id`.
- `frontend/src/routes/mail/MailboxPage.test.tsx` — Rewrote fixtures for `/mail/mailboxes` + `address_id`-required contract; added pending/rejected/approved, selector-switch, personal-pending+company-approved, and shared-badge tests.
- `frontend/src/routes/company/CompanyMailSection.tsx` (new) — Owner-only 회사 이메일 section: register form / pending / rejected+재신청 / approved read-only+copy; POSTs `/companies/:id/mail-address`.
- `frontend/src/routes/company/CompanyMailSection.test.tsx` (new) — none/approved/rejected render + POST.
- `frontend/src/routes/company/CompanyDetailPage.tsx` — Renders `&lt;CompanyMailSection&gt;` when `isOwner`.
- `frontend/src/routes/admin/AdminMailAddressesPage.tsx` (new) — Tabs: "승인 대기" (개인/회사 구분 rows via `owner_type`/`owner_name`, 승인/반려 POST) and "공용 메일함" (create form, grants list, user-search grant-add via `/users/search`, revoke).
- `frontend/src/routes/admin/AdminMailAddressesPage.test.tsx` (new) — pending render/approve/reject/empty + shared create/grant/revoke.
- `frontend/src/routes/admin/AdminPage.tsx` — Added `pending_mail` stat (fetch `/admin/mail/addresses?status=pending`) + "메일 주소 승인" menu entry with pending-count badge.
- `frontend/src/App.tsx` — Route `/admin/mail-addresses` inside `AdminGuard`.
- `frontend/src/routes/notifications/NotificationsPage.tsx` — `mail_address_approved` → MailCheck (green), `mail_address_rejected` → MailX (coral); `mail → /mail` mapping already present (verified).

### Tests / verification
- `npx tsc -b --noEmit` — clean.
- `npx vitest run` — 29 files, 234 tests, all pass (FeedPage flake passed; no rerun needed). Targeted new/changed files: 22 tests pass.
- `npm run build` — success (the &gt;500 kB chunk warning is pre-existing, unrelated).

New/updated test names: MailboxPage — 메일함 없음→신청화면, 신청→승인대기 전환, pending→대기화면(메일함 아님), rejected→재신청 POST, 승인→메일함+address_id, selector 전환 address_id, 공용 뱃지+전환, 개인pending+회사approved 사용가능 (12 total). Admin — 대기 개인/회사 구분, approve POST, reject POST, 빈상태, 공용 생성 POST, grant 부여 POST, revoke POST (7). Company — 등록폼 POST, 승인 읽기전용, 반려 재신청 (3).

### Contract assumptions the mocked tests can't verify (flag before promoting to prod)
1. **Highest risk — legacy compatibility.** The gate shows the mailbox UI only when an entry has `status:"approved"`. Any already-onboarded student whose existing address returns from `/mail/mailboxes` without `status:"approved"` will be bounced to the claim form on deploy. The backend migration must mark pre-existing personal addresses `approved`. Not fixable/verifiable from frontend.
2. `owner_name` must be populated for both `owner_type` values — the admin pending row renders `{owner_name}` as the primary label; if empty for personal requests the name shows blank.
3. Notifications assume the backend emits `reference_type:"mail"` for `mail_address_approved`/`mail_address_rejected` (brief-stated).

### Known limitation (design, not a bug; not in brief/test matrix)
A user with an approved shared/company mailbox but **no personal entry at all** goes straight to the mailbox UI, and the selector offers no path to claim a personal address (personal claim form only appears when `approved.length === 0`). This is the deliberate reading — the literal "no personal → claim form" would strand shared-only users away from a mailbox they were granted. Surface if personal-claim-while-granted is required.

No design decisions beyond the three briefs were made.</result>
<usage><subagent_tokens>138373</subagent_tokens><tool_uses>46</tool_uses><duration_ms>1165188</duration_ms></usage>
</task-notification>

---

## 3. 2026-07-21 00:51

<task-notification>
<task-id>a78181e3687b16d7f</task-id>
<tool-use-id>toolu_017rXMpc7dwNUbzxo27eWihM</tool-use-id>
<output-file>/tmp/claude-1000/-home-cycorld-Workspace-earnlearning/c4bd23fd-626a-4b72-b744-b4bf09b7d7a2/tasks/a78181e3687b16d7f.output</output-file>
<status>completed</status>
<summary>Agent "Backend mail address approval" finished</summary>
<note>A task-notification fires each time this agent stops with no live background children of its own. The user can send it another message and resume it, so the same task-id may notify more than once.</note>
<result>All verification passes. Here is the final report.

## Summary

Added an admin-approval flow to the #166 mail feature and, per two mid-task scope deltas, expanded it to multi-mailbox (personal + company + shared) with per-user grants. Backend only; frontend/docs/tasks in `git status` are parallel work I did not touch.

### Files changed (one line each)
- `backend/internal/domain/notification/entity.go` — added `NotifMailAddressApproved`/`NotifMailAddressRejected` consts + both to `PushEligibleTypes` (matching `NotifMailReceived`).
- `backend/internal/domain/mail/entity.go` — `Address` gains `OwnerType/OwnerID/DisplayName/Status`; `Email`/`Attachment` gain `AddressID`; owner/status consts; `ErrNotApproved`/`ErrAlreadyApproved`; expanded reserved-word set; split `ValidateLocalPartFormat` (admin) vs `ValidateLocalPart`; new `MailboxItem`/`AddressAdminItem`/`Grant`/`SharedAddressItem`.
- `backend/internal/domain/mail/repository.go` — new repo interface (owner-keyed addresses, shared create, grants, mailbox/admin lists, company-owner lookup, `address_id`-scoped emails).
- `backend/internal/infrastructure/persistence/mail_repo.go` — implemented all of the above; emails key off `address_id`; grant upsert/revoke; shared list with grants.
- `backend/internal/infrastructure/persistence/sqlite.go` — restructured `mail_addresses` CREATE (owner_type/owner_id/user_id/display_name/status), partial unique index `(owner_type,owner_id) WHERE owner_type!='shared'`, `emails.address_id`, `mail_address_grants` table; idempotent ALTERs + owner_id backfill for local dev DBs.
- `backend/internal/application/mail_usecase.go` — approval-gated claim/re-request (owner-keyed self-exemption), approve/reject with notifications, address-based access checks, company/shared claim, kind-aware From display, approved-only inbound with shared fan-out, shared create/grant/revoke/list.
- `backend/internal/interfaces/http/handler/mail_handler.go` — `status` in payloads; `GetMailboxes`; company claim; `address_id`-required list/send; admin approve/reject + shared create/list/grant/revoke; error mappings.
- `backend/internal/interfaces/http/router/router.go` — registered new user + admin routes.
- Tests: updated `mail_address_test.go`, `mail_inbound_test.go` (helper now claims+approves, returns `address_id`), `mail_send_test.go`, `mail_scoping_test.go`; new `mail_address_approval_test.go`.

### Admin endpoint paths + response shapes
- `GET /api/admin/mail/addresses?status=pending|all` → `[{id,owner_type,owner_id,user_id,user_name,user_email,owner_name,local_part,status,created_at}]`
- `POST /api/admin/mail/addresses/:id/approve` → `{id,local_part,email,status}` (200)
- `POST /api/admin/mail/addresses/:id/reject` → same shape (200); approve/reject on already-`approved` → 409
- `POST /api/admin/mail/shared` `{local_part,display_name}` → `{address_id,local_part,display_name,email,status:"approved"}` (201)
- `GET /api/admin/mail/shared` → `[{address_id,local_part,display_name,email,grants:[{user_id,user_name,revoked}]}]`
- `POST /api/admin/mail/shared/:addressId/grants` `{user_id}` → `{granted:true}`
- `POST /api/admin/mail/shared/:addressId/grants/:userId/revoke` → `{revoked:true}`
- User-facing: `GET /api/mail/mailboxes` → `{mailboxes:[{address_id,kind,company_id,name,local_part,email,status}]}`; `POST /api/companies/:id/mail-address`; `GET /api/mail?box=&amp;address_id=` and `POST /api/mail/send {address_id,...}`.

### Exact notif_type strings
`mail_address_approved` (title "메일 주소 승인"), `mail_address_rejected` (title "메일 주소 반려"), both `reference_type="mail"`, `reference_id=`address id. Inbound stays `mail_received`.

### Tests (Red→Green)
New in `mail_address_approval_test.go` (all fail without the new behavior, pass after): `TestMailAddressPendingUnusable`, `TestMailAddressRerequest`, `TestMailAddressApprove`, `TestMailAddressReject`, `TestMailAdminAddressAuth`, `TestMailApproveRejectConflicts`, `TestMailReservedWordExpanded`, `TestMailCompanyAddress`, `TestMailCompanyAddressNonOwner`, `TestMailPersonalAndCompanyCoexist`, `TestMailCompanySendDisplayName`, `TestMailSharedReservedWordAdminOK`, `TestMailSharedGrantRevoke`, `TestMailSharedInboundFanout`, `TestMailAdminListShared`, `TestMailSharedSendDisplayName`.
Updated (kept assertions, added approval to setup): `TestMailAddressClaim` (pending status), `TestMailAddressImmutable` (approve-first → 409), `TestMailInboundDelivery`, `TestMailScoping` (+cross-mailbox 403), `TestMailSendReply`, `TestMailSendSESFailure`.

### Verification (exact commands run, all from `backend/`, `GO`=go1.24 toolchain)
- `$GO build -tags fts5 ./...` → BUILD_OK
- `$GO test -tags fts5 ./tests/integration/ -run 'TestSmoke|TestMail' -timeout 180s` → ok (29 mail tests + smoke, 0 fail)
- `$GO test -tags fts5 ./tests/integration/ -timeout 600s` → ok, **186 top-level PASS, 0 FAIL**

### Blockers / notes / deliberate deviations
- Two non-integration test packages fail to build — `internal/infrastructure/persistence/exchange_repo_test.go` (`GetListedCompanies` arg-count) and `internal/infrastructure/ragindex/loader_test.go` (`chat.WikiRepository.GetDoc`). I verified by stashing only my backend changes that both fail identically without my work — pre-existing breakage from the #159 branch, on symbols I never touched. Not caused by this task.
- Dev-DB limitation (per plan): the old `mail_addresses.user_id UNIQUE` cannot be dropped via ALTER, so a pre-existing local dev DB would block one user holding both a personal and a company mailbox. Fresh DBs (prod-never-shipped, all tests) use the new CREATE without that constraint. Report note only, not a test target.
- Deliberate deviation: `Send` passes `isAdmin=false` into the access check (owner/grant-holder only). Delta-3 listed send under "grant holder or admin," but threading `isAdmin=true` would let an admin send-as from any personal/company mailbox. I kept the stricter behavior on purpose; list/detail/attachments correctly grant admin full read.
- `ErrNoAddress` is now unreachable from `Send` (address_id required) — harmless dead mapping left in place. `status=all` admin path is correct on inspection (covered indirectly).</result>
<usage><subagent_tokens>192052</subagent_tokens><tool_uses>109</tool_uses><duration_ms>2140113</duration_ms></usage>
</task-notification>

---

## 4. 2026-07-21 01:01

관리자 기능에서 메일 계정들 전부 관리 가능해야해. 개인/회사 메일 승인 및 상태 확인, 공용메일 생성 및 공유 관리 등.

---

## 5. 2026-07-21 01:02

각 유저들이 권한 있는 메일만 확인 가능한지 백엔드/프론트엔드 관점에서 모두 테스트 제대로 해주고 보안점검해줘.

---

## 6. 2026-07-21 01:15

<task-notification>
<task-id>a489f4df8180fdfbe</task-id>
<tool-use-id>toolu_019iQvcgaKDw7LJyvjxgRTZS</tool-use-id>
<output-file>/tmp/claude-1000/-home-cycorld-Workspace-earnlearning/c4bd23fd-626a-4b72-b744-b4bf09b7d7a2/tasks/a489f4df8180fdfbe.output</output-file>
<status>completed</status>
<summary>Agent "E2E multi-mailbox approval QA" finished</summary>
<note>A task-notification fires each time this agent stops with no live background children of its own. The user can send it another message and resume it, so the same task-id may notify more than once.</note>
<result>J is now airtight. Notification routing confirmed (`mail`→`/mail`, icons registered per CLAUDE.md). All steps A-L complete with evidence. Here is the QA report.

# EarnLearning Mail Feature QA (#166: multi-mailbox + admin approval)

Environment: local, fresh DB. Setup created via curl (admin=user1, student1 김학생=user2 w/ company id1, student2 이학생=user3, classroom code `3LGUNN`). No servers/docker/git touched; source read-only.

Address IDs: personal `kim.student`=**1**, company `kim-company`=**2**, shared `hello`=**3**.

## PASS/FAIL per step

| Step | Result | Evidence |
|---|---|---|
| A. student1 claim form → 승인 대기 | **PASS** | Claim form with "관리자 승인" note + kim.student preview [40]; pending view "관리자 승인 대기 중" [41] |
| B. inbound to pending kim.student@ | **PASS** | curl → `404 {"error":"unknown recipient"}` |
| C. admin "메일 주소 승인" (대기 1) → 승인 | **PASS** | Menu badge "1건 대기" [42]; approval row 김학생/개인/대기 [43]; approve → toast, badge→0 |
| D. student1 mailbox usable | **PASS** | Address bar `kim.student@earnlearning.com`, inbox/sent tabs [44] |
| E. company address kim-company → 회사 approve | **PASS** | 회사 이메일 claim form [45]; admin row 김학생컴퍼니/**회사** badge [46]; approved |
| F. shared "hello"/"언러닝 안내" + grant student2 | **PASS** | Reserved word "hello" **succeeded** (toast "공용 메일함을 만들었습니다") [47]; grant to 이학생 verified (API: address_id 3, grant user_id 3, revoked false) |
| G. 3 inbound (personal/company/shared) | **PASS** | curl → all **201** (ids 1,2,3) |
| H. selector 개인+회사, isolation | **PASS** | 개인 shows only G1 [48]; 회사 shows only G2 [49]; both badges in selector, no cross-leak |
| I. student2 shared-only mailbox | **PASS** (not the feared gate) | Lands directly in shared inbox, G3 visible [50] — see exact render below |
| J. notifications | **PASS** | student2: 1 mail_received (공용 G3) [51]; student1: 4 = 2 approvals + 2 mail [53]; mail notif click routes to `/mail` |
| K. reply from 개인 (SES off) | **PASS on hard bar (no crash) + FINDING** | See below [52] |
| L. negative checks | **PASS** | L1 s2→s1 personal inbox = **403**; L2 s2 claim s1 company = **403**; L3 s1 claim "billing" = **400** reserved |

## Exact behavior — Step I (the flagged edge)
student2 (no personal claim, one shared grant) does **not** hit a claim-form gate. `MailboxPage` renders `MailboxShell` because the granted shared box has status `approved` (`approved.length===1`). Result [50]: 메일함 heading, address bar showing the raw `hello@earnlearning.com`, 받은편지함/보낸편지함 tabs, and the G3 shared mail (`visitor@web.com · 공용 메일 G3`). **No mailbox selector and no "공용/언러닝 안내" label are shown** — `MailboxSelector` returns null when boxes ≤ 1, so the shared display-name "언러닝 안내" is never surfaced to a shared-only user; only the bare address appears. Fully functional. API confirms `/mail/mailboxes` returns the shared box with kind `shared`, status `approved`.

## Exact behavior — Step K (headline finding)
Reply from the 개인 mailbox does **not crash**. It shows toast **"메일을 보냈습니다"** (success), auto-switches to 보낸편지함, and records the reply `Re: 개인 메일 G1 → prof@univ.ac.kr` in the sent box [52] (confirmed via `GET /mail?box=sent&amp;address_id=1`, total=1). Root cause: SES is disabled locally, and `SESService.SendMailFrom` returns `nil` when `!s.enabled` (`ses.go:135-137`) — a **no-op that reports success**. So the app tells the user the mail was sent even though nothing was dispatched externally, and persists a sent-box record.

## Findings

1. **[Medium — K] Silent false-success on disabled/failed sender.** With the sender disabled, a reply reports "sent" and creates a sent-box record without delivering. Harmless in local/dev, but the same code path means a **misconfigured or disabled sender in production would tell students their mail sent when it did not, with no signal**. Flagging neutrally for the human to decide whether `!enabled → return nil` is the intended contract vs. surfacing a "발송 비활성" state. Symptom at `backend/internal/infrastructure/email/ses.go:134-137`. Screenshot [52].

2. **[Low — I, UX] Shared-only users never see the mailbox's display name.** For a user with exactly one accessible mailbox, the selector (and thus the "언러닝 안내"/공용 label) is hidden; only `hello@earnlearning.com` shows. Where: `frontend/src/routes/mail/MailboxPage.tsx` `MailboxSelector` (`if (boxes.length &lt;= 1) return null`). Screenshot [50].

3. **[Low — F, UX] Admin mail page tab resets after each create/grant.** After creating a shared mailbox or granting access, the uncontrolled Radix `Tabs` (`defaultValue="pending"`) re-renders back to 승인 대기; admin must re-click 공용 메일함 to see the result. Where: `frontend/src/routes/admin/AdminMailAddressesPage.tsx`. Not blocking.

4. **[Env note] Running admin mail page has a 3rd tab "전체 계정 (N)"** not present in the source file version I read — the deployed frontend is slightly ahead of that source. No functional impact observed.

## Screenshots (all Read/verified)
- `.../scratchpad/e2e-shots/40-s1-mail-claim-form.png` — student1 claim form, kim.student filled, "관리자 승인" note (A)
- `.../41-s1-mail-pending.png` — 승인 대기 view (A)
- `.../42-admin-menu-mail-badge.png` — admin menu, "메일 주소 승인 1건 대기" badge (C)
- `.../43-admin-approval-list.png` — pending row 김학생 / 개인 / 대기 (C)
- `.../44-s1-mail-usable.png` — mailbox usable, address bar kim.student@ (D)
- `.../45-s1-company-mail-claim.png` — 회사 이메일 section, kim-company filled (E)
- `.../46-admin-company-approval.png` — pending row 김학생컴퍼니 / **회사** badge (E)
- `.../47-admin-shared-created.png` — shared "언러닝 안내" hello@ created (F)
- `.../48-s1-personal-inbox.png` — 개인 inbox = G1 only (H)
- `.../49-s1-company-inbox.png` — 회사 inbox = G2 only (H)
- `.../50-s2-shared-mailbox.png` — student2 shared inbox, G3, no selector (I)
- `.../51-s2-notifications.png` — student2 mail_received G3 (J)
- `.../52-s1-reply-error.png` — **reply SUCCEEDED** → sent box + "메일을 보냈습니다" toast (K; filename is a misnomer, actual = graceful success)
- `.../53-s1-notifications.png` — student1 4 notifications: 2 mail + 2 approvals (J)

Overall: all 12 scenario steps A–L pass on their hard bars. The one substantive risk to weigh is Finding #1 (silent false-success send). No crashes, no isolation leaks, reserved-word admin override works, and cross-user access is correctly rejected (403/400).</result>
<usage><subagent_tokens>180756</subagent_tokens><tool_uses>105</tool_uses><duration_ms>1338400</duration_ms></usage>
</task-notification>

---

## 7. 2026-07-21 01:18

<task-notification>
<task-id>aec151677c735f2b9</task-id>
<tool-use-id>toolu_01R6suPcNWFizxzgZh7ozPea</tool-use-id>
<output-file>/tmp/claude-1000/-home-cycorld-Workspace-earnlearning/c4bd23fd-626a-4b72-b744-b4bf09b7d7a2/tasks/aec151677c735f2b9.output</output-file>
<status>completed</status>
<summary>Agent "Frontend mail approval UI" finished</summary>
<note>A task-notification fires each time this agent stops with no live background children of its own. The user can send it another message and resume it, so the same task-id may notify more than once.</note>
<result>Both fixes are done and verified green.

## What changed (this addition)

`frontend/src/routes/mail/MailboxPage.tsx` — In `MailboxSelector`, replaced the `boxes.length &lt;= 1 → null` early-return with: `length === 0 → null`, and `length === 1 →` a non-interactive header row (same visual language as selector items: kind icon + name + kind badge + email). The Tabs selector for 2+ mailboxes is unchanged. This means a single-granted/shared-only user now sees the display name (e.g. "언러닝 안내") and the 공용 badge instead of only the bare address.

`frontend/src/routes/admin/AdminMailAddressesPage.tsx` — Made Tabs controlled: added `const [tab, setTab] = useState('pending')` and changed `&lt;Tabs defaultValue="pending"&gt;` to `&lt;Tabs value={tab} onValueChange={setTab}&gt;`. The active tab now survives the loading-spinner swap that `fetchData()` triggers after create/grant/revoke, so it no longer snaps back to 승인 대기.

Test additions:
- `MailboxPage.test.tsx` — "메일함이 하나면 이름과 구분 뱃지를 비대화형 헤더로 보여준다": single shared mailbox renders name + 공용 badge, no selector tab, mailbox usable.
- `AdminMailAddressesPage.test.tsx` — "권한 부여 후 refetch 되어도 공용 메일함 탭이 유지된다": after a grant action + refetch, asserts the 공용 메일함 tab has `data-state="active"` and the shared-tab content ("새 공용 메일함") is still shown.

## Verification
- `npx tsc -b --noEmit` — clean.
- `npx vitest run` — 29 files, 238 tests, all pass (was 236; +2 new: MailboxPage now 13, admin now 10).
- `npm run build` — success (pre-existing chunk-size warning only).

Constraints honored: changes confined to `frontend/src/**`, Korean UI, no new deps, no commit/push. The prior backend integration assumptions still stand for the coordinator to confirm before promoting.</result>
<usage><subagent_tokens>165494</subagent_tokens><tool_uses>7</tool_uses><duration_ms>151643</duration_ms></usage>
</task-notification>

---

## 8. 2026-07-21 01:30

<task-notification>
<task-id>a78181e3687b16d7f</task-id>
<tool-use-id>toolu_01Rf7UxwqQ8scP5YztwhZbRh</tool-use-id>
<output-file>/tmp/claude-1000/-home-cycorld-Workspace-earnlearning/c4bd23fd-626a-4b72-b744-b4bf09b7d7a2/tasks/a78181e3687b16d7f.output</output-file>
<status>completed</status>
<summary>Agent "Backend mail address approval" finished</summary>
<note>A task-notification fires each time this agent stops with no live background children of its own. The user can send it another message and resume it, so the same task-id may notify more than once.</note>
<result>Everything is complete and verified. Nothing was committed (HEAD unchanged).

## Security batch — complete

All 5 findings addressed TDD-style; no commit/push (task overrides the global "commit every prompt" rule).

### What changed
- **Fix 1 — mail routes reachable by OAuth-app tokens (the real finding).** Added `middleware.RejectOAuth()` in `backend/internal/interfaces/http/middleware/oauth.go` — returns 403 `OAUTH_FORBIDDEN` when the request carries `oauth_scopes` (the context key set only for OAuth-token auth, never first-party session JWT). Applied it via subgroups to **every** mail route in `router.go`: the user mail group (`/mail/*`, `/companies/:id/mail-address`) and all admin mail routes. `/api/mail/inbound` deliberately untouched (no JWT; secret-authed).
- **Fix 2 — From display-name header injection.** `mail_usecase.go` now wraps the From name in an RFC 5322 quoted-string via `quoteDisplayName()` (strip CR/LF, escape `\` then `"`, always quote). User/company/shared names are attacker-influenced.
- **Fix 3 — company-mail notification to current owner.** `ReceiveInbound` resolves the company's **current** `owner_id` at notification time instead of the frozen `addr.UserID`, so a former owner stops receiving sender+subject after ownership transfer. (Access via `userOwnsAddress` already resolved current owner.)
- **Fix 4 — thread-header injection.** `Send` strips CR/LF from `orig.MessageID`/`orig.Refs` before building `In-Reply-To`/`References` (`stripHeaderNewlines`).
- **Fix 5 — SES silent false-success.** `backend/internal/infrastructure/email/ses.go`: `SendMailFrom` now returns the new `ErrSenderDisabled` when disabled (was `return nil`). Usecase maps it to new domain `mail.ErrSendDisabled` → handler 503 `SEND_DISABLED`; store happens only after a successful send, so nothing is persisted on failure. `SendEmail` (notification/password-reset path) keeps its silent no-op — verified those callers.

### New tests (Red→Green)
Integration (`mail_address_approval_test.go`, `mail_send_test.go` spy) — Red reasoned (each fails without its fix; e.g. OAuth token would pass JWT-fallback→200, disabled spy would return nil→201+stored):
- `TestMailRejectsOAuthTokens` — OAuth token → 403 `OAUTH_FORBIDDEN` on mailboxes/address/list/send; same token still works on `/api/wallet` (sanity).
- `TestMailAdminRejectsOAuthTokens` — admin OAuth token → 403 `OAUTH_FORBIDDEN` (proves RejectOAuth fires after AdminOnly passes).
- `TestMailSendQuotesDisplayName` — CRLF+quote+angle-bracket display name → From is CRLF-free, quoted, inner quotes escaped, real address is the trailing angle-addr.
- `TestMailCompanyOwnerChangeNotifiesCurrentOwner` — transfer `companies.owner_id` in DB, inbound → new owner notified, old owner gets none.
- `TestMailSendDisabledSenderNoStore` — disabled spy → 503, `box=sent` stays empty.

Unit (`backend/internal/infrastructure/email/ses_test.go`) — Red **demonstrated** (toggled ses.go line 143 to `return nil` → `TestSESSendMailFromDisabledIsLoud` FAILs with `got &lt;nil&gt;`; restored → passes):
- `TestSESSendMailFromDisabledIsLoud` — pins the actual ses.go fix (`errors.Is(err, ErrSenderDisabled)`).
- `TestSESSendEmailDisabledStaysSilent` — guards that the notification path stays a silent no-op.

### Verification (from `backend/`, `GO`=go1.24 toolchain, `-tags fts5`)
- `$GO build -tags fts5 ./...` → BUILD_OK
- `$GO test -tags fts5 -count=1 ./internal/infrastructure/email/` → ok
- `$GO test -tags fts5 ./tests/integration/ -run 'TestSmoke|TestMail|TestOAuth' -timeout 300s` → ok (34 mail tests + OAuth suites + smoke)
- Full integration suite `$GO test -tags fts5 ./tests/integration/ -timeout 600s` → ok, **192 top-level PASS, 0 FAIL**

### Notes
- Pre-existing, unrelated build failures remain in `internal/infrastructure/persistence` (`exchange_repo_test.go` `GetListedCompanies` arg-count) and `internal/infrastructure/ragindex` (`chat.WikiRepository.GetDoc`) — confirmed earlier via stash that they fail without any of my changes (#159 tech debt). Not in scope.
- Dev-DB limitation unchanged (old `mail_addresses.user_id UNIQUE` can't be dropped by ALTER) — fresh DBs and all tests unaffected.
- Relevant files: `backend/internal/interfaces/http/middleware/oauth.go`, `backend/internal/interfaces/http/router/router.go`, `backend/internal/application/mail_usecase.go`, `backend/internal/infrastructure/email/ses.go`, `backend/internal/domain/mail/entity.go`, `backend/internal/interfaces/http/handler/mail_handler.go`, `backend/internal/infrastructure/email/ses_test.go`, `backend/tests/integration/mail_address_approval_test.go`.</result>
<usage><subagent_tokens>245293</subagent_tokens><tool_uses>44</tool_uses><duration_ms>1035802</duration_ms></usage>
</task-notification>

---
