import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { MessageCircle, Paperclip, Send, Sparkles, Trash2, X, ZapOff, Zap } from 'lucide-react'
import { toast } from 'sonner'

import { api, ApiError } from '@/lib/api'
import { getToken } from '@/lib/auth'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { MarkdownContent } from '@/components/MarkdownContent'
import { useAuth } from '@/hooks/use-auth'

interface Skill {
  id: number
  slug: string
  name: string
  description: string
  admin_only: boolean
}

interface ToolCall {
  id: string
  name: string
  raw_args?: string
}

interface Message {
  id: number
  role: 'user' | 'assistant' | 'system' | 'tool'
  content: string
  tool_calls?: ToolCall[]
  tool_call_id?: string
  attachments?: string[] // #106 학생 첨부 이미지 URL
  created_at: string
}

interface Session {
  id: number
  title: string
  active_skill_id?: number
  created_at: string
  last_message_at: string
  messages?: Message[]
  active_skill?: Skill
}

interface AskResponse {
  message: Message
  tool_logs?: Message[]
}

type Mode = 'fast' | 'deep'

interface StreamEvent {
  type: 'tool_call' | 'tool_result' | 'text_delta' | 'done' | 'error' | 'close' | 'queued'
  delta?: string
  tool_name?: string
  tool_id?: string
  tool_args?: string
  tool_content?: string
  message_id?: number
  tokens?: number
  error?: string
  queue_waiting?: number
}

interface StreamHandlers {
  onToolCall?: (ev: StreamEvent) => void
  onToolResult?: (ev: StreamEvent) => void
  onTextDelta?: (delta: string) => void
  onError?: (err: string) => void
  onDone?: (ev: StreamEvent) => void
  onQueued?: (waiting: number) => void
}

// streamAsk — POST /chat/sessions/:id/ask/stream 을 fetch + ReadableStream 으로 소비.
// EventSource 는 GET 만 지원하므로 fetch 를 직접 사용.
async function streamAsk(
  sessionID: number,
  message: string,
  mode: Mode,
  skillSlug: string,
  attachments: string[],
  handlers: StreamHandlers,
): Promise<void> {
  const token = getToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'Accept': 'text/event-stream',
  }
  if (token) headers['Authorization'] = `Bearer ${token}`

  const resp = await fetch(`/api/chat/sessions/${sessionID}/ask/stream`, {
    method: 'POST',
    headers,
    body: JSON.stringify({
      message,
      mode,
      skill_slug: skillSlug || undefined,
      attachments: attachments.length > 0 ? attachments : undefined,
    }),
  })
  if (!resp.ok) {
    const text = await resp.text().catch(() => '')
    throw new Error(`HTTP ${resp.status}: ${text || resp.statusText}`)
  }
  const reader = resp.body?.getReader()
  if (!reader) throw new Error('스트리밍을 지원하지 않는 환경입니다.')
  const decoder = new TextDecoder()
  let buffer = ''
  while (true) {
    const { value, done } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    // SSE event boundary: \n\n
    let idx = buffer.indexOf('\n\n')
    while (idx !== -1) {
      const event = buffer.slice(0, idx)
      buffer = buffer.slice(idx + 2)
      processSseEvent(event, handlers)
      idx = buffer.indexOf('\n\n')
    }
  }
}

function processSseEvent(event: string, handlers: StreamHandlers): void {
  for (const line of event.split('\n')) {
    if (!line.startsWith('data:')) continue
    const payload = line.slice(5).trim()
    if (!payload) continue
    let parsed: StreamEvent
    try {
      parsed = JSON.parse(payload) as StreamEvent
    } catch {
      continue
    }
    switch (parsed.type) {
      case 'tool_call':
        handlers.onToolCall?.(parsed)
        break
      case 'tool_result':
        handlers.onToolResult?.(parsed)
        break
      case 'text_delta':
        if (parsed.delta) handlers.onTextDelta?.(parsed.delta)
        break
      case 'queued':
        handlers.onQueued?.(parsed.queue_waiting ?? 0)
        break
      case 'error':
        handlers.onError?.(parsed.error || '알 수 없는 오류')
        break
      case 'done':
        handlers.onDone?.(parsed)
        break
      case 'close':
        // end of stream — caller exits read loop on EOF
        break
    }
  }
}

