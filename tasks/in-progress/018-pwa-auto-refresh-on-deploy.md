---
id: 018
title: PWA / 브라우저 자동 업데이트 (배포 시 즉시 반영)
priority: high
type: feat
branch: feat/pwa-auto-refresh
created: 2026-04-10
---

## 배경

스테이지/프로덕션에 새 버전을 배포해도 이미 페이지를 띄워둔 사용자에게는
**완전 새로고침 (Cmd+Shift+R)** 을 하지 않으면 새 코드가 안 보임. 두 가지 원인:

1. **vite-plugin-pwa autoUpdate** 가 새 service worker 를 install 하지만 활성화는
   "모든 탭이 닫힐 때" — 사용자가 탭을 안 닫으면 업데이트 안 됨.
2. 활성 탭의 React 앱은 이미 로드된 JS 번들로 동작 → 새 번들 hash 를 받으려면
   페이지 reload 필요.

## 목표

배포가 일어나면 **온라인인 모든 사용자가** 60초 안에 새 버전을 보게 한다.
일반 브라우저 (PWA 설치 안 한 케이스) 와 **PWA 설치된 케이스 양쪽**.

## 설계

### 1. Service Worker 즉시 활성화
\`vite.config.ts\` 의 PWA workbox 옵션에 추가:
\`\`\`ts
workbox: {
  skipWaiting: true,    // 새 SW 가 install 즉시 activate (waiting 단계 skip)
  clientsClaim: true,   // 활성화 후 즉시 모든 클라이언트(탭)를 새 SW 가 제어
  cleanupOutdatedCaches: true,  // 옛 캐시 정리
  ...
}
\`\`\`

### 2. 버전 폴링 + 새로고침 UI
\`__COMMIT_SHA__\` (vite define 으로 빌드 시점에 인젝션됨) 와 \`/api/version\` 의
응답 \`commit_sha\` 를 비교하는 React 컴포넌트 \`<VersionWatcher>\`:

- 마운트 시: 한 번 체크
- 60초 간격: 폴링
- 탭 focus 시: 즉시 체크
- SHA 가 다르면: \`sonner\` toast 로 "🚀 새 버전이 배포됐어요" + [새로고침] 버튼 표시
  - 5분 동안 무시 시: 자동 새로고침 (선택)
- 새로고침은 \`window.location.reload()\` (브라우저가 \`cache: "reload"\` 로 fetch)

### 3. SW 업데이트도 강제 트리거
앱 마운트 시 + focus 시 \`navigator.serviceWorker.getRegistration()\` 의 \`.update()\` 호출
→ vite-plugin-pwa 가 추가로 polling 하지 않아도 새 SW 다운로드 시도.

### 4. 시각 UX
- 토스트: 우상단, 빨간색 또는 강조 색상
- 본문: "새 버전이 배포됐어요. 새로고침해서 적용해주세요."
- 액션: [지금 새로고침] (primary)
- 닫기: 가능하지만 60초 후 다시 노출

## 작업

- [ ] vite.config.ts: skipWaiting/clientsClaim/cleanupOutdatedCaches 추가
- [ ] src/lib/version-watcher.ts (또는 hook): 폴링 + reload 트리거
- [ ] App.tsx 에 마운트
- [ ] /api/version 응답에 commit_sha 가 이미 있는지 확인 (없으면 추가)
- [ ] vitest 회귀 (mock fetch) — 새 버전 감지 시 toast 호출 확인
- [ ] 스테이지 배포 후 시나리오 테스트:
  - 탭 열어둔 상태에서 새 deploy → 60초 안에 토스트 노출 확인
  - 토스트의 새로고침 버튼 클릭 → 새 commit_sha 적용 확인

## 비-목표

- 데이터 마이그레이션 알림 (DB 스키마 변경 시 알림)
- WebSocket 으로 push (HTTP 폴링이면 충분)
- 다국어 (한국어 only)
- 자동 새로고침 강제 (사용자 선택 우선)
