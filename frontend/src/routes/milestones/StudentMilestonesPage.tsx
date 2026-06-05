import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  ArrowLeft,
  Award,
  CheckCircle2,
  Clock,
  ExternalLink,
  FileText,
  Mic,
  Rocket,
  Send,
  XCircle,
} from 'lucide-react'
import { toast } from 'sonner'

import { api } from '@/lib/api'
import {
  GROUP_DESCRIPTIONS,
  MILESTONE_DEADLINES,
  MILESTONE_LABELS,
  MILESTONE_TYPES,
  type Milestone,
  type MilestoneType,
  type StudentProgress,
  isValidMilestoneURL,
} from '@/lib/milestone'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import { Textarea } from '@/components/ui/textarea'

const TYPE_ICONS: Record<MilestoneType, typeof Rocket> = {
  mvp1: Rocket,
  mvp2: Rocket,
  business_plan: FileText,
  retrospective: Mic,
}

const STATUS_META: Record<
  Milestone['status'],
  { label: string; chip: string; Icon: typeof CheckCircle2 }
> = {
  pending: { label: '검토 대기', chip: 'bg-amber-100 text-amber-700', Icon: Clock },
  approved: { label: '승인됨', chip: 'bg-emerald-100 text-emerald-700', Icon: CheckCircle2 },
  rejected: { label: '반려됨', chip: 'bg-red-100 text-red-700', Icon: XCircle },
}

export default function StudentMilestonesPage() {
  const [data, setData] = useState<StudentProgress | null>(null)
  const [loading, setLoading] = useState(true)

  const load = () => {
    setLoading(true)
    api
      .get<StudentProgress>('/milestones/mine')
      .then(setData)
      .catch((e) => toast.error(e instanceof Error ? e.message : '조회 실패'))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
  }, [])

  if (loading) {
    return (
      <div className="flex justify-center p-16">
        <Spinner />
      </div>
    )
  }

  if (!data) return null

  return (
    <div className="container mx-auto max-w-2xl space-y-4 p-4 pb-24">
      <Link
        to="/feed"
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" /> 피드로
      </Link>

      <div>
        <h1 className="text-2xl font-bold">평가지표</h1>
        <p className="text-sm text-muted-foreground">
          syllabus 의 4가지 절대평가 지표. 승인 개수로 그룹(A/B/C/D)이 결정됩니다.
        </p>
      </div>

      <ProgressSummary data={data} />

      <div className="space-y-3">
        {MILESTONE_TYPES.map((t, i) => (
          <MilestoneCard
            key={t}
            type={t}
            milestone={data.milestones[i]}
            onSubmitted={load}
          />
        ))}
      </div>

      <p className="text-xs text-muted-foreground">
        ℹ️ MVP는 회사의 service_url 또는 정부과제 응모 본문에서 자동 집계됩니다.
        AI Studio · Claude · ChatGPT · Gemini · localhost 등 연습용 URL은 제외됩니다.
      </p>
    </div>
  )
}

function ProgressSummary({ data }: { data: StudentProgress }) {
  const group = data.group || ''
  const groupColor =
    group === 'A'
      ? 'bg-emerald-100 text-emerald-700'
      : group === 'B'
        ? 'bg-blue-100 text-blue-700'
        : group === 'C'
          ? 'bg-amber-100 text-amber-700'
          : group === 'D'
            ? 'bg-orange-100 text-orange-700'
            : 'bg-muted text-muted-foreground'

  return (
    <Card>
      <CardContent className="flex items-center gap-4 p-4">
        <div className="flex h-14 w-14 items-center justify-center rounded-full bg-primary/10">
          <Award className="h-7 w-7 text-primary" />
        </div>
        <div className="flex-1">
          <div className="flex items-baseline gap-2">
            <span className="text-2xl font-bold">{data.approved_count}</span>
            <span className="text-sm text-muted-foreground">/ 4 승인됨</span>
          </div>
          <p className="text-xs text-muted-foreground">
            {GROUP_DESCRIPTIONS[group]}
          </p>
        </div>
        <Badge
          variant="secondary"
          className={`text-base font-bold px-3 py-1 ${groupColor}`}
        >
          {group ? `${group} 그룹` : '미진입'}
        </Badge>
      </CardContent>
    </Card>
  )
}

function MilestoneCard({
  type,
  milestone,
  onSubmitted,
}: {
  type: MilestoneType
  milestone: Milestone | null
  onSubmitted: () => void
}) {
  const Icon = TYPE_ICONS[type]
  const [editing, setEditing] = useState(false)
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center justify-between text-base">
          <span className="flex items-center gap-2">
            <Icon className="h-4 w-4 text-muted-foreground" />
            {MILESTONE_LABELS[type]}
          </span>
          {milestone && <StatusChip status={milestone.status} />}
        </CardTitle>
        <div className="text-xs text-muted-foreground">{MILESTONE_DEADLINES[type]}</div>
      </CardHeader>
      <CardContent className="space-y-2">
        {milestone ? (
          <SubmittedView
            milestone={milestone}
            editing={editing}
            setEditing={setEditing}
            type={type}
            onSubmitted={() => {
              setEditing(false)
              onSubmitted()
            }}
          />
        ) : (
          <EmptyView
            type={type}
            editing={editing}
            setEditing={setEditing}
            onSubmitted={() => {
              setEditing(false)
              onSubmitted()
            }}
          />
        )}
      </CardContent>
    </Card>
  )
}

