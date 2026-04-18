---
id: 076
title: 챗봇 /v1/chat/completions 401 — admin key 로 호출 불가, service key 자동 프로비저닝
priority: high
type: fix
branch: fix/chatbot-service-key
created: 2026-04-19
---

## 버그
스테이지 챗봇 `/api/chat/sessions/:id/ask` → 500 INTERNAL
`llm call: chat 401: {"detail":"invalid api key"}`

## 원인
llm.cycorld.com 의 `admin-*` 키는 `/admin/api/*` 만 쓸 수 있고 `/v1/chat/completions`
는 `sk-stu-*` 키를 요구한다. 현재 백엔드는 `LLM_ADMIN_API_KEY` 하나로 양쪽을 모두
호출하고 있어서 chat 이 401.

## 수정
1. 서버 기동 시 admin key 로:
   - email=`chatbot-svc@earnlearning.com` 학생이 llm-proxy 에 없으면 생성
   - 그 학생에게 sk-stu-* 키 발급
   - 평문 키를 DB `chat_service_config` 테이블에 저장
2. `/v1/chat/completions` 호출은 이 service key 사용
3. DB 에 이미 있으면 재사용

## 테스트
- 재배포 후 backend 로그에 `chatbot service key provisioned` 확인
- 브라우저에서 질문 → 정상 응답
