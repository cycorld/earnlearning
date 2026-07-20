# Prompt History: feat/166-student-mailbox

**브랜치**: `feat/166-student-mailbox`
**시작일**: 2026-07-20

---

## 1. 2026-07-20 23:46

<task-notification>
<task-id>a224a4c615d0c4598</task-id>
<tool-use-id>toolu_01H9H1FMw9psgwfpWBihdj1K</tool-use-id>
<output-file>/tmp/claude-1000/-home-cycorld-Workspace-earnlearning/c4bd23fd-626a-4b72-b744-b4bf09b7d7a2/tasks/a224a4c615d0c4598.output</output-file>
<status>completed</status>
<summary>Agent "Frontend mailbox UI implementation" finished</summary>
<note>A task-notification fires each time this agent stops with no live background children of its own. The user can send it another message and resume it, so the same task-id may notify more than once.</note>
<result>Done. Everything is implemented, wired, and verified.

## Files changed

Created:
- `/home/cycorld/Workspace/earnlearning/frontend/src/routes/mail/MailboxPage.tsx` — mailbox page: address-claim gate (live validation on `^[a-z0-9][a-z0-9._-]{2,29}$`, `@earnlearning.com` preview, immutability warning "한 번 정하면 바꿀 수 없습니다"), then inbox/sent Tabs, list rows (from/to, subject, snippet, relative time, unread bold+dot, paperclip when `has_attachments`), "더 보기" offset-append pagination, detail view (headers + `body_text` in pre-wrap, no `dangerouslySetInnerHTML`, attachment download buttons), compose + reply (inline views). Address bar with copy button up top.
- `/home/cycorld/Workspace/earnlearning/frontend/src/routes/mail/MailboxPage.test.tsx` — 6 tests.

Edited (frontend/src only):
- `/home/cycorld/Workspace/earnlearning/frontend/src/App.tsx` — import + `/mail` route inside authed `MainLayout`.
- `/home/cycorld/Workspace/earnlearning/frontend/src/components/layout/BottomNav.tsx` — 메일함 entry (Mail icon) in the 더보기 menu.
- `/home/cycorld/Workspace/earnlearning/frontend/src/routes/notifications/NotificationsPage.tsx` — `mail` → `/mail` in `getReferencePath()`, `mail_received` → Mail icon in `getNotifIcon()`.

## Contract wiring (the easy-to-cross-wire bits, confirmed correct)
- Send payload uses exact keys `to`, `subject`, `body_text`, `in_reply_to_id`.
- Reply `in_reply_to_id` = the email's own `id` (not the detail's `in_reply_to`).
- Re: dedup is case-insensitive after trim (`/^re:/i`).
- Send success → switches to 보낸편지함 and refetches it; tab switch resets offset to 0 (replace, not append).
- Attachment download follows the milestone auth pattern (fetch + Bearer → blob → anchor download).

## Tests (`MailboxPage.test.tsx`, 6/6 pass)
- 주소가 없으면 주소 만들기 화면을 보여준다
- 주소를 만들면 POST 후 메일함으로 전환된다
- 목록의 읽지 않은 메일은 굵게 + 점 표시된다
- 행을 클릭하면 상세를 조회한다
- 답장은 Re: 제목이 채워지고 in_reply_to_id 로 전송한다
- 새 메일을 보내면 보낸편지함에 나타난다

NotificationsPage has no existing test file → mapping test skipped per brief.

## Verification gates (all green)
- `npx tsc -b --noEmit` → exit 0, clean.
- `npx vitest run` → final run **218 passed / 27 files**.
- `npm run build` → exit 0 (only the pre-existing &gt;500 kB chunk-size advisory, not an error).

## Important for your independent re-run — read before you see red
`FeedPage.test.tsx` (like-button optimistic-update test, ~line 532) is a **pre-existing intermittent flake**, not caused by this change:
- Passes 24/24 in isolation, every time.
- `vitest` runs with default `isolate: true` (per-file jsdom env), so my test's `beforeAll` globals cannot leak into FeedPage.
- Observed 0 FeedPage failures across my last 6 full runs; final gate run was clean 218/218. It only occasionally times out under full-suite parallel load. If you hit it, re-run — it is not the mail change.

