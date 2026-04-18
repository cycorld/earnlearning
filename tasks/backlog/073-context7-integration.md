---
id: 073
title: Context7 HTTP 게이트웨이 통합 — 공식 라이브러리 문서 검색
priority: medium
type: feat
branch: feat/chatbot-context7
created: 2026-04-19
---

## 배경
#073 (chatbot followups) 에서 DuckDuckGo 기반 `web_search` 를 임시 붙였음. Context7 은
공식 라이브러리 문서에 특화된 MCP/API 서비스 — 코드 예제 품질이 웹 검색보다 훨씬
정확. 운영 중 `web_search` 품질이 아쉬우면 병행 또는 교체.

## 스코프
1. Context7 공식 HTTP API 조사 (`https://context7.com/api/...`) — 엔드포인트 / 인증 / 한도
2. `backend/internal/infrastructure/context7/client.go` — HTTP 클라이언트
3. 새 도구 `context7_library_docs(library, topic)` — 라이브러리별 최신 문서
4. `dev_helper` / `code_review` 스킬에 도구 추가
5. 환경변수 `CONTEXT7_API_KEY` (필요 시) — 없으면 도구 조용히 비활성

## 확인 필요
- Context7 가 한국어 쿼리 지원하나? 영어만이면 내부에서 번역 거치는 레이어 필요한지
- 월간 request 한도 / 가격
- 라이브러리 슬러그 체계 (`react`, `next`, `tanstack/react-query` 등) 조사

## 대안
Context7 통합이 어려우면 Brave Search API (월 2000 쿼리 무료) 로 `web_search` 품질만
개선.
