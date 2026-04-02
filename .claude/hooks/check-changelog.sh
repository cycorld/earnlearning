#!/bin/bash
# Claude Code PreToolUse hook: PR 생성 전 changelog 엔트리 존재 여부 확인
# stdin으로 tool_input JSON이 들어옴

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | grep -o '"command":"[^"]*"' | head -1 | sed 's/"command":"//;s/"//')

# gh pr create 명령인지 확인
if echo "$COMMAND" | grep -q "gh pr create"; then
  # 현재 브랜치에서 main 대비 changelog 변경 확인
  CHANGELOG_FILES=$(git diff --name-only origin/main...HEAD 2>/dev/null | grep '^changelog/' || true)

  if [ -z "$CHANGELOG_FILES" ]; then
    echo "DENIED: changelog/ 엔트리가 없습니다."
    echo ""
    echo "PR 생성 전에 changelog/NNN-slug.md 파일을 추가해주세요."
    echo "changelog/index.json도 업데이트해야 합니다."
    echo ""
    echo "참고: CLAUDE.md '개발일지 (Changelog)' 섹션"
    exit 2
  fi

  # index.json 업데이트 여부 확인
  INDEX_UPDATED=$(echo "$CHANGELOG_FILES" | grep 'index.json' || true)
  if [ -z "$INDEX_UPDATED" ]; then
    echo "DENIED: changelog/index.json이 업데이트되지 않았습니다."
    echo ""
    echo "새 changelog 엔트리를 index.json에도 추가해주세요."
    exit 2
  fi
fi
