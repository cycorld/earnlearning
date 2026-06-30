---
id: 116
title: LLM 바운티 grant 14 남은 4명 fix + 보상 지급 (#108 후속)
priority: high
type: chore
branch: chore/llm-bounty-grant14-payout
created: 2026-06-05
---

## 배경
Grant 14 (LLM API 연동 버그바운티, 5명×500k) 첫 1명(#325)은 #108 에서 처리됨.
남은 4명 신청 (#329, #338, #341, #343) 검토 + 발견 버그 수정 + 보상 지급.

## 발견 버그 → cycorld FastAPI llm-proxy 패치

### Tier 1 — 보안
- 🔥 빈 messages 배열 → 500 + Jinja 템플릿 소스 노출 (정보 누출 #341)

### Tier 2 — 입력 검증 + OpenAI 호환
- max_tokens 음수 silent fallback (#341)
- role 오타 silent pass (#341)
- image_url → 500 "cannot make GET" 내부 메시지 누출 (#338)
- 에러 응답 포맷 불일치 ({detail} vs {error:{...}}) (#329, #338, #341)
- 401 invalid key → 빈 바디 (#343)

## 적용 (cycorld /home/cycorld/llm-proxy/main.py)
- `_openai_error_response()` + 글로벌 `HTTPException` 핸들러 → 모든 에러를 OpenAI 표준 `{error:{message,type,code}}` 로 통일
- `_validate_chat_body()` — messages 비어있음·role 잘못·max_tokens 음수·image_url 사전 검증 → 400
- `_sanitize_upstream_error()` — upstream 5xx 의 Jinja 소스/스택 트레이스 → generic 메시지로 sanitize
- backup: main.py.bak.20260605-111211

## 보상 지급
| AppID | Student | 발견 | 상태 |
|---|---|---|---|
| 329 | user 9 | CORS 미설정 / 에러 바디 누락 | approved → 500k |
| 338 | user 23 | Vision 500 / max_tokens silent / 에러 포맷 / /v1/models 메타 | approved → 500k |
| 341 | user 14 | **빈 messages 500 + Jinja 소스 노출** / 4 slot 500 / max_tokens / role | approved → 500k |
| 343 | user 10 | 401 빈 바디 / 응답 품질 | approved → 500k |

Grant 14 = 5/5 슬롯 충족, 자동 마감.

## 미포함 (다음 티켓)
- 4 slot 초과 → 429 + Retry-After (현재 500/큐잉 혼재)
- /v1/models 메타데이터 확장 (context_length, capabilities)
- Vision base64 inline image 지원 (현재 image_url reject)
- 사용량 조회 API `/v1/usage`, `/v1/me`

## 검증
end-to-end 5 시나리오 — 모두 400/401 + OpenAI 포맷 ✅
정상 호출 regression — content 정상, reasoning_content strip 유지 ✅
