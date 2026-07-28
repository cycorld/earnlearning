# Prompt History: feat/184-dm-attachments

**브랜치**: `feat/184-dm-attachments`
**시작일**: 2026-07-28

---

## 1. 2026-07-28 19:47

Implement ticket 184 DM file/image attachments end-to-end in this worktree. Read CLAUDE.md and strictly use TDD (write/run failing tests first), inspect ticket conventions/changelog index, existing DM domain/repo/usecase/handler/router/frontend and secure upload patterns; do not invent unrelated routes. Requirements from user: sender may send text, attachments, or both; only sender/receiver list/download; private storage never /uploads static; safe filenames; extension/MIME/actual-byte validation; 10MB each; bounded count; cleanup on DB/write failure; Content-Disposition attachment + nosniff for non-image; validated safe image MIME can render inline; no HTML execution; additive SQLite migrations; notifications remain valid. Add backend integration/unit tests authorization, malicious/spoofed, oversize/count, cleanup, empty, persistence. Add frontend tests/UI picker preview download. Go 1.24.13 and sqlite_fts5 contract. Run focused tests, backend smoke+integration, frontend focused+build. Do not commit/push/deploy. Include ticket 184 and changelog/index. Work autonomously until all gates pass; report exact commands/results and modified files.

---

## 2. 2026-07-28 20:00

Continue ticket 184 DM file/image attachments. Ponytail full mode is active. Read CLAUDE.md, tasks/in-progress/184-dm-attachments.md, and docs/prompts/037-feat-184-dm-attachments.md. Implement end-to-end with TDD and all security requirements already documented. Run ./scripts/test-backend.sh smoke and relevant integration tests plus frontend tests/build. Do not commit, push, or deploy. Keep working until implementation and verification are complete, then summarize exact files and test outputs.

---

## 3. 2026-07-28 20:35

Fix all independent review findings before release. Read /home/cycorld/.hermes/cache/delegation/subagent-summary-0-20260728_203451_088852.txt and /home/cycorld/.hermes/cache/delegation/subagent-summary-1-20260728_203451_089007.txt. Treat backend P1 and frontend high/medium as blockers: pre-parse request body cap; private dirs 0700/files 0600; popup-safe authenticated private-file opening; visible send errors; common token refresh path; image failure/retry/download fallback; keyboard accessibility; max-w-full object-contain mobile; disable file controls while sending; synchronous duplicate-submit lock. Add regression tests first, fix, run focused/full backend and frontend tests/build. Do not commit/push/deploy. Report exact outcomes.

---
