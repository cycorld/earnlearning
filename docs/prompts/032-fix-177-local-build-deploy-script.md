# Prompt History: fix/177-local-build-deploy-script

**브랜치**: `fix/177-local-build-deploy-script`
**시작일**: 2026-07-23

---

## 1. 2026-07-23 21:01

Fix ticket #177 in this worktree. Follow CLAUDE.md. deploy-remote.sh is now always run on the actual cycorld build host, so remove the redundant ssh back into BUILD_HOST and execute git/update plus deploy/build-and-push.sh locally from PROJECT_DIR. Design safely: refuse deploy if current branch/worktree is dirty or not synchronized with origin/main rather than silently building stale or unrelated code; do not mutate a developer branch by pull. Keep ssh earnlearning for stage/promote/status/rollback. Fix promote tag detection so it promotes the exact image currently deployed to Stage rather than reading EC2 git HEAD. Add a script-level regression test that mocks git/ssh/build commands and proves no ssh cycorld occurs, stage receives the built tag, promote reads the Stage container image tag, and dirty/non-main state fails before build. Update docs/DEPLOY.md and CLAUDE.md wording, create changelog/177-local-build-deploy-script.md, update changelog/index.json. Run bash syntax and regression tests. Do not commit/push/PR/deploy. Summarize exact results.

---

## 2. 2026-07-23 21:09

<task-notification>
<task-id>a3d2a817f0cac3b07</task-id>
<tool-use-id>toolu_01SYS5j5HsKrS8XZzUtMcrf5</tool-use-id>
<output-file>/tmp/claude-1000/-home-cycorld-Workspace-earnlearning-worktrees-fix-177-local-build-deploy-script/20d2acaa-ba56-4860-8546-6766f3ec79d6/tasks/a3d2a817f0cac3b07.output</output-file>
<status>completed</status>
<summary>Agent "Docs + changelog updates" finished</summary>
<note>A task-notification fires each time this agent stops with no live background children of its own. The user can send it another message and resume it, so the same task-id may notify more than once.</note>
<result>All done. H1 and index.json title match byte-for-byte, and JSON validates.

## Files changed/created (all absolute paths)

