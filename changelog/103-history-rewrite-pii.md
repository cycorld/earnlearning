# 103. history rewrite — 학생 PII 전체 git 히스토리에서 제거 (#102 후속)

**날짜**: 2026-04-20
**태그**: 보안, PII, 운영, history-rewrite

## 배경
#102 에서 새 커밋으로 학생 실명·앱명을 익명화했지만 **이전 커밋 history 에는 그대로 남아있었음**.
공개 저장소이기 때문에 단순 새 커밋만으로는 부족 — `git log -p` 로 누구나 옛날 데이터를 볼 수 있었음.

## 작업
1. **mirror clone audit** (`git clone --mirror`):
   - deep history 에서 학생 이름 4~6 commits 씩 발견 확인
   - API 키 (sk-ant-, sk-proj-, AKIA, ghp_, BEGIN PRIVATE KEY 등): **0 commits — 깨끗** ✅
2. **`git filter-repo --replace-text`** 로 8 패턴 redact (이름 4명 + 앱 4개)
3. main 브랜치 protection 일시 해제 → `git push --force` → protection 복원
4. **빌드 서버 (cycorld) 재 clone**: `~/Workspace/earnlearning` 백업 후 새 clone (deploy-remote.sh 가 git pull 하던 디렉토리)
5. **EC2 prod 재 clone**: `/home/ubuntu/lms` 백업 후 새 clone (`tasks/` bind mount source — 컨테이너 재시작 불필요, bind 즉시 반영)
6. **로컬 worktree 동기화**: `git fetch origin` + `reset --hard origin/main`

## 검증 결과 (모두 0 commits)
| 패턴 | github origin/main | 빌드 서버 | EC2 prod |
|---|---|---|---|
| 임서원/김나연/우해든/이서현 | 0 | 0 | 0 |
| 엄마맘/Swipe2Eat/디핑/수능체험 | 0 | 0 | 0 |

## 백업 위치 (확인 후 삭제 예정)
- 빌드서버: `cycorld:~/Workspace/earnlearning.bak.20260420-103149`
- EC2: `earnlearning:/home/ubuntu/lms.bak.20260420-013226`

## 알아두기
- **GitHub PR 페이지의 diff snapshot 은 갱신 불가** — 이전 PR (특히 #108, #109, #113) 의 diff 페이지에는 옛 이름이 남아있을 수 있음. PR 자체를 close + 새로 만들지 않는 한 GitHub UI 에서는 보임.
- **이미 클론한 다른 사람의 사본** (있다면) 에는 옛 history 가 그대로 — fork 모니터링 권장.
- main 외 force push 차단된 다른 branches/PR refs 는 그대로 유지 (모두 closed PR 들).

## 미포함 (별도 결정)
- prod admin 비번 회전 — 사용자 결정: **안 함** (memory 상 prod 비번 ≠ admin1234 확인됨)
- gitleaks pre-commit hook — 별도 티켓 권장
