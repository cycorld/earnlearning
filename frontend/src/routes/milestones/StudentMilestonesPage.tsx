import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  ArrowLeft,
  Award,
  Bot,
  CheckCircle2,
  Clock,
  Download,
  ExternalLink,
  FileText,
  Loader2,
  Mic,
  Paperclip,
  Rocket,
  Send,
  Sparkles,
  Trash2,
  TrendingUp,
  Upload,
  XCircle,
} from 'lucide-react'
import { toast } from 'sonner'

import { api } from '@/lib/api'
import { formatFileSize, openMilestoneFile } from '@/lib/milestoneFiles'
import {
  GROUP_DESCRIPTIONS,
  MILESTONE_DEADLINES,
  MILESTONE_LABELS,
  MILESTONE_TYPES,
  type EssayScoreResult,
  type Milestone,
  type MilestoneFile,
  type MilestoneType,
  type StudentProgress,
  aiScoreMeta,
  isValidMilestoneURL,
} from '@/lib/milestone'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { MarkdownEditor } from '@/components/MarkdownEditor'
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

      <GradeAssetCard data={data} />

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

const GRADE_COLOR: Record<string, string> = {
  A: 'bg-emerald-500 text-white',
  B: 'bg-blue-500 text-white',
  C: 'bg-amber-500 text-white',
  D: 'bg-orange-500 text-white',
  '': 'bg-muted text-muted-foreground',
}

// #125 성적 평가 — 그레이드(A/B/C/D) + 같은 그룹 내 자산 상위 %.
function GradeAssetCard({ data }: { data: StudentProgress }) {
  const grade = data.group || ''
  const pct = data.asset_percentile ?? 0
  const size = data.group_size ?? 0
  const rank = data.asset_rank ?? 0
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-base">
          <Award className="h-4 w-4 text-primary" /> 성적 평가
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex items-center justify-between gap-3">
          <div>
            <p className="text-xs text-muted-foreground">그레이드 (승인 평가지표 기준)</p>
            <p className="text-sm">{GROUP_DESCRIPTIONS[grade]}</p>
          </div>
          <div
            className={`flex h-14 w-14 flex-shrink-0 items-center justify-center rounded-xl text-2xl font-bold ${GRADE_COLOR[grade]}`}
          >
            {grade || '–'}
          </div>
        </div>
        <div className="rounded-md bg-muted/50 p-3">
          <p className="flex items-center gap-1 text-xs text-muted-foreground">
            <TrendingUp className="h-3.5 w-3.5" /> 자산가치 ({grade ? `${grade} 그룹` : '미진입 그룹'} 내 비교)
          </p>
          {size <= 0 ? (
            <p className="mt-0.5 text-sm text-muted-foreground">산정할 그룹원이 없습니다.</p>
          ) : size === 1 ? (
            <p className="mt-0.5 text-sm">그룹 내 유일한 멤버입니다.</p>
          ) : (
            <p className="mt-0.5 text-lg font-bold">
              상위 {pct}%{' '}
              <span className="text-xs font-normal text-muted-foreground">
                ({size}명 중 {rank}위)
              </span>
            </p>
          )}
        </div>
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
        {type === 'business_plan' && <BusinessPlanFiles />}
      </CardContent>
    </Card>
  )
}

