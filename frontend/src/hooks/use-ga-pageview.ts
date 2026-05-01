import { useEffect } from 'react'
import { useLocation } from 'react-router-dom'
import { trackPageView } from '@/lib/analytics'

/**
 * BrowserRouter 안에서 mount 되어, react-router 라우트 변경마다 GA page_view 1회 발사.
 * production + 측정 ID 있을 때만 실제 발사 (analytics.ts 내부에서 가드).
 */
export function useGAPageView(): void {
  const location = useLocation()
  useEffect(() => {
    trackPageView(location.pathname + location.search)
  }, [location.pathname, location.search])
}

/** App.tsx 에서 hooks-wrapper 로 mount 하기 위한 컴포넌트. */
export function GAPageViewTracker(): null {
  useGAPageView()
  return null
}
