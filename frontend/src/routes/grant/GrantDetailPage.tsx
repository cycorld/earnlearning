import { useState, useEffect, useCallback } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import type { Grant, GrantApplication } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { toast } from 'sonner'
import { ArrowLeft, Loader2, CheckCircle, Send, Pencil, Trash2, X } from 'lucide-react'
import { MarkdownEditor } from '@/components/MarkdownEditor'
import { MarkdownContent } from '@/components/MarkdownContent'
import { formatMoney, displayName } from '@/lib/utils'

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

  // Edit state
  const [editingAppId, setEditingAppId] = useState<number | null>(null)
  const [editProposal, setEditProposal] = useState('')

  // Delete state
  const [deleteAppId, setDeleteAppId] = useState<number | null>(null)
  const [deleteConfirmText, setDeleteConfirmText] = useState('')

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

  const handleRevoke = async (appId: number) => {
    if (!confirm('승인을 취소하시겠습니까? 지급된 보상금이 회수됩니다.')) return
    setActionLoading(true)
    try {
      await api.post(`/admin/grants/${id}/revoke/${appId}`, {})
      toast.success('승인이 취소되었습니다. 보상금이 회수됩니다.')
      await fetchGrant()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '승인 취소에 실패했습니다.')
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

  const handleUpdate = async (appId: number) => {
    setActionLoading(true)
    try {
      await api.put(`/grants/${id}/applications/${appId}`, { proposal: editProposal })
      toast.success('지원서가 수정되었습니다.')
      setEditingAppId(null)
      await fetchGrant()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '수정에 실패했습니다.')
    } finally {
      setActionLoading(false)
    }
  }

  const handleDelete = async () => {
    if (!deleteAppId || deleteConfirmText !== '삭제') return
    const appId = deleteAppId
    setActionLoading(true)
    try {
      await api.del(`/grants/${id}/applications/${appId}`)
      toast.success('지원서가 삭제되었습니다.')
      setDeleteAppId(null)
      setDeleteConfirmText('')
      await fetchGrant()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '삭제에 실패했습니다.')
    } finally {
      setActionLoading(false)
    }
  }

  const startEdit = (app: GrantApplication) => {
    setEditingAppId(app.id)
    setEditProposal(app.proposal || '')
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
      <div className="sticky top-14 z-40 -mx-4 bg-background px-4 py-1">
        <Button variant="ghost" size="sm" asChild>
          <Link to="/grant">
            <ArrowLeft className="mr-1 h-4 w-4" />
            과제 목록으로
          </Link>
        </Button>
      </div>

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
            {applications.map((app) => {
              const isOwner = app.user?.id === user?.id
              const canEdit = isOwner && app.status === 'pending'
              const isEditing = editingAppId === app.id

              return (
                <div key={app.id} className="rounded-lg border p-3">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium">{displayName(app.user)}</span>
                    <div className="flex items-center gap-2">
                      <Badge
                        variant={app.status === 'approved' ? 'default' : 'secondary'}
                        className="text-xs"
                      >
                        {appStatusLabels[app.status] || app.status}
                      </Badge>
                      {canEdit && !isEditing && (
                        <>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7"
                            onClick={() => startEdit(app)}
                          >
                            <Pencil className="h-3.5 w-3.5" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7 text-destructive hover:text-destructive"
                            onClick={() => { setDeleteAppId(app.id); setDeleteConfirmText('') }}
                            disabled={actionLoading}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </>
                      )}
                    </div>
                  </div>
                  {isEditing ? (
                    <div className="mt-2 space-y-2">
                      <MarkdownEditor
                        value={editProposal}
                        onChange={setEditProposal}
                        placeholder="지원서를 수정해 주세요"
                        rows={6}
                      />
                      <div className="flex gap-2 justify-end">
                        <Button
                          size="sm"
                          onClick={() => handleUpdate(app.id)}
                          disabled={actionLoading}
                        >
                          {actionLoading ? (
                            <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                          ) : (
                            <CheckCircle className="mr-1 h-3 w-3" />
                          )}
                          저장
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => setEditingAppId(null)}
                        >
                          <X className="mr-1 h-3 w-3" />
                          취소
                        </Button>
                      </div>
                    </div>
                  ) : (
                    app.proposal && (
                      <MarkdownContent content={app.proposal} className="mt-2 text-sm" />
                    )
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
                  {isAdmin && app.status === 'approved' && (
                    <div className="mt-2 flex justify-end">
                      <Button
                        size="sm"
                        variant="destructive"
                        onClick={() => handleRevoke(app.id)}
                        disabled={actionLoading}
                      >
                        {actionLoading ? (
                          <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                        ) : (
                          <X className="mr-1 h-3 w-3" />
                        )}
                        승인 취소 (보상 회수)
                      </Button>
                    </div>
                  )}
                </div>
              )
            })}
          </CardContent>
        </Card>
      )}

      {/* Delete confirmation dialog */}
      <Dialog open={deleteAppId !== null} onOpenChange={(open) => { if (!open) { setDeleteAppId(null); setDeleteConfirmText('') } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>지원서 삭제</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              지원서를 정말 삭제하시겠습니까? 삭제된 지원서는 복구할 수 없습니다.
            </p>
            <p className="text-sm font-medium">
              확인을 위해 아래에 <span className="text-destructive">"삭제"</span>를 입력하세요.
            </p>
            <Input
              value={deleteConfirmText}
              onChange={(e) => setDeleteConfirmText(e.target.value)}
              placeholder="삭제"
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setDeleteAppId(null); setDeleteConfirmText('') }}>
              취소
            </Button>
            <Button
              variant="destructive"
              disabled={deleteConfirmText !== '삭제' || actionLoading}
              onClick={handleDelete}
            >
              {actionLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              삭제
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
