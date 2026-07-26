import { Bell, MessageSquare } from 'lucide-react'
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { useAuth } from '@/hooks/use-auth'
import { useEffect, useState, useCallback } from 'react'
import { api } from '@/lib/api'
import { useWebSocket } from '@/hooks/use-ws'
import ClassroomSwitcher from './ClassroomSwitcher'

export default function Header() {
  const { user } = useAuth()
  const [unreadCount, setUnreadCount] = useState(0)
  const [dmUnreadCount, setDmUnreadCount] = useState(0)

  const fetchUnread = useCallback(async () => {
    try {
      const data = await api.get<{ unread_count: number }>(
        '/notifications?is_read=false&limit=1',
      )
      setUnreadCount(data.unread_count ?? 0)
    } catch {
      // ignore
    }
  }, [])

  const fetchDMUnread = useCallback(async () => {
    try {
      const data = await api.get<{ unread_count: number }>('/dm/unread-count')
      setDmUnreadCount(data.unread_count ?? 0)
    } catch {
      // ignore
    }
  }, [])

  useEffect(() => {
    if (user) {
      fetchUnread()
      fetchDMUnread()
    }
  }, [user, fetchUnread, fetchDMUnread])

  const handleNotification = useCallback(() => {
    setUnreadCount((prev) => prev + 1)
  }, [])

  const handleDM = useCallback(() => {
    fetchDMUnread()
  }, [fetchDMUnread])

  useWebSocket('notification', handleNotification)
  useWebSocket('dm', handleDM)

  return (
    <header className="sticky top-0 z-50 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      {/* #178 모바일: 로고/아이콘 1줄 + 강의실 스위처 전용 2줄(flex-wrap). 데스크톱: 한 줄 유지. */}
      <div className="flex min-h-14 flex-wrap items-center gap-x-2 px-4 py-1.5 sm:h-14 sm:flex-nowrap sm:py-0">
        <Link to="/feed" className="flex min-w-0 shrink-0 items-center gap-2">
          <img src="/favicon.svg" alt="" aria-hidden className="h-7 w-7" />
          <span className="text-lg font-bold tracking-tight text-primary">EarnLearning</span>
          <span className="text-[10px] text-muted-foreground leading-tight mt-0.5">
            <span>{__BUILD_NUMBER__ !== 'dev' ? `#${__BUILD_NUMBER__}` : 'dev'}</span>
            {/* 커밋 sha 는 모바일 공간 확보를 위해 데스크톱에서만 노출 */}
            <span className="hidden sm:inline">
              {' · '}
              {__COMMIT_SHA__ !== 'local' ? __COMMIT_SHA__.slice(0, 7) : 'local'}
            </span>
          </span>
        </Link>
        <div className="order-last min-w-0 basis-full sm:order-none sm:basis-auto">
          <ClassroomSwitcher />
        </div>
        <div className="ml-auto flex items-center gap-1">
          <Button variant="ghost" size="icon" asChild className="relative h-11 w-11">
            <Link to="/messages" aria-label="메시지">
              <MessageSquare className="h-5 w-5" />
              {dmUnreadCount > 0 && (
                <Badge
                  variant="highlight"
                  className="absolute -top-1 -right-1 h-5 min-w-5 px-1 text-xs"
                >
                  {dmUnreadCount > 99 ? '99+' : dmUnreadCount}
                </Badge>
              )}
            </Link>
          </Button>
          <Button variant="ghost" size="icon" asChild className="relative h-11 w-11">
            <Link to="/notifications" aria-label="알림">
              <Bell className="h-5 w-5" />
              {unreadCount > 0 && (
                <Badge
                  variant="highlight"
                  className="absolute -top-1 -right-1 h-5 min-w-5 px-1 text-xs"
                >
                  {unreadCount > 99 ? '99+' : unreadCount}
                </Badge>
              )}
            </Link>
          </Button>
        </div>
      </div>
    </header>
  )
}
