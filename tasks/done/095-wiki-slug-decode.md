---
id: 095
title: 위키 slug `/` 포함 시 path param URL-decode 누락 수정
priority: high
type: fix
branch: fix/wiki-slug-decode
created: 2026-04-19
---

## 증상
`POST /api/admin/chat/wiki/notion-manuals%2Fwallet/notion-sync` 호출 시
`wiki doc not found: notion-manuals%2Fwallet` — slug 가 URL-encoded 상태로 사용됨.

## 원인
Echo 의 `c.Param("slug")` 는 path segment 의 `%2F` 를 자동 decode 하지 않음.
이는 라우팅 안전성을 위한 기본 동작이지만 우리는 slug 에 `/` 가 포함되는 케이스가
정상이라 명시적 decode 가 필요.

## 수정
`AdminGetWikiDoc`, `AdminUpdateWikiDoc`, `AdminSyncNotionOne` 에서 `url.PathUnescape(slug)` 추가.
