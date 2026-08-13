import { useState, useEffect, useCallback } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { api } from '@/lib/api'
import type { DMConversation } from '@/types'
import { Card, CardContent } from '@/components/ui/card'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { ArrowLeft, MessageSquare, Plus, Search } from 'lucide-react'
import { useWebSocket } from '@/hooks/use-ws'
import { Spinner } from '@/components/ui/spinner'
import { useAuth } from '@/hooks/use-auth'

interface MessageRecipient {
  user_id: number
  name: string
  department: string
  student_id: string
  avatar_url: string
}

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

export default function MessagesPage() {
  const navigate = useNavigate()
  const { user } = useAuth()
  const [conversations, setConversations] = useState<DMConversation[]>([])
  const [loading, setLoading] = useState(true)
  const [composeOpen, setComposeOpen] = useState(false)
  const [recipients, setRecipients] = useState<MessageRecipient[]>([])
  const [recipientsLoading, setRecipientsLoading] = useState(false)
  const [recipientsError, setRecipientsError] = useState('')
  const [query, setQuery] = useState('')

  const fetchConversations = useCallback(async () => {
    try {
      const data = await api.get<DMConversation[]>('/dm/conversations')
      setConversations(data ?? [])
    } catch {
      setConversations([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchConversations()
  }, [fetchConversations])

  const handleDM = useCallback(() => {
    fetchConversations()
  }, [fetchConversations])

  useWebSocket('dm', handleDM)

  const fetchRecipients = useCallback(async () => {
    if (!user?.active_classroom_id) {
      setRecipientsError('활성 강의실을 먼저 선택해주세요.')
      return
    }
    setRecipientsLoading(true)
    setRecipientsError('')
    try {
      setRecipients(await api.get<MessageRecipient[]>(`/classrooms/${user.active_classroom_id}/students`) ?? [])
    } catch {
      setRecipients([])
      setRecipientsError('학생 목록을 불러오지 못했습니다.')
    } finally {
      setRecipientsLoading(false)
    }
  }, [user?.active_classroom_id])

  useEffect(() => {
    if (composeOpen) void fetchRecipients()
  }, [composeOpen, fetchRecipients])

  const normalizedQuery = query.trim().toLocaleLowerCase()
  const filteredRecipients = recipients.filter((recipient) =>
    [recipient.name, recipient.department, recipient.student_id].some((value) =>
      value?.toLocaleLowerCase().includes(normalizedQuery),
    ),
  )

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Spinner />
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="icon" asChild>
          <Link to="/feed">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <h1 className="flex items-center gap-2 text-xl font-bold">
          <MessageSquare className="h-5 w-5" />
          메시지
        </h1>
        <Button className="ml-auto" size="sm" onClick={() => setComposeOpen(true)}>
          <Plus className="mr-1 h-4 w-4" /> 메시지 쓰기
        </Button>
      </div>

      {conversations.length === 0 ? (
        <Card>
          <CardContent className="p-8 text-center text-sm text-muted-foreground">
            아직 메시지가 없습니다.
            <br />
            다른 사용자의 프로필에서 메시지를 보내보세요.
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {conversations.map((conv) => (
            <Card
              key={conv.peer_id}
              className="cursor-pointer transition-colors hover:bg-muted/50"
              onClick={() => navigate(`/messages/${conv.peer_id}`)}
            >
              <CardContent className="flex items-center gap-3 p-4">
                <Avatar className="h-10 w-10 shrink-0">
                  <AvatarImage src={conv.peer_avatar_url} />
                  <AvatarFallback>{conv.peer_name?.charAt(0) || '?'}</AvatarFallback>
                </Avatar>
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium">{conv.peer_name}</p>
                  <p className="truncate text-xs text-muted-foreground">
                    {conv.last_message || '📎 첨부파일'}
                  </p>
                </div>
                <div className="shrink-0 text-right">
                  <p className="text-[10px] text-muted-foreground">
                    {timeAgo(conv.last_message_at)}
                  </p>
                  {conv.unread_count > 0 && (
                    <Badge variant="destructive" className="mt-1 text-[10px]">
                      {conv.unread_count}
                    </Badge>
                  )}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Dialog open={composeOpen} onOpenChange={setComposeOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>메시지 쓰기</DialogTitle>
            <DialogDescription>현재 강의실에서 메시지를 보낼 학생을 선택하세요.</DialogDescription>
          </DialogHeader>
          <div className="relative">
            <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input aria-label="학생 검색" className="pl-9" placeholder="이름, 학과, 학번 검색" value={query} onChange={(event) => setQuery(event.target.value)} />
          </div>
          <div className="max-h-80 space-y-1 overflow-y-auto">
            {recipientsLoading ? (
              <div className="flex justify-center py-8"><Spinner /></div>
            ) : recipientsError ? (
              <div className="space-y-3 py-6 text-center text-sm text-muted-foreground">
                <p>{recipientsError}</p>
                {user?.active_classroom_id && <Button variant="outline" size="sm" onClick={fetchRecipients}>다시 시도</Button>}
              </div>
            ) : filteredRecipients.length === 0 ? (
              <p className="py-8 text-center text-sm text-muted-foreground">{recipients.length === 0 ? '메시지를 보낼 학생이 없습니다.' : '검색 결과가 없습니다.'}</p>
            ) : filteredRecipients.map((recipient) => (
              <button key={recipient.user_id} type="button" className="flex w-full items-center gap-3 rounded-md p-3 text-left hover:bg-muted" onClick={() => { setComposeOpen(false); navigate(`/messages/${recipient.user_id}`) }}>
                <Avatar className="h-9 w-9"><AvatarImage src={recipient.avatar_url} /><AvatarFallback>{recipient.name.charAt(0) || '?'}</AvatarFallback></Avatar>
                <div className="min-w-0"><p className="text-sm font-medium">{recipient.name}</p><p className="truncate text-xs text-muted-foreground">{[recipient.department, recipient.student_id].filter(Boolean).join(' · ')}</p></div>
              </button>
            ))}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