function StatusChip({ status }: { status: Milestone['status'] }) {
  const meta = STATUS_META[status]
  const Icon = meta.Icon
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs ${meta.chip}`}
    >
      <Icon className="h-3 w-3" />
      {meta.label}
    </span>
  )
}

function SourceLabel({ milestone }: { milestone: Milestone }) {
  if (milestone.source_type === 'company')
    return <span className="text-xs text-muted-foreground">회사 service_url 에서 자동 집계</span>
  if (milestone.source_type === 'grant')
    return <span className="text-xs text-muted-foreground">정부과제 응모 본문에서 자동 추출</span>
  return <span className="text-xs text-muted-foreground">직접 제출</span>
}

function SubmittedView({
  milestone,
  editing,
  setEditing,
  type,
  onSubmitted,
}: {
  milestone: Milestone
  editing: boolean
  setEditing: (e: boolean) => void
  type: MilestoneType
  onSubmitted: () => void
}) {
  if (editing) {
    return <SubmitForm type={type} initial={milestone} onSubmitted={onSubmitted} onCancel={() => setEditing(false)} />
  }
  return (
    <>
      {milestone.url && (
        <a
          href={milestone.url}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 text-sm text-primary hover:underline break-all"
        >
          {milestone.url} <ExternalLink className="h-3 w-3 flex-shrink-0" />
        </a>
      )}
      {milestone.content && (
        <div className="whitespace-pre-wrap rounded-md bg-muted/50 p-2 text-sm">
          {milestone.content}
        </div>
      )}
      <SourceLabel milestone={milestone} />
      {milestone.admin_note && (
        <div className="rounded-md border border-amber-200 bg-amber-50 p-2 text-xs">
          <strong>교수님 코멘트:</strong> {milestone.admin_note}
        </div>
      )}
      {milestone.status !== 'approved' && (
        <Button size="sm" variant="outline" onClick={() => setEditing(true)}>
          다시 제출
        </Button>
      )}
    </>
  )
}

function EmptyView({
  type,
  editing,
  setEditing,
  onSubmitted,
}: {
  type: MilestoneType
  editing: boolean
  setEditing: (e: boolean) => void
  onSubmitted: () => void
}) {
  if (editing) {
    return <SubmitForm type={type} onSubmitted={onSubmitted} onCancel={() => setEditing(false)} />
  }
  return (
    <>
      <p className="text-sm text-muted-foreground">
        {type === 'mvp1' || type === 'mvp2'
          ? '회사 페이지에서 service_url 을 등록하거나, 정부과제 응모 본문에 배포 URL을 적으면 자동 집계됩니다.'
          : '아직 제출하지 않았습니다.'}
      </p>
      <Button size="sm" variant="outline" onClick={() => setEditing(true)}>
        <Send className="mr-1 h-3 w-3" /> 직접 제출
      </Button>
    </>
  )
}

function SubmitForm({
  type,
  initial,
  onSubmitted,
  onCancel,
}: {
  type: MilestoneType
  initial?: Milestone
  onSubmitted: () => void
  onCancel: () => void
}) {
  const [url, setUrl] = useState(initial?.url ?? '')
  const [content, setContent] = useState(initial?.content ?? '')
  const [submitting, setSubmitting] = useState(false)
  const needsURL = type === 'mvp1' || type === 'mvp2'
  const urlInvalid = url.trim() !== '' && !isValidMilestoneURL(url.trim())

  const submit = async () => {
    if (needsURL && !url.trim()) {
      toast.error('배포 URL을 입력해주세요')
      return
    }
    if (urlInvalid) {
      toast.error('vercel.app 또는 자체 도메인만 인정됩니다 (AI Studio 등 제외)')
      return
    }
    setSubmitting(true)
    try {
      await api.post('/milestones', { type, url: url.trim(), content: content.trim() })
      toast.success('제출 완료. 교수님 승인 대기 중')
      onSubmitted()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '제출 실패')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="space-y-2">
      {(needsURL || type === 'business_plan' || type === 'retrospective') && (
        <div>
          <label className="text-xs text-muted-foreground">
            {needsURL ? '배포 URL (필수)' : '참고 URL (선택)'}
          </label>
          <Input
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://my-mvp.vercel.app"
          />
          {urlInvalid && (
            <p className="mt-1 text-xs text-red-600">
              연습용 도메인(AI Studio·Claude·ChatGPT·localhost 등)은 제외됩니다.
            </p>
          )}
        </div>
      )}
      {!needsURL && (
        <div>
          <label className="text-xs text-muted-foreground">본문</label>
          <Textarea
            value={content}
            onChange={(e) => setContent(e.target.value)}
            placeholder={
              type === 'business_plan' ? '사업계획서 요약/링크' : '회고 발표 요약/링크'
            }
            rows={4}
          />
        </div>
      )}
      <div className="flex gap-2">
        <Button size="sm" onClick={submit} disabled={submitting}>
          제출
        </Button>
        <Button size="sm" variant="ghost" onClick={onCancel}>
          취소
        </Button>
      </div>
    </div>
  )
}
