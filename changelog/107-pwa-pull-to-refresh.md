# 107. PWA 당겨 내려 리프레시 (pull-to-refresh)

**날짜**: 2026-04-20
**태그**: PWA, 프론트엔드, UX

## 배경
PWA standalone 모드 (홈 화면에 설치해 쓰는 경우) 는 브라우저 기본 pull-to-refresh 가 비활성화 → 학생이 새로고침하려면 URL 바 접근 또는 앱 껐다 켜야 하는 불편함.

## 추가
### `PullToRefresh` 컴포넌트 (`frontend/src/components/PullToRefresh.tsx`)
- 재사용 가능한 wrapper. touch events (`touchstart/move/end/cancel`) 로 직접 구현
- 조건: `pointer: coarse` 미디어쿼리로 터치 기기만 활성, 데스크톱 mouse noop
- 동작 흐름:
  1. `window.scrollY === 0` 에서 `touchstart`
  2. 아래 방향 drag → 임계값까지 저항감 있게 (0.5x 배율) 당겨짐
  3. 임계값 이상(기본 80px) 에서 release → `onRefresh()` 호출 (기본 `window.location.reload()`)
  4. 표시기: 상단 중앙에 회전하는 `RotateCw` 아이콘 + 당김에 비례해 opacity/회전
- Props: `threshold`, `maxDistance`, `onRefresh` 모두 선택

### `MainLayout.tsx` 래핑
모든 인증 후 페이지에 자동 적용. 별도 opt-in 불필요.

## 미포함 (의도)
- react-query 기반 invalidate — 프로젝트가 react-query 안 씀. 단순 reload 가 SW cache 덕분에 충분히 빠름
- 페이지별 커스텀 refresh — 필요 시 각 페이지에서 `<PullToRefresh onRefresh={...}>` 로 override
- visionOS / desktop 지원 — 터치 기기 전용
