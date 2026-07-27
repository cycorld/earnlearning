# Prompt History: feat/176-html-assignment-download

**브랜치**: `feat/176-html-assignment-download`
**시작일**: 2026-07-27

---

## 1. 2026-07-27 15:43

Implement and release this urgent EarnLearning feature end-to-end. User explicitly approved deployment through Production: government-grant submissions must accept .html business-plan files, but HTML must be download-only and never rendered same-origin. Work only in this isolated branch/worktree. Follow CLAUDE.md TDD/ticket/changelog/PR workflow. First create ticket 176 if absent. Trace upload and government grant submission paths. Add server-authoritative .html acceptance with bounded 10MB full-content validation (UTF-8, reject NUL; do not unnecessarily reject normal business-plan HTML because it will never execute), safe filename checks, cleanup, and a protected/download endpoint that always sends Content-Disposition attachment plus safe Content-Type (application/octet-stream or text/plain) and nosniff. Do NOT expose HTML through the existing /uploads static same-origin route. Keep existing formats compatible. Update frontend accept/UI. Tests must prove accepted HTML, spoof/path/oversize rejection as applicable, and download headers/body; run smoke, targeted/full relevant backend, frontend tests/build. Independently security-review diff. Commit, push, create PR, merge after checks/local gates (user has approved full chain), sync a clean main worktree without touching the dirty primary checkout, build/deploy Stage, verify health/image/rendering and authenticated upload/download if credentials/session are safely available, then promote identical image to Production and verify health/images/logs. Never print secrets or raw sensitive logs. If any gate fails, stop promotion and report exact blocker. Return PR URL, SHAs/images, tests and live verification.

---

## 2. 2026-07-27 15:49

Continue the interrupted task. Inspect current diff first. Finish implementation and tests, especially ensure frontend grant upload accepts .html and HTML is private/download-only with attachment+nosniff. Run required gates. Then commit/push/PR/merge and deploy Stage then Production as already authorized. Do not redo completed work. Stop on failed gate. Return concise concrete evidence.

---
