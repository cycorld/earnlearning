#!/bin/bash
# Claude Code UserPromptSubmit hook:
# 1) main 브랜치면 → 브랜치 생성 유도 (context injection)
# 2) feature 브랜치면 → docs/prompts/NNN-브랜치명.md에 프롬프트 저장
set -e

INPUT=$(cat)
PROMPT=$(echo "$INPUT" | jq -r '.prompt // empty')

# 빈 프롬프트, 슬래시 커맨드 무시
if [ -z "$PROMPT" ]; then
  exit 0
fi
if echo "$PROMPT" | grep -qE '^\s*/'; then
  exit 0
fi

CWD=$(echo "$INPUT" | jq -r '.cwd // empty')

# 브랜치명 가져오기
BRANCH=$(git -C "$CWD" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
if [ "$BRANCH" = "HEAD" ]; then
  BRANCH="detached"
fi

# main 브랜치에서는 티켓 생성 + 브랜치 생성을 강제
if [ "$BRANCH" = "main" ] || [ "$BRANCH" = "master" ]; then
  TASK_DIRS="$CWD/tasks/backlog $CWD/tasks/todo $CWD/tasks/in-progress"

  # 프롬프트에서 키워드 추출 (처음 5단어)
  KEYWORDS=$(echo "$CLEAN_PROMPT" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-zA-Z0-9가-힣 ]//g' | awk '{for(i=1;i<=NF&&i<=8;i++) printf "%s\n",$i}')

  # 모든 상태의 티켓에서 프롬프트 키워드와 매칭되는 티켓 검색
  MATCHED_TICKETS=""
  for dir in $TASK_DIRS; do
    [ ! -d "$dir" ] && continue
    STATUS=$(basename "$dir")
    for f in "$dir"/*.md; do
      [ ! -f "$f" ] && continue
      FNAME=$(basename "$f")
      TITLE=$(grep '^title:' "$f" 2>/dev/null | head -1 | sed 's/title:[[:space:]]*//')
      CONTENT=$(cat "$f" 2>/dev/null | tr '[:upper:]' '[:lower:]')
      for kw in $KEYWORDS; do
        [ ${#kw} -lt 2 ] && continue
        if echo "$CONTENT" | grep -q "$kw"; then
          MATCHED_TICKETS="${MATCHED_TICKETS}  - [${STATUS}] ${FNAME}: ${TITLE}\n"
          break
        fi
      done
    done
  done

  # 현재 in-progress 티켓 목록
  ACTIVE_TASKS=""
  if [ -d "$CWD/tasks/in-progress" ]; then
    ACTIVE_TASKS=$(ls "$CWD/tasks/in-progress"/*.md 2>/dev/null | while read f; do
      TITLE=$(grep '^title:' "$f" 2>/dev/null | head -1 | sed 's/title:[[:space:]]*//')
      BRNCH=$(grep '^branch:' "$f" 2>/dev/null | head -1 | sed 's/branch:[[:space:]]*//')
      FNAME=$(basename "$f")
      echo "  - $FNAME: $TITLE (branch: $BRNCH)"
    done)
  fi

  MATCH_SECTION=""
  if [ -n "$MATCHED_TICKETS" ]; then
    MATCH_SECTION=$(printf "\n**프롬프트와 관련 있을 수 있는 기존 티켓:**\n${MATCHED_TICKETS}")
    MATCH_SECTION="${MATCH_SECTION}기존 티켓이 있으면 해당 티켓을 tasks/in-progress/로 이동하여 재사용하세요.\n"
  fi

  cat <<INJECT
<user-prompt-submit-hook>
[TASK + BRANCH REQUIRED] 현재 main 브랜치입니다.

새 작업을 시작하려면 아래 절차를 따르세요:

1. **기존 티켓 확인**: 아래 목록에서 관련 티켓이 있는지 확인
${MATCH_SECTION:-   (프롬프트와 매칭되는 기존 티켓 없음)}
**현재 진행 중인 티켓:**
${ACTIVE_TASKS:-   (없음)}

2. 기존 티켓이 있으면 tasks/in-progress/로 이동, 없으면 새 티켓 생성
3. 티켓의 branch 필드에 맞는 브랜치 생성 (\`git checkout -b feat/xxx\`)
4. 브랜치에서 작업 시작

티켓 형식:
\`\`\`
---
id: NNN
title: 작업 제목
priority: high | medium | low
type: feat | fix | chore
branch: feat/xxx
created: YYYY-MM-DD
---
작업 내용 설명
\`\`\`

사용자의 원래 요청을 무시하지 말고, 티켓 + 브랜치 생성 후 이어서 처리하세요.
</user-prompt-submit-hook>
INJECT
  exit 0
fi

# --- feature 브랜치: 프롬프트 저장 ---
PROMPTS_DIR="$CWD/docs/prompts"
mkdir -p "$PROMPTS_DIR"

TIMESTAMP=$(date "+%Y-%m-%d %H:%M")
DATE_SHORT=$(date "+%Y-%m-%d")

SAFE_BRANCH=$(echo "$BRANCH" | tr '/' '-')

# 이 브랜치의 기존 파일 찾기 (NNN-브랜치명.md 패턴)
EXISTING=$(ls "$PROMPTS_DIR"/*-"${SAFE_BRANCH}.md" 2>/dev/null | head -1 || true)

if [ -n "$EXISTING" ]; then
  # 기존 파일에 추가
  FILENAME="$EXISTING"
else
  # 새 파일: 전역 최대 번호 + 1
  LAST_NUM=$(ls "$PROMPTS_DIR"/*.md 2>/dev/null | grep -oE '/[0-9]+' | grep -oE '[0-9]+' | sort -n | tail -1 || echo "0")
  [ -z "$LAST_NUM" ] && LAST_NUM=0
  NEXT_NUM=$(printf "%03d" $((LAST_NUM + 1)))
  FILENAME="${PROMPTS_DIR}/${NEXT_NUM}-${SAFE_BRANCH}.md"

  cat > "$FILENAME" <<EOF
# Prompt History: ${BRANCH}

**브랜치**: \`${BRANCH}\`
**시작일**: ${DATE_SHORT}

---
EOF
fi

# 파일 내 순번 계산
FILE_LAST=$(grep -oE '^## [0-9]+\.' "$FILENAME" 2>/dev/null | tail -1 | grep -oE '[0-9]+' || echo "0")
PROMPT_NUM=$((FILE_LAST + 1))

# 프롬프트 추가
CLEAN_PROMPT=$(echo "$PROMPT" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')

cat >> "$FILENAME" <<EOF

## ${PROMPT_NUM}. ${TIMESTAMP}

${CLEAN_PROMPT}

---
EOF

exit 0
