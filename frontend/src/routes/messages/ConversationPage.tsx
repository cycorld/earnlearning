import { useState, useEffect, useCallback, useRef } from 'react'
import { useParams, Link } from 'react-router-dom'
import { toast } from 'sonner'
import { api, apiFetch, ApiError } from '@/lib/api'
import { formatFileSize, openPrivateFile } from '@/lib/milestoneFiles'
import { useAuth } from '@/hooks/use-auth'
import { useWebSocket } from '@/hooks/use-ws'
import type { DMAttachment, DMMessage, User } from '@/types'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Spinner } from '@/components/ui/spinner'
import { ArrowLeft, Send, Loader2, Paperclip, X } from 'lucide-react'

// #184 DM 첨부 — 서버와 같은 한도(개수/크기/확장자). 서버 검증이 최종 방어선이고 여기는 UX 용.
const ALLOWED_EXT = [
  '.jpg', '.jpeg', '.png', '.gif', '.webp', '.pdf',
  '.doc', '.docx', '.xls', '.xlsx', '.ppt', '.pptx',
  '.zip', '.mp4', '.mp3',
  '.html', '.htm',
]
const MAX_FILES = 4
const MAX_SIZE = 10 * 1024 * 1024
const MAX_TOTAL = 20 * 1024 * 1024

function attachmentURL(id: number): string {
  return `/api/dm/attachments/${id}`
}

// inline 렌더 여부는 서버가 확장자로 판단한다(stored_name). mime 컬럼은 업로더 입력이라 신뢰하지 않는다.
function isImageAttachment(att: DMAttachment): boolean {
  const ext = att.stored_name.toLowerCase().match(/\.[^.]+$/)?.[0] ?? ''
  return ['.jpg', '.jpeg', '.png', '.gif', '.webp'].includes(ext)
}

// 이미지 첨부는 인증 헤더가 필요해 <img src> 로 바로 못 건다 → fetch 후 blob URL.
function DMImage({ attachment }: { attachment: DMAttachment }) {
  const [url, setUrl] = useState('')
  const [failed, setFailed] = useState(false)
  const [attempt, setAttempt] = useState(0) // 재시도 = effect 재실행

  useEffect(() => {
    let objectURL = ''
    let alive = true
    setFailed(false)
    apiFetch(attachmentURL(attachment.id))
      .then((res) => {
        if (!res.ok) throw new Error(String(res.status))
        return res.blob()
      })
      .then((blob) => {
        if (!alive) return
        objectURL = URL.createObjectURL(blob)
        setUrl(objectURL)
      })
      .catch(() => {
        if (alive) setFailed(true)
      })
    return () => {
      alive = false
      if (objectURL) URL.revokeObjectURL(objectURL)
    }
  }, [attachment.id, attempt])

  // 실패해도 파일명과 대안(재시도/다운로드)은 남긴다.
  if (failed) {
    return (
      <div className="mt-1 rounded-lg bg-background/70 px-2 py-1.5 text-xs">
        <p className="truncate">{attachment.filename}</p>
        <p className="opacity-70">이미지를 불러오지 못했습니다.</p>
        <div className="mt-1 flex gap-3">
          <button type="button" className="underline" onClick={() => setAttempt((n) => n + 1)}>
            재시도
          </button>
          <button
            type="button"
            className="underline"
            onClick={() => openPrivateFile(attachmentURL(attachment.id), attachment.filename)}
          >
            다운로드
          </button>
        </div>
      </div>
    )
  }

  if (!url) return null
  return (
    <button
      type="button"
      aria-label={`${attachment.filename} 원본 보기`}
      onClick={() => openPrivateFile(attachmentURL(attachment.id), attachment.filename)}
      className="mt-1 block cursor-pointer"
    >
      <img
        src={url}
        alt={attachment.filename}
        className="max-h-60 max-w-full rounded-lg object-contain"
      />
    </button>
  )
}

// 전송 전 미리보기 칩. 이미지면 썸네일(자기 objectURL 을 스스로 정리).
function FilePreview({
  file,
  onRemove,
  disabled,
}: {
  file: File
  onRemove: () => void
  disabled?: boolean
}) {
  const [thumb, setThumb] = useState('')

  useEffect(() => {
    if (!file.type.startsWith('image/')) return
    const url = URL.createObjectURL(file)
    setThumb(url)
    return () => URL.revokeObjectURL(url)
  }, [file])

  return (
    <div className="flex items-center gap-2 rounded-lg border bg-muted/50 px-2 py-1 text-xs">
      {thumb && <img src={thumb} alt="" className="h-8 w-8 rounded object-cover" />}
      <span className="max-w-32 truncate">{file.name}</span>
      <span className="opacity-60">{formatFileSize(file.size)}</span>
      <button
        type="button"
        aria-label={`${file.name} 제거`}
        onClick={onRemove}
        disabled={disabled}
        className="text-muted-foreground hover:text-foreground disabled:opacity-50"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  )
}

// 서버 envelope 이 아닌 응답(413 등)은 message 가 raw statusText 라 그대로 보여주면 안 된다.
function sendErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.status === 413) return '첨부 용량이 너무 큽니다. 파일 크기를 줄여주세요.'
    if (err.code !== 'UNKNOWN' && err.message) return err.message
  }
  return '메시지를 보내지 못했습니다. 잠시 후 다시 시도해주세요.'
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

