# 044. PWA / 브라우저 자동 업데이트 (배포 시 즉시 반영)

> **날짜**: 2026-04-10
> **태그**: `feat`, `PWA`, `프론트엔드`, `UX`

## 무엇을 했나요?

배포 직후 **모든 사용자가 60초 안에** 새 버전을 보게 만들었어요. 토스트로 "🚀 새 버전이
배포됐어요" 알리고 [지금 새로고침] 버튼 한 번이면 끝. PWA 로 설치한 사용자도 동일.

## 왜 필요했나요?

스테이지 사용 중 사용자가 발견:
> "아까 스테이지에서 프론트엔드 반영된게 완전 새로고침하지 않으니 안나타났었어"

이유:
1. **vite-plugin-pwa autoUpdate** 는 새 SW 를 install 만 하고 활성화는 "모든 탭 닫힐 때"
2. 활성 탭의 React 앱은 이미 로드된 JS 번들로 동작 → 새 번들 hash 받으려면 reload 필요
3. PWA 는 캐시 우선이라 더 끈질김

특히 프로덕션 학생들은 한 번 페이지 띄워두고 며칠 안 닫는 패턴이 흔해서 새 기능을
영원히 못 볼 수도 있었어요.

## 어떻게 만들었나요?

### 1. Service Worker 즉시 활성화 (vite.config.ts)

```ts
VitePWA({
  registerType: 'autoUpdate',
  workbox: {
    skipWaiting: true,        // ← 새 SW install 즉시 activate (waiting skip)
    clientsClaim: true,       // ← 모든 클라이언트 즉시 새 SW 가 제어
    cleanupOutdatedCaches: true, // ← 옛 캐시 자동 삭제
    // ...기존 설정
  }
})
```

이 세 옵션만으로도 새 탭 / 새로고침 시 새 SW 를 강제로 잡지만, 활성 탭이 이미 로드된
JS 번들에는 영향 없음. **그래서 2단계 필요.**

### 2. 버전 폴링 + 토스트 (`use-version-check.ts`)

기존에도 라우트 변경 시 버전 확인하는 hook 이 있었지만 한계가 많았어요:
- ❌ 한 페이지에 머무르면 영영 체크 안 함
- ❌ 첫 렌더에서 baseline 만 잡고 비교 안 함 (낡은 탭 못 잡음)
- ❌ 탭 focus 이벤트 무시
- ❌ 사일런트 reload (사용자가 작성 중인 폼 날아감)

새 동작:
```ts
EMBEDDED_VERSION = `${__BUILD_NUMBER__}-${__COMMIT_SHA__}`  // 빌드 시 vite define

useEffect(() => {
  // 1. 마운트 직후 즉시 체크
  checkAndNotify()

  // 2. 60초 폴링
  setInterval(checkAndNotify, 60_000)

  // 3. 탭 focus / visibilitychange
  document.addEventListener('visibilitychange', onVisible)
  window.addEventListener('focus', onVisible)
}, [])

useEffect(() => {
  // 4. 라우트 변경 시 가벼운 체크
  checkAndNotify()
}, [location.pathname])
```

`checkAndNotify` 는 `/api/version` 응답의 commit_sha 를 임베드된 SHA 와 비교 →
다르면:

```ts
toast('🚀 새 버전이 배포됐어요', {
  id: 'version-update-available',
  description: '새로고침해서 최신 화면으로 업데이트하세요.',
  duration: Infinity,           // 사용자 닫기 전까지 유지
  action: {
    label: '지금 새로고침',
    onClick: () => forceRefresh(),
  },
})
```

### 3. forceRefresh() — 캐시 박멸 + hard reload

```ts
async function forceRefresh() {
  // (1) SW 캐시 모두 삭제
  const cacheNames = await caches.keys()
  await Promise.all(cacheNames.map(name => caches.delete(name)))

  // (2) SW unregister (다음 로드에서 새로 등록)
  const regs = await navigator.serviceWorker.getRegistrations()
  await Promise.all(regs.map(r => r.unregister()))

  // (3) 캐시 버스팅 쿼리 + replace 로 hard reload
  const url = new URL(window.location.href)
  url.searchParams.set('_v', Date.now().toString())
  window.location.replace(url.toString())
}
```

세 단계가 모두 필요한 이유:
- (1) 안 하면 SW 가 다음 fetch 에 캐시된 옛 응답 줌
- (2) 안 하면 unregister 안 된 SW 가 다음 페이지에서 다시 캐시 우선
- (3) 단순 `location.reload()` 도 잘 동작하지만, 쿼리 추가하면 서버 측 ETag/proxy 캐시도 우회

### 4. 세부 동작 디테일

- **`toastShown` 모듈 변수**: 같은 토스트가 60초마다 중첩으로 뜨지 않도록 가드. 사용자가 닫으면 다시 띄울 수 있음
- **`isDevBuild()`**: `__COMMIT_SHA__ === 'local'` 또는 `__BUILD_NUMBER__ === 'dev'` 면 폴링 안 함 (개발 서버에서 의미 없음)
- **lazy 평가**: `embeddedVersion()` / `isDevBuild()` 를 함수로 만들어 테스트에서 globalThis stub 후 호출 가능

### 5. 회귀 테스트 (vitest)

`use-version-check.test.tsx` 3 케이스:
1. 서버 버전 == 임베드 → 토스트 안 뜸
2. 서버 버전 != 임베드 → 토스트 1회 + action 라벨에 "새로고침" 포함
3. fetch 실패 → 토스트 안 뜸 (silent failure)

## 검증

- [x] tsc + build + vitest 75/75 통과 (기존 72 + 신규 3)
- [x] 백엔드 smoke 회귀 없음
- [ ] 스테이지 배포 후 시나리오:
  - 탭 열어둔 채 새 deploy → 60초 안에 토스트 노출
  - [지금 새로고침] 클릭 → 새 commit_sha 로 페이지 갱신

## 배운 점

### 1. PWA autoUpdate 만으론 부족
`registerType: 'autoUpdate'` 는 새 SW 를 자동으로 다운로드 하지만, 활성 탭이
새 코드를 보려면 결국 페이지가 reload 돼야 해요. SW 만으로 끝나는 줄 알면 함정.

### 2. skipWaiting + clientsClaim 의 트레이드오프
새 SW 가 즉시 활성화되면 활성 탭이 일시적으로 옛 JS + 새 SW 의 불일치 상태가
생길 수 있어요. 보통은 안전하지만, 극단 케이스에서 예상 못 한 동작 가능.
사용자 토스트로 reload 권장하는 게 안전망.

### 3. 캐시 박멸은 3단계
SW 캐시, SW 등록, 브라우저 HTTP 캐시 — 셋 다 따로따로 다뤄야 진짜 fresh fetch.
한 단계만 빠뜨려도 옛 콘텐츠가 어딘가에서 살아남아요.

### 4. ES Module hoisting + vi.stubGlobal
ES module `import` 는 파일 상단으로 hoist 돼서 stubGlobal 보다 먼저 평가됨.
모듈 top-level 에서 `__BUILD_NUMBER__` 를 const 로 잡으면 테스트에서 못 바꿈.
함수로 감싸서 lazy 평가하면 stub 가능.

## 사용한 AI 프롬프트

```
아까 스테이지에서 프론트엔드 반영된게 완전 새로고침하지 않으니 안나타났었어.
버전 올라갈 때, 강제 리프레시 할 좋은 방법 없을까?

→ 프러덕션에서 모든 유저가 업데이트 반영을 즉각할 수 있도록. pwa 에서도.
```
