import { useState, useEffect, useCallback } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import type { FreelanceJob, JobApplication } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { toast } from 'sonner'
import {
  ArrowLeft,
  Clock,
  User,
  Loader2,
  CheckCircle,
  Send,
  Star,
  FileText,
} from 'lucide-react'
import { MarkdownEditor } from '@/components/MarkdownEditor'
import { MarkdownContent } from '@/components/MarkdownContent'

function formatMoney(amount: number): string {
  return new Intl.NumberFormat('ko-KR').format(amount) + '원'
}

const statusLabels: Record<string, string> = {
  open: '모집 중',
  in_progress: '진행 중',
  completed: '완료',
  cancelled: '취소됨',
}

const statusVariant: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  open: 'default',
  in_progress: 'secondary',
  completed: 'outline',
  cancelled: 'destructive',
}

const appStatusLabels: Record<string, string> = {
  pending: '대기 중',
  accepted: '수락됨',
  rejected: '거절됨',
}

export default function MarketDetailPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const [job, setJob] = useState<FreelanceJob | null>(null)
  const [applications, setApplications] = useState<JobApplication[]>([])
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)

  // Apply form state
  const [showApplyForm, setShowApplyForm] = useState(false)
  const [proposal, setProposal] = useState('')
  const [price, setPrice] = useState('')

  // Complete work report form
  const [showReportForm, setShowReportForm] = useState(false)
  const [reportContent, setReportContent] = useState('')

  const fetchJob = useCallback(async () => {
    try {
      const data = await api.get<FreelanceJob>(`/freelance/jobs/${id}`)
      setJob(data)
    } catch {
      setJob(null)
    }
  }, [id])

  const fetchApplications = useCallback(async () => {
    try {
      const data = await api.get<JobApplication[]>(`/freelance/jobs/${id}/applications`)
      setApplications(data)
    } catch {
      setApplications([])
    }
  }, [id])

  useEffect(() => {
    Promise.all([fetchJob(), fetchApplications()]).finally(() => setLoading(false))
  }, [fetchJob, fetchApplications])

  const isClient = user?.id === job?.client?.id
  const isFreelancer = user?.id === job?.freelancer_id

  const handleApply = async (e: React.FormEvent) => {
    e.preventDefault()
    setActionLoading(true)
    try {
      await api.post(`/freelance/jobs/${id}/apply`, {
        proposal,
        price: Number(price),
      })
      toast.success('지원이 완료되었습니다.')
      setShowApplyForm(false)
      setProposal('')
      setPrice('')
      await fetchApplications()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '지원에 실패했습니다.')
    } finally {
      setActionLoading(false)
    }
  }

  const handleAccept = async (applicationId: number) => {
    setActionLoading(true)
    try {
      await api.post(`/freelance/jobs/${id}/accept`, { application_id: applicationId })
      toast.success('지원자를 수락했습니다.')
      await Promise.all([fetchJob(), fetchApplications()])
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '수락에 실패했습니다.')
    } finally {
      setActionLoading(false)
    }
  }

  const handleComplete = async () => {
    if (!reportContent.trim()) {
      toast.error('완료 보고서를 작성해주세요.')
      return
    }
    setActionLoading(true)
    try {
      await api.post(`/freelance/jobs/${id}/complete`, {
        report: reportContent,
      })
      toast.success('작업 완료를 보고했습니다. 외주마켓 게시판에 자동 포스팅됩니다.')
      setShowReportForm(false)
      setReportContent('')
      await fetchJob()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '완료 처리에 실패했습니다.')
    } finally {
      setActionLoading(false)
    }
  }

  const handleApprove = async () => {
    setActionLoading(true)
    try {
      await api.post(`/freelance/jobs/${id}/approve`)
      toast.success('작업을 승인했습니다.')
      await fetchJob()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '승인에 실패했습니다.')
    } finally {
      setActionLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  if (!job) {
    return (
      <div className="p-4 text-center text-muted-foreground">의뢰를 찾을 수 없습니다.</div>
    )
  }

  return (
    <div className="mx-auto max-w-lg space-y-4 p-4">
      <Button variant="ghost" size="sm" asChild>
        <Link to="/market">
          <ArrowLeft className="mr-1 h-4 w-4" />
          마켓으로 돌아가기
        </Link>
      </Button>

      {/* Job Detail */}
      <Card>
        <CardHeader>
          <div className="flex items-start justify-between">
            <CardTitle className="text-lg">{job.title}</CardTitle>
            <Badge variant={statusVariant[job.status] || 'secondary'}>
              {statusLabels[job.status] || job.status}
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-4 text-sm text-muted-foreground">
            <span className="flex items-center gap-1">
              <User className="h-4 w-4" />
              {job.client?.name}
            </span>
            {job.client?.rating != null && (
              <span className="flex items-center gap-1">
                <Star className="h-3 w-3 fill-yellow-400 text-yellow-400" />
                {job.client.rating.toFixed(1)}
              </span>
            )}
            {job.deadline && (
              <span className="flex items-center gap-1">
                <Clock className="h-4 w-4" />
                {new Date(job.deadline).toLocaleDateString('ko-KR')}
              </span>
            )}
          </div>
          <Separator />
          <div className="flex gap-6">
            <div>
              <p className="text-sm font-medium text-muted-foreground">예산</p>
              <p className="text-lg font-bold text-primary">{formatMoney(job.budget)}</p>
            </div>
            {job.agreed_price > 0 && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">합의 금액</p>
                <p className="text-lg font-bold">{formatMoney(job.agreed_price)}</p>
              </div>
            )}
          </div>
          <Separator />
          <div>
            <p className="mb-2 text-sm font-medium">상세 설명</p>
            <MarkdownContent content={job.description} maxLines={10} className="text-sm" />
          </div>
          {(() => {
            const skills = typeof job.required_skills === 'string'
              ? (job.required_skills as string).split(',').map(s => s.trim()).filter(Boolean)
              : Array.isArray(job.required_skills)
                ? job.required_skills
                : []
            return skills.length > 0 ? (
              <>
                <Separator />
                <div>
                  <p className="mb-2 text-sm font-medium">필요 기술</p>
                  <div className="flex flex-wrap gap-1">
                    {skills.map((skill) => (
                      <Badge key={skill} variant="outline">
                        {skill}
                      </Badge>
                    ))}
                  </div>
                </div>
              </>
            ) : null
          })()}
        </CardContent>
      </Card>

      {/* Action Buttons */}
      {job.status === 'open' && !isClient && (
        <Card>
          <CardContent className="p-4">
            {showApplyForm ? (
              <form onSubmit={handleApply} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="proposal">제안서</Label>
                  <MarkdownEditor
                    value={proposal}
                    onChange={setProposal}
                    placeholder="자신의 강점과 작업 계획을 설명해 주세요 (마크다운 지원)"
                    rows={8}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="price">희망 금액 (원)</Label>
                  <Input
                    id="price"
                    type="number"
                    placeholder="제안 금액"
                    value={price}
                    onChange={(e) => setPrice(e.target.value)}
                    required
                    min={1}
                  />
                </div>
                <div className="flex gap-2">
                  <Button type="submit" className="flex-1" disabled={actionLoading}>
                    {actionLoading ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <Send className="mr-2 h-4 w-4" />
                    )}
                    지원하기
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setShowApplyForm(false)}
                  >
                    취소
                  </Button>
                </div>
              </form>
            ) : (
              <Button className="w-full" onClick={() => setShowApplyForm(true)}>
                <Send className="mr-2 h-4 w-4" />
                이 의뢰에 지원하기
              </Button>
            )}
          </CardContent>
        </Card>
      )}

      {/* Freelancer: Complete work */}
      {job.status === 'in_progress' && isFreelancer && !job.work_completed && (
        <Card>
          <CardContent className="p-4">
            {showReportForm ? (
              <div className="space-y-4">
                <div className="flex items-center gap-2">
                  <FileText className="h-4 w-4" />
                  <h3 className="text-sm font-semibold">작업 완료 보고서</h3>
                </div>
                <p className="text-xs text-muted-foreground">
                  작업 내용을 정리하여 보고서를 작성해주세요. 이 보고서는 외주마켓 게시판에 자동 포스팅됩니다.
                </p>
                <MarkdownEditor
                  value={reportContent}
                  onChange={setReportContent}
                  placeholder="작업 내용, 결과물, 특이사항 등을 마크다운으로 작성하세요. 파일 첨부도 가능합니다."
                  rows={8}
                />
                <div className="flex gap-2">
                  <Button
                    className="flex-1"
                    onClick={handleComplete}
                    disabled={actionLoading || !reportContent.trim()}
                  >
                    {actionLoading ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <CheckCircle className="mr-2 h-4 w-4" />
                    )}
                    완료 보고 제출
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => setShowReportForm(false)}
                  >
                    취소
                  </Button>
                </div>
              </div>
            ) : (
              <Button className="w-full" onClick={() => setShowReportForm(true)}>
                <FileText className="mr-2 h-4 w-4" />
                작업 완료 보고
              </Button>
            )}
          </CardContent>
        </Card>
      )}

      {/* Client: Approve completed work */}
      {job.status === 'in_progress' && isClient && job.work_completed && (
        <Button className="w-full" onClick={handleApprove} disabled={actionLoading}>
          {actionLoading ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <CheckCircle className="mr-2 h-4 w-4" />
          )}
          작업 승인하기
        </Button>
      )}

      {/* Applications List */}
      {applications.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">
              지원자 ({applications.length}명)
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {applications.map((app) => (
              <div
                key={app.id}
                className="rounded-lg border p-3"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-sm">{app.user?.name}</span>
                    {app.user?.rating != null && (
                      <span className="flex items-center gap-0.5 text-xs text-muted-foreground">
                        <Star className="h-3 w-3 fill-yellow-400 text-yellow-400" />
                        {app.user.rating.toFixed(1)}
                      </span>
                    )}
                  </div>
                  <Badge
                    variant={app.status === 'accepted' ? 'default' : 'secondary'}
                    className="text-xs"
                  >
                    {appStatusLabels[app.status] || app.status}
                  </Badge>
                </div>
                <p className="mt-2 text-sm text-muted-foreground">{app.proposal}</p>
                <div className="mt-2 flex items-center justify-between">
                  <span className="text-sm font-medium">
                    희망 금액: {formatMoney(app.price)}
                  </span>
                  {isClient && job.status === 'open' && app.status === 'pending' && (
                    <Button
                      size="sm"
                      onClick={() => handleAccept(app.id)}
                      disabled={actionLoading}
                    >
                      {actionLoading ? (
                        <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                      ) : (
                        <CheckCircle className="mr-1 h-3 w-3" />
                      )}
                      수락
                    </Button>
                  )}
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