export default function ConversationPage() {
  const { userId } = useParams<{ userId: string }>()
  const peerID = Number(userId)
  const { user } = useAuth()
  const [messages, setMessages] = useState<DMMessage[]>([])
  const [peer, setPeer] = useState<{ name: string; avatar_url: string } | null>(null)
  const [loading, setLoading] = useState(true)
  const [sending, setSending] = useState(false)
  const [content, setContent] = useState('')
  const [files, setFiles] = useState<File[]>([])
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const fileRef = useRef<HTMLInputElement>(null)
  const sendingRef = useRef(false)

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

  const handlePick = (e: React.ChangeEvent<HTMLInputElement>) => {
    const picked = Array.from(e.target.files ?? [])
    e.target.value = '' // 같은 파일 다시 고를 수 있게
    if (picked.length === 0) return

    const accepted: File[] = []
    for (const f of picked) {
      const ext = f.name.toLowerCase().match(/\.[^.]+$/)?.[0] ?? ''
      if (!ALLOWED_EXT.includes(ext)) {
        toast.error(`${f.name}: 지원하지 않는 형식입니다.`)
        continue
      }
      if (f.size > MAX_SIZE) {
        toast.error(`${f.name}: 파일은 10MB 이하만 가능합니다.`)
        continue
      }
      accepted.push(f)
    }
    if (accepted.length === 0) return

    const next = [...files]
    let total = files.reduce((s, f) => s + f.size, 0)
    for (const f of accepted) {
      if (next.length >= MAX_FILES) {
        toast.error(`파일은 최대 ${MAX_FILES}개까지 첨부할 수 있습니다.`)
        break
      }
      if (total + f.size > MAX_TOTAL) {
        toast.error('첨부 파일 합계는 20MB 이하만 가능합니다.')
        break
      }
      next.push(f)
      total += f.size
    }
    setFiles(next)
  }

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault()
    const text = content.trim()
    // sending 상태만으로는 같은 렌더에서 Enter 연타 + 클릭이 동시에 통과할 수 있다 → ref 로 동기 잠금.
    if ((!text && files.length === 0) || sendingRef.current) return

    sendingRef.current = true
    setSending(true)
    try {
      let body: unknown
      if (files.length > 0) {
        const fd = new FormData()
        fd.append('receiver_id', String(peerID))
        fd.append('content', text)
        for (const f of files) fd.append('files', f)
        body = fd
      } else {
        body = { receiver_id: peerID, content: text }
      }
      await api.post<DMMessage>('/dm/messages', body)
      // WS echo will add the message to the list
      setContent('')
      setFiles([])
      inputRef.current?.focus()
    } catch (err) {
      // 실패해도 본문/첨부는 그대로 두어 재시도할 수 있게 한다.
      toast.error(sendErrorMessage(err))
    } finally {
      sendingRef.current = false
      setSending(false)
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Spinner />
      </div>
    )
  }

  return (
    <div className="mx-auto flex h-[calc(100dvh-7rem)] max-w-lg flex-col">
      {/* Header */}
      <div className="sticky top-14 z-40 flex items-center gap-3 border-b bg-background p-3">
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
                className={`max-w-[75%] whitespace-pre-wrap break-words rounded-2xl px-3 py-2 text-sm ${
                  isMine
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-muted'
                }`}
              >
                {msg.content}
                {msg.attachments?.map((att) =>
                  isImageAttachment(att) ? (
                    <DMImage key={att.id} attachment={att} />
                  ) : (
                    <button
                      key={att.id}
                      type="button"
                      onClick={() => openPrivateFile(attachmentURL(att.id), att.filename)}
                      className={`mt-1 flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-xs ${
                        isMine ? 'bg-primary-foreground/15' : 'bg-background'
                      }`}
                    >
                      <Paperclip className="h-3.5 w-3.5 shrink-0" />
                      <span className="truncate">{att.filename}</span>
                      <span className="ml-auto shrink-0 opacity-70">
                        {formatFileSize(att.size)}
                      </span>
                    </button>
                  ),
                )}
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
        {files.length > 0 && (
          <div className="mb-2 flex flex-wrap gap-2">
            {files.map((f, i) => (
              <FilePreview
                key={`${f.name}-${i}`}
                file={f}
                disabled={sending}
                onRemove={() => setFiles((prev) => prev.filter((_, j) => j !== i))}
              />
            ))}
          </div>
        )}
        <div className="flex gap-2">
          <input
            ref={fileRef}
            type="file"
            multiple
            data-testid="dm-file-input"
            accept={ALLOWED_EXT.join(',')}
            onChange={handlePick}
            disabled={sending}
            className="hidden"
          />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label="파일 첨부"
            disabled={sending}
            onClick={() => fileRef.current?.click()}
          >
            <Paperclip className="h-4 w-4" />
          </Button>
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
          <Button
            type="submit"
            size="icon"
            aria-label="보내기"
            disabled={sending || (!content.trim() && files.length === 0)}
          >
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
