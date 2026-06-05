# 117. 게시글 상세 페이지에서 댓글이 안 보이던 버그 수정

**날짜**: 2026-06-05
**태그**: 핫픽스, 댓글, frontend, 회귀

## 증상
`/post/:id` 상세 페이지에서 댓글 목록이 **항상 0개** 로 보임. 실제로 댓글이 4개+ 있는 게시글도 "댓글 0개" 로 표시. FeedPage 의 인라인 댓글은 정상.

## 진단
- Backend `GET /api/posts/:id/comments` 응답: `{success, data: {data: [...], pagination}}`
- `api.get` 가 `data.data` 한 번 unwrap → frontend 가 받는 건 `{data: [...], pagination: {...}}` (객체)
- **PostDetailPage.tsx:46** 가 `api.get<Comment[]>` 로 타입 잘못 명시 → `Array.isArray(cs)` false → `setComments([])` → 댓글 0개
- **FeedPage.tsx:336** 는 이미 `PaginatedData<Comment>` 로 받고 `.data` 추출 → 정상. 즉 두 페이지 패턴이 일관되지 않았음.

## Fix
`frontend/src/routes/post/PostDetailPage.tsx`:
```ts
- const cs = await api.get<Comment[]>(`/posts/${id}/comments?...`).catch(() => [])
- setComments(Array.isArray(cs) ? cs : [])
+ const cs = await api
+   .get<PaginatedData<Comment>>(`/posts/${id}/comments?...`)
+   .catch(() => ({ data: [], pagination: {...} }))
+ setComments(Array.isArray(cs?.data) ? cs.data : [])
```

## 회귀 테스트
`frontend/src/routes/post/PostDetailPage.test.tsx` 신규 — 3 tests:
- 백엔드 `{data:[],pagination}` shape 을 unwrap 해서 30개 정확히 카운트
- 빈 배열이면 "댓글 0개"
- API 실패 시 catch fallback (앱 crash X)

이 테스트가 깨지면 학생들이 댓글 못 보는 회귀 → 즉시 알림.

frontend 182 pass · backend smoke + comment 34 pass · vite build OK.

## 영향
- LMS 학생 전원이 게시글 상세 페이지 댓글 못 보던 상황 (피드 인라인은 정상이었음)
- 학생들이 "댓글 어디에 있어요?" 라며 헷갈리던 원인 해소
