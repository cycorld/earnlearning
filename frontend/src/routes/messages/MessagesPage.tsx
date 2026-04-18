import { useState, useEffect, useCallback } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { api } from '@/lib/api'
import type { DMConversation } from '@/types'
import { Card, CardContent } from '@/components/ui/card'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ArrowLeft, MessageSquare } from 'lucide-react'
import { useWebSocket } from '@/hooks/use-ws'
import { Spinner } from '@/components/ui/spinner'

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
  const [conversations, setConversations] = useState<DMConversation[]>([])
  const [loading, setLoading] = useState(true)

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
        <div className="space-y-2">
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
                    {conv.last_message}
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
    </div>
  )
}
