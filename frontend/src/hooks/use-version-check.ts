import { useEffect, useRef } from 'react'
import { useLocation } from 'react-router-dom'
import { toast } from 'sonner'

// 빌드 시점에 vite define 으로 주입되는 commit SHA / build number.
// __COMMIT_SHA__ 가 'local' 이면 dev 환경 → 버전 체크 비활성.
// 함수로 감싸서 lazy 평가 — 테스트에서 vi.stubGlobal 로 override 가능하게 함.
function embeddedVersion(): string {
  return `${__BUILD_NUMBER__}-${__COMMIT_SHA__}`
}
function isDevBuild(): boolean {
  return __COMMIT_SHA__ === 'local' || __BUILD_NUMBER__ === 'dev'
}

const POLL_INTERVAL_MS = 60_000 // 60초 폴링
const TOAST_ID = 'version-update-available'

let toastShown = false

/**
 * useVersionCheck — 새 버전이 배포되면 사용자에게 토스트로 알려주고 새로고침 안내한다.
 *
 * 트리거:
 *  1. 마운트 직후 한 번 (현재 EMBEDDED 와 서버 응답 비교)
 *  2. 60초 간격 폴링
 *  3. 라우트 변경 시
 *  4. 탭 focus 시 (visibilitychange visible)
 *  5. window focus event
 *
 * 새 버전 감지 시:
 *  - sonner toast 를 띄움 (action 버튼: "지금 새로고침")
 *  - 사용자가 무시해도 폴링은 계속 — 다음 경로 변경 시 다시 노출
 *
 * 새로고침은 forceRefresh() 가:
 *  - 모든 SW 캐시 삭제
 *  - SW unregister
 *  - 캐시 버스팅 쿼리 + window.location.replace 로 hard reload
 */
export function useVersionCheck() {
  const location = useLocation()
  const intervalRef = useRef<number | null>(null)

  useEffect(() => {
    if (isDevBuild()) return

    // 마운트 시 한 번
    void checkAndNotify()

    // 폴링
    intervalRef.current = window.setInterval(() => {
      void checkAndNotify()
    }, POLL_INTERVAL_MS)

    // 탭이 다시 보일 때
    const onVisibility = () => {
      if (document.visibilityState === 'visible') {
        void checkAndNotify()
      }
    }
    document.addEventListener('visibilitychange', onVisibility)
    window.addEventListener('focus', onVisibility)

    return () => {
      if (intervalRef.current) window.clearInterval(intervalRef.current)
      document.removeEventListener('visibilitychange', onVisibility)
      window.removeEventListener('focus', onVisibility)
    }
  }, [])

  // 라우트 변경 시 가벼운 체크
  useEffect(() => {
    if (isDevBuild()) return
    void checkAndNotify()
  }, [location.pathname])
}

async function checkAndNotify(): Promise<void> {
  const serverVersion = await fetchVersion()
  if (!serverVersion) return
  if (serverVersion === embeddedVersion()) return

  // 새 버전 발견 — 토스트 한 번만 띄움 (이미 보이면 무시)
  if (toastShown) return
  toastShown = true

  toast('🚀 새 버전이 배포됐어요', {
    id: TOAST_ID,
    description: '새로고침해서 최신 화면으로 업데이트하세요.',
    duration: Infinity, // 사용자가 닫기 전까지 유지
    action: {
      label: '지금 새로고침',
      onClick: () => {
        void forceRefresh()
      },
    },
    onDismiss: () => {
      // 닫으면 다시 노출 가능하도록
      toastShown = false
    },
  })

  // SW 도 강제 업데이트 시도
  if ('serviceWorker' in navigator) {
    try {
      const reg = await navigator.serviceWorker.getRegistration()
      if (reg) await reg.update()
    } catch {
      // ignore
    }
  }
}

async function fetchVersion(): Promise<string | null> {
  try {
    const res = await fetch('/api/version', { cache: 'no-store' })
    if (!res.ok) return null
    const data = await res.json()
    const { build_number, commit_sha } = data.data
    return `${build_number}-${commit_sha}`
  } catch {
    return null
  }
}

async function forceRefresh(): Promise<void> {
  // 1. SW 캐시 모두 삭제
  if ('caches' in window) {
    try {
      const cacheNames = await caches.keys()
      await Promise.all(cacheNames.map((name) => caches.delete(name)))
    } catch {
      // ignore
    }
  }

  // 2. SW unregister (다음 로드에서 새로 등록됨)
  if ('serviceWorker' in navigator) {
    try {
      const registrations = await navigator.serviceWorker.getRegistrations()
      await Promise.all(registrations.map((r) => r.unregister()))
    } catch {
      // ignore
    }
  }

  // 3. 캐시 버스팅 쿼리 + hard reload
  const url = new URL(window.location.href)
  url.searchParams.set('_v', Date.now().toString())
  window.location.replace(url.toString())
}