export default function ChatDock() {
  const { user } = useAuth()
  const [open, setOpen] = useState(false)
  const [session, setSession] = useState<Session | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [sending, setSending] = useState(false)
  const [mode, setMode] = useState<Mode>('fast')
  const [skills, setSkills] = useState<Skill[]>([])
  const [activeSkillSlug, setActiveSkillSlug] = useState<string>('')
  const [queueWaiting, setQueueWaiting] = useState(0)
  const [attachments, setAttachments] = useState<string[]>([]) // #106 첨부 이미지 URL
  const [uploadingFile, setUploadingFile] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const scrollRef = useRef<HTMLDivElement>(null)

  const isAdmin = user?.role === 'admin'
  const isApproved = user?.status === 'approved'

  const currentSkill = useMemo(
    () => skills.find((s) => s.slug === activeSkillSlug) ?? null,
    [skills, activeSkillSlug],
  )

  // Load skills once when opened
  useEffect(() => {
    if (!open || skills.length > 0) return
    api.get<Skill[]>('/chat/skills')
      .then((items) => {
        setSkills(items ?? [])
        if (items && items.length > 0 && !activeSkillSlug) {
          // 기본: general_ta, 없으면 첫 번째
          const def = items.find((s) => s.slug === 'general_ta') ?? items[0]
          setActiveSkillSlug(def.slug)
        }
      })
      .catch(() => {
        // 챗봇 비활성 상태면 조용히 무시
      })
  }, [open, skills.length, activeSkillSlug])

  // Load or create session when opened
  const ensureSession = useCallback(async () => {
    if (session) return session
    setLoading(true)
    try {
      const s = await api.post<Session>('/chat/sessions', { skill_slug: activeSkillSlug || undefined })
      setSession(s)
      setMessages([])
      return s
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        // 백엔드에 챗봇 라우트 없음 (LLM_ADMIN_API_KEY 미설정)
        toast.error('챗봇 서비스가 아직 활성화되지 않았습니다.')
      } else {
        toast.error(err instanceof Error ? err.message : '세션 생성 실패')
      }
      return null
    } finally {
      setLoading(false)
    }
  }, [session, activeSkillSlug])

  const handleFilePick = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files ?? [])
    e.target.value = '' // reset so picking the same file again triggers change
    if (files.length === 0) return
    setUploadingFile(true)
    try {
      const newURLs: string[] = []
      for (const f of files) {
        if (!f.type.startsWith('image/')) {
          toast.error(`이미지 파일만 첨부할 수 있어요: ${f.name}`)
          continue
        }
        if (f.size > 5 * 1024 * 1024) {
          toast.error(`이미지가 너무 큽니다 (5MB 이하): ${f.name}`)
          continue
        }
        const fd = new FormData()
        fd.append('file', f)
        const token = getToken()
        const resp = await fetch('/api/upload', {
          method: 'POST',
          headers: token ? { Authorization: `Bearer ${token}` } : {},
          body: fd,
        })
        if (!resp.ok) throw new Error(`업로드 실패: ${resp.status}`)
        const j = await resp.json()
        const url = j?.data?.url
        if (typeof url === 'string') newURLs.push(url)
      }
      if (newURLs.length > 0) setAttachments((prev) => [...prev, ...newURLs])
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '이미지 업로드 실패')
    } finally {
      setUploadingFile(false)
    }
  }

  const removeAttachment = (url: string) => {
    setAttachments((prev) => prev.filter((u) => u !== url))
  }

  const send = async () => {
    const text = input.trim()
    if ((!text && attachments.length === 0) || sending) return
    const s = await ensureSession()
    if (!s) return

    // Optimistic user message
    const optimisticId = -Date.now()
    const userMsg: Message = {
      id: optimisticId,
      role: 'user',
      content: text,
      attachments: attachments.length > 0 ? [...attachments] : undefined,
      created_at: new Date().toISOString(),
    }
    // assistant placeholder — streaming 으로 채워짐
    const assistantId = -Date.now() - 1
    const assistantPlaceholder: Message = {
      id: assistantId,
      role: 'assistant',
      content: '',
      created_at: new Date().toISOString(),
    }
    setMessages((prev) => [...prev, userMsg, assistantPlaceholder])
    const sentAttachments = [...attachments]
    setInput('')
    setAttachments([])
    setSending(true)

    try {
      await streamAsk(s.id, text, mode, activeSkillSlug, sentAttachments, {
        onToolCall: (ev) => {
          setMessages((prev) => {
            // 기존 placeholder 의 tool_calls 에 추가
            return prev.map((m) => {
              if (m.id !== assistantId) return m
              const tc = { id: ev.tool_id || '', name: ev.tool_name || '' }
              return { ...m, tool_calls: [...(m.tool_calls ?? []), tc] }
            })
          })
        },
        onToolResult: (ev) => {
          // tool 메시지를 placeholder 직전에 끼워넣음
          const toolMsg: Message = {
            id: -Date.now() - Math.random(),
            role: 'tool',
            content: ev.tool_content || '',
            tool_call_id: ev.tool_id,
            created_at: new Date().toISOString(),
          }
          setMessages((prev) => {
            const idx = prev.findIndex((m) => m.id === assistantId)
            if (idx === -1) return [...prev, toolMsg]
            return [...prev.slice(0, idx), toolMsg, ...prev.slice(idx)]
          })
        },
        onTextDelta: (delta) => {
          if (queueWaiting > 0) setQueueWaiting(0)
          setMessages((prev) =>
            prev.map((m) =>
              m.id === assistantId ? { ...m, content: m.content + delta } : m,
            ),
          )
        },
        onQueued: (waiting) => {
          setQueueWaiting(waiting)
        },
        onError: (err) => {
          toast.error(err)
          setMessages((prev) =>
            prev.map((m) =>
              m.id === assistantId
                ? { ...m, content: m.content || `❌ ${err}` }
                : m,
            ),
          )
        },
      })
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '답변을 받지 못했습니다.')
    } finally {
      setSending(false)
      setQueueWaiting(0)
    }
  }

  const clearSession = () => {
    if (!window.confirm('이 대화를 삭제할까요? 서버에도 영구 삭제됩니다.')) return
    if (!session) {
      setMessages([])
      return
    }
    api.del(`/chat/sessions/${session.id}`)
      .then(() => {
        setSession(null)
        setMessages([])
        toast.success('대화가 삭제되었습니다.')
      })
      .catch((err) => toast.error(err instanceof Error ? err.message : '삭제 실패'))
  }

  // Auto-scroll on new messages
  useEffect(() => {
    if (!scrollRef.current) return
    scrollRef.current.scrollTop = scrollRef.current.scrollHeight
  }, [messages, sending])

  if (!isApproved) return null

  // FAB (closed state)
  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className={cn(
          'fixed bottom-[calc(5rem_+_env(safe-area-inset-bottom))] right-4 z-50 flex h-12 w-12 items-center justify-center rounded-full',
          'bg-primary text-primary-foreground shadow-[0_4px_0_0_var(--primary-shadow)]',
          'transition-transform hover:scale-105 active:translate-y-[2px] active:shadow-[0_2px_0_0_var(--primary-shadow)]',
          'sm:bottom-6',
        )}
        aria-label="챗봇 조교 열기"
      >
        <MessageCircle className="h-5 w-5" />
      </button>
    )
  }

  return (
    <div
      className={cn(
        'fixed inset-x-0 bottom-0 z-50 flex h-[82vh] flex-col border-t bg-background shadow-xl',
        'sm:right-4 sm:bottom-6 sm:left-auto sm:h-[600px] sm:w-[420px] sm:rounded-2xl sm:border',
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between border-b p-3">
        <div className="flex items-center gap-2">
          <Sparkles className="h-4 w-4 text-highlight" />
          <h2 className="text-sm font-semibold">챗봇 조교</h2>
          {currentSkill && (
            <span className="rounded-full bg-muted px-2 py-0.5 text-[10px] text-muted-foreground">
              {currentSkill.name}
            </span>
          )}
        </div>
        <div className="flex items-center gap-1">
          {messages.length > 0 && (
            <button
              onClick={clearSession}
              className="rounded p-1.5 text-muted-foreground hover:bg-muted"
              aria-label="대화 삭제"
            >
              <Trash2 className="h-4 w-4" />
            </button>
          )}
          <button
            onClick={() => setOpen(false)}
            className="rounded p-1.5 text-muted-foreground hover:bg-muted"
            aria-label="닫기"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Skill + mode selector */}
      <div className="flex items-center gap-2 border-b bg-muted/30 px-3 py-2 text-xs">
        <select
          value={activeSkillSlug}
          onChange={(e) => setActiveSkillSlug(e.target.value)}
          className="flex-1 rounded border bg-background px-2 py-1 text-xs"
        >
          {skills.map((s) => (
            <option key={s.slug} value={s.slug}>
              {s.name}
              {s.admin_only ? ' (관리자)' : ''}
            </option>
          ))}
        </select>
        {!isAdmin && (
          <button
            onClick={() => setMode(mode === 'fast' ? 'deep' : 'fast')}
            className={cn(
              'flex items-center gap-1 rounded border px-2 py-1 text-[11px] transition-colors',
              mode === 'deep'
                ? 'border-highlight bg-highlight/10 text-highlight'
                : 'border-border text-muted-foreground hover:bg-muted',
            )}
            aria-label={mode === 'fast' ? '빠른 모드' : '깊이 생각'}
            title={mode === 'fast' ? '클릭: 깊이 생각 모드로' : '클릭: 빠른 모드로'}
          >
            {mode === 'fast' ? <ZapOff className="h-3 w-3" /> : <Zap className="h-3 w-3" />}
            {mode === 'fast' ? '빠름' : '깊이'}
          </button>
        )}
        {isAdmin && (
          <span className="rounded bg-primary/15 px-2 py-1 text-[11px] text-primary">관리자 (깊이 자동)</span>
        )}
      </div>

      {/* Messages */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto p-3 space-y-3">
        {messages.length === 0 && !loading && (
          <EmptyState skill={currentSkill} />
        )}
        {messages.map((m, idx) => (
          <MessageBubble key={`${m.id}-${idx}`} message={m} />
        ))}
        {sending && (
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Spinner className="h-3 w-3" />
            {queueWaiting > 0
              ? `대기 중… 현재 ${queueWaiting}명이 함께 기다리고 있어요`
              : mode === 'deep' || isAdmin
                ? '깊이 생각하는 중…'
                : '답변 생성 중…'}
          </div>
        )}
      </div>

      {/* Composer */}
      <div className="border-t p-2 pb-[max(0.5rem,env(safe-area-inset-bottom))]">
        {/* #106 첨부 이미지 미리보기 chips */}
        {attachments.length > 0 && (
          <div className="mb-2 flex flex-wrap gap-2">
            {attachments.map((url) => (
              <div key={url} className="relative">
                <img
                  src={url}
                  alt="첨부"
                  className="h-16 w-16 rounded-md border object-cover"
                />
                <button
                  type="button"
                  onClick={() => removeAttachment(url)}
                  aria-label="첨부 제거"
                  className="absolute -right-1.5 -top-1.5 inline-flex h-5 w-5 items-center justify-center rounded-full bg-background text-muted-foreground shadow ring-1 ring-border hover:text-foreground"
                >
                  <X className="h-3 w-3" />
                </button>
              </div>
            ))}
          </div>
        )}
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          multiple
          className="hidden"
          onChange={(e) => void handleFilePick(e)}
        />
        <div className="relative">
          <textarea
            value={input}
            onChange={(e) => {
              setInput(e.target.value)
              // auto-grow up to 5 lines
              const el = e.currentTarget
              el.style.height = 'auto'
              el.style.height = Math.min(el.scrollHeight, 24 * 5 + 16) + 'px'
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                void send()
              }
            }}
            placeholder="질문을 입력하세요…"
            rows={1}
            // text-base (16px) — iOS focus auto-zoom 방지 (WCAG 호환, viewport meta 안 건드림)
            className="w-full resize-none rounded-md border bg-background py-2 pl-12 pr-12 text-base leading-6 focus:outline-none focus:ring-1 focus:ring-ring"
          />
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            disabled={sending || uploadingFile}
            aria-label="이미지 첨부"
            className="absolute bottom-1.5 left-1.5 inline-flex h-9 w-9 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:opacity-50"
          >
            {uploadingFile ? <Spinner className="h-4 w-4" /> : <Paperclip className="h-4 w-4" />}
          </button>
          <button
            type="button"
            onClick={() => void send()}
            disabled={sending || (!input.trim() && attachments.length === 0)}
            aria-label="전송"
            className="absolute bottom-1.5 right-1.5 inline-flex h-9 w-9 items-center justify-center rounded-md bg-primary text-primary-foreground shadow-sm transition-colors hover:bg-primary/90 disabled:bg-muted disabled:text-muted-foreground"
          >
            <Send className="h-4 w-4" />
          </button>
        </div>
        <p className="mt-1 hidden text-[11px] text-muted-foreground sm:block">
          Enter 전송 · Shift+Enter 줄바꿈
        </p>
      </div>
    </div>
  )
}

