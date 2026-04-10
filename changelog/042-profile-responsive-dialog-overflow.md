# 042. 프로필 페이지 반응형 + DB 다이얼로그 모바일 오버플로우 수정

> **날짜**: 2026-04-10
> **태그**: `fix`, `반응형`, `UI`, `프론트엔드`, `학생DB`

## 무엇을 했나요?

스테이지에서 #013 을 써보다가 발견한 두 가지 UI 문제를 고쳤어요:

1. **PC 화면에서 프로필이 너무 좁음**: 데스크톱에서도 모바일 폭(`max-w-lg ≈ 512px`)
   으로 고정되어 한 컬럼이 화면 한가운데에만 표시됐어요. 1920px 모니터에서 양쪽 여백이
   너무 큼.

2. **모바일에서 "접속 정보" 다이얼로그가 깨짐**: DB 생성 후 비밀번호/접속정보 모달의
   `password` (24자), `.env DATABASE_URL` (~100자), `psql 명령어` (긴 한 줄) 가 모달
   폭을 넘어서면서 **복사 버튼이 화면 밖으로 밀려나** 사용 불가.

## 어떻게 고쳤나요?

### 1. 데스크톱 2열 그리드 (`ProfilePage.tsx`)

기존:
```tsx
<div className="mx-auto max-w-lg space-y-4 p-4">
  {/* 모든 카드 단일 컬럼 */}
</div>
```

수정:
```tsx
<div className="mx-auto max-w-lg p-4 lg:max-w-5xl">
  <div className="flex flex-col gap-4 lg:grid lg:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)] lg:items-start lg:gap-4">
    <UserCard />
    <WalletCard />
    <UserDatabasesSection className="lg:col-start-2 lg:row-start-1 lg:row-span-4" />
    <NavCard />
    <LogoutButton />
  </div>
</div>
```

**핵심**:
- 모바일(`<lg`): `flex flex-col gap-4` → 단일 컬럼 (기존 동작 유지)
- 데스크톱(`lg+`): `grid` 2열, 좌측 1fr / 우측 1.2fr (DB 영역에 더 많은 공간)
- DB 섹션은 `lg:col-start-2 lg:row-start-1 lg:row-span-4` 로 우측 컬럼 전체 차지
- 다른 카드들은 자연스럽게 좌측 컬럼에 위에서부터 쌓임
- `minmax(0, *)` 는 그리드 자식이 콘텐츠 폭으로 부풀어 가로 스크롤 생기는 것 차단

### 2. KV 컴포넌트 — flex truncate 수정

기존:
```tsx
<div className="flex items-center gap-2">
  <span className="w-16 shrink-0">{label}</span>
  <code className="flex-1 truncate">{value}</code>  // 안 줄어듦!
  <button>copy</button>
</div>
```

`truncate` (=`overflow-hidden text-overflow-ellipsis whitespace-nowrap`) 가 적용되긴
하지만, **flex 자식의 기본 `min-width: auto`** 때문에 최소 폭이 콘텐츠 폭으로
설정돼서 실제로는 줄어들지 않아요. 부모 flex 컨테이너를 밀어서 가로 스크롤이
생겨요.

수정:
```tsx
<div className="flex min-w-0 items-center gap-2">
  <span className="w-16 shrink-0">{label}</span>
  <code className="min-w-0 flex-1 truncate select-all" title={value}>
    {value}
  </code>
  <button className="shrink-0">copy</button>
</div>
```

- `min-w-0` 추가 (부모 flex 와 자식 code 양쪽) → flex 자식이 콘텐츠 폭 이하로 줄어들 수 있게 함
- `select-all` → 클릭 한 번으로 전체 선택 (드래그 안 해도 됨)
- `title={value}` → 호버 시 전체 값 확인
- 복사 버튼에 `shrink-0` → 항상 우측 노출 보장

### 3. CopyBlock 컴포넌트 — 멀티라인 wrap + 절대 위치 복사 버튼

기존:
```tsx
<div className="flex gap-1 p-2">
  <code className="flex-1 overflow-x-auto whitespace-nowrap">{value}</code>
  <Button>copy</Button>
</div>
```

