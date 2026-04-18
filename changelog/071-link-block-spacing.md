# 071. 리스트 카드 간격 2차 정리 — Link 가 inline 이라 space-y 가 안 먹던 케이스

**날짜**: 2026-04-18
**태그**: 프론트엔드, 버그수정, CSS, 디자인

## 무엇을 했나
`/grant`, `/company`, `/market`, `/admin/classroom`, `/admin/classroom/:id` 의
카드 리스트에서 카드들이 여백 없이 붙어 있던 문제를 수정. 5개 파일의 `<Link>`
태그에 `className="block"` 을 추가했다.

## 왜 필요했나
#066 에서 `space-y-2` → `space-y-4` 로 올렸는데, **왜 어떤 페이지는 여전히 간격이
없었는가** 가 이 PR 의 진짜 진단.

Tailwind `space-y-*` 유틸리티는 `& > :not([hidden]) ~ :not([hidden])` 선택자로
**형제 요소에 margin-top 을 주는** 방식인데, CSS 스펙상 **margin-top 은 inline
요소에는 적용되지 않는다**.

react-router-dom 의 `<Link>` 는 기본적으로 `<a>` 로 렌더링되고, `<a>` 는 기본
`display: inline`. 그래서 아래 같은 구조에서:

```tsx
<div className="space-y-4">
  {items.map(i => (
    <Link key={i.id} to={...}>
      <Card>...</Card>
    </Link>
  ))}
</div>
```

margin-top 이 Link(inline) 에 안 붙으면서 카드들이 다닥다닥 붙는 모양이 됨.
Card 자체는 block 이지만, 부모 `<a>` 가 inline 이라 inline-level 박스 안에 block
자식이 있는 애매한 상황 — 시각적으로는 세로로 쌓이지만 `space-y` margin 은
inline 인 Link 기준이라 무시됨.

## 어떻게 만들었나
각 `<Link>` 에 `className="block"` (Tailwind `display: block`) 추가. 한 줄 바뀜이
문제를 완전히 해결한다.

### 대상 파일
- `frontend/src/routes/grant/GrantListPage.tsx`
- `frontend/src/routes/company/CompanyListPage.tsx`
- `frontend/src/routes/market/MarketPage.tsx`
- `frontend/src/routes/admin/AdminClassroomPage.tsx`
- `frontend/src/routes/admin/AdminClassroomDetailPage.tsx`

### 제외 (이미 block-level)
- `UserProfilePage.tsx` 의 Link 들은 `className="flex ..."` 또는 `className="block ..."`
  가 이미 붙어 있어 정상 — 역시 flex 컨테이너는 block-level.
- `NotificationsPage.tsx`, `MessagesPage.tsx` 는 children 이 `<Card>` 직접 (div)
  이라 문제 없음.

## 왜 #066 때 못 잡았나
- #066 은 **부모 컨테이너의 `space-y-*` 값**만 훑어서 `space-y-2` 이하를 `space-y-4`
  로 올렸음.
- 하지만 이 문제의 원인은 **자식이 inline 요소**라는 것이지 부모 값이 아니었음.
  `space-y-4` 로 올려도 Link 가 여전히 inline 이면 효과 없음.
- 스테이지 브라우저 검수 때 `/feed`·`/wallet` 같은 "Card 를 직접 나열" 하는 페이지만
  확인했고, "Link 로 감싸서 Card 를 나열" 하는 `/grant`·`/company`·`/market` 은
  놓쳤음.

## 배운 점
- **Tailwind `space-y-*` 는 block-level 형제에만 동작한다.** Link/span 같은 inline
  요소를 children 으로 쓸 땐 반드시 `block` 또는 `flex` 를 함께 지정할 것.
- **원인 진단 vs 증상 완화**: #066 에서 "space-y-2 → space-y-4" 는 증상 완화였고,
  진짜 원인은 inline Link. 이제야 근본 원인까지 내려왔음.
- **검수 스코프 확대**: 리스트 페이지 QA 체크리스트에 "children 의 display
  속성까지 눈으로 확인" 을 추가해야.
