---
id: 108
title: LLM 바운티 #325 후속 — 모델명 오기재 + proxy 개선 + 보상
priority: high
type: chore
branch: chore/llm-bounty-followup-325
created: 2026-04-21
---

## 배경
grant 14 (LLM API 연동 버그바운티) 첫 제출 — Student-#325 (임서원) 3 bugs.
- **#1 모델명 오기재** (valid): 공지/공고/LLM 페이지 "claude-opus-4-7" 실제로는 Qwen3.6 1개만 있음
- **#2 미지원 모델 silent fallback** (valid, cycorld proxy 개선 필요)
- **#3 reasoning_content 응답 노출** (cycorld proxy 에서 strip 옵션)

## 작업
### LMS (PR)
- `frontend/src/routes/llm/LlmPage.tsx`: 모델명 표시 수정 — 실제 model list API 연동 or 하드코딩 "qwen" 계열로
- **prod 데이터** (admin API 또는 SQL 로 직접):
  - post 102 (공지 LLM API 사용법): claude-opus-4-7 → qwen-chat, 가격도 Qwen 기준으로 (또는 표현만 일반화)
  - grant 14 description: claude-opus-4-7 → qwen-chat

### cycorld 서버 (직접 수정)
- `/home/cycorld/bin/llama-proxy.ts` 확장:
  - model validation: 허용 모델 목록 아닌 경우 400 + 명확한 에러
  - reasoning_content: 응답에서 제거 (request 에 `include_reasoning: true` 없으면)
- 재시작

### 보상
- admin API `/api/admin/grants/14/approve/325` → 500k 자동 지급
