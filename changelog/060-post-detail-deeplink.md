---
slug: 060-post-detail-deeplink
title: 포스트 상세 딥링크 페이지 (/post/:id) 추가
date: 2026-04-18
tags: [feat, 피드, 딥링크, 알림, 회귀테스트]
---

# 포스트 상세 딥링크 페이지 (/post/:id) 추가

## 왜 필요했는가

전체 이용가이드 매뉴얼을 작성하면서 발견한 구멍: **포스트를 단일 URL로 공유할 방법이 없었습니다**.

상황별 문제:

- "A가 당신의 글에 댓글을 달았어요" 알림 클릭 → `/feed` 로만 이동. 내 글을 찾으려면 스크롤 지옥
- 인상적인 외주 완료 보고를 친구에게 공유하고 싶음 → 보낼 URL 없음
- 교재/매뉴얼에 "이런 식으로 글을 쓰세요" 예시로 실제 포스트를 링크 걸고 싶음 → 불가능

피드는 전부 `/feed` 한 URL 안에 모여 있었고, 개별 포스트로 들어가는 **딥링크**가 아예 없었어요. #034 가이드 작성 중 이 구조를 깨달아서 티켓 #035로 정리해두었고, 지금 구현합니다.

## 무엇을 했는가

### 1. 백엔드 — `GET /api/posts/:id` 단건 조회

기존 `/api/posts` 는 리스트만 반환했습니다. 단건 조회 엔드포인트를 추가하고, **viewer-specific** 필드(`is_liked`)까지 계산해서 주는 게 핵심.

```go
// persistence/post_repo.go — 리스트 쿼리와 동일 구조로 단일 로우만
func (r *PostRepo) FindPostByIDWithViewer(postID, viewerUserID int) (*post.Post, error) {
    // LEFT JOIN post_likes pl ON pl.post_id = p.id AND pl.user_id = ?
    // → 이 viewer가 좋아요 눌렀는지 즉석에서 계산
}
```

```go
// interfaces/http/handler/post_handler.go
func (h *PostHandler) GetPost(c echo.Context) error {
    // 404 가 아닌 경우도 구분: DB 에러 vs 존재하지 않음
    if p == nil { return 404 }
    return p  // MarshalJSON 이 author/channel 자동 중첩
}
```

라우트 등록: `approved.GET("/posts/:id", h.Post.GetPost, ...)`

### 2. 프론트엔드 — `PostDetailPage`

새 라우트 `/post/:id` 에 연결되는 경량 디테일 뷰. 구성:

- 상단: **← 피드** 버튼 + **링크 복사** 버튼
- 본문 카드: 프로필·채널·시각, 마크다운 렌더된 본문, 태그, 좋아요/댓글 카운트
- 하단: 댓글 목록 + 댓글 작성 폼

**의도적으로 간결한 UX**:
피드 안의 포스트 카드는 편집/삭제/이미지 업로드/스레드 대댓글 등 풍부한 인터랙션이 있지만, 상세 페이지는 **"읽고 반응하기"** 에 집중. 더 많은 기능이 필요하면 "← 피드" 링크로 돌아갑니다.

### 3. 피드 카드의 시각 → 상세 페이지 링크

Twitter 스타일: 포스트 카드의 **작성 시각 텍스트**를 `/post/:id` 링크로 감쌌습니다.

```tsx
<Link to={`/post/${post.id}`} className="hover:underline" title="이 글의 고유 링크">
  {timeAgo(post.created_at)}
</Link>
```

시간은 본래 정적 텍스트라 클릭할 필요가 없어 보이지만, 트위터·메스토돈 등 소셜 서비스는 모두 이 관행을 씁니다. 학생들이 "공유 링크" 를 찾을 때 시간을 건드려보는 게 표준 휴리스틱.

### 4. 알림 클릭 동선 갱신

`NotificationsPage.getReferencePath` 에서 `post`/`posts` reference_type 을 `/feed` → `/post/:id` 로 전환. `refId > 0` 인 경우에만 적용하고, `refId = 0` 인 구버전 알림은 `/feed` 로 폴백.

```ts
case 'post':
case 'posts':
  return refId > 0 ? `/post/${refId}` : '/feed'
```

### 5. 회귀 테스트 2종 (TDD)

`TestPostDetail_Success_ReturnsSinglePostWithAuthor` — 포스트 생성 → 단건 조회 → 응답 필드 검증 + `is_liked` 상태 변화 검증 (좋아요 전/후)

`TestPostDetail_NotFound_Returns404` — 존재하지 않는 ID는 실패

수정 전 코드(라우트 없음)로 돌리면 두 테스트 모두 실패. 수정 후 통과 — Red→Green 직접 검증.

## 배운 점

### "리스트만 있으면 충분하다" 는 함정

피드는 "타임라인"이라 **가장 최근 글부터 쭉 보여주는 리스트 뷰**가 자연스러워 보입니다. 하지만 글 한 편 한 편은 **개체(entity)** 이고, 개체는 **고유 URL** 을 가질 때 비로소 인터넷 시민권을 얻습니다. 공유, 인용, 검색, 알림 링크 — 모두 URL 기반 작동이에요.

교훈: **리스트 엔드포인트를 만들 때마다 "이 요소가 단독으로 참조될 수 있는가?" 를 묻기**. 거의 항상 YES 고, 그러면 단건 엔드포인트도 같이 만드는 게 좋습니다.

### Viewer-specific 필드의 책임

`is_liked` 같은 필드는 "누가 보고 있느냐" 에 따라 달라지는 값입니다. SQL 레벨에서 `LEFT JOIN post_likes ON user_id = ?` 한 줄로 계산해 주면 API는 **무상태** 를 유지할 수 있어요.

이 책임을 반대 방향으로 밀면 (예: 프론트가 "내가 누른 like 목록" 을 별도로 쿼리해서 클라이언트에서 조인) 네트워크 라운드트립이 늘고 race condition 여지가 생깁니다. 백엔드에서 viewer 컨텍스트까지 처리하고 깨끗한 객체를 돌려주는 패턴이 안정적.

### "간결한 UX" 는 중복 구현을 피하는 설계

상세 페이지에서 편집/삭제/스레드 대댓글 같은 풍부한 인터랙션을 "모두 복붙하면" 피드와 코드 중복이 심해져요. 대신 **"여기는 공유·빠른 확인 용도"** 로 성격을 분명히 하고 "더 많은 기능은 피드" 로 자연스럽게 유도하면, 코드도 가볍고 사용자도 혼란 없습니다.

## 사용한 프롬프트

> 2) 포스트 상세는 원래 없었어. 아 그런데 알림 통해 들어오면 필요하겠구나. 상세 페이지 추가해줘야겠다. ux 는 자연스럽게 해줘. 피드에서 모두 확인 가능하긴 하니까.
