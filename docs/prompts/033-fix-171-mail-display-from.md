# Prompt History: fix/171-mail-display-from

**브랜치**: `fix/171-mail-display-from`
**시작일**: 2026-07-21

---

## 1. 2026-07-21 04:54

cycorld@cycorld-B650-LiveMixer:~/Workspace/earnlearning$ gh pr merge 170 --merge
X Pull request cycorld/earnlearning#170 is not mergeable: the merge commit cannot be cleanly created.
To have the pull request merged after all the requirements have been met, add the `--auto` flag.
Run the following to resolve the merge conflicts locally:
gh pr checkout 170 && git fetch origin main && git merge origin/main
cycorld@cycorld-B650-LiveMixer:~/Workspace/earnlearning$   gh pr checkout 170 && git fetch origin main && git merge origin/main
Already on 'fix/171-mail-display-from'
Your branch is up to date with 'origin/fix/171-mail-display-from'.
Already up to date.
remote: Enumerating objects: 1, done.
remote: Counting objects: 100% (1/1), done.
remote: Total 1 (delta 0), reused 0 (delta 0), pack-reused 0 (from 0)
Unpacking objects: 100% (1/1), 1003 bytes | 1003.00 KiB/s, done.
From https://github.com/cycorld/earnlearning
* branch            main       -> FETCH_HEAD
b8c84fe..f97d47f  main       -> origin/main
Auto-merging backend/internal/domain/mail/entity.go
Auto-merging changelog/index.json
CONFLICT (content): Merge conflict in changelog/index.json
Automatic merge failed; fix conflicts and then commit the

---

## 2. 2026-07-21 05:16

gh pr merge 170 --merge 머지 완료

---
