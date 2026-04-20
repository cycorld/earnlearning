import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowLeft, Bug, Lightbulb, MessageSquare, Copy } from 'lucide-react'
import { toast } from 'sonner'

import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import { Textarea } from '@/components/ui/textarea'

interface Proposal {
  id: number
  category: 'feature' | 'bug' | 'general'
  title: string
  body: string
  attachments?: string[]
  status: 'open' | 'reviewing' | 'resolved' | 'wontfix'
  admin_note: string
  ticket_link: string
  created_at: string
  updated_at: string
  user?: { id: number; name: string; student_id: string; department: string }
}

const CATEGORY_META: Record<Proposal['category'], { label: string; icon: typeof Bug; chip: string }> = {
  feature: { label: '기능 제안', icon: Lightbulb, chip: 'bg-blue-100 text-blue-700' },
  bug: { label: '버그 신고', icon: Bug, chip: 'bg-red-100 text-red-700' },
  general: { label: '일반 의견', icon: MessageSquare, chip: 'bg-muted text-muted-foreground' },
}

const STATUS_LABEL: Record<Proposal['status'], string> = {
  open: '신규',
  reviewing: '검토 중',
  resolved: '해결됨',
  wontfix: '보류',
}

export default function AdminProposalsPage() {
  const [list, setList] = useState<Proposal[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [categoryFilter, setCategoryFilter] = useState<string>('')
  const [selected, setSelected] = useState<Proposal | null>(null)

  const load = () => {
    setLoading(true)
    const qs = new URLSearchParams()
    if (statusFilter) qs.set('status', statusFilter)
    if (categoryFilter) qs.set('category', categoryFilter)
    qs.set('limit', '100')
    api.get<{ items: Proposal[]; total: number }>(`/admin/proposals?${qs.toString()}`)
      .then((d) => {
        setList(d?.items ?? [])
        setTotal(d?.total ?? 0)
      })
      .catch((err) => toast.error(err instanceof Error ? err.message : '조회 실패'))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [statusFilter, categoryFilter])

  return (
    <div className="container mx-auto p-4 pb-24">
      <Link to="/admin" className="mb-4 inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="h-4 w-4" /> 관리자 홈
      </Link>
      <h1 className="mb-4 text-2xl font-bold">학생 제안 (#{total})</h1>

      <div className="mb-4 flex flex-wrap gap-2">
        <Filter label="전체" value="" current={statusFilter} onClick={setStatusFilter} />
        {(['open', 'reviewing', 'resolved', 'wontfix'] as const).map((s) => (
          <Filter key={s} label={STATUS_LABEL[s]} value={s} current={statusFilter} onClick={setStatusFilter} />
        ))}
        <span className="mx-2 self-center text-muted-foreground">·</span>
        <Filter label="모든 종류" value="" current={categoryFilter} onClick={setCategoryFilter} />
        {(['feature', 'bug', 'general'] as const).map((c) => (
          <Filter key={c} label={CATEGORY_META[c].label} value={c} current={categoryFilter} onClick={setCategoryFilter} />
        ))}
      </div>

      {loading ? (
        <div className="flex justify-center p-8"><Spinner /></div>
      ) : list.length === 0 ? (
        <Card><CardContent className="p-8 text-center text-muted-foreground">조건에 맞는 제안이 없습니다.</CardContent></Card>
      ) : (
        <div className="space-y-2">
          {list.map((p) => {
            const meta = CATEGORY_META[p.category]
            const Icon = meta.icon
            return (
              <button
                key={p.id}
                onClick={() => setSelected(p)}
                className="block w-full rounded-lg border bg-card p-3 text-left transition-colors hover:bg-muted/50"
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="flex-1 min-w-0">
                    <div className="mb-1 flex items-center gap-2">
                      <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs ${meta.chip}`}>
                        <Icon className="h-3 w-3" />{meta.label}
                      </span>
                      <span className="text-xs text-muted-foreground">{STATUS_LABEL[p.status]}</span>
                      {p.attachments && p.attachments.length > 0 && (
                        <span className="text-xs text-muted-foreground">📎{p.attachments.length}</span>
                      )}
                    </div>
                    <div className="font-medium truncate">{p.title}</div>
                    <div className="text-xs text-muted-foreground">
                      {p.user?.name} · {p.user?.department} · {new Date(p.created_at).toLocaleDateString('ko-KR')}
                    </div>
                  </div>
                </div>
              </button>
            )
          })}
        </div>
      )}

      {selected && (
        <ProposalDetailModal
          p={selected}
          onClose={() => setSelected(null)}
          onUpdated={(np) => {
            setList((prev) => prev.map((x) => (x.id === np.id ? np : x)))
            setSelected(np)
          }}
        />
      )}
    </div>
  )
}

function Filter({ label, value, current, onClick }: { label: string; value: string; current: string; onClick: (v: string) => void }) {
  const active = value === current
  return (
    <button
      type="button"
      onClick={() => onClick(value)}
      className={`rounded-full px-3 py-1 text-xs ${active ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground hover:bg-muted/70'}`}
    >
      {label}
    </button>
  )
}

function ProposalDetailModal({
  p,
  onClose,
  onUpdated,
}: {
  p: Proposal
  onClose: () => void
  onUpdated: (np: Proposal) => void
}) {
  const meta = CATEGORY_META[p.category]
  const [status, setStatus] = useState<Proposal['status']>(p.status)
  const [adminNote, setAdminNote] = useState(p.admin_note)
  const [ticketLink, setTicketLink] = useState(p.ticket_link)
  const [saving, setSaving] = useState(false)

  const save = async () => {
    setSaving(true)
    try {
      const np = await api.patch<Proposal>(`/admin/proposals/${p.id}`, {
        status,
        admin_note: adminNote,
        ticket_link: ticketLink,
      })
      onUpdated(np)
      toast.success('업데이트 완료')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '저장 실패')
    } finally {
      setSaving(false)
    }
  }

  const copyTicketMd = () => {
    const slug = p.title.toLowerCase().replace(/[^\w가-힣]+/g, '-').replace(/^-|-$/g, '').slice(0, 40) || `proposal-${p.id}`
    const branchPrefix = p.category === 'bug' ? 'fix' : p.category === 'feature' ? 'feat' : 'chore'
    const md = `---\nid: NNN\ntitle: ${p.title}\npriority: medium\ntype: ${branchPrefix === 'fix' ? 'fix' : branchPrefix === 'feat' ? 'feat' : 'chore'}\nbranch: ${branchPrefix}/${slug}\ncreated: ${new Date().toISOString().slice(0,10)}\n---\n\n## 배경 (학생 제안 #${p.id})\n${p.body}\n\n${p.attachments && p.attachments.length > 0 ? `## 첨부\n${p.attachments.map((u) => `- ${u}`).join('\n')}\n\n` : ''}## 작업\n- [ ]\n`
    void navigator.clipboard.writeText(md).then(
      () => toast.success('티켓 markdown 복사됨 — tasks/in-progress/ 에 NNN-' + slug + '.md 로 저장'),
      () => toast.error('복사 실패'),
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-2 sm:items-center" onClick={onClose}>
      <div className="w-full max-w-2xl max-h-[90vh] overflow-y-auto rounded-lg bg-background shadow-xl" onClick={(e) => e.stopPropagation()}>
        <Card className="border-0">
          <CardHeader>
            <div className="flex items-start justify-between">
              <div className="min-w-0">
                <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs ${meta.chip}`}>
                  {meta.label}
                </span>
                <CardTitle className="mt-2 break-words">{p.title}</CardTitle>
                <p className="mt-1 text-xs text-muted-foreground">
                  #{p.id} · {p.user?.name} ({p.user?.department}) · {new Date(p.created_at).toLocaleString('ko-KR')}
                </p>
              </div>
              <Button variant="ghost" size="sm" onClick={onClose}>닫기</Button>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="rounded-md bg-muted/50 p-3 text-sm whitespace-pre-wrap">{p.body}</div>

            {p.attachments && p.attachments.length > 0 && (
              <div>
                <div className="mb-1 text-xs font-medium text-muted-foreground">첨부 이미지</div>
                <div className="flex flex-wrap gap-2">
                  {p.attachments.map((u) => (
                    <a key={u} href={u} target="_blank" rel="noopener noreferrer">
                      <img src={u} alt="첨부" className="h-32 rounded-md border object-cover" />
                    </a>
                  ))}
                </div>
              </div>
            )}

            <div>
              <label className="text-xs font-medium text-muted-foreground">상태</label>
              <select
                value={status}
                onChange={(e) => setStatus(e.target.value as Proposal['status'])}
                className="mt-1 block w-full rounded-md border bg-background px-2 py-1.5 text-sm"
              >
                {(['open', 'reviewing', 'resolved', 'wontfix'] as const).map((s) => (
                  <option key={s} value={s}>{STATUS_LABEL[s]}</option>
                ))}
              </select>
            </div>

            <div>
              <label className="text-xs font-medium text-muted-foreground">학생에게 보낼 메모 (선택)</label>
              <Textarea
                value={adminNote}
                onChange={(e) => setAdminNote(e.target.value)}
                rows={2}
                placeholder="학생이 챗봇의 get_my_proposals 로 볼 수 있는 메모"
              />
            </div>

            <div>
              <label className="text-xs font-medium text-muted-foreground">티켓 링크 (선택, GitHub Issue/PR URL)</label>
              <input
                type="url"
                value={ticketLink}
                onChange={(e) => setTicketLink(e.target.value)}
                placeholder="https://github.com/cycorld/earnlearning/pull/..."
                className="mt-1 block w-full rounded-md border bg-background px-2 py-1.5 text-sm"
              />
            </div>

            <div className="flex flex-wrap gap-2">
              <Button onClick={save} disabled={saving}>{saving ? '저장 중…' : '저장'}</Button>
              {(p.category === 'bug' || p.category === 'feature') && (
                <Button variant="outline" onClick={copyTicketMd}>
                  <Copy className="mr-1 h-4 w-4" /> 티켓 markdown 복사
                </Button>
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
