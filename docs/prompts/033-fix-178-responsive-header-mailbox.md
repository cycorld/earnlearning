# Prompt History: fix/178-responsive-header-mailbox

**브랜치**: `fix/178-responsive-header-mailbox`
**시작일**: 2026-07-23

---

## 1. 2026-07-23 21:37

Implement ticket #178 in this worktree, following CLAUDE.md and TDD. User wants: wider desktop content, especially Mailbox; mobile mailbox selector that stays usable with many mailboxes; mobile Header currently overlaps after adding ClassroomSwitcher; classroom selection should exist only in global Header, remove any duplicate selector from Feed/board if present. First inspect existing Header, ClassroomSwitcher, MailboxPage, FeedPage, layout and tests. Add failing regression tests first. Preferred UX: responsive two-row Header on narrow screens if needed, compact brand/build metadata, full-width or otherwise non-overlapping classroom selector with clear current classroom; preserve message/notification 44px tap targets. On desktop keep one-row header. Mailbox should use substantially wider max width on desktop (including list/detail/compose consistently); for many mailboxes on mobile use an accessible single dropdown/select-style chooser rather than wrapped tiny tabs, while desktop can use clearly scannable choices. Ensure only approved mailboxes are selectable but pending/rejected states remain understandable if shown. Search for duplicate classroom selector in Feed; remove only if actually present, preserving header SSOT. Add/update component tests for header responsive classes/structure, mailbox mobile selector and wider width, and absence of feed duplicate selector. Create changelog #178 and update index; include prompt doc if convention requires. Run targeted tests, full frontend tests, and production build. Do not commit, push, PR or deploy. Return exact files and test results.

---

## 2. 2026-07-23 21:52

Continue #178. Your previous pass only added tests and they now fail: 4 test files / 11 tests. Implement production code in Header.tsx, ClassroomSwitcher.tsx, MailboxPage.tsx and FeedPage.tsx as required by those tests and the user request. Then run targeted affected tests, full npm test -- --run, npm run build. Create missing changelog/178-responsive-header-mailbox.md and changelog index entry. Do not stop after tests; inspect git diff to ensure production files changed. Do not commit/push/deploy.

---