1. `/home/cycorld/Workspace/earnlearning-worktrees/fix-177-local-build-deploy-script/docs/DEPLOY.md` — Architecture diagram now shows 빌드서버 cycorld doing native amd64 `docker build` (x86_64 16코어) → GHCR push → EC2 pull. Retitled "배포 명령어 (로컬에서 실행)" → "(빌드서버 cycorld에서 실행)" with a one-line intro. Added "### 배포 전 안전 검증" subsection (three guards + no auto-pull + `git pull --ff-only` guidance; promote's Stage-container tag detection via `docker inspect earnlearning-stage-backend-1` + `IMAGE_TAG` override + `latest`/empty rejection; regression test path &amp; run command). Updated 권장 배포 플로우 line. Everything below (server setup, VAPID, troubleshooting) untouched.

2. `/home/cycorld/Workspace/earnlearning-worktrees/fix-177-local-build-deploy-script/CLAUDE.md` — Retitled deploy-command heading to "(빌드서버 cycorld에서 실행)"; 배포 플로우 step 1 now says pull main on cycorld first + notes the main/clean/origin verification and promote's tag promotion; step 3 notes Stage-container tag promotion; 개발 워크플로우 배포 bullet → 빌드서버에서; 빌드서버 리포 bullet drops the "자동 git pull" claim and states the verify-then-local-build behavior.

3. `/home/cycorld/Workspace/earnlearning-worktrees/fix-177-local-build-deploy-script/tasks/in-progress/177-local-build-deploy-script.md` — 완료 조건 rewritten: no auto-pull + main/clean/origin verify before local `build-and-push.sh`; promote reads Stage container's real image tag; added mock regression test item. Frontmatter unchanged.

4. `/home/cycorld/Workspace/earnlearning-worktrees/fix-177-local-build-deploy-script/changelog/177-local-build-deploy-script.md` — New educational dev-log mirroring 175's structure (H1 + 무엇을/왜/어떻게/배운 점), covering the self-SSH bug, fail-fast over auto-pull, promote "ask the container, don't guess," PATH-mock shell testing, and lessons.

5. `/home/cycorld/Workspace/earnlearning-worktrees/fix-177-local-build-deploy-script/changelog/index.json` — Appended entry at end of array (slug `177-local-build-deploy-script`, title = exact H1, date `2026-07-23`, tags `["배포","셸스크립트","테스트","인프라"]`, `github: null`), matching existing 2-space indent.

## Verification
- `python3 -m json.tool changelog/index.json &gt; /dev/null` → JSON OK (exit 0).
- H1 vs index.json title byte-for-byte → MATCH.

## Note for the reviewing Advisor
`git status` also shows `deploy-remote.sh` (modified), `deploy/tests/` (untracked), and `docs/prompts/032-fix-177-local-build-deploy-script.md` (untracked). Those are from the **parallel script rewrite**, not my edits — per the brief I did not touch the script and only documented its described final behavior. I did confirm the referenced test file `deploy/tests/test-deploy-remote.sh` exists so the doc path is accurate. No commits/pushes were made.</result>
<usage><subagent_tokens>59759</subagent_tokens><tool_uses>17</tool_uses><duration_ms>273558</duration_ms></usage>
</task-notification>

---

## 3. 2026-07-23 21:11

<task-notification>
<task-id>aac217e9b41b99bbb</task-id>
<tool-use-id>toolu_01FwBWCJ69nbUm61vRkiU4sS</tool-use-id>
<output-file>/tmp/claude-1000/-home-cycorld-Workspace-earnlearning-worktrees-fix-177-local-build-deploy-script/20d2acaa-ba56-4860-8546-6766f3ec79d6/tasks/aac217e9b41b99bbb.output</output-file>
<status>completed</status>
<summary>Agent "Rewrite deploy script + test" finished</summary>
<note>A task-notification fires each time this agent stops with no live background children of its own. The user can send it another message and resume it, so the same task-id may notify more than once.</note>
<result>Confirmed. The `backend` service has no `container_name:` override anywhere in `deploy/`, project name is `earnlearning-stage` (from `slot_project stage` in deploy.sh), and Compose v2 naming `{project}-{service}-{index}` yields exactly `earnlearning-stage-backend-1`. The constant is correct.

## Summary

Two files changed, both made executable (`chmod +x`), no commit/push per instructions.

### `/home/cycorld/Workspace/earnlearning-worktrees/fix-177-local-build-deploy-script/deploy-remote.sh` (rewritten)
- Removed `BUILD_HOST`/`BUILD_REPO_DIR` and every `ssh cycorld` — build now runs locally on this host (cycorld).
- Added `STAGE_BACKEND_CONTAINER="earnlearning-stage-backend-1"` (verified against `deploy/docker-compose.stage.yml`).
- New `ensure_clean_main()` called at the top of `deploy_to_stage` before any build: requires branch=main, clean worktree, and `origin/main` sync; only `git fetch` (never pull/merge — never mutates the branch); errors with SHAs and manual `git pull --ff-only` instruction on mismatch. `local` declarations are split from command-substitution assignments so `set -e` catches failures.
- `deploy_to_stage` builds via `IMAGE_TAG="$("$PROJECT_DIR/deploy/build-and-push.sh")"`, then unchanged `ssh earnlearning ... ./deploy.sh stage`.
- `promote_to_prod` no longer reads EC2 git HEAD. It honors `IMAGE_TAG` env, else reads the live Stage image via `ssh earnlearning "sudo docker inspect --format '{{.Config.Image}}' earnlearning-stage-backend-1"` and strips the tag; refuses empty/`latest` with an instruction to pass `IMAGE_TAG=&lt;sha&gt;`.
- `rollback_prod`/`show_status` unchanged. Header comment and `*)` usage text updated (deploy = "로컬 빌드(이 서버=cycorld) → GHCR push → Stage 배포", promote = "Stage에 떠 있는 이미지 태그를 그대로 Prod blue-green 배포").

### `/home/cycorld/Workspace/earnlearning-worktrees/fix-177-local-build-deploy-script/deploy/tests/test-deploy-remote.sh` (new)
Sandbox-based regression test (no real git/ssh/docker/network): per-scenario mktemp dir with the real script copied in plus mock `git`, `ssh`, and `build-and-push.sh` on a prepended PATH, logging calls to `$MOCK_LOG`. Covers all 8 required scenarios including the two regressions (deploy has no `ssh cycorld`; promote has no `rev-parse --short HEAD`). Fails with nonzero exit if any assertion fails.

### Verification (exact output)
- `bash -n deploy-remote.sh` → OK
- `bash -n deploy/tests/test-deploy-remote.sh` → OK
- `bash deploy/tests/test-deploy-remote.sh` → `=== 결과: 33 passed, 0 failed ===`, `EXIT=0`

No deviations from spec. The only `git pull` strings remaining in the script are inside error-message text instructing the user to run it manually (not actual pull calls), which matches the brief's intent that the script never mutates the branch.</result>
<usage><subagent_tokens>49302</subagent_tokens><tool_uses>17</tool_uses><duration_ms>420915</duration_ms></usage>
</task-notification>

---
