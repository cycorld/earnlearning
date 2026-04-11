# 047. 서브페이지 네비게이션 헤더 고정 (sticky)

> **날짜**: 2026-04-11
> **태그**: `fix`, `UX`, `네비게이션`

## 무엇을 했나요?

정부과제 상세, 투자 상세, 마켓 상세 등 서브페이지에서 스크롤이 길어지면
"← 과제 목록으로" 같은 뒤로가기 버튼이 화면 밖으로 사라져서
다시 목록으로 돌아가기 힘들었던 문제를 수정했습니다.

## 어떻게 만들었나요?

CSS `position: sticky`를 사용해서 네비게이션 헤더를 화면 상단에 고정했습니다.

메인 Header가 `sticky top-0 z-50 h-14` (56px)이므로,
서브페이지 네비게이션은 그 바로 아래인 `sticky top-14 z-40`으로 설정했습니다.
`bg-background`로 배경색을 지정해서 스크롤 시 콘텐츠가 비치지 않도록 했습니다.

### 적용된 페이지 (7개)
- GrantDetailPage — "과제 목록으로"
- GrantNewPage — "과제 목록으로"
- InvestDetailPage — "투자 목록으로"
- MarketDetailPage — "마켓으로 돌아가기"
- MarketNewPage — "마켓으로 돌아가기"
- BusinessCardPage — "← 명함"
- ConversationPage — 채팅 상대방 헤더

## 배운 점

- `position: sticky`는 부모 요소에 `overflow: hidden`이 없어야 동작합니다.
- `z-index`를 메인 Header보다 낮게 설정해야 겹침 문제가 생기지 않습니다.
- `-mx-4 px-4` 트릭으로 padding 있는 부모 안에서도 배경을 전체 너비로 확장할 수 있습니다.
