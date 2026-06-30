---
id: 117
title: 게시글 상세보기 페이지에서 댓글이 안 보이는 버그
priority: high
type: fix
branch: fix/post-detail-comments-shape
created: 2026-06-05
---

## 증상
`/post/:id` 상세 페이지에서 댓글 목록이 항상 0개로 보임 (실제 댓글 4개+ 있어도).

## 재현
1. Post #84 (comment_count=4) 상세 페이지 진입
2. "댓글 0개" 표시 + 빈 영역

## 진단
- Backend `GET /api/posts/:id/comments` 응답: `{success, data: {data: [...], pagination: {...}}}`
- `api.get` 가 `data.data` 한 번 unwrap → frontend 가 받는 건 `{data: [...], pagination}`
- PostDetailPage.tsx:46: `api.get<Comment[]>(...)` 로 type 잘못 → `Array.isArray(cs)` false → `setComments([])` → 댓글 0개 표시
- FeedPage.tsx:336 에선 이미 `PaginatedData<Comment>` 타입으로 받고 `.data` 추출 → 정상 동작

## Fix
PostDetailPage.tsx:46 — `api.get<PaginatedData<Comment>>` 로 변경 + `setComments(p.data ?? [])`

## 회귀 테스트
PostDetailPage 가 별도 테스트 없으면 추가. FeedPage 처럼 mock 으로 comments shape 검증.
