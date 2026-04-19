---
id: 093
title: 챗봇 service key revoked 자동 복구
priority: high
type: fix
branch: fix/chat-key-auto-reprovision
created: 2026-04-19
---

## 배경
프로덕션에서 챗봇 service key 가 revoked 되어 모든 채팅이 "api key revoked" 로 실패.
지금까진 수동 처리: DB chat_service_config DELETE → backend 재시작 → 자동 재발급.

이미 두 번 발생 (스테이지 1회, 프로덕션 1회).

## 원인 추정
- llm.cycorld.com 의 자동 키 rotation 정책
- 또는 외부에서 수동 revoke

## 수정
1. **ChatComplete / ChatCompleteStream 응답이 403 + "api key revoked" 면**:
   - singleflight 으로 한 번만 재발급 (다중 in-flight 가 동시 재발급 시도 방지)
   - 재발급 성공 시 SetUserKey + 원래 요청 1회 재시도
   - DB chat_service_config 도 새 키로 업데이트
2. **메트릭**: 재발급 횟수 카운트 (#089 stats endpoint 에 노출)
3. **알림**: 재발급 발생 시 admin Slack/이메일 (선택적)

## 확인 필요
- llm.cycorld.com 에서 key revoke 의 실제 패턴 (시간/조건)
- 무한 루프 방지 (재발급된 키도 즉시 revoked 면? 백오프 + 한 번만 시도)
