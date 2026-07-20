import { useCallback, useEffect, useState } from 'react'
import {
  ArrowLeft,
  Building2,
  Check,
  Clock,
  Copy,
  Download,
  Loader2,
  Mail,
  Paperclip,
  Reply,
  Send,
  User,
  Users,
} from 'lucide-react'
import { toast } from 'sonner'

import { api } from '@/lib/api'
import { getToken } from '@/lib/auth'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'

// ─── 타입 (백엔드 계약과 1:1) ────────────────────────────────
// status: null(미신청) | 'pending'(승인대기) | 'rejected'(반려) | 'approved'(승인)
type AddressStatus = null | 'pending' | 'rejected' | 'approved'

// GET /api/mail/mailboxes 의 각 항목
// kind: 'user'(개인) | 'company'(내 회사) | 'shared'(관리자가 부여한 공용 메일함)
type MailboxKind = 'user' | 'company' | 'shared'

interface Mailbox {
  address_id: number
  kind: MailboxKind
  company_id: number | null
  name: string
  local_part: string | null
  email: string | null
  status: AddressStatus
}

const KIND_LABEL: Record<MailboxKind, string> = {
  user: '개인',
  company: '회사',
  shared: '공용',
}

function kindIcon(kind: MailboxKind) {
  if (kind === 'company') return <Building2 className="h-3 w-3" />
  if (kind === 'shared') return <Users className="h-3 w-3" />
  return <User className="h-3 w-3" />
}

