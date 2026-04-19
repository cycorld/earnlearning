import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { MessageCircle, Send, Sparkles, Trash2, X, ZapOff, Zap } from 'lucide-react'
import { toast } from 'sonner'

import { api, ApiError } from '@/lib/api'
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

  const send = async () => {
    const text = input.trim()
    if (!text || sending) return
    const s = await ensureSession()
    if (!s) return

    // Optimistic user message
    const optimisticId = -Date.now()
    const userMsg: Message = {
      id: optimisticId,
      role: 'user',
      content: text,
      created_at: new Date().toISOString(),
    }
    setMessages((prev) => [...prev, userMsg])
    setInput('')
    setSending(true)
    try {
      const resp = await api.post<AskResponse>(`/chat/sessions/${s.id}/ask`, {
        message: text,
        mode,
        skill_slug: activeSkillSlug || undefined,
      })
      // Replace optimistic with actual + add assistant
      setMessages((prev) => {
        const withoutOpt = prev.filter((m) => m.id !== optimisticId)
        // append the optimistic as "real" (no ID from server for user msg here; just keep local one)
        const next = [...withoutOpt, { ...userMsg, id: optimisticId }]
        // tool logs (show collapsed)
        if (resp.tool_logs && resp.tool_logs.length > 0) {
          next.push(...resp.tool_logs)
        }
        if (resp.message) next.push(resp.message)
        return next
      })
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '답변을 받지 못했습니다.')
      // keep the user message (don't remove) so they can retry
    } finally {
      setSending(false)
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
            {mode === 'deep' || isAdmin ? '깊이 생각하는 중…' : '답변 생성 중…'}
          </div>
        )}
      </div>

      {/* Composer */}
      <div className="border-t p-2">
        <div className="flex items-end gap-2">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                void send()
              }
            }}
            placeholder="질문을 입력하세요… (Enter 전송, Shift+Enter 줄바꿈)"
            rows={2}
            className="flex-1 resize-none rounded-md border bg-background px-2 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
          />
          <Button
            size="sm"
            onClick={() => void send()}
            disabled={sending || !input.trim()}
          >
            <Send className="h-4 w-4" />
          </Button>
        </div>
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
          {message.content}
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
