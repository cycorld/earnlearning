import { useState, useEffect, useCallback, useRef } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import { useWebSocket } from '@/hooks/use-ws'
import type { DMMessage, User } from '@/types'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { ArrowLeft, Send, Loader2 } from 'lucide-react'

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

export default function ConversationPage() {
  const { userId } = useParams<{ userId: string }>()
  const peerID = Number(userId)
  const { user } = useAuth()
  const [messages, setMessages] = useState<DMMessage[]>([])
  const [peer, setPeer] = useState<{ name: string; avatar_url: string } | null>(null)
  const [loading, setLoading] = useState(true)
  const [sending, setSending] = useState(false)
  const [content, setContent] = useState('')
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  // Fetch peer profile
  useEffect(() => {
    if (!peerID) return
    api.get<User>(`/users/${peerID}/profile`)
      .then((data) => setPeer({ name: data.name, avatar_url: data.avatar_url }))
      .catch(() => setPeer({ name: '알 수 없음', avatar_url: '' }))
  }, [peerID])

  // Fetch messages
  const fetchMessages = useCallback(async () => {
    if (!peerID) return
    try {
      const data = await api.get<DMMessage[]>(`/dm/messages/${peerID}?limit=50`)
      setMessages(data ?? [])
    } catch {
      setMessages([])
    } finally {
      setLoading(false)
    }
  }, [peerID])

  useEffect(() => {
    fetchMessages()
  }, [fetchMessages])

  // Mark as read on enter
  useEffect(() => {
    if (peerID) {
      api.put(`/dm/messages/${peerID}/read`).catch(() => {})
    }
  }, [peerID])

  // Scroll to bottom on new messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  // Real-time message reception
  const handleDM = useCallback(
    (data: unknown) => {
      const msg = data as DMMessage
      // Avoid duplicates (sender gets WS echo of own message)
      setMessages((prev) => {
        if (prev.some((m) => m.id === msg.id)) return prev
        if (msg.sender_id === peerID || msg.receiver_id === peerID) {
          if (msg.sender_id === peerID) {
            api.put(`/dm/messages/${peerID}/read`).catch(() => {})
          }
          return [...prev, msg]
        }
        return prev
      })
    },
    [peerID],
  )
  useWebSocket('dm', handleDM)

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault()
    const text = content.trim()
    if (!text || sending) return

    setSending(true)
    try {
      await api.post<DMMessage>('/dm/messages', {
        receiver_id: peerID,
        content: text,
      })
      // WS echo will add the message to the list
      setContent('')
      inputRef.current?.focus()
    } catch {
      // ignore
    } finally {
      setSending(false)
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="mx-auto flex h-[calc(100dvh-7rem)] max-w-lg flex-col">
      {/* Header */}
      <div className="flex items-center gap-3 border-b p-3">
        <Button variant="ghost" size="icon" asChild>
          <Link to="/messages">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <Avatar className="h-8 w-8">
          <AvatarImage src={peer?.avatar_url} />
          <AvatarFallback>{peer?.name?.charAt(0) || '?'}</AvatarFallback>
        </Avatar>
        <span className="text-sm font-medium">{peer?.name}</span>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        {messages.length === 0 && (
          <p className="py-8 text-center text-sm text-muted-foreground">
            첫 메시지를 보내보세요!
          </p>
        )}
        {messages.map((msg) => {
          const isMine = msg.sender_id === user?.id
          return (
            <div
              key={msg.id}
              className={`flex ${isMine ? 'justify-end' : 'justify-start'}`}
            >
              <div
                className={`max-w-[75%] rounded-2xl px-3 py-2 text-sm ${
                  isMine
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-muted'
                }`}
              >
                {msg.content}
                <p
                  className={`mt-1 text-right text-[10px] ${
                    isMine ? 'text-primary-foreground/60' : 'text-muted-foreground'
                  }`}
                >
                  {timeAgo(msg.created_at)}
                </p>
              </div>
            </div>
          )
        })}
        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <form onSubmit={handleSend} className="border-t p-3">
        <div className="flex gap-2">
          <textarea
            ref={inputRef}
            value={content}
            onChange={(e) => setContent(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                handleSend(e)
              }
            }}
            placeholder="메시지를 입력하세요"
            rows={1}
            className="flex-1 resize-none rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
          />
          <Button type="submit" size="icon" disabled={sending || !content.trim()}>
            {sending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Send className="h-4 w-4" />
            )}
          </Button>
        </div>
      </form>
    </div>
  )
}
