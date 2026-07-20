import { useCallback, useEffect, useState } from 'react'
import {
  ArrowLeft,
  Check,
  Copy,
  Download,
  Loader2,
  Mail,
  Paperclip,
  Reply,
  Send,
} from 'lucide-react'
import { toast } from 'sonner'

import { api } from '@/lib/api'
import { getToken } from '@/lib/auth'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'

// ─── 타입 (백엔드 계약과 1:1) ────────────────────────────────
interface MailAddress {
  local_part: string | null
  email: string | null
}

interface MailListItem {
  id: number
  direction: string
  from_addr: string
  to_addr: string
  subject: string
  snippet: string
  read: boolean
  has_attachments: boolean
  created_at: string
}

interface MailAttachment {
  id: number
  filename: string
  mime: string
  size: number
}

interface MailDetail extends MailListItem {
  body_text: string
  body_html: string
  in_reply_to: number | null
  attachments: MailAttachment[]
}

interface MailListResponse {
  emails: MailListItem[]
  total: number
}

type Box = 'inbox' | 'sent'

const PAGE_SIZE = 20
const LOCAL_PART_RE = /^[a-z0-9][a-z0-9._-]{2,29}$/

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

function formatFileSize(n: number): string {
  if (n < 1024) return `${n}B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)}KB`
  return `${(n / (1024 * 1024)).toFixed(1)}MB`
}

// 인증 헤더가 필요한 첨부 다운로드 — milestoneFiles.ts 패턴과 동일.
async function downloadAttachment(att: MailAttachment) {
  try {
    const res = await fetch(`/api/mail/attachments/${att.id}`, {
      headers: { Authorization: `Bearer ${getToken()}` },
    })
    if (!res.ok) throw new Error(String(res.status))
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = att.filename
    document.body.appendChild(a)
    a.click()
    a.remove()
    setTimeout(() => URL.revokeObjectURL(url), 60_000)
  } catch {
    toast.error('첨부 파일을 내려받을 수 없습니다.')
  }
}

export default function MailboxPage() {
  const [addr, setAddr] = useState<MailAddress | null>(null)
  const [loadingAddr, setLoadingAddr] = useState(true)

  useEffect(() => {
    api
      .get<MailAddress>('/mail/address')
      .then((a) => setAddr(a ?? { local_part: null, email: null }))
      .catch(() => setAddr({ local_part: null, email: null }))
      .finally(() => setLoadingAddr(false))
  }, [])

  if (loadingAddr) {
    return (
      <div className="flex justify-center py-16">
        <Spinner />
      </div>
    )
  }

  if (!addr?.email) {
    return <ClaimAddressView onClaimed={setAddr} />
  }

  return <Mailbox address={addr.email} />
}

