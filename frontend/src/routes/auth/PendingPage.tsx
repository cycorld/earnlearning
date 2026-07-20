import { useEffect, useCallback, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/hooks/use-auth'
import { useWebSocket } from '@/hooks/use-ws'
import { api } from '@/lib/api'
import { setToken } from '@/lib/auth'
import type { User } from '@/types'
import { Hourglass, LogOut } from 'lucide-react'

// 승인 대기 화면 폴링 간격 (#167)
const POLL_INTERVAL_MS = 5000

export default function PendingPage() {
  const navigate = useNavigate()
  const { user, logout } = useAuth()

  // 승인 확정 시 SPA 이동이 아니라 전체 새로고침(window.location.replace)으로 앱을
  // 재부팅한다. 그래야 새 approved 토큰으로 WebSocket 을 포함한 모든 연결이
  // 정상 재수립되어 실시간 알림이 승인 직후부터 동작한다 (#167).
  const handleApproved = useCallback(() => {
    window.location.replace('/feed')
  }, [])

  // WebSocket 알림 (승인 시 서버가 이벤트를 보내면 즉시 반응).
  // 현재 pending 유저는 WS 연결이 거부되므로 폴링이 실제 트리거지만,
  // 향후 WS 게이트가 열리면 자동으로 동작하도록 리스너를 유지한다.
  useWebSocket('user_approved', handleApproved)

  // #167 승인 상태 폴링 — 5초 주기 + 탭 복귀 시 즉시 확인.
  // /auth/refresh 는 최신 DB 상태로 새 토큰을 발급하므로,
  // approved 로 바뀌면 UI 이동과 JWT 갱신이 한 번에 해결된다.
  const inFlight = useRef(false)
  const approvedRef = useRef(false)

  useEffect(() => {
    let cancelled = false

    const checkApproval = async () => {
      if (inFlight.current || approvedRef.current || cancelled) return
      inFlight.current = true
      try {
        const result = await api.post<{ token: string; user: User }>(
          '/auth/refresh',
        )
        if (cancelled) return
        if (result.user.status === 'approved') {
          approvedRef.current = true
          setToken(result.token)
          handleApproved()
        }
      } catch {
        // 네트워크 오류·토큰 문제 등은 무시하고 다음 주기에 재시도
      } finally {
        inFlight.current = false
      }
    }

    const interval = setInterval(checkApproval, POLL_INTERVAL_MS)
    const onVisibility = () => {
      if (document.visibilityState === 'visible') checkApproval()
    }
    document.addEventListener('visibilitychange', onVisibility)

    return () => {
      cancelled = true
      clearInterval(interval)
      document.removeEventListener('visibilitychange', onVisibility)
    }
  }, [handleApproved])

  useEffect(() => {
    if (user && user.status === 'approved') {
      navigate('/feed', { replace: true })
    }
  }, [user, navigate])

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-background px-4">
      <div className="flex flex-col items-center gap-6 text-center">
        <Hourglass className="h-16 w-16 text-muted-foreground" />
        <div className="space-y-2">
          <h1 className="text-xl font-semibold">
            관리자 승인을 기다리고 있습니다.
          </h1>
          <p className="text-sm text-muted-foreground">
            승인이 완료되면 자동으로 이동합니다.
          </p>
          <p className="text-sm text-muted-foreground">
            문의:{' '}
            <a
              href={`mailto:${import.meta.env.VITE_CONTACT_EMAIL || 'admin@earnlearning.com'}`}
              className="text-primary hover:underline"
            >
              {import.meta.env.VITE_CONTACT_EMAIL || 'admin@earnlearning.com'}
            </a>
          </p>
        </div>
        <Button variant="outline" onClick={logout} className="gap-2">
          <LogOut className="h-4 w-4" />
          로그아웃
        </Button>
      </div>
    </div>
  )
}
