import { useState, useEffect, useCallback } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import type { Grant, GrantApplication } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { toast } from 'sonner'
import { ArrowLeft, Loader2, CheckCircle, Send } from 'lucide-react'
import { MarkdownEditor } from '@/components/MarkdownEditor'
import { MarkdownContent } from '@/components/MarkdownContent'
import { formatMoney } from '@/lib/utils'

const statusLabels: Record<string, string> = {
  open: '모집 중',
  closed: '종료',
}

const appStatusLabels: Record<string, string> = {
  pending: '심사 중',
  approved: '승인됨',
  rejected: '거절됨',
}

export default function GrantDetailPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const [grant, setGrant] = useState<Grant | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)

  const [showApplyForm, setShowApplyForm] = useState(false)
  const [proposal, setProposal] = useState('')

  const isAdmin = user?.role === 'admin'

  const fetchGrant = useCallback(async () => {
    try {
      const data = await api.get<Grant>(`/grants/${id}`)
      setGrant(data)
    } catch {
      setGrant(null)
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    fetchGrant()
  }, [fetchGrant])

  const hasApplied = grant?.applications?.some((a) => a.user?.id === user?.id) ?? false

  const handleApply = async (e: React.FormEvent) => {
    e.preventDefault()
    setActionLoading(true)
    try {
      await api.post(`/grants/${id}/apply`, { proposal })
      toast.success('지원이 완료되었습니다.')
      setShowApplyForm(false)
      setProposal('')
      await fetchGrant()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '지원에 실패했습니다.')
    } finally {
      setActionLoading(false)
    }
  }

  const handleApprove = async (appId: number) => {
    setActionLoading(true)
    try {
      await api.post(`/admin/grants/${id}/approve/${appId}`, {})
      toast.success('지원자를 승인했습니다. 보상이 지급됩니다.')
      await fetchGrant()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '승인에 실패했습니다.')
    } finally {
      setActionLoading(false)
    }
  }

  const handleClose = async () => {
    setActionLoading(true)
    try {
      await api.post(`/admin/grants/${id}/close`, {})
      toast.success('과제가 종료되었습니다.')
      await fetchGrant()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '종료에 실패했습니다.')
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

  if (!grant) {
    return (
      <div className="p-4 text-center text-muted-foreground">과제를 찾을 수 없습니다.</div>
    )
  }

  const applications: GrantApplication[] = grant.applications ?? []

  return (
    <div className="mx-auto max-w-lg space-y-4 p-4">
      <Button variant="ghost" size="sm" asChild>
        <Link to="/grant">
          <ArrowLeft className="mr-1 h-4 w-4" />
          과제 목록으로
        </Link>
      </Button>

      <Card>
        <CardHeader>
          <div className="flex items-start justify-between">
            <CardTitle className="text-lg">{grant.title}</CardTitle>
            <Badge variant={grant.status === 'open' ? 'default' : 'secondary'}>
              {statusLabels[grant.status] || grant.status}
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-6">
            <div>
              <p className="text-sm font-medium text-muted-foreground">보상</p>
              <p className="text-lg font-bold text-primary">{formatMoney(grant.reward)}</p>
            </div>
            {grant.max_applicants > 0 && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">정원</p>
                <p className="text-lg font-bold">{grant.max_applicants}명</p>
              </div>
            )}
          </div>
          <Separator />
          <div>
            <p className="mb-2 text-sm font-medium">상세 설명</p>
            <MarkdownContent content={grant.description} maxLines={10} className="text-sm" />
          </div>
        </CardContent>
      </Card>

      {/* Apply button for non-admin users */}
      {grant.status === 'open' && !isAdmin && !hasApplied && (
        <Card>
          <CardContent className="p-4">
            {showApplyForm ? (
              <form onSubmit={handleApply} className="space-y-4">
                <div className="space-y-2">
                  <p className="text-sm font-medium">지원서</p>
                  <MarkdownEditor
                    value={proposal}
                    onChange={setProposal}
                    placeholder="지원 동기와 계획을 작성해 주세요"
                    rows={6}
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
                  <Button type="button" variant="outline" onClick={() => setShowApplyForm(false)}>
                    취소
                  </Button>
                </div>
              </form>
            ) : (
              <Button className="w-full" onClick={() => setShowApplyForm(true)}>
                <Send className="mr-2 h-4 w-4" />
                이 과제에 지원하기
              </Button>
            )}
          </CardContent>
        </Card>
      )}

      {grant.status === 'open' && !isAdmin && hasApplied && (
        <Card>
          <CardContent className="p-4 text-center text-sm text-muted-foreground">
            이미 지원한 과제입니다. 승인을 기다려주세요.
          </CardContent>
        </Card>
      )}

      {/* Admin: Close grant */}
      {isAdmin && grant.status === 'open' && (
        <Button
          variant="outline"
          className="w-full"
          onClick={handleClose}
          disabled={actionLoading}
        >
          과제 종료
        </Button>
      )}

      {/* Applications list */}
      {applications.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">지원자 ({applications.length}명)</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {applications.map((app) => (
              <div key={app.id} className="rounded-lg border p-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">{app.user?.name}</span>
                  <Badge
                    variant={app.status === 'approved' ? 'default' : 'secondary'}
                    className="text-xs"
                  >
                    {appStatusLabels[app.status] || app.status}
                  </Badge>
                </div>
                {app.proposal && (
                  <MarkdownContent content={app.proposal} className="mt-2 text-sm" />
                )}
                {isAdmin && app.status === 'pending' && grant.status === 'open' && (
                  <div className="mt-2 flex justify-end">
                    <Button
                      size="sm"
                      onClick={() => handleApprove(app.id)}
                      disabled={actionLoading}
                    >
                      {actionLoading ? (
                        <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                      ) : (
                        <CheckCircle className="mr-1 h-3 w-3" />
                      )}
                      승인 (보상 지급)
                    </Button>
                  </div>
                )}
              </div>
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
