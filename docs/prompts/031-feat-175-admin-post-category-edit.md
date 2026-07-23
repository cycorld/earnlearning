# Prompt History: feat/175-admin-post-category-edit

**브랜치**: `feat/175-admin-post-category-edit`
**시작일**: 2026-07-23

---

## 1. 2026-07-23 13:59

EarnLearning에 관리자가 학생 게시물을 수정할 때 카테고리(channel)도 변경할 수 있는 기능을 구현하세요. 반드시 CLAUDE.md를 따르세요. 먼저 다음 번호 175 티켓 tasks/in-progress/175-admin-post-category-edit.md와 changelog/175-admin-post-category-edit.md를 만들고 changelog/index.json도 갱신하세요. TDD: 현재 관리자도 타인 게시물 본문/태그 수정 가능하지만 UpdatePostInput에는 channel_id가 없습니다. 관리자만 유효한 동일 classroom 채널로 이동 가능하게 backend domain/repository/usecase/handler를 최소 변경하고 권한·존재·classroom 경계 회귀 테스트를 먼저 작성하세요. 일반 작성자는 category를 변경할 수 없어야 합니다. frontend FeedPage 기존 수정 dialog에 관리자에게만 카테고리 selector를 노출하고 channels 목록을 재사용/조회하여 PUT /posts/:id payload에 channel_id를 보내며, 성공 시 local post channel도 갱신하세요. 수정 버튼 권한도 관리자가 학생 포스트에 볼 수 있는지 검증하세요. 관련 테스트를 추가하세요. public repo이므로 PII/secret 금지. 먼저 smoke test, 관련 backend/frontend tests, build를 실행하세요. 작업을 완료하면 정확한 변경 파일과 테스트 결과를 요약하되 commit/push/PR/deploy는 하지 마세요.

---

## 2. 2026-07-23 14:11

<task-notification>
<task-id>a7b0dd38c2b7eadd6</task-id>
<tool-use-id>toolu_01XzF6jw4ovwrCqeG3TbbUbt</tool-use-id>
<output-file>/tmp/claude-1000/-home-cycorld-Workspace-earnlearning-worktrees-feat-175-admin-post-category-edit/931bb630-85c3-4851-a8b6-c274939ae185/tasks/a7b0dd38c2b7eadd6.output</output-file>
<status>completed</status>
<summary>Agent "Backend TDD channel edit" finished</summary>
<note>A task-notification fires each time this agent stops with no live background children of its own. The user can send it another message and resume it, so the same task-id may notify more than once.</note>
<result>Full integration suite passes (60.6s), no regressions. Task complete.

## Summary

Task #175 implemented via TDD: only admins can change a post's channel (category) when editing. All verification green, nothing committed.

### Files I changed (4 — the only ones attributable to me)

