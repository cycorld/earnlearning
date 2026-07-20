---
id: 157
title: 프롬프트 캡처 로그 커밋 + 로컬 데이터 디렉토리 gitignore 보강
priority: medium
type: chore
branch: chore/157-prompt-logs-gitignore
created: 2026-07-20
---

## 작업 내용

1. **gitignore 보강**: `backend/data/`(로컬 SQLite DB + 업로드 파일), `.claude/worktrees/`(에이전트 워크트리)가 untracked 상태로 노출되어 있어 `git add -A` 시 실수로 커밋될 위험. 공개 저장소 보안 규칙(#102)상 DB 파일 유출 방지를 위해 `.gitignore`에 추가.
2. **프롬프트 캡처 로그 커밋**: `docs/prompts/011~021` 11건 + `backend/docs/prompts/001~002` 2건. 보안 점검 완료 — 학생 실명/학번/이메일/토큰 없음, 전부 운영자 지시 프롬프트만 포함.
