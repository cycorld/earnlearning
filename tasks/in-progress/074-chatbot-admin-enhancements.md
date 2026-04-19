---
id: 074
title: 챗봇 관리자 UI 향상 — 유저 이름 표시 + 비용 대시보드 + 세션 검색
priority: low
type: feat
branch: feat/chatbot-admin-ui
created: 2026-04-19
---

## 배경
#073 에서 추가한 관리자 대화 조회는 MVP. 다음 개선 항목:

## 스코프
1. **세션 목록에 user 이름 조인** — 현재 `user_id=43` 만 표시, 실제 이름 표시
   (SessionRepository.ListAll SQL 에 users 조인, UserName 필드 추가)
2. **사용량 대시보드** — `chat_usage` 테이블을 읽어 일별 / 학생별 토큰·원화 그래프
   (chart 라이브러리 또는 간단 bar). 월 누적 예산 감시용.
3. **세션 검색** — title / 메시지 내용 키워드로 필터 (기존 FTS5 쓸지, LIKE 로 쓸지)
4. **대화 내보내기** — 특정 세션을 JSON / PDF 로 관리자가 다운로드

## 관련
- #072 SSE 스트리밍과 독립
- #073 Context7 와 독립

## 제외
- 학생 이름 공개는 관리자 한정 (학생끼리는 공개되지 않음)
