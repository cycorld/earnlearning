# 099. 마크다운 내부 SPA 링크 비활성화 회귀 (#086 over-fix)

**날짜**: 2026-04-19
**태그**: 핫픽스, 프론트엔드, 마크다운, 회귀

## 현상
공지글 (post 102, post 103) 의 `[지원하러 가기](/grant/14)` 링크가 클릭이 안 됐다.
점선 underline span 으로 렌더되어 `유효하지 않은 링크: /grant/14` 라는 툴팁만 떴다.

## 원인
`MarkdownContent.tsx` 의 #086 핫픽스가 너무 광범위했다. #086 은 챗봇이
`/wiki/존재안함` 같은 phantom 경로를 출력하면 SPA 의 catch-all 라우트 (`*` →
`/feed`) 가 메인으로 튕기는 문제를 막기 위해, **모든 비-절대 경로를 비활성화**
했다.

부작용으로 정상 SPA 라우트 (`/grant`, `/feed`, `/wallet`, `/llm` 등) 까지 모두
링크가 죽어버림.

## 해결
알려진 SPA 라우트 prefix 화이트리스트로 분기:

```ts
const SPA_ROUTES = new Set([
  'admin', 'bank', 'changelog', 'company', 'developer', 'exchange',
  'feed', 'grant', 'grants', 'invest', 'llm', 'login', 'market',
  'messages', 'notifications', 'oauth', 'pending', 'post', 'profile',
  'register', 'wallet',
])
```

분기 로직:
- `http(s)`/`mailto`/`tel`/`/uploads/` → 외부 링크 (`target=_blank`)
- `/<known-prefix>/...` → react-router `useNavigate()` 로 SPA 클라이언트 네비
  (`<a href>` 도 유지 → `cmd+click` 으로 새 탭 가능)
- 그 외 비-절대 경로 → 기존 #086 처럼 비활성화 span (phantom 보호 유지)

## 회귀 테스트
`MarkdownContent.test.tsx` 11 케이스:
- `/grant/14`, `/feed`, `/wallet`, `/llm`, `/profile/3` 등 → `<a>` 활성
- `/wiki/없는문서` → 비활성 span (#086 보호)
- `https://...` → 외부 링크
- `/uploads/foo.pdf` → 외부 링크 + download

## 검증
- prod post 102, 103 배포 후 링크 클릭 → grant 14 상세로 이동
- 챗봇 출력의 phantom 경로는 여전히 비활성

## 미포함
- 모든 라우트를 자동 인식 — App.tsx 수정 시 SPA_ROUTES 도 갱신 필요. 라우트
  정의가 자주 바뀌지 않아 수동 동기화로 충분.
