---
id: 101
title: wiki seed — 디렉토리 단위 incremental (#100 후속)
priority: medium
type: feat
branch: feat/wiki-incremental-seed
created: 2026-04-20
---

## 배경
현재 `seedWikiDirIfEmpty(dst, src)` 는 dst 가 **완전 비어있을 때만** 통째로 src 에서 복사.
→ 한 번 시드 후 새 서브디렉토리(예: `lecture-notes/`) 추가되면 자동 복사 안 됨.
→ #100 작업 시 prod volume 에 수동 `docker cp` 필요했음.

## 해결
`seedWikiDir(dst, src)` 로 변경 — image src 의 파일별로 dst 에 없으면 복사 (overwrite 안 함).
- 기존 dst 파일은 절대 덮어쓰지 않음 (운영자 수정 보호)
- 새로 추가된 파일·디렉토리만 시드 (incremental)
- 결과 로그: "seeded N new files, skipped M existing"

## 회귀 안전
- 기존 동작 (빈 dst → 전체 시드) 동일하게 유지
- 운영자가 수정한 파일은 안 건드림
