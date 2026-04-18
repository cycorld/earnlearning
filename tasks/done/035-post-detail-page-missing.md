---
id: 035
title: 피드 포스트 상세 딥링크 페이지(/post/:id) 부재
priority: low
type: feat
branch: feat/post-detail-page
created: 2026-04-18
---

## 배경
#034 전체 이용가이드 작성 중 Playwright 스크린샷 스크립트가 "포스트 상세" 화면을 찍으려고 `/post/:id` 링크를 클릭했으나, **해당 라우트 자체가 존재하지 않음**을 발견.

- `frontend/src/App.tsx` 에 `/post/:id` 라우트 없음
- `FeedPage.tsx` 에서 포스트 클릭 시 `toggleComments(post.id)` 만 호출 (같은 페이지 내 인라인 확장)
- 포스트 상세가 **독립 URL로 존재하지 않아** 알림/외부 링크에서 특정 포스트로 바로 이동 불가

## 왜 필요한가
- **알림 심화**: "A가 당신의 글에 댓글을 달았습니다" 알림에서 해당 포스트로 바로 들어가려면 딥링크가 필요. 현재는 `/feed` 로만 이동 (스크롤 필요).
- **공유성**: 학생들이 인상적인 공시/외주 완료 보고 포스트를 서로 공유하려면 URL이 있어야 함.
- **SEO/교재 연계**: 가이드 매뉴얼에서 "이런 형태로 쓰면 됩니다" 예시로 특정 포스트를 링크로 걸고 싶을 때.

## 제안 스펙
- 라우트: `GET /post/:id` → `PostDetailPage`
- 렌더: 포스트 단일 + 전체 댓글 스레드 펼쳐진 상태 + 공유 버튼
- 기존 `FeedPage` 의 포스트 카드 헤더에 `#id` 링크(또는 `···` 메뉴에 "링크 복사") 추가
- 알림 `reference_type=post` 의 URL 매핑을 `/feed` → `/post/:id` 로 변경 (`NotificationsPage.getReferencePath`)

## 영향
- 기존 기능 비파괴. 새 페이지 추가 + 알림 매핑 변경 1곳.
- 회귀 테스트: 알림 클릭 → 올바른 URL 이동 E2E

## 관련
- 매뉴얼 작성 시 발견 (#034)
- CLAUDE.md "알림 연동 체크리스트" — `reference_type` 맵에 `post` 항목이 이미 있음(`/feed`), 이 티켓으로 실제 상세 페이지로 연결