interface MailListItem {
  id: number
  direction: string
  from_addr: string
  header_from: string
  header_from_name: string
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
  const [mailboxes, setMailboxes] = useState<Mailbox[] | null>(null)
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api.get<{ mailboxes: Mailbox[] }>('/mail/mailboxes')
      setMailboxes(data?.mailboxes ?? [])
    } catch {
      setMailboxes([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  if (loading) {
    return (
      <div className="flex justify-center py-16">
        <Spinner />
      </div>
    )
  }

  const boxes = mailboxes ?? []
  const approved = boxes.filter((b) => b.status === 'approved')
  const personal = boxes.find((b) => b.kind === 'user') ?? null

  // 사용 가능한(승인된) 메일함이 하나도 없으면 개인 주소 신청/상태 화면
  if (approved.length === 0) {
    if (personal) {
      return <PersonalStatusView personal={personal} onChanged={load} />
    }
    return <ClaimAddressView mode="new" onClaimed={load} />
  }

  return <MailboxShell boxes={boxes} />
}

// ─── 개인 주소 신청 / 재신청 폼 ──────────────────────────────
type ClaimMode = 'new' | 'rejected' | 'change'

function ClaimAddressView({
  mode,
  initialLocalPart = '',
  onClaimed,
  onCancel,
}: {
  mode: ClaimMode
  initialLocalPart?: string
  onClaimed: () => void
  onCancel?: () => void
}) {
  const [localPart, setLocalPart] = useState(initialLocalPart)
  const [submitting, setSubmitting] = useState(false)

  const trimmed = localPart.trim().toLowerCase()
  const valid = LOCAL_PART_RE.test(trimmed)
  const showError = trimmed.length > 0 && !valid

  const heading =
    mode === 'change' ? '주소 변경(재신청)' : '내 이메일 주소 신청'

  const submit = async () => {
    if (!valid) return
    setSubmitting(true)
    try {
      await api.post('/mail/address', { local_part: trimmed })
      toast.success('이메일 주소를 신청했습니다. 관리자 승인 후 사용할 수 있어요.')
      onClaimed()
    } catch (e) {
      const msg = e instanceof Error ? e.message : '주소 신청에 실패했습니다.'
      toast.error(msg)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      <div className="flex items-center gap-2">
        <Mail className="h-6 w-6 text-primary" />
        <h1 className="text-lg font-bold">{heading}</h1>
      </div>

      <Card>
        <CardContent className="space-y-4 p-4">
          {mode === 'rejected' && (
            <div className="rounded-md border border-red-200 bg-red-50 p-3 text-xs text-red-700">
              이전 주소 신청이 반려되었습니다. 다른 주소로 다시 신청해주세요.
            </div>
          )}

          <p className="text-sm text-muted-foreground">
            나만의 이메일 주소를 신청하면 관리자 승인 후 앱 안에서 메일을 주고받을 수
            있어요.
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
            승인 후에는 변경할 수 없습니다. 신중히 정해주세요.
          </div>

          <div className="flex gap-2">
            <Button
              className="flex-1"
              onClick={submit}
              disabled={!valid || submitting}
            >
              {submitting ? (
                <Loader2 className="mr-1 h-4 w-4 animate-spin" />
              ) : (
                <Check className="mr-1 h-4 w-4" />
              )}
              이 주소로 신청하기
            </Button>
            {onCancel && (
              <Button variant="ghost" onClick={onCancel} disabled={submitting}>
                취소
              </Button>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

// ─── 개인 주소 상태(승인 대기 / 반려) 화면 ───────────────────
function PersonalStatusView({
  personal,
  onChanged,
}: {
  personal: Mailbox
  onChanged: () => void
}) {
  const [changing, setChanging] = useState(false)

  // 반려·미신청 상태이거나 변경(재신청) 모드면 폼을 보여준다.
  if (personal.status !== 'pending' || changing) {
    const mode: ClaimMode =
      personal.status === 'rejected'
        ? 'rejected'
        : changing
          ? 'change'
          : 'new'
    return (
      <ClaimAddressView
        mode={mode}
        initialLocalPart={personal.local_part ?? ''}
        onClaimed={onChanged}
        onCancel={changing ? () => setChanging(false) : undefined}
      />
    )
  }

  // 승인 대기 화면
  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      <div className="flex items-center gap-2">
        <Mail className="h-6 w-6 text-primary" />
        <h1 className="text-lg font-bold">이메일 주소 신청</h1>
      </div>

      <Card>
        <CardContent className="space-y-4 p-4">
          <div className="flex items-center gap-2 rounded-md border bg-muted/40 px-3 py-2">
            <Mail className="h-4 w-4 shrink-0 text-muted-foreground" />
            <span className="min-w-0 flex-1 truncate text-sm font-medium">
              {personal.email}
            </span>
          </div>

          <div className="flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800">
            <Clock className="mt-0.5 h-4 w-4 shrink-0" />
            <span>
              관리자 승인 대기 중입니다. 승인되면 이 주소로 메일을 주고받을 수 있어요.
            </span>
          </div>

          <Button
            variant="outline"
            className="w-full"
            onClick={() => setChanging(true)}
          >
            주소 변경(재신청)
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}

// ─── 메일함 셸: 메일함 선택기 + 선택된 메일함 ────────────────
function MailboxShell({ boxes }: { boxes: Mailbox[] }) {
  const approved = boxes.filter((b) => b.status === 'approved')
  const [selectedId, setSelectedId] = useState<number>(approved[0].address_id)

  // 선택된 메일함이 사라진 경우(재조회 등) 첫 승인 메일함으로 복귀
  const exists = boxes.some(
    (b) => b.address_id === selectedId && b.status === 'approved',
  )
  const effectiveId = exists ? selectedId : approved[0].address_id

  return (
    <Mailbox
      key={effectiveId}
      boxes={boxes}
      selectedId={effectiveId}
      onSelect={setSelectedId}
    />
  )
}

// ─── 메일함 선택기 ───────────────────────────────────────────
function MailboxSelector({
  boxes,
  value,
  onChange,
}: {
  boxes: Mailbox[]
  value: number
  onChange: (id: number) => void
}) {
  if (boxes.length === 0) return null

  // 메일함이 하나뿐이면 전환할 게 없으니 비대화형 헤더로 이름·구분·주소를 보여준다.
  if (boxes.length === 1) {
    const b = boxes[0]
    return (
      <div className="flex flex-col items-start gap-0.5 rounded-md border bg-muted/40 px-3 py-2">
        <span className="flex items-center gap-1 text-xs font-medium">
          {kindIcon(b.kind)}
          {b.name}
          <Badge variant="outline" className="px-1 py-0 text-[10px]">
            {KIND_LABEL[b.kind]}
          </Badge>
        </span>
        <span className="text-[10px] text-muted-foreground">{b.email}</span>
      </div>
    )
  }

  return (
    <Tabs value={String(value)} onValueChange={(v) => onChange(Number(v))}>
      {/* 베이스 TabsList 는 가로형 h-8 고정 → 2줄(이름+주소) 항목이 잘려서 variant 까지 h-auto 로 오버라이드 (#171 비전검사) */}
      <TabsList className="flex h-auto w-full flex-wrap justify-start gap-1 group-data-horizontal/tabs:h-auto">
        {boxes.map((b) => (
          <TabsTrigger
            key={b.address_id}
            value={String(b.address_id)}
            disabled={b.status !== 'approved'}
            className="h-auto flex-none flex-col items-start gap-0.5 px-3 py-2"
          >
            <span className="flex items-center gap-1 text-xs font-medium">
              {kindIcon(b.kind)}
              {b.name}
              <Badge variant="outline" className="px-1 py-0 text-[10px]">
                {KIND_LABEL[b.kind]}
              </Badge>
              {b.status === 'pending' && (
                <Badge variant="secondary" className="px-1 py-0 text-[10px]">
                  대기중
                </Badge>
              )}
              {b.status === 'rejected' && (
                <Badge variant="destructive" className="px-1 py-0 text-[10px]">
                  반려됨
                </Badge>
              )}
            </span>
            <span className="text-[10px] text-muted-foreground">{b.email}</span>
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  )
}

// ─── 메일함 ──────────────────────────────────────────────────
type ComposeInit = {
  to: string
  subject: string
  inReplyToId: number | null
}

function Mailbox({
  boxes,
  selectedId,
  onSelect,
}: {
  boxes: Mailbox[]
  selectedId: number
  onSelect: (id: number) => void
}) {
  const selected =
    boxes.find((b) => b.address_id === selectedId) ??
    boxes.find((b) => b.status === 'approved')!
  const address = selected.email ?? ''
  const addressId = selected.address_id

  const [box, setBox] = useState<Box>('inbox')
  const [emails, setEmails] = useState<MailListItem[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)

  const [detail, setDetail] = useState<MailDetail | null>(null)
  const [compose, setCompose] = useState<ComposeInit | null>(null)

  const fetchBox = useCallback(
    async (b: Box, offset: number) => {
      const data = await api.get<MailListResponse>(
        `/mail?box=${b}&limit=${PAGE_SIZE}&offset=${offset}&address_id=${addressId}`,
      )
      return data ?? { emails: [], total: 0 }
    },
    [addressId],
  )

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
    setCompose({ to: d.header_from || d.from_addr, subject, inReplyToId: d.id })
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
        addressId={addressId}
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

      <MailboxSelector boxes={boxes} value={selectedId} onChange={onSelect} />

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
  // 표시용 발신자 (#171): 헤더 From 우선 (SES 봉투 VERP 주소 노출 방지), 이름 있으면 이름.
  const sender = mail.header_from_name || mail.header_from || mail.from_addr
  const who = box === 'inbox' ? sender : mail.to_addr
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
          <p>
            보낸사람:{' '}
            {detail.header_from
              ? `${detail.header_from_name ? detail.header_from_name + ' ' : ''}<${detail.header_from}>`
              : detail.from_addr}
          </p>
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
  addressId,
  onCancel,
  onSent,
}: {
  init: ComposeInit
  addressId: number
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
        address_id: addressId,
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
