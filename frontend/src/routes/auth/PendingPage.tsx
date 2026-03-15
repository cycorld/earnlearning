import { useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/hooks/use-auth'
import { useWebSocket } from '@/hooks/use-ws'
import { Hourglass, LogOut } from 'lucide-react'

export default function PendingPage() {
  const navigate = useNavigate()
  const { user, logout, refreshUser } = useAuth()

  const handleApproved = useCallback(() => {
    refreshUser().then(() => {
      navigate('/feed', { replace: true })
    })
  }, [refreshUser, navigate])

  useWebSocket('user_approved', handleApproved)

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
