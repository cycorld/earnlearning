# 074. 챗봇 /v1/chat/completions 401 수정 — 서비스 키 자동 프로비저닝

**날짜**: 2026-04-19
**태그**: 챗봇, 버그수정, LLM

## 무엇을 했나
`/api/chat/sessions/:id/ask` 가 500 INTERNAL `llm call: chat 401: {"detail":"invalid api key"}` 실패. 원인은 admin key(`admin-*`) 로는 `/v1/chat/completions` 호출이 불가능하다는 것 — 학생 키(`sk-stu-*`) 만 허용. 백엔드가 이 둘을 구분 안 하고 admin key 로 양쪽을 다 호출하고 있었음.

## 수정
1. `llmproxy.Client` 에 `userKey` 필드 추가, `SetUserKey()` 메서드 + chat.go 에서 userKey 우선 사용 (없으면 adminKey fallback)
2. 새 DB 테이블 `chat_service_config` (key-value)
3. 기동 시 `ProvisionServiceKey(ctx, client, configStore)` 호출:
   - 캐시 확인 (`chat_service_config.chatbot_service_key`)
   - 없으면 llm-proxy 에 `chatbot-svc@earnlearning.com` 학생 조회/생성
   - 기존 활성 키 revoke + 신규 발급
   - 평문 키를 config 에 저장
4. 발급된 service key 를 `proxy.SetUserKey()` 주입 → chat 완성

## 왜 이 구조
- 전용 "service 학생" 분리 → 학교 내부 학생 usage 와 **비용 로그가 섞이지 않음**
- DB 캐시 → 재기동 시 키 재발급 없이 재사용 (프록시 쪽 revoke 없는 한 영구)
- env 에 평문 키 저장 안 함 → 관리자 개입 없이 자동 셀프-서빙