// ─── 주소 만들기 ─────────────────────────────────────────────
function ClaimAddressView({ onClaimed }: { onClaimed: (a: MailAddress) => void }) {
  const [localPart, setLocalPart] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const trimmed = localPart.trim().toLowerCase()
  const valid = LOCAL_PART_RE.test(trimmed)
  const showError = trimmed.length > 0 && !valid

  const submit = async () => {
    if (!valid) return
    setSubmitting(true)
    try {
      const a = await api.post<MailAddress>('/mail/address', {
        local_part: trimmed,
      })
      toast.success('이메일 주소를 만들었습니다.')
      onClaimed(a)
    } catch (e) {
      const msg = e instanceof Error ? e.message : '주소 생성에 실패했습니다.'
      toast.error(msg)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      <div className="flex items-center gap-2">
        <Mail className="h-6 w-6 text-primary" />
        <h1 className="text-lg font-bold">내 이메일 주소 만들기</h1>
      </div>

      <Card>
        <CardContent className="space-y-4 p-4">
          <p className="text-sm text-muted-foreground">
            나만의 이메일 주소를 정하면 앱 안에서 메일을 주고받을 수 있어요.
          </p>

          <div className="space-y-1.5">
            <Label htmlFor="local-part">주소</Label>
            <div className="flex items-center gap-2">
              <Input
                id="local-part"
                value={localPart}
                onChange={(e) => setLocalPart(e.target.value)}
                placeholder="jane99"
                autoCapitalize="none"
                autoCorrect="off"
                spellCheck={false}
              />
              <span className="shrink-0 text-sm text-muted-foreground">
                @earnlearning.com
              </span>
            </div>
            {showError ? (
              <p className="text-xs text-red-600">
                3~30자, 영소문자·숫자로 시작하고 영소문자·숫자·.·_·- 만 쓸 수 있어요.
              </p>
            ) : (
              <p className="text-xs text-muted-foreground">
                미리보기:{' '}
                <span className="font-medium text-foreground">
                  {trimmed || 'jane99'}@earnlearning.com
                </span>
              </p>
            )}
          </div>

          <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800">
            한 번 정하면 바꿀 수 없습니다. 신중히 정해주세요.
          </div>

          <Button
            className="w-full"
            onClick={submit}
            disabled={!valid || submitting}
          >
            {submitting ? (
              <Loader2 className="mr-1 h-4 w-4 animate-spin" />
            ) : (
              <Check className="mr-1 h-4 w-4" />
            )}
            이 주소로 만들기
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}

// ─── 메일함 ──────────────────────────────────────────────────
type ComposeInit = {
  to: string
  subject: string
  inReplyToId: number | null
}

function Mailbox({ address }: { address: string }) {
  const [box, setBox] = useState<Box>('inbox')
  const [emails, setEmails] = useState<MailListItem[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)

  const [detail, setDetail] = useState<MailDetail | null>(null)
  const [compose, setCompose] = useState<ComposeInit | null>(null)

  const fetchBox = useCallback(async (b: Box, offset: number) => {
    const data = await api.get<MailListResponse>(
      `/mail?box=${b}&limit=${PAGE_SIZE}&offset=${offset}`,
    )
    return data ?? { emails: [], total: 0 }
  }, [])

  const loadBox = useCallback(
    async (b: Box) => {
      setLoading(true)
      try {
        const data = await fetchBox(b, 0)
        setEmails(data.emails ?? [])
        setTotal(data.total ?? 0)
      } catch {
        setEmails([])
        setTotal(0)
      } finally {
        setLoading(false)
      }
    },
    [fetchBox],
  )

  useEffect(() => {
    loadBox(box)
  }, [box, loadBox])

  const loadMore = async () => {
    setLoadingMore(true)
    try {
      const data = await fetchBox(box, emails.length)
      setEmails((prev) => [...prev, ...(data.emails ?? [])])
      setTotal(data.total ?? total)
    } catch {
      // ignore
    } finally {
      setLoadingMore(false)
    }
  }

  const openDetail = async (id: number) => {
    try {
      const d = await api.get<MailDetail>(`/mail/${id}`)
      setDetail(d)
      // 상세 조회 시 서버가 읽음 처리 → 목록도 반영
      setEmails((prev) =>
        prev.map((e) => (e.id === id ? { ...e, read: true } : e)),
      )
    } catch {
      toast.error('메일을 열 수 없습니다.')
    }
  }

  const startReply = (d: MailDetail) => {
    const base = d.subject.trim()
    const subject = /^re:/i.test(base) ? base : `Re: ${base}`
    setCompose({ to: d.from_addr, subject, inReplyToId: d.id })
    setDetail(null)
  }

  const handleSent = async () => {
    setCompose(null)
    setBox('sent')
    // box 가 이미 sent 였다면 useEffect 가 안 돌아 직접 새로고침
    await loadBox('sent')
  }

  // ── 작성 화면 ──
  if (compose) {
    return (
      <ComposeView
        init={compose}
        onCancel={() => setCompose(null)}
        onSent={handleSent}
      />
    )
  }

  // ── 상세 화면 ──
  if (detail) {
    return (
      <DetailView
        detail={detail}
        onBack={() => setDetail(null)}
        onReply={() => startReply(detail)}
      />
    )
  }

  // ── 목록 화면 ──
  return (
    <div className="mx-auto max-w-lg space-y-4 p-4">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-bold">메일함</h1>
        <Button
          size="sm"
          onClick={() => setCompose({ to: '', subject: '', inReplyToId: null })}
        >
          <Send className="mr-1 h-4 w-4" />새 메일
        </Button>
      </div>

      <AddressBar address={address} />

      <Tabs value={box} onValueChange={(v) => setBox(v as Box)}>
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="inbox">받은편지함</TabsTrigger>
          <TabsTrigger value="sent">보낸편지함</TabsTrigger>
        </TabsList>
      </Tabs>

      {loading ? (
        <div className="flex justify-center py-10">
          <Spinner />
        </div>
      ) : emails.length === 0 ? (
        <div className="flex flex-col items-center py-12 text-muted-foreground">
          <Mail className="mb-2 h-10 w-10" />
          <p className="text-sm">
            {box === 'inbox' ? '받은 메일이 없습니다.' : '보낸 메일이 없습니다.'}
          </p>
        </div>
      ) : (
        <>
          <div className="space-y-2">
            {emails.map((mail) => (
              <MailRow
                key={mail.id}
                mail={mail}
                box={box}
                onClick={() => openDetail(mail.id)}
              />
            ))}
          </div>
          {emails.length < total && (
            <div className="flex justify-center pt-1">
              <Button
                variant="outline"
                size="sm"
                onClick={loadMore}
                disabled={loadingMore}
              >
                {loadingMore && <Loader2 className="mr-1 h-4 w-4 animate-spin" />}
                더 보기
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  )
}

function AddressBar({ address }: { address: string }) {
  const [copied, setCopied] = useState(false)

  const copy = async () => {
    try {
      await navigator.clipboard?.writeText(address)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      toast.error('복사에 실패했습니다.')
    }
  }

  return (
    <div className="flex items-center gap-2 rounded-md border bg-muted/40 px-3 py-2">
      <Mail className="h-4 w-4 shrink-0 text-muted-foreground" />
      <span className="min-w-0 flex-1 truncate text-sm font-medium">{address}</span>
      <Button
        variant="ghost"
        size="sm"
        className="h-7 gap-1 px-2 text-xs"
        onClick={copy}
        aria-label="주소 복사"
      >
        {copied ? (
          <Check className="h-3.5 w-3.5 text-emerald-600" />
        ) : (
          <Copy className="h-3.5 w-3.5" />
        )}
        {copied ? '복사됨' : '복사'}
      </Button>
    </div>
  )
}

function MailRow({
  mail,
  box,
  onClick,
}: {
  mail: MailListItem
  box: Box
  onClick: () => void
}) {
  const unread = box === 'inbox' && !mail.read
  const who = box === 'inbox' ? mail.from_addr : mail.to_addr
  return (
    <Card
      className={`cursor-pointer transition-colors hover:bg-accent/30 ${
        unread ? 'border-primary/30 bg-primary/5' : ''
      }`}
      onClick={onClick}
    >
      <CardContent className="flex items-start gap-3 p-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span
              className={`truncate text-sm ${unread ? 'font-bold' : 'font-medium'}`}
            >
              {who}
            </span>
            {mail.has_attachments && (
              <Paperclip className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
            )}
            {unread && (
              <span
                data-testid="unread-dot"
                className="ml-auto h-2 w-2 shrink-0 rounded-full bg-primary"
              />
            )}
          </div>
          <p className={`truncate text-sm ${unread ? 'font-semibold' : ''}`}>
            {mail.subject || '(제목 없음)'}
          </p>
          <p className="truncate text-xs text-muted-foreground">{mail.snippet}</p>
        </div>
        <span className="shrink-0 text-xs text-muted-foreground">
          {timeAgo(mail.created_at)}
        </span>
      </CardContent>
    </Card>
  )
}

// ─── 상세 ────────────────────────────────────────────────────
function DetailView({
  detail,
  onBack,
  onReply,
}: {
  detail: MailDetail
  onBack: () => void
  onReply: () => void
}) {
  return (
    <div className="mx-auto max-w-lg space-y-4 p-4">
      <button
        onClick={onBack}
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" /> 메일함
      </button>

      <div className="space-y-1">
        <h1 className="text-lg font-bold">{detail.subject || '(제목 없음)'}</h1>
        <div className="text-xs text-muted-foreground">
          <p>보낸사람: {detail.from_addr}</p>
          <p>받는사람: {detail.to_addr}</p>
          <p>{new Date(detail.created_at).toLocaleString('ko-KR')}</p>
        </div>
      </div>

      {/* 본문은 text 만 렌더 (body_html 원시 렌더는 XSS 위험이라 사용 안 함) */}
      <div className="whitespace-pre-wrap rounded-md bg-muted/40 p-3 text-sm">
        {detail.body_text}
      </div>

      {detail.attachments?.length > 0 && (
        <div className="space-y-2">
          <p className="flex items-center gap-1 text-xs font-medium text-muted-foreground">
            <Paperclip className="h-3.5 w-3.5" /> 첨부 파일 ({detail.attachments.length})
          </p>
          <ul className="space-y-1">
            {detail.attachments.map((att) => (
              <li
                key={att.id}
                className="flex items-center gap-2 rounded-md border bg-background px-2 py-1.5 text-xs"
              >
                <span className="min-w-0 flex-1 truncate" title={att.filename}>
                  {att.filename}
                </span>
                <span className="shrink-0 text-muted-foreground">
                  {formatFileSize(att.size)}
                </span>
                <button
                  type="button"
                  onClick={() => downloadAttachment(att)}
                  className="rounded p-0.5 hover:bg-muted"
                  aria-label={`${att.filename} 내려받기`}
                >
                  <Download className="h-4 w-4" />
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}

      <Button variant="outline" onClick={onReply}>
        <Reply className="mr-1 h-4 w-4" /> 답장
      </Button>
    </div>
  )
}

// ─── 작성 ────────────────────────────────────────────────────
function ComposeView({
  init,
  onCancel,
  onSent,
}: {
  init: ComposeInit
  onCancel: () => void
  onSent: () => void
}) {
  const [to, setTo] = useState(init.to)
  const [subject, setSubject] = useState(init.subject)
  const [bodyText, setBodyText] = useState('')
  const [sending, setSending] = useState(false)

  const send = async () => {
    if (!to.trim()) {
      toast.error('받는 사람을 입력해주세요.')
      return
    }
    setSending(true)
    try {
      await api.post('/mail/send', {
        to: to.trim(),
        subject: subject.trim(),
        body_text: bodyText,
        in_reply_to_id: init.inReplyToId,
      })
      toast.success('메일을 보냈습니다.')
      onSent()
    } catch (e) {
      const msg = e instanceof Error ? e.message : '메일 전송에 실패했습니다.'
      toast.error(msg)
    } finally {
      setSending(false)
    }
  }

  return (
    <div className="mx-auto max-w-lg space-y-4 p-4">
      <button
        onClick={onCancel}
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" /> 메일함
      </button>

      <h1 className="text-lg font-bold">
        {init.inReplyToId ? '답장 쓰기' : '새 메일'}
      </h1>

      <div className="space-y-1.5">
        <Label htmlFor="mail-to">받는 사람</Label>
        <Input
          id="mail-to"
          value={to}
          onChange={(e) => setTo(e.target.value)}
          placeholder="someone@earnlearning.com"
          autoCapitalize="none"
          autoCorrect="off"
          spellCheck={false}
        />
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="mail-subject">제목</Label>
        <Input
          id="mail-subject"
          value={subject}
          onChange={(e) => setSubject(e.target.value)}
          placeholder="제목"
        />
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="mail-body">내용</Label>
        <Textarea
          id="mail-body"
          value={bodyText}
          onChange={(e) => setBodyText(e.target.value)}
          placeholder="내용을 입력하세요"
          rows={10}
        />
      </div>

      <div className="flex gap-2">
        <Button onClick={send} disabled={sending}>
          {sending ? (
            <Loader2 className="mr-1 h-4 w-4 animate-spin" />
          ) : (
            <Send className="mr-1 h-4 w-4" />
          )}
          보내기
        </Button>
        <Button variant="ghost" onClick={onCancel}>
          취소
        </Button>
      </div>
    </div>
  )
}