`overflow-x-auto` + `whitespace-nowrap` → 가로 스크롤. 모바일에서 사용자가 가로
스크롤을 인지하지 못하면 값이 잘려 보임. 복사 버튼은 항상 우측 끝.

수정:
```tsx
<div className="relative rounded border bg-background">
  <pre className="whitespace-pre-wrap break-all p-2 pr-9 font-mono text-[10px] leading-snug">
    <code className="select-all">{value}</code>
  </pre>
  <Button className="absolute top-1 right-1 h-6 w-6 shrink-0 p-0" ...>
    copy
  </Button>
</div>
```

- `whitespace-pre-wrap break-all` → 긴 단일 줄이 모달 폭에서 자동 줄바꿈 (URL 처럼
  공백이 없는 문자열도 강제 래핑)
- `pr-9` → 우측에 여백 확보 (복사 버튼 자리)
- 복사 버튼은 `absolute top-1 right-1` → 항상 우상단 노출, 가로 스크롤 안 발생
- `select-all` → 클릭으로 전체 선택

### 4. 카드 액션 버튼 안정화 (`DatabaseCard`)

```tsx
<div className="flex shrink-0 gap-1">
  <Button>eye</Button>
  <Button>rotate</Button>
  <Button>delete</Button>
</div>
```

`shrink-0` 추가 → 좁은 폭에서 액션 버튼들이 압축되지 않도록.

## 검증

- [x] `npx tsc --noEmit` 통과
- [x] `npm run build` 통과 (dist 생성)
- [x] `npm test` 72 passed
- [ ] 스테이지 배포 후 PC/모바일 양쪽 시각 확인 (이 작업과 함께)

## 배운 점

### 1. flex + truncate 의 함정
`truncate` 만 붙이면 동작 안 함. **flex 자식의 기본 `min-width: auto`** 가
콘텐츠 폭이라서, 자식이 부모를 밀어 가로 스크롤을 만들어요.

해결: 부모 또는 자식에 `min-w-0` 추가 → "이 자식은 콘텐츠 폭보다 작아져도 OK"
라는 신호.

### 2. 가로 스크롤 vs 멀티라인 wrap
긴 코드 스니펫을 보여줄 때:
- **가로 스크롤** (`overflow-x-auto whitespace-nowrap`): 데스크톱에선 OK, 모바일에선
  스크롤바 인지 어려움 + 끝 부분 안 보임
- **멀티라인 wrap** (`whitespace-pre-wrap break-all`): 줄바꿈은 못생겨도 모든 글자가
  보임 + 복사 버튼 위치 안정

복사 가능한 값이라면 wrap 이 모바일 친화적. 시각적 균일성 < 사용성.

### 3. 절대 위치 복사 버튼 패턴
다양한 길이의 값을 보여주면서 복사 버튼을 항상 우상단에 두고 싶을 때:
```css
.relative > pre { padding-right: 2.25rem; }
.relative > button { position: absolute; top: 0.25rem; right: 0.25rem; }
```
패딩으로 텍스트가 버튼 아래로 들어가지 않게 공간 확보.

### 4. CSS Grid 의 `minmax(0, ...)`
그리드 트랙 정의에서 `1fr` 만 쓰면 콘텐츠 폭으로 트랙이 늘어날 수 있어요. 자식이
긴 코드/URL 같은 거 갖고 있으면 그리드 전체가 부풀어요.

`minmax(0, 1fr)` 로 하면 트랙의 최소 폭이 0 으로 강제돼서 콘텐츠가 truncate/wrap
돼요. 안전한 기본값.

## 사용한 AI 프롬프트

```
PC 보기를 우리 기존 화면을 더 넓게 반응형으로 고쳐줘야할거 같아.
그리고 db 생성시 모바일에서 저런현상을 보완해줘.(사용성 해치지 않으면서)
```

(스크린샷 첨부: 모바일에서 모달 안의 .env / psql 줄이 화면 밖으로 삐져나간 모습)
