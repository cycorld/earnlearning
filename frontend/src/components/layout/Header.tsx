import { Bell, MessageSquare } from 'lucide-react'
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { useAuth } from '@/hooks/use-auth'
import { useEffect, useState, useCallback } from 'react'
import { api } from '@/lib/api'
import { useWebSocket } from '@/hooks/use-ws'

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
      <div className="flex h-14 items-center justify-between px-4">
        <Link to="/feed" className="flex items-center gap-2">
          <span className="text-lg font-bold text-primary">EarnLearning</span>
          <span className="text-[10px] text-muted-foreground leading-tight mt-0.5">
            {__BUILD_NUMBER__ !== 'dev' ? `#${__BUILD_NUMBER__}` : 'dev'}
            {' · '}
            {__COMMIT_SHA__ !== 'local' ? __COMMIT_SHA__.slice(0, 7) : 'local'}
          </span>
        </Link>
        <div className="flex items-center gap-1">
          <Button variant="ghost" size="icon" asChild className="relative">
            <Link to="/messages">
              <MessageSquare className="h-5 w-5" />
              {dmUnreadCount > 0 && (
                <Badge
                  className="absolute -top-1 -right-1 h-5 min-w-5 bg-coral px-1 text-xs text-white"
                >
                  {dmUnreadCount > 99 ? '99+' : dmUnreadCount}
                </Badge>
              )}
            </Link>
          </Button>
          <Button variant="ghost" size="icon" asChild className="relative">
            <Link to="/notifications">
              <Bell className="h-5 w-5" />
              {unreadCount > 0 && (
                <Badge
                  className="absolute -top-1 -right-1 h-5 min-w-5 bg-coral px-1 text-xs text-white"
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
