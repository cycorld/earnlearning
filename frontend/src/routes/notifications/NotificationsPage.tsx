import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import type { Notification, PaginatedData } from '@/types'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  AtSign,
  Bell,
  CheckCheck,
  Loader2,
  MessageCircle,
  Wallet,
  Building2,
  TrendingUp,
  ShieldCheck,
  Sparkles,
} from 'lucide-react'
import { toast } from 'sonner'

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return '방금'
  if (mins < 60) return `${mins}분 전`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}시간 전`
  const days = Math.floor(hours / 24)
  return `${days}일 전`
}

function getNotifIcon(type: string) {
  switch (type) {
    case 'mention':
      return <AtSign className="h-5 w-5 text-primary" />
    case 'comment':
    case 'new_comment':
    case 'post':
    case 'assignment_graded':
      return <MessageCircle className="h-5 w-5 text-info" />
    case 'wallet':
    case 'transaction':
    case 'reward':
    case 'admin_transfer':
    case 'transfer':
      return <Wallet className="h-5 w-5 text-success" />
    case 'company':
      return <Building2 className="h-5 w-5 text-entity" />
    case 'investment':
    case 'dividend':
      return <TrendingUp className="h-5 w-5 text-highlight" />
    case 'approval':
    case 'admin':
      return <ShieldCheck className="h-5 w-5 text-coral" />
    case 'grant_applied':
    case 'grant_approved':
    case 'grant_closed':
      return <ShieldCheck className="h-5 w-5 text-emerald-500" />
    case 'new_dm':
      return <MessageCircle className="h-5 w-5 text-teal-500" />
    case 'job_applied':
    case 'job_accepted':
    case 'job_work_done':
    case 'job_completed':
    case 'job_cancelled':
    case 'job_disputed':
      return <Building2 className="h-5 w-5 text-indigo-500" />
    case 'disclosure_approved':
    case 'disclosure_rejected':
      return <Building2 className="h-5 w-5 text-teal-500" />
    case 'proposal_started':
    case 'proposal_closed':
      return <Building2 className="h-5 w-5 text-fuchsia-500" />
    case 'liquidation_payout':
      return <Wallet className="h-5 w-5 text-warning" />
    case 'llm_billed':
      return <Sparkles className="h-5 w-5 text-highlight" />
    case 'milestone_approved':
    case 'milestone_rejected':
      return <ShieldCheck className="h-5 w-5 text-emerald-500" />
    default:
      return <Bell className="h-5 w-5 text-muted-foreground" />
  }
}

function getReferencePath(refType: string, refId: number): string | null {
  switch (refType) {
    case 'post':
    case 'posts':
      return refId > 0 ? `/post/${refId}` : '/feed'
    case 'assignment':
    case 'submission':
      return '/feed'
    case 'company':
      return `/company/${refId}`
    case 'investment':
      return `/invest/${refId}`
    case 'dividend':
      return '/invest'
    case 'transaction':
    case 'wallet':
    case 'admin_transfer':
      return '/wallet'
    case 'loan':
      return '/bank'
    case 'job':
    case 'freelance_job':
      return `/market/${refId}`
    case 'grant':
      return `/grant/${refId}`
    case 'proposal':
      // #106: 챗봇 학생 제안 알림 → admin 전용 페이지. 일반 학생은 fallback.
      // shareholder proposal (주주총회) 도 reference_type=proposal 이라 충돌 가능 — 추후 분리 권장.
      return refId > 0 ? `/admin/proposals` : '/admin/proposals'
    case 'dm':
      return `/messages/${refId}`
    case 'user':
      return `/profile/${refId}`
    case 'milestone':
      // #119 학생 본인은 /milestones 로, admin 은 동일 알림에서 매트릭스로 이동하길 원할 수 있음.
      // 현재는 학생용 경로로 통일.
      return '/milestones'
    default:
      return null
  }
}

export default function NotificationsPage() {
  const navigate = useNavigate()
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [loading, setLoading] = useState(true)
  const [markingAll, setMarkingAll] = useState(false)
  // #132 멘션 탭: 'all' | 'mention'
  const [tab, setTab] = useState('all')

  const fetchNotifications = useCallback(async () => {
    setLoading(true)
    try {
      const typeParam = tab === 'mention' ? '&type=mention' : ''
      const data = await api.get<PaginatedData<Notification>>(
        `/notifications?page=1&limit=20${typeParam}`,
      )
      setNotifications(data?.data ?? [])
    } catch {
      setNotifications([])
    } finally {
      setLoading(false)
    }
  }, [tab])

  useEffect(() => {
    fetchNotifications()
  }, [fetchNotifications])

  const handleMarkRead = async (notif: Notification) => {
    if (!notif.is_read) {
      try {
        await api.put(`/notifications/${notif.id}/read`)
        setNotifications((prev) =>
          prev.map((n) => (n.id === notif.id ? { ...n, is_read: true } : n)),
        )
      } catch {
        // ignore
      }
    }

    const path = getReferencePath(notif.reference_type, notif.reference_id)
    if (path) {
      // #132 anchor — 댓글 멘션이면 /post/3#comment-12 로 이동해 해당 댓글로 스크롤
      navigate(notif.anchor ? `${path}#${notif.anchor}` : path)
    }
  }

  const handleMarkAllRead = async () => {
    setMarkingAll(true)
    try {
      await api.put('/notifications/read-all')
      setNotifications((prev) => prev.map((n) => ({ ...n, is_read: true })))
      toast.success('모든 알림을 읽음 처리했습니다.')
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : '처리에 실패했습니다.'
      toast.error(message)
    } finally {
      setMarkingAll(false)
    }
  }

  const unreadCount = notifications.filter((n) => !n.is_read).length

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Spinner />
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h1 className="text-lg font-semibold">알림</h1>
          {unreadCount > 0 && (
            <Badge variant="destructive" className="text-xs">
              {unreadCount}
            </Badge>
          )}
        </div>
        {unreadCount > 0 && (
          <Button
            variant="ghost"
            size="sm"
            onClick={handleMarkAllRead}
            disabled={markingAll}
            className="gap-1 text-xs"
          >
            {markingAll ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <CheckCheck className="h-4 w-4" />
            )}
            모두 읽음
          </Button>
        )}
      </div>

      {/* #132 멘션 탭 */}
      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="all">전체</TabsTrigger>
          <TabsTrigger value="mention" className="gap-1">
            <AtSign className="h-3.5 w-3.5" />
            멘션
          </TabsTrigger>
        </TabsList>
      </Tabs>

      {notifications.length === 0 ? (
        <div className="flex flex-col items-center py-12 text-muted-foreground">
          <Bell className="mb-2 h-10 w-10" />
          <p className="text-sm">
            {tab === 'mention' ? '멘션 알림이 없습니다.' : '알림이 없습니다.'}
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {notifications.map((notif) => (
            <Card
              key={notif.id}
              className={`cursor-pointer transition-colors hover:bg-accent/30 ${
                !notif.is_read ? 'border-primary/30 bg-primary/5' : ''
              }`}
              onClick={() => handleMarkRead(notif)}
            >
              <CardContent className="flex items-start gap-3 p-4">
                <div className="mt-0.5 shrink-0">
                  {getNotifIcon(notif.notif_type)}
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-start justify-between gap-2">
                    <p className="text-sm font-medium">{notif.title}</p>
                    {!notif.is_read && (
                      <div className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-primary" />
                    )}
                  </div>
                  <p className="mt-0.5 text-xs text-muted-foreground line-clamp-2">
                    {notif.body}
                  </p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {timeAgo(notif.created_at)}
                  </p>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
