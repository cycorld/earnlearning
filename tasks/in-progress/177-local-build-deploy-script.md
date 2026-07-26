---
id: 177
title: 로컬 빌드 서버 기준 배포 스크립트 수정
priority: high
type: fix
branch: fix/177-local-build-deploy-script
created: 2026-07-23
---

## 문제

`deploy-remote.sh`가 빌드 서버 자신에서 실행되면서도 `ssh cycorld`로 다시 접속한다. 현재 호스트에는 해당 SSH alias/DNS가 없어 Stage 배포가 시작되지 않는다.

## 완료 조건

- 자동 pull 없이 main·clean·origin/main 동기화 상태를 검증한 뒤 `deploy/build-and-push.sh`를 로컬에서 직접 실행한다 (어긋나면 빌드 전 실패, 사용자에게 `git pull --ff-only` 안내).
- Stage/Production 원격 배포는 기존 `ssh earnlearning` 경로를 유지한다.
- `promote`는 Stage 컨테이너의 실제 이미지 태그(`docker inspect`)를 읽어 Prod로 승격한다 (EC2 git HEAD 추측 금지, `latest`/빈 태그 거부).
- 로컬 작업 트리가 dirty이거나 비-main·비동기화이면 빌드 전에 안전하게 실패한다.
- mock 기반 회귀 테스트 `deploy/tests/test-deploy-remote.sh`를 추가한다.
- shell syntax 및 dry-run 가능한 검증을 추가한다.
- 문서와 실제 동작이 일치한다.