// BusinessPlanFiles — 사업계획서 비공개 첨부 (#125). 본인 + 관리자만 접근.
function BusinessPlanFiles() {
  const [files, setFiles] = useState<MilestoneFile[]>([])
  const [uploading, setUploading] = useState(false)

  const load = useCallback(() => {
    api
      .get<MilestoneFile[]>('/milestones/files')
      .then((f) => setFiles(f ?? []))
      .catch(() => {})
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const onPick = async (list: FileList | null) => {
    if (!list || list.length === 0) return
    setUploading(true)
    let ok = 0
    for (const file of Array.from(list)) {
      try {
        const fd = new FormData()
        fd.append('file', file)
        await api.post('/milestones/files', fd)
        ok++
      } catch (e) {
        toast.error(`${file.name}: ${e instanceof Error ? e.message : '업로드 실패'}`)
      }
    }
    setUploading(false)
    if (ok > 0) {
      toast.success(`${ok}개 파일을 첨부했습니다.`)
      load()
    }
  }

  const remove = async (id: number) => {
    try {
      await api.del(`/milestones/files/${id}`)
      load()
    } catch {
      toast.error('삭제에 실패했습니다.')
    }
  }

  return (
    <div className="rounded-md border border-dashed bg-muted/30 p-2">
      <div className="mb-1.5 flex items-center justify-between">
        <span className="flex items-center gap-1 text-xs font-medium text-muted-foreground">
          <Paperclip className="h-3.5 w-3.5" /> 첨부 파일 (본인·교수만 열람)
        </span>
        <label className="inline-flex cursor-pointer items-center gap-1 rounded-md border bg-background px-2 py-1 text-xs hover:bg-muted">
          {uploading ? <Loader2 className="h-3 w-3 animate-spin" /> : <Upload className="h-3 w-3" />}
          파일 추가
          <input
            type="file"
            multiple
            className="hidden"
            accept=".pdf,.doc,.docx,.ppt,.pptx,.xls,.xlsx,.hwp,.hwpx,.txt,.md,.csv,.zip,.png,.jpg,.jpeg,.gif,.webp"
            disabled={uploading}
            onChange={(e) => {
              onPick(e.target.files)
              e.target.value = ''
            }}
          />
        </label>
      </div>
      {files.length === 0 ? (
        <p className="text-xs text-muted-foreground">아직 첨부한 파일이 없습니다. (PDF·DOCX·PPTX·HWP 등, 최대 20MB)</p>
      ) : (
        <ul className="space-y-1">
          {files.map((f) => (
            <li
              key={f.id}
              className="flex items-center gap-2 rounded-md bg-background px-2 py-1 text-xs"
            >
              <FileText className="h-3.5 w-3.5 flex-shrink-0 text-muted-foreground" />
              <button
                type="button"
                onClick={() => openMilestoneFile(f)}
                className="flex-1 truncate text-left hover:underline"
                title={f.filename}
              >
                {f.filename}
              </button>
              <span className="flex-shrink-0 text-muted-foreground">{formatFileSize(f.size)}</span>
              <button
                type="button"
                onClick={() => openMilestoneFile(f)}
                className="rounded p-0.5 hover:bg-muted"
                title="열기/다운로드"
              >
                <Download className="h-3.5 w-3.5" />
              </button>
              <button
                type="button"
                onClick={() => remove(f.id)}
                className="rounded p-0.5 text-destructive hover:bg-destructive/10"
                title="삭제"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
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
      {type === 'retrospective' && typeof milestone.ai_score === 'number' && (
        <AIScoreBadge score={milestone.ai_score} reasoning={milestone.ai_reasoning} />
      )}
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

function AIScoreBadge({ score, reasoning }: { score: number; reasoning?: string }) {
  const meta = aiScoreMeta(score)
  return (
    <div className={`flex items-center gap-2 rounded-md p-2 text-xs ${meta.chip}`}>
      <Bot className="h-4 w-4 flex-shrink-0" />
      <div className="flex-1">
        <div className="font-medium">
          AI 작성 확률 {score}점 — {meta.label}
        </div>
        {reasoning && <div className="mt-0.5 text-[11px] opacity-80">{reasoning}</div>}
      </div>
    </div>
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
  const isEssay = type === 'retrospective'
  const urlInvalid = url.trim() !== '' && !isValidMilestoneURL(url.trim())

  // #120 회고 에세이 — AI 점수 셀프체크 상태
  const [scoreResult, setScoreResult] = useState<EssayScoreResult | null>(null)
  const [scoring, setScoring] = useState(false)
  const charCount = content.trim().length
  const minChars = isEssay ? 800 : 0

  const checkAIScore = async () => {
    if (charCount < 200) {
      toast.error('200자 이상이어야 평가 가능합니다')
      return
    }
    setScoring(true)
    try {
      const r = await api.post<EssayScoreResult>('/milestones/essay/score', { text: content })
      setScoreResult(r)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'AI 점수 평가 실패')
    } finally {
      setScoring(false)
    }
  }

  const submit = async () => {
    if (needsURL && !url.trim()) {
      toast.error('배포 URL을 입력해주세요')
      return
    }
    if (urlInvalid) {
      toast.error('vercel.app 또는 자체 도메인만 인정됩니다 (AI Studio 등 제외)')
      return
    }
    if (isEssay && charCount < 200) {
      toast.error('회고 에세이는 200자 이상 써주세요 (권장 800자 이상)')
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
          <label className="text-xs text-muted-foreground">
            {isEssay ? '회고 에세이 본문' : '본문'}
          </label>
          {isEssay ? (
            <MarkdownEditor
              value={content}
              onChange={(v) => {
                setContent(v)
                setScoreResult(null) // 텍스트 바뀌면 점수 무효화
              }}
              rows={18}
              placeholder='"한 학기 동안 어떤 경험을 했고, 무엇을 배웠고, 어떻게 변했는지" 본인 말투로 솔직하게 써주세요. 800자 이상 권장.'
            />
          ) : (
            <Textarea
              value={content}
              onChange={(e) => {
                setContent(e.target.value)
                setScoreResult(null)
              }}
              placeholder={type === 'business_plan' ? '사업계획서 요약 또는 참고 링크 (파일은 아래에서 첨부)' : '본문'}
              rows={4}
            />
          )}
          {isEssay && (
            <div className="mt-1 flex items-center justify-between text-xs text-muted-foreground">
              <span className={charCount < minChars ? 'text-amber-600' : 'text-emerald-600'}>
                {charCount}자{charCount < minChars && ` (${minChars}자 권장)`}
              </span>
              <button
                type="button"
                onClick={checkAIScore}
                disabled={scoring || charCount < 200}
                className="inline-flex items-center gap-1 rounded-md border px-2 py-1 hover:bg-muted disabled:opacity-50"
              >
                <Sparkles className="h-3 w-3" />
                {scoring ? '평가 중…' : 'AI 작성 확률 셀프체크'}
              </button>
            </div>
          )}
        </div>
      )}

      {isEssay && scoreResult && <EssayScorePreview result={scoreResult} />}

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

function EssayScorePreview({ result }: { result: EssayScoreResult }) {
  const meta = aiScoreMeta(result.combined_score)
  return (
    <div className={`space-y-2 rounded-md border p-3 text-xs ${meta.chip}`}>
      <div className="flex items-center gap-2">
        <Bot className="h-4 w-4" />
        <span className="font-medium text-sm">
          AI 작성 확률 {result.combined_score}점 — {meta.label}
        </span>
      </div>
      <div className="flex flex-wrap gap-2 opacity-90">
        <span>휴리스틱 {result.heuristic_score}</span>
        {result.llm_score >= 0 && <span>· LLM {result.llm_score}</span>}
      </div>
      {result.llm_reasoning && (
        <div className="rounded bg-white/50 p-2">
          <strong>LLM:</strong> {result.llm_reasoning}
        </div>
      )}
      {(result.signals ?? []).length > 0 && (
        <details className="rounded bg-white/50 p-2">
          <summary className="cursor-pointer font-medium">개선 가이드 ({result.signals.length}개)</summary>
          <ul className="mt-1 space-y-1">
            {result.signals.map((s, i) => (
              <li key={i}>
                <strong>{s.label}</strong>: {s.hint}
              </li>
            ))}
          </ul>
        </details>
      )}
    </div>
  )
}