- `backend/internal/application/post_usecase.go` — Added `ChannelID *int \`json:"channel_id"\`` to `UpdatePostInput` (pointer so absent ≠ 0). In `UpdatePost`, after the author/admin permission check and BEFORE any content/tag write, added a channel-move validation block guarded by `input.ChannelID != nil &amp;&amp; *input.ChannelID != p.ChannelID`: non-admin → `카테고리는 관리자만 변경할 수 있습니다`; unknown channel → propagates `FindChannelByID` error; cross-classroom (`newCh.ClassroomID != oldCh.ClassroomID`) → `같은 강의실의 채널로만 이동할 수 있습니다`. The actual `UpdatePostChannel` DB write happens after the existing content `UpdatePost` succeeds.
- `backend/internal/domain/post/repository.go` — Added `UpdatePostChannel(postID int, channelID int) error` to the `PostRepository` interface.
- `backend/internal/infrastructure/persistence/post_repo.go` — Implemented `UpdatePostChannel` (`UPDATE posts SET channel_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, error wrapped as `update post channel: %w` like neighbors).
- `backend/tests/integration/post_category_edit_test.go` (new) — `TestAdminPostCategoryEdit` with 7 subtests covering the 5 spec cases plus the two backward-compat/no-op variants.

No handler or migration changes needed (handler already binds `UpdatePostInput` and passes through).

### TDD evidence
- **Red** (before impl): cases 1-4 failed for the right reasons, e.g. `expected channel_id 4 after move, got 2`, `author changing channel should fail, but succeeded`, plus nonexistent-channel and cross-classroom moves wrongly succeeding. Cases 5/no-op/content-only were already green (unknown JSON fields dropped by `c.Bind`, channel stays put).
- **Green** (after impl): all 7 subtests PASS.

### Verification commands run (all pass)
```
go test -tags sqlite_fts5 ./tests/integration/ -run TestAdminPostCategoryEdit   # ok, 7/7
go test -tags sqlite_fts5 ./tests/integration/ -run TestSmoke                    # ok
go build -tags sqlite_fts5 ./...                                                 # exit 0
go test -tags sqlite_fts5 ./tests/integration/   (full package, no -run filter)  # ok, 60.6s, no regressions
```

### Notes / risks / assumptions
- **Toolchain workaround (parent will hit this):** local `go` is 1.22.2 but `go.mod` requires 1.24, and `GOTOOLCHAIN=auto` fails to download (sandbox, no network → `toolchain not available`). I ran everything with the cached toolchain binary at `$GOPATH/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.0.linux-amd64/bin/go` with `GOTOOLCHAIN=local`.
- **Content-overwrite contract (frontend implication, not my scope):** `UpdatePost` still overwrites `content` unconditionally (pre-existing behavior). My feature makes this newly load-bearing — an admin "move to channel X" UI action that sends `channel_id` WITHOUT the current post body will wipe the content. My tests send matching content alongside `channel_id` (mirrors the real edit form), so this is safe as long as the move UI includes content. Whoever wires the frontend must send content together with `channel_id`. I deliberately did NOT add "only update content if non-empty" logic (out of scope, per brief).
- **Not mine:** git status also shows `changelog/index.json` (modified) and untracked `changelog/175-admin-post-category-edit.md`, `docs/prompts/031-*.md`, `tasks/in-progress/175-*.md` — these were pre-existing worktree scaffolding; I did not create or edit them.
- As instructed, I did not commit.</result>
<usage><subagent_tokens>55685</subagent_tokens><tool_uses>28</tool_uses><duration_ms>518498</duration_ms></usage>
</task-notification>

---

## 3. 2026-07-23 14:18

<task-notification>
<task-id>a9197a077c200ea57</task-id>
<tool-use-id>toolu_014iZTThaEQy6ADqAJCcw1Ni</tool-use-id>
<output-file>/tmp/claude-1000/-home-cycorld-Workspace-earnlearning-worktrees-feat-175-admin-post-category-edit/931bb630-85c3-4851-a8b6-c274939ae185/tasks/a9197a077c200ea57.output</output-file>
<status>completed</status>
<summary>Agent "Frontend admin category selector" finished</summary>
<note>A task-notification fires each time this agent stops with no live background children of its own. The user can send it another message and resume it, so the same task-id may notify more than once.</note>
<result>Done. Task #175 frontend is implemented and verified.

## Files changed (mine — frontend only)

- `/home/cycorld/Workspace/earnlearning-worktrees/feat-175-admin-post-category-edit/frontend/src/routes/feed/FeedPage.tsx`
- New state `editPostChannelId` (`number | null`).
- `openEditPost` seeds it from `post.channel?.id ?? null`.
- `handleUpdatePost` builds payload `{ content, tags }` and adds `channel_id` only when `user?.role === 'admin' &amp;&amp; editPostChannelId != null`. Local update resolves the new channel as `updated.channel ?? (admin ? channels.find(c =&gt; c.id === editPostChannelId) ?? p.channel : p.channel)` — so it uses the local `channels` list because the PUT response's `channel` is null. Non-admin path leaves `channel` untouched. Resets `editPostChannelId` in the success block.
- Edit dialog: admin-only labeled native `&lt;select&gt;` (`카테고리`, `id="edit-post-channel"` associated to `&lt;Label htmlFor&gt;`) listing all `channels`, styled like the create-post select.
- `/home/cycorld/.../frontend/src/routes/feed/FeedPage.test.tsx`
- Wired a real `mockApiPut` (was an unreachable `vi.fn()`), added its default mock.
- New describe block with 5 tests: admin sees category select (with options) on a student's post; student sees no select on own post; admin change+save sends `channel_id` and the badge flips 자유→공지 (proving the local-channels fallback, since the mocked PUT response has `channel: null`); student save payload has no `channel_id` key; regression — admin sees the edit pencil on another author's post. `afterEach` resets the mock user to student.
- `/home/cycorld/.../frontend/src/test/setup.ts`
- Added guarded jsdom polyfills (`hasPointerCapture`/`setPointerCapture`/`releasePointerCapture`/`scrollIntoView`) required for Radix DropdownMenu/Dialog interaction in tests. Additive no-ops, safe for all test files.

The backend + changelog changes visible in `git diff` are from the parallel backend worker — I did not touch them.

## Verification (all run in `.../frontend`)

- `npm ci` — deps were missing in the worktree; installed.
- `npm test -- --run FeedPage` — 29/29 pass (24 pre-existing + 5 new); ran multiple times, always green.
- `npm run build` (`tsc -b &amp;&amp; vite build`) — success. The only warning is the pre-existing chunk-size (&gt;500 kB) notice, unrelated.
- Full suite `npm test -- --run` — ran 9 times: 8 runs green at 244/244; one run had a single intermittent failure in a pre-existing test `이미 좋아요한 게시글 클릭 시 좋아요가 취소된다` (~1036 ms, racing the 1000 ms default `waitFor` timeout under CPU contention on this build server). My 5 new tests never failed in any run. The base (my changes stashed) passed 3/3.

## Blockers / risks

- Intermittent full-suite flake noted above is a pre-existing borderline test (its ~1 s runtime sits at the default `waitFor` timeout); it is not one of my new tests and not caused by my logic. My extra tests add ~2 s to the FeedPage file, which marginally increases contention and can occasionally tip that borderline test over. I did not modify the unrelated test or global timeouts (out of scope / would mask). Recommend the Advisor decide whether to nudge that pre-existing test's timeout separately.
- Not committed, per the brief.</result>
<usage><subagent_tokens>101016</subagent_tokens><tool_uses>34</tool_uses><duration_ms>893607</duration_ms></usage>
</task-notification>

---

## 4. 2026-07-23 14:20

Review and correct your #175 implementation. Important P1: UpdatePost currently writes content/tags first and channel in a second UPDATE; if the second DB write fails, request returns failure after a partial mutation. Refactor repository to expose one atomic UpdatePost call that updates content, tags, and optionally channel_id in a single SQL statement (or transaction), preserving nil=no channel change. Update interface/tests accordingly. Also inspect exact Go toolchain requirement: this host reports toolchain auto-download failure for go1.24, so do not claim backend tests passed unless actually run; try the project-documented Docker or installed-compatible test route without changing go.mod. Re-run frontend tests/build. Return concise results. Do not commit/push/PR/deploy.

---
