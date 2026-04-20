---
id: 099
title: 마크다운 내부 SPA 링크 비활성화 회귀 (#086 over-fix)
priority: high
type: fix
branch: fix/markdown-internal-link
created: 2026-04-19
---

## 현상
공지글의 `[지원하러 가기](/grant/14)` 링크가 클릭 안 됨 (점선 underline span 으로 렌더).

## 원인
`MarkdownContent.tsx:46-60` (#086 fix) 가 모든 비-절대 URL 을 span 처리.
- 의도: 챗봇이 만들어낸 `/wiki/존재안함` 같은 phantom 경로 → catch-all → `/feed` 로 튕기는 문제 방지
- 부작용: 정상 SPA 경로 (`/grant`, `/feed`, `/wallet`, `/llm` 등) 도 모두 비활성화

## 해결
알려진 SPA 라우트 prefix 화이트리스트로 분기:
- 알려진 prefix `/grant`, `/feed`, `/wallet`, ... → react-router Link/navigate 로 SPA 이동
- 알려지지 않은 path → 기존대로 span (regression 안 만들기)

## 회귀 테스트
- `/grant/14` 클릭 → SPA 내 grant 상세로 이동
- `/wiki/없는문서` 같은 phantom → 여전히 비활성화 (#086 케이스)

## 검증
prod post 102, 103 의 `[지원하러 가기]` 링크 클릭 동작 확인
