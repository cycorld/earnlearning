# 108. LLM 바운티 #325 후속 — 모델명 오기재 + cycorld proxy 개선

**날짜**: 2026-04-21
**태그**: 바운티, LLM, proxy, 문서

## 배경
grant 14 (LLM API 연동 버그바운티) 첫 제출 — Student-#325, 3개 버그:
1. 문서/공지의 모델명 `claude-opus-4-7` 오기재 (실제는 Qwen3.6)
2. 미지원 모델명 요청 시 silent fallback (에러 안 반환)
3. 응답에 `reasoning_content` 내부 추론 노출

## 수정
### cycorld 서버 (`/home/cycorld/llm-proxy/main.py`, FastAPI)
* `proxy_chat_completions`:
  - `alias is None` 이면 **400 `model '...' is not supported. Allowed: [...]`** 반환
  - `include_reasoning: true` 옵션 추가 (기본 false)
  - non-stream 응답: `choices[].message.reasoning_content` strip
  - stream 응답: 신규 `_strip_reasoning_sse()` 헬퍼로 chunk 단위 strip (안전하게 event 경계 맞을 때만)
* 원본 `main.py` 는 `main.py.bak.20260421-122137` 로 백업
* `systemctl --user restart llm-proxy.service` 로 반영

### LMS 코드 (이 PR)
* `frontend/src/routes/llm/LlmPage.tsx` PricingCard: "Anthropic Claude Opus 4.7 공식 가격" → "실제 모델 Qwen3.6-35B-A3B, 가격은 Anthropic Opus 4.7 기준 환산" 으로 명확화

### prod 데이터 (SQL)
* `posts.content`: id 102(공지), 103(자동 grant 포스트) — `claude-opus-4-7` → `qwen-chat`
* `grants.description`: id 14 — 동일

## 검증 (end-to-end, llm.cycorld.com)
```
# Bug #2
$ curl .../v1/chat/completions -d '{"model":"claude-opus-4-7",...}'
{"detail":"model 'claude-opus-4-7' is not supported. Allowed: ['qwen','qwen-chat','qwen-reasoning']"}

# Bug #3
$ curl .../v1/chat/completions -d '{"model":"qwen-chat","messages":...}'
message keys: ['role', 'content']  # reasoning_content 없음 ✅
```

## 보상
* Student-#325 approved → 500,000 KRW 지급 (transaction 802)

## 미포함 (의도)
* FastAPI MODEL_ALIASES 에 upstream 실제 파일명(`Qwen3.6-35B-A3B-UD-Q3_K_XL.gguf`) 추가 여부 — LLM 클라이언트가 alias 로만 쓰게 유도. 상용 패턴과 일치
* `include_reasoning` 플래그 문서화 — `/llm` 페이지 추후 반영