function EmptyState({ skill }: { skill: Skill | null }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 text-center text-sm text-muted-foreground">
      <Sparkles className="h-8 w-8 text-highlight/60" />
      <div>
        <p className="font-medium text-foreground">
          {skill?.name ?? '챗봇 조교'} 에게 물어보세요
        </p>
        {skill?.description && (
          <p className="mt-1 text-xs">{skill.description}</p>
        )}
      </div>
      <div className="mt-2 flex max-w-xs flex-wrap justify-center gap-1 text-[11px]">
        {['지갑 잔액 알려줘', 'LLM API 키 어떻게 발급받아?', '청산하면 세금이 얼마?', '투자 라운드 여는 법'].map(
          (q) => (
            <span key={q} className="rounded-full bg-muted px-2 py-0.5">
              "{q}"
            </span>
          ),
        )}
      </div>
    </div>
  )
}

function MessageBubble({ message }: { message: Message }) {
  if (message.role === 'tool') {
    let parsed: unknown = message.content
    try {
      parsed = JSON.parse(message.content)
    } catch { /* raw text */ }
    return (
      <details className="rounded border border-border/60 bg-muted/40 px-2 py-1 text-[11px] text-muted-foreground">
        <summary className="cursor-pointer select-none">
          🔧 도구 응답 {message.tool_call_id ? `(${message.tool_call_id.slice(0, 8)})` : ''}
        </summary>
        <pre className="mt-1 max-h-40 overflow-auto whitespace-pre-wrap break-all text-[10px] text-muted-foreground">
          {typeof parsed === 'string' ? parsed : JSON.stringify(parsed, null, 2)}
        </pre>
      </details>
    )
  }

  if (message.role === 'user') {
    return (
      <div className="flex justify-end">
        <div className="max-w-[85%] rounded-2xl bg-primary px-3 py-2 text-sm text-primary-foreground">
          {message.attachments && message.attachments.length > 0 && (
            <div className="mb-2 flex flex-wrap gap-1.5">
              {message.attachments.map((url) => (
                <a key={url} href={url} target="_blank" rel="noopener noreferrer">
                  <img src={url} alt="첨부" className="h-20 w-20 rounded-md object-cover" />
                </a>
              ))}
            </div>
          )}
          {message.content && <span className="whitespace-pre-wrap">{message.content}</span>}
        </div>
      </div>
    )
  }

  if (message.role === 'assistant') {
    // Tool call visualization + content
    const hasToolCalls = message.tool_calls && message.tool_calls.length > 0
    return (
      <div className="flex justify-start">
        <div className="max-w-[90%] rounded-2xl bg-muted px-3 py-2 text-sm">
          {hasToolCalls && (
            <div className="mb-1 flex flex-wrap gap-1">
              {message.tool_calls!.map((tc) => (
                <span key={tc.id} className="rounded-full bg-highlight/15 px-2 py-0.5 text-[10px] text-highlight">
                  🔧 {tc.name}
                </span>
              ))}
            </div>
          )}
          {message.content ? (
            <MarkdownContent content={message.content} className="text-sm" />
          ) : hasToolCalls ? (
            <span className="text-xs text-muted-foreground">도구 호출 중…</span>
          ) : null}
        </div>
      </div>
    )
  }

  return null
}
