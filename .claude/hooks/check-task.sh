#!/bin/bash
# Claude Code PreToolUse hook: 작업 전 티켓 존재 여부 확인
# - git checkout -b: in-progress 티켓이 있어야 브랜치 생성 가능
# - gh pr create: 브랜치명과 매칭되는 티켓이 있어야 PR 생성 가능
set -e

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | grep -o '"command":"[^"]*"' | head -1 | sed 's/"command":"//;s/"//')

# git checkout -b 명령 감지 (새 브랜치 생성)
if echo "$COMMAND" | grep -qE 'git checkout -b|git switch -c'; then
  # in-progress 티켓이 있는지 확인
  TASK_DIR="tasks/in-progress"
  if [ -d "$TASK_DIR" ]; then
    TASKS=$(ls "$TASK_DIR"/*.md 2>/dev/null | wc -l | tr -d ' ')
    if [ "$TASKS" -gt "0" ]; then
      exit 0  # 티켓 있음 → 허용
    fi
  fi

  # backlog이나 todo에서 in-progress로 옮길 티켓이 있는지 확인
  TODO_COUNT=$(ls tasks/todo/*.md 2>/dev/null | wc -l | tr -d ' ')
  BACKLOG_COUNT=$(ls tasks/backlog/*.md 2>/dev/null | wc -l | tr -d ' ')

  cat <<'MSG'
[TASK REQUIRED] 브랜치 생성 전에 작업 티켓이 필요합니다.

아래 절차를 따르세요:
1. tasks/backlog/ 또는 tasks/todo/ 에 티켓이 있으면 tasks/in-progress/ 로 이동
2. 새 작업이면 tasks/in-progress/NNN-slug.md 티켓을 먼저 생성
3. 티켓 생성 후 브랜치를 만들어주세요

티켓 형식:
```
---
id: NNN
title: 작업 제목
priority: high | medium | low
type: feat | fix | chore
branch: feat/xxx 또는 fix/xxx
created: YYYY-MM-DD
---
작업 내용 설명
```
MSG
  exit 2
fi

# gh pr create 명령 감지
if echo "$COMMAND" | grep -q "gh pr create"; then
  BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

  # main이면 PR 불가
  if [ "$BRANCH" = "main" ] || [ "$BRANCH" = "master" ]; then
    echo "DENIED: main 브랜치에서는 직접 PR을 생성할 수 없습니다."
    exit 2
  fi

  # in-progress 티켓 중 현재 브랜치와 매칭되는 티켓이 있는지 확인
  TASK_DIR="tasks/in-progress"
  if [ -d "$TASK_DIR" ]; then
    MATCHING=$(grep -rl "branch:.*${BRANCH}" "$TASK_DIR"/*.md 2>/dev/null || true)
    if [ -n "$MATCHING" ]; then
      exit 0  # 매칭 티켓 있음 → 허용
    fi
  fi

  # 브랜치명의 슬러그와 티켓 파일명 비교
  SAFE_BRANCH=$(echo "$BRANCH" | sed 's|.*/||')  # feat/xxx → xxx
  MATCHING_FILE=$(ls "$TASK_DIR"/*-"${SAFE_BRANCH}"*.md 2>/dev/null | head -1 || true)
  if [ -n "$MATCHING_FILE" ]; then
    exit 0  # 파일명 매칭 → 허용
  fi

  # 아무 in-progress 티켓이라도 있으면 허용 (엄격 모드 대신 완화)
  if [ -d "$TASK_DIR" ]; then
    TASKS=$(ls "$TASK_DIR"/*.md 2>/dev/null | wc -l | tr -d ' ')
    if [ "$TASKS" -gt "0" ]; then
      exit 0
    fi
  fi

  echo "DENIED: PR 생성 전에 tasks/in-progress/ 에 작업 티켓이 필요합니다."
  echo ""
  echo "작업이 완료되면 티켓을 tasks/done/ 으로 이동하고 PR을 생성하세요."
  exit 2
fi