## Scope note (not mine)
`changelog/index.json` (modified) and untracked `tasks/`, `docs/prompts/027-*`, `workers/`, `generated/`, `WORKLOG.md`, `whitepaper-2026/`, `changelog/166-student-mailbox.md` appeared mid-session from a concurrent process — I did not create or edit them. My edits are strictly the 4 frontend files above plus the new `frontend/src/routes/mail/` directory. I did not commit or push.

## Untested edges (implemented and correct, just not covered by unit tests)
- Radix `TabsTrigger` click path (tab switch is exercised programmatically via send→sent; wired identically to NotificationsPage).
- Re:-when-already-Re dedup branch and the authenticated attachment download helper.

No new npm dependencies. All UI text Korean. No blockers.</result>
<usage><subagent_tokens>83010</subagent_tokens><tool_uses>44</tool_uses><duration_ms>797715</duration_ms></usage>
</task-notification>

---

## 2. 2026-07-21 00:00

<task-notification>
<task-id>a34767c9d652cacc2</task-id>
<tool-use-id>toolu_014H5XaKdjKKfd4QNvnGPNHB</tool-use-id>
<output-file>/tmp/claude-1000/-home-cycorld-Workspace-earnlearning/c4bd23fd-626a-4b72-b744-b4bf09b7d7a2/tasks/a34767c9d652cacc2.output</output-file>
<status>completed</status>
<summary>Agent "Backend mailbox TDD implementation" finished</summary>
<note>A task-notification fires each time this agent stops with no live background children of its own. The user can send it another message and resume it, so the same task-id may notify more than once.</note>
<result>Backend for feature #166 is complete. Build passes, full integration suite is green (171 top-level passes, 0 failures), and all required TDD cases are present and passing.

## Files created
- `/home/cycorld/Workspace/earnlearning/backend/internal/domain/mail/entity.go` — entities (Address/Email/Attachment/EmailListItem), domain errors, reserved-word list, `ValidateLocalPart`, `EmailFor`, rune-safe `Snippet`.
- `/home/cycorld/Workspace/earnlearning/backend/internal/domain/mail/repository.go` — `Repository` interface.
- `/home/cycorld/Workspace/earnlearning/backend/internal/infrastructure/persistence/mail_repo.go` — SQLite repo (addresses, emails, attachments, admin list, user name/email lookup).
- `/home/cycorld/Workspace/earnlearning/backend/internal/application/mail_usecase.go` — `MailUseCase` + `MailSender` DI interface + `Send`/`ReceiveInbound`/`ClaimAddress`/`GetEmail`/`GetAttachmentForAccess`/`ListBox`/`ListAll`; recipient parsing via `net/mail.ParseAddress`, 10MB attachment cap, notification emit.
- `/home/cycorld/Workspace/earnlearning/backend/internal/interfaces/http/handler/mail_handler.go` — all 8 endpoints, error→status mapping, webhook secret constant-time check, `stored_path` hidden behind a dedicated response struct.
- `/home/cycorld/Workspace/earnlearning/backend/tests/integration/mail_address_test.go`, `mail_inbound_test.go`, `mail_scoping_test.go`, `mail_send_test.go` — integration tests (incl. the `spyMailSender` DI seam).

## Files changed
- `internal/infrastructure/email/ses.go` — added `OutgoingMail` + `SendMailFrom` (From override, In-Reply-To/References headers, unverified-identity fallback to configured From + Reply-To). Existing `SendEmail` untouched.
- `internal/infrastructure/config/config.go` — added `MailWebhookSecret` (`MAIL_WEBHOOK_SECRET`, default empty).
- `internal/domain/notification/entity.go` — added `NotifMailReceived = "mail_received"` and registered it in `PushEligibleTypes`.
- `internal/infrastructure/persistence/sqlite.go` — added idempotent `CREATE TABLE IF NOT EXISTS` for `mail_addresses`, `emails` (+ owner/created_at index), `mail_attachments` (+ index). 001_init.sql untouched; no ALTER on existing tables; no DROP/DELETE.
- `internal/interfaces/http/router/router.go` — `Handlers.Mail` field + public `/mail/inbound`, approved `/mail/*` (static before `:id`), admin `/admin/mail`.
- `cmd/server/main.go` — wired `mailRepo` + `mailUC` + `Mail` handler.
- `tests/integration/setup_test.go` — variadic `setupTestServer(t, opts...)` with `testConfig{mailWebhookSecret}`, `mailSpy` field, mail wiring, `testMailWebhookSecret` const.

