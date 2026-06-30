---
id: 109
title: cycorld LLM 서버 아키텍처 문서 — 실수 방지 (#108 후속)
priority: medium
type: chore
branch: chore/cycorld-arch-doc
created: 2026-04-21
---

## 배경
#108 에서 Bug #2/#3 fix 시 `/home/cycorld/bin/llama-proxy.ts` (:8080) 를 먼저 수정했는데, 실제 요청 경로에 없어서 헛수고. 원상복구 후 FastAPI `llm-proxy/main.py` (:8100) 에 반영 성공.

다음 유지보수자(혹은 미래의 나)가 같은 실수 안 하도록 **서버에 아키텍처 문서** + 헷갈리는 포인트 경고를 남긴다.

## 작업
### cycorld:`/home/cycorld/llm-proxy/ARCHITECTURE.md` 신규
- 요청 흐름 다이어그램 (nginx → FastAPI 8100 → llama-server 8099)
- `llama-proxy.ts` 는 **경로 밖** 이라는 명확한 경고
- 변경 시 어디 고쳐야 하는지 의사결정 표 (model 검증, auth, usage 로깅, streaming, stop tokens)
- 재시작 명령 + 로그 경로

### cycorld:`/home/cycorld/bin/llama-proxy.ts` 상단 배너 주석
- "이 파일은 대부분의 경우 경로에 없음" 경고 + ARCHITECTURE.md 참조

### LMS repo (이 PR)
- `docs/DEPLOY.md` 또는 `docs/LLM_ARCHITECTURE.md` 에 cycorld 서버 구조 요약 추가
