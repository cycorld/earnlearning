# 095. 위키 slug `/` 포함 path param URL-decode 수정

**날짜**: 2026-04-19
**태그**: 챗봇, 위키, 핫픽스

## 증상
`POST /api/admin/chat/wiki/notion-manuals%2Fwallet/notion-sync` →
`wiki doc not found: notion-manuals%2Fwallet`

## 원인
Echo `c.Param("slug")` 는 `%2F` 를 자동 decode 하지 않음 (path segment 안전성).
우리는 slug 에 `/` 포함되는 케이스 (`notion-manuals/wallet`) 가 정상 → 명시 decode 필요.

## 수정
`AdminGetWikiDoc`, `AdminUpdateWikiDoc`, `AdminSyncNotionOne` 진입 시
`url.PathUnescape(slug)` 추가.