## TDD Red → Green
Red was compile-level: `go vet` on `tests/integration/` failed with `undefined: email.OutgoingMail` (new symbols/routes absent) before implementation. After implementation, all Green:
`TestMailAddressClaim`, `TestMailAddressInvalid` (too_short/uppercase/consecutive_dot/leading_symbol/too_long), `TestMailAddressReserved`, `TestMailAddressDuplicate`, `TestMailAddressImmutable`, `TestMailInboundAuth` (401 wrong/missing secret + 503 disabled server), `TestMailInboundUnknownRecipient`, `TestMailInboundDelivery` (store + snippet + has_attachments + read-side-effect + notification + attachment download owner-OK/other-403 + detail other-403 + admin-read + admin list owner_name), `TestMailScoping`, `TestMailSendNoAddress`, `TestMailSendReply` (threading headers from original + From display + sent box), `TestMailSendSESFailure` (502, not stored).

## Verification (exact commands, `GO` = the go1.24 toolchain)
- `$GO build -tags fts5 ./...` → build OK.
- `$GO test -tags fts5 ./tests/integration/ -run 'TestSmoke|TestMail' -timeout 120s` → ok.
- `$GO test -tags fts5 ./tests/integration/ -timeout 600s` → ok, 53s, **171 top-level passes, 0 failures**.
- gofmt clean on all new/changed files.

## Notification emitted (exact)
`title "새 메일"`, `body = "&lt;from&gt; · &lt;subject truncated to 60 runes&gt;"`, `notif_type "mail_received"`, `reference_type "mail"`, `reference_id = email id`.

## Contract deviations / deliberate decisions
1. JWT endpoints return the contract shapes inside the standard `{success,data,error}` envelope (every other endpoint does this; the shared test `parseResponse` and the parallel frontend client expect it). `POST /mail/inbound` returns raw JSON (`201 {"id":N}`, `404 {"error":...}`, `503`, `401`, `413`) to match the literal worker-facing contract.
2. `mail_received` added to `PushEligibleTypes` — beyond the literal "create notification": inbound mail also fires a web-push and a fallback account-email notification (to the student's real account email, not the mailbox). Defensible UX; flag if you want it minimal (drop it from `PushEligibleTypes` and it stays WS+DB only).
3. Mail routes carry no OAuth `RequireScope` — modeled on DM/classrooms (internal-only).
4. Reserved local parts map to 400 (grouped under validation), matching the brief's "400 invalid"; 409 reserved only for taken / already-claimed.

## Required follow-ups (out of my scope)
- **Frontend notification mapping** (CLAUDE.md "알림 연동 체크리스트"): the new `reference_type "mail"` and `notif_type "mail_received"` are NOT yet in `frontend/src/routes/notifications/NotificationsPage.tsx`. The parallel frontend work (untracked `frontend/src/routes/mail/`) must add `getReferencePath()` `"mail"` → `/mail/:id` and a `getNotifIcon()` entry for `"mail_received"`, or clicking a new-mail notification dead-ends. I did not touch frontend/ per constraints.
- Set `MAIL_WEBHOOK_SECRET` in stage/prod env (empty disables `/mail/inbound` with 503).

## Pre-existing failures (not caused by this work)
- `go vet ./...` surfaces two pre-existing test-compile failures outside `tests/integration/`: the known `ragindex` one, plus `internal/infrastructure/persistence/exchange_repo_test.go` (`GetListedCompanies` signature). Both are separate test binaries from `tests/integration/`, so neither affects the brief's verification command; I did not modify either file (confirmed via git).

No commit/push performed. Branch `feat/166-student-mailbox` unchanged in git state.</result>
<usage><subagent_tokens>180757</subagent_tokens><tool_uses>74</tool_uses><duration_ms>1650210</duration_ms></usage>
</task-notification>

---

## 3. 2026-07-21 00:13

gh pr merge 165 --merge  완료함.

---
