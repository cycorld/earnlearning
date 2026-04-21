# 109. cycorld LLM 서버 아키텍처 문서 (#108 후속)

**날짜**: 2026-04-21
**태그**: 운영, 문서, LLM, infra

## 배경
#108 작업 시 `/home/cycorld/bin/llama-proxy.ts` (`:8080`) 를 먼저 수정했는데, 실제 요청 경로에 없어 헛수고. 원상복구 후 FastAPI `/home/cycorld/llm-proxy/main.py` (`:8100`) 에 반영해 성공. 같은 실수를 방지하기 위한 다층 문서화.

## 추가
### 서버 (cycorld)
- **`/home/cycorld/llm-proxy/ARCHITECTURE.md`** — 다이어그램, 경로 경고, 변경 의사결정 표, 재시작 cheat sheet, 검증 프로토콜, 변경 이력
- **`/home/cycorld/bin/llama-proxy.ts`** 상단 배너 주석 — "이 파일은 경로 밖. ARCHITECTURE.md 참조"

### LMS repo (이 PR)
- **`docs/LLM_ARCHITECTURE.md`** — LMS 관점 요약 (외부 인프라이지만 LMS 가 의존하므로 docs/ 에 둠)

### Claude memory
- **`memory/reference_cycorld_llm.md`** — 다음 세션에서도 바로 보임. MEMORY.md 인덱스에도 추가.

## 핵심 메시지
> 변경은 거의 모두 `/home/cycorld/llm-proxy/main.py` (FastAPI). `llama-proxy.ts` 는 dead code.

## 미포함 (의도)
- `llama-proxy.service` 자체 비활성화 — 안전 차원에서 일단 살려둠 (도움 안 되지만 해 끼치지도 않음). 정리 시 별도 티켓.
- `llama-qwen.service` 모델 파일명 fix — 실제 동작 중인 llama-server 는 별도 프로세스라 우선순위 낮음.
