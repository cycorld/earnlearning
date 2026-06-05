# 116. LLM 바운티 grant 14 fix + 보상 지급 4명 (#108 후속)

**날짜**: 2026-06-05
**태그**: 바운티, LLM, proxy, 운영, 보안

## 배경
Grant 14 (LLM API 연동 버그바운티, 5명×500k). 첫 1명(Student #325) 은 #108 에서 처리. 남은 4명 신청 검토 + 발견 버그 fix + 보상 지급.

## 발견 버그 → cycorld FastAPI llm-proxy 패치

### Tier 1 — 보안 🔥
- **빈 messages 배열 → 500 + Jinja 템플릿 소스 노출** (정보 누출) — 응답 body 에 서버 내부 Jinja 라인·컬럼·소스 일부 그대로 노출됨

### Tier 2 — 입력 검증 + OpenAI 호환
- max_tokens 음수 → 200 + 1토큰 silent fallback
- role 오타 (예: `"users"`) → 검증 없이 통과 → 모델이 중국어/엉뚱 응답
- image_url 콘텐츠 → upstream 500 `"cannot make GET request"` 내부 메시지 누출
- 에러 응답 포맷 불일치 (FastAPI `{detail}` vs OpenAI `{error:{message,type,code}}`) — OpenAI Python/Node SDK 의 APIError 파싱 깨짐
- 잘못된 API 키 → 401 + 빈 바디 → 학생이 원인 파악 불가

## 적용 (cycorld `/home/cycorld/llm-proxy/main.py`)

### 추가
- `OPENAI_ERR_TYPE_MAP` + `_openai_error_response()` 헬퍼
- 글로벌 `@app.exception_handler(HTTPException)` → 모든 에러를 `{"error":{"message","type","code"}}` 로 변환
- `_validate_chat_body(body)` — 호출 진입점에서 사전 검증
  - messages: 비어있지 않은 list, 각 dict, role ∈ {system,user,assistant,tool,function}
  - max_tokens: 양의 정수, ≤ 131072
  - image_url 콘텐츠 사전 reject (400 + roadmap 안내)
- `_sanitize_upstream_error(data, status_code)` — upstream 5xx 응답의 Jinja 소스/스택 트레이스 leak 패턴 매칭해서 generic 메시지로 교체
- 비-JSON upstream 에러 응답도 OpenAI 포맷으로 wrap

### 변경
- proxy_chat_completions 진입 직후 `_validate_chat_body(body)` 호출
- non-stream 응답 return 부분 sanitize + OpenAI 포맷화

### 백업
- `main.py.bak.20260605-111211`

## 검증 (end-to-end via prod LMS 학생 키)

| # | 시나리오 | Before | After |
|---|---|---|---|
| 1 | 빈 messages | 500 + Jinja 소스 | **400** `{"error":{"message":"messages must be a non-empty array","type":"invalid_request_error"...}}` |
| 2 | max_tokens=-10 | 200 + 1토큰 | **400** `max_tokens must be >= 1` |
| 3 | role="users" | 200 + 중국어 | **400** `messages[0].role must be one of [...], got 'users'` |
| 4 | image_url | 500 cannot make GET | **400** `image_url input is not supported by the current model pipeline...` |
| 5 | 잘못된 키 | 401 빈 바디 | **401** `{"error":{"message":"invalid api key","type":"authentication_error"...}}` |
| 6 | 정상 호출 | 200 | **200** (regression OK), reasoning_content strip 유지 |

## 보상 지급

| AppID | Student (id) | 핵심 발견 | 지급 |
|---|---|---|---|
| 325 | user 9 (#108) | 모델명 오기재 / silent fallback / reasoning_content | 500,000 (#108) |
| 329 | user 9 (※) | CORS 미설정 안내 / 에러 바디 누락 | **500,000** ✅ |
| 338 | user 23 | Vision 500 / max_tokens silent / 에러 포맷 / /v1/models 메타 | **500,000** ✅ |
| 341 | user 14 | 🔥 빈 messages Jinja 소스 노출 / 4 slot 500 / role 검증 누락 등 5건 | **500,000** ✅ |
| 343 | user 10 | 401 빈 바디 / 응답 품질 | **500,000** ✅ |

**Grant 14 = 5/5 슬롯 충족** → 자동 마감.
**합계: 4 × 500,000 = 2,000,000 KRW 지급**.

## 미포함 (다음 티켓 분리)

- 4 slot 초과 → 429 + `Retry-After` (현재 500 또는 무한 큐잉)
- `/v1/models` 메타데이터 확장 (context_length, max_output_tokens, capabilities, pricing)
- Vision 실제 지원 (base64 inline image, mmproj 활용)
- 사용량 조회 `/v1/usage`, `/v1/me` 엔드포인트
- 결제 알림 (월 한도 soft/hard cap, 80%/100% 알림)
- 학생 키 발급 페이지 코드 스니펫 자동 생성

## 영향 평가

- LMS repo 코드 변경 0줄 — cycorld 서버 측만 패치
- 변경된 에러 응답 포맷은 **OpenAI 표준** 으로 통일. OpenAI SDK 사용 학생들의 호환성 ↑
- 기존 클라이언트 (FastAPI `{detail}` 만 보던) 는 새 `{error:{message,...}}` 로 변경됨 — backward-incompatible 이지만 사실상 모든 OpenAI SDK 가 후자 포맷을 기본 처리

## 학습 포인트

- 보안 이슈는 **다음 정기 라운드를 기다리지 말고 즉시 fix**. 4 신청자가 모두 다른 각도에서 같은 류의 입력 검증 부재를 지적한 게 핵심 신호.
- "OpenAI 호환" 표방 시엔 **에러 포맷도 호환** 해야 진정한 호환. FastAPI 기본 `HTTPException → {detail}` 패턴이 가장 큰 함정.
- 사용자 입력 검증은 **upstream 에 보내기 전** 단계에서 — upstream 의 에러 메시지가 종종 내부 구현 디테일 (Jinja, llama.cpp internals) 을 누출.
