import { useEffect, useRef } from 'react'
import { toast } from 'sonner'
import { wsClient } from '@/lib/ws'
import { forceRefresh } from '@/lib/force-refresh'

// 관리자가 POST /api/admin/force-reload 를 호출하면 서버가 WS 로
// event=force_reload 브로드캐스트. 이 훅이 그걸 받아서 카운트다운 후
// 강제 새로고침한다. 사용자는 카운트다운 동안 "취소" 를 눌러 보류할 수 있다.
//
// 관련: #027 (WS force-reload broadcast)

const COUNTDOWN_SECONDS = 5
const TOAST_ID = 'force-reload'

export function useForceReload(): void {
  // StrictMode 이중 마운트 / 다중 구독 방지.
  const timerRef = useRef<number | null>(null)

  useEffect(() => {
    function handler(payload: unknown) {
      // 이미 카운트다운 진행 중이면 무시 (중복 브로드캐스트 대비)
      if (timerRef.current !== null) return

      const data = (payload ?? {}) as { reason?: string }
      const reason = data.reason?.trim()

      let remaining = COUNTDOWN_SECONDS

      const description = () =>
        `${remaining}초 후 자동으로 새로고침됩니다${reason ? ` — ${reason}` : ''}`

      toast('⚠️ 관리자 강제 새로고침', {
        id: TOAST_ID,
        description: description(),
        duration: Infinity,
        action: {
          label: '취소',
          onClick: () => {
            if (timerRef.current !== null) {
              window.clearInterval(timerRef.current)
              timerRef.current = null
            }
            toast.dismiss(TOAST_ID)
          },
        },
      })

      timerRef.current = window.setInterval(() => {
        remaining -= 1
        if (remaining <= 0) {
          if (timerRef.current !== null) {
            window.clearInterval(timerRef.current)
            timerRef.current = null
          }
          void forceRefresh()
          return
        }
        // 남은 시간 갱신 (같은 id 로 재호출하면 description 업데이트)
        toast('⚠️ 관리자 강제 새로고침', {
          id: TOAST_ID,
          description: description(),
          duration: Infinity,
          action: {
            label: '취소',
            onClick: () => {
              if (timerRef.current !== null) {
                window.clearInterval(timerRef.current)
                timerRef.current = null
              }
              toast.dismiss(TOAST_ID)
            },
          },
        })
      }, 1000)
    }

    const unsubscribe = wsClient.on('force_reload', handler)

    return () => {
      unsubscribe()
      if (timerRef.current !== null) {
        window.clearInterval(timerRef.current)
        timerRef.current = null
      }
    }
  }, [])
}
