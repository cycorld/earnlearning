---
id: 103
title: history rewrite — PII redact across all commits + 서버 재연결 (#102 후속)
priority: high
type: chore
branch: chore/history-rewrite-pii
created: 2026-04-20
---

## 작업
1. **mirror clone audit**: `git clone --mirror` → deep history 에서 추가 발견 항목 정리
2. **filter-repo** 로 학생 실명·앱명 history 전체에서 redact
3. **force push to origin** (사용자 명시 승인 받음)
4. **빌드 서버 (cycorld) 재clone**: `~/Workspace/earnlearning` 폴더 새로 받기
5. **EC2 prod**: docker image 배포라 git 연결 없음 확인만

## 매핑 (이미 #102 에서 사용)
- 임서원 → Student-#266, 엄마맘 → App-#266
- 김나연 → Student-#267, Swipe2Eat → App-#267
- 우해든 → Student-#271, 수능체험 → App-#271
- 이서현 → Student-#276, 디핑 → App-#276

## 비번 회전: 사용자 결정 — 안 함 (memory 상 prod ≠ admin1234 확인)
