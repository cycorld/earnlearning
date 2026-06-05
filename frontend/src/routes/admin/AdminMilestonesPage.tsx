import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  ArrowLeft,
  CheckCircle2,
  Clock,
  ExternalLink,
  RefreshCw,
  X,
  XCircle,
} from 'lucide-react'
import { toast } from 'sonner'

import { api } from '@/lib/api'
import {
  MILESTONE_LABELS,
  MILESTONE_TYPES,
  type Milestone,
  type StudentProgress,
} from '@/lib/milestone'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import { Textarea } from '@/components/ui/textarea'

const STATUS_META: Record<
  Milestone['status'],
  { label: string; chip: string; Icon: typeof CheckCircle2 }
> = {
  pending: { label: '대기', chip: 'bg-amber-100 text-amber-700', Icon: Clock },
  approved: { label: '승인', chip: 'bg-emerald-100 text-emerald-700', Icon: CheckCircle2 },
  rejected: { label: '반려', chip: 'bg-red-100 text-red-700', Icon: XCircle },
}

const GROUP_COLOR: Record<string, string> = {
  A: 'bg-emerald-100 text-emerald-700',
  B: 'bg-blue-100 text-blue-700',
  C: 'bg-amber-100 text-amber-700',
  D: 'bg-orange-100 text-orange-700',
  '': 'bg-muted text-muted-foreground',
}

export default function AdminMilestonesPage() {
  const [rows, setRows] = useState<StudentProgress[]>([])
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<{ student: StudentProgress; milestone: Milestone } | null>(null)

  const load = (sync: boolean = false) => {
    setLoading(true)
    api
      .get<StudentProgress[]>(`/admin/milestones${sync ? '?sync=1' : ''}`)
      .then((d) => setRows(d ?? []))
      .catch((e) => toast.error(e instanceof Error ? e.message : '조회 실패'))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load(true)
  }, [])

  const counts = useMemo(() => {
    const c: Record<string, number> = { A: 0, B: 0, C: 0, D: 0, '': 0 }
    for (const r of rows) c[r.group] = (c[r.group] ?? 0) + 1
    return c
  }, [rows])

  return (
    <div className="container mx-auto max-w-5xl p-4 pb-24">
      <Link
        to="/admin"
        className="mb-4 inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" /> 관리자 홈
      </Link>

      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-2xl font-bold">평가지표 매트릭스</h1>
        <Button size="sm" variant="outline" onClick={() => load(true)}>
          <RefreshCw className="mr-1 h-3 w-3" /> 재집계
        </Button>
      </div>

      <Card className="mb-4">
        <CardContent className="flex flex-wrap items-center gap-3 p-4 text-sm">
          {(['A', 'B', 'C', 'D'] as const).map((g) => (
            <span key={g} className={`rounded-full px-3 py-1 ${GROUP_COLOR[g]}`}>
              {g}그룹 <strong>{counts[g] ?? 0}</strong>명
            </span>
          ))}
          <span className={`rounded-full px-3 py-1 ${GROUP_COLOR['']}`}>
            미진입 <strong>{counts[''] ?? 0}</strong>명
          </span>
        </CardContent>
      </Card>

      {loading ? (
        <div className="flex justify-center p-8">
          <Spinner />
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr className="border-b bg-muted/50">
                <th className="p-2 text-left">학생</th>
                {MILESTONE_TYPES.map((t) => (
                  <th key={t} className="p-2 text-center text-xs">
                    {MILESTONE_LABELS[t]}
                  </th>
                ))}
                <th className="p-2 text-center">그룹</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.student.id} className="border-b">
                  <td className="p-2">
                    <div className="font-medium">{r.student.name}</div>
                    <div className="text-xs text-muted-foreground">
                      {r.student.student_id} · {r.student.department}
                    </div>
                  </td>
                  {r.milestones.map((m, i) =>
                    m ? (
                      <td key={i} className="p-2 text-center">
                        <button
                          onClick={() => setSelected({ student: r, milestone: m })}
                          className="cursor-pointer rounded p-1 hover:bg-muted"
                          title={m.url || m.content || ''}
                        >
                          <StatusChip status={m.status} />
                        </button>
                      </td>
                    ) : (
                      <td key={i} className="p-2 text-center text-muted-foreground">
                        —
                      </td>
                    ),
                  )}
                  <td className="p-2 text-center">
                    <Badge variant="secondary" className={GROUP_COLOR[r.group]}>
                      {r.group || '-'}
                    </Badge>
                  </td>
                </tr>
              ))}
              {rows.length === 0 && (
                <tr>
                  <td
                    colSpan={MILESTONE_TYPES.length + 2}
                    className="p-8 text-center text-muted-foreground"
                  >
                    데이터 없음
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      {selected && (
        <ReviewDialog
          student={selected.student.student}
          milestone={selected.milestone}
          onClose={() => setSelected(null)}
          onDone={() => {
            setSelected(null)
            load(false)
          }}
        />
      )}
    </div>
  )
}

function StatusChip({ status }: { status: Milestone['status'] }) {
  const meta = STATUS_META[status]
  const Icon = meta.Icon
  return (
    <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs ${meta.chip}`}>
      <Icon className="h-3 w-3" />
      {meta.label}
    </span>
  )
}

function ReviewDialog({
  student,
  milestone,
  onClose,
  onDone,
}: {
  student: StudentProgress['student']
  milestone: Milestone
  onClose: () => void
  onDone: () => void
}) {
  const [note, setNote] = useState(milestone.admin_note ?? '')
  const [busy, setBusy] = useState(false)

  const decide = async (action: 'approve' | 'reject') => {
    setBusy(true)
    try {
      await api.post(`/admin/milestones/${milestone.id}/${action}`, { admin_note: note })
      toast.success(action === 'approve' ? '승인 완료' : '반려 완료')
      onDone()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '처리 실패')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <Card className="w-full max-w-md" onClick={(e) => e.stopPropagation()}>
        <CardContent className="space-y-3 p-4">
          <div className="flex items-start justify-between">
            <div>
              <div className="text-xs text-muted-foreground">
                {student.name} · {student.student_id}
              </div>
              <div className="text-lg font-bold">{MILESTONE_LABELS[milestone.type]}</div>
            </div>
            <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
              <X className="h-4 w-4" />
            </button>
          </div>

          <div className="space-y-2">
            <StatusChip status={milestone.status} />
            {milestone.url && (
              <a
                href={milestone.url}
                target="_blank"
                rel="noopener noreferrer"
                className="block break-all text-sm text-primary hover:underline"
              >
                {milestone.url} <ExternalLink className="ml-1 inline h-3 w-3" />
              </a>
            )}
            {milestone.content && (
              <div className="whitespace-pre-wrap rounded bg-muted/50 p-2 text-sm">
                {milestone.content}
              </div>
            )}
            <div className="text-xs text-muted-foreground">
              출처:{' '}
              {milestone.source_type === 'company'
                ? '회사 service_url'
                : milestone.source_type === 'grant'
                  ? '정부과제 응모 본문'
                  : '학생 직접 제출'}
            </div>
          </div>

          <div>
            <label className="text-xs text-muted-foreground">코멘트 (선택)</label>
            <Textarea value={note} onChange={(e) => setNote(e.target.value)} rows={2} />
          </div>

          <div className="flex gap-2">
            <Button onClick={() => decide('approve')} disabled={busy} className="flex-1">
              승인
            </Button>
            <Button onClick={() => decide('reject')} disabled={busy} variant="outline" className="flex-1">
              반려
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
