import { useEffect, useRef, useState, type ReactNode } from 'react'
import { RotateCw } from 'lucide-react'

interface PullToRefreshProps {
  children: ReactNode
  /** 당김 임계값(px). 넘으면 release 시 onRefresh 호출. 기본 80. */
  threshold?: number
  /** 최대 당김 거리(px). 기본 120. */
  maxDistance?: number
  /** 리프레시 콜백. 기본: window.location.reload() */
  onRefresh?: () => void | Promise<void>
}

/**
 * PWA 당겨 내려 새로고침 (#107).
 *
 * 브라우저 기본 pull-to-refresh 는 standalone PWA 에서 동작 안 하므로 직접 구현.
 * 데스크톱에서는 touch 없어 noop.
 *
 * 동작:
 * 1. scrollY === 0 에서 touchstart
 * 2. 아래 방향 drag 감지 → 화면 최상단에 표시기 표시
 * 3. threshold 이상에서 release → onRefresh (기본 reload)
 */
export function PullToRefresh({
  children,
  threshold = 80,
  maxDistance = 120,
  onRefresh,
}: PullToRefreshProps) {
  const [pullDistance, setPullDistance] = useState(0)
  const [refreshing, setRefreshing] = useState(false)
  const startYRef = useRef<number | null>(null)
  const activeRef = useRef(false)

  useEffect(() => {
    // coarse pointer 가 없거나 hoverable (desktop mouse) 면 skip
    const isTouch = window.matchMedia('(pointer: coarse)').matches
    if (!isTouch) return

    const handleStart = (e: TouchEvent) => {
      if (refreshing) return
      if (window.scrollY > 0) return
      startYRef.current = e.touches[0]?.clientY ?? null
      activeRef.current = true
    }

    const handleMove = (e: TouchEvent) => {
      if (!activeRef.current || startYRef.current === null || refreshing) return
      const currentY = e.touches[0]?.clientY ?? 0
      const delta = currentY - startYRef.current
      if (delta <= 0) {
        // 위로 올리면 취소
        setPullDistance(0)
        if (window.scrollY > 0) activeRef.current = false
        return
      }
      // resistance — 멀리 당길수록 점점 느려짐
      const resisted = Math.min(delta * 0.5, maxDistance)
      setPullDistance(resisted)
    }

    const handleEnd = () => {
      if (!activeRef.current) return
      activeRef.current = false
      startYRef.current = null
      if (pullDistance >= threshold && !refreshing) {
        setRefreshing(true)
        setPullDistance(threshold)
        void Promise.resolve(onRefresh?.() ?? window.location.reload()).finally(() => {
          // reload() 가 호출됐다면 이 코드는 실행 안 됨. 커스텀 onRefresh 만 해당.
          setRefreshing(false)
          setPullDistance(0)
        })
      } else {
        setPullDistance(0)
      }
    }

    document.addEventListener('touchstart', handleStart, { passive: true })
    document.addEventListener('touchmove', handleMove, { passive: true })
    document.addEventListener('touchend', handleEnd, { passive: true })
    document.addEventListener('touchcancel', handleEnd, { passive: true })
    return () => {
      document.removeEventListener('touchstart', handleStart)
      document.removeEventListener('touchmove', handleMove)
      document.removeEventListener('touchend', handleEnd)
      document.removeEventListener('touchcancel', handleEnd)
    }
  }, [threshold, maxDistance, onRefresh, pullDistance, refreshing])

  const active = pullDistance > 0 || refreshing
  const progress = Math.min(pullDistance / threshold, 1)
  const rotate = progress * 360

  return (
    <>
      {active && (
        <div
          className="pointer-events-none fixed left-0 right-0 top-0 z-[60] flex justify-center"
          style={{
            transform: `translateY(${Math.max(pullDistance - 40, 0)}px)`,
            transition: refreshing ? 'transform 150ms ease' : undefined,
          }}
        >
          <div
            className="flex h-10 w-10 items-center justify-center rounded-full bg-background shadow-lg ring-1 ring-border"
            style={{ opacity: refreshing ? 1 : Math.max(progress, 0.4) }}
          >
            <RotateCw
              className="h-4 w-4 text-primary"
              style={{
                transform: `rotate(${refreshing ? 0 : rotate}deg)`,
                transition: refreshing ? 'none' : 'transform 80ms linear',
                animation: refreshing ? 'ptr-spin 700ms linear infinite' : undefined,
              }}
            />
          </div>
        </div>
      )}
      <style>{`@keyframes ptr-spin { to { transform: rotate(360deg); } }`}</style>
      {children}
    </>
  )
}
