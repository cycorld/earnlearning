import { useState, useEffect, useCallback } from 'react'
import { api } from '@/lib/api'
import type { Disclosure } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { MarkdownContent } from '@/components/MarkdownContent'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { toast } from 'sonner'
import { ArrowLeft, CheckCircle2, FileText, Loader2, XCircle } from 'lucide-react'
import { Link } from 'react-router-dom'
import { formatMoney, formatDate } from '@/lib/utils'
import { Spinner } from '@/components/ui/spinner'

const statusLabels: Record<string, string> = {
  pending: '심사 대기',
  approved: '승인',
  rejected: '거절',
}

const statusVariant: Record<string, 'default' | 'secondary' | 'destructive'> = {
  pending: 'secondary',
  approved: 'default',
  rejected: 'destructive',
}

export default function AdminDisclosuresPage() {
  const [disclosures, setDisclosures] = useState<Disclosure[]>([])
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<Disclosure | null>(null)
  const [reward, setReward] = useState('')
  const [adminNote, setAdminNote] = useState('')
  const [actionLoading, setActionLoading] = useState(false)

  const fetchAll = useCallback(async () => {
    try {
      const data = await api.get<Disclosure[]>('/admin/disclosures')
      setDisclosures(data ?? [])
    } catch {
      toast.error('공시 목록을 불러오는데 실패했습니다.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchAll()
  }, [fetchAll])

  function openReview(d: Disclosure) {
    setSelected(d)
    setReward(d.reward > 0 ? String(d.reward) : '')
    setAdminNote(d.admin_note || '')
  }

  async function handleApprove() {
    if (!selected) return
    setActionLoading(true)
    try {
      await api.post(`/admin/disclosures/${selected.id}/approve`, {
        reward: Number(reward) || 0,
        admin_note: adminNote.trim(),
      })
      toast.success('공시가 승인되었습니다.')
      setSelected(null)
      await fetchAll()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '승인에 실패했습니다.')
    } finally {
      setActionLoading(false)
    }
  }

  async function handleReject() {
    if (!selected) return
    setActionLoading(true)
    try {
      await api.post(`/admin/disclosures/${selected.id}/reject`, {
        admin_note: adminNote.trim(),
      })
      toast.success('공시가 거절되었습니다.')
      setSelected(null)
      await fetchAll()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '거절에 실패했습니다.')
    } finally {
      setActionLoading(false)
    }
  }

  const pendingCount = disclosures.filter((d) => d.status === 'pending').length

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Spinner />
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="icon" asChild>
          <Link to="/admin">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <h1 className="text-xl font-bold">공시 관리</h1>
        {pendingCount > 0 && (
          <Badge variant="secondary">{pendingCount}건 대기</Badge>
        )}
      </div>

      {disclosures.length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            등록된 공시가 없습니다.
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {disclosures.map((d) => (
            <Card
              key={d.id}
              className="cursor-pointer transition-colors hover:bg-accent"
              onClick={() => openReview(d)}
            >
              <CardContent className="p-4">
                <div className="flex items-start justify-between">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <FileText className="h-4 w-4 text-muted-foreground shrink-0" />
                      <span className="text-sm font-medium">{d.company_name}</span>
                      <Badge variant={statusVariant[d.status] || 'secondary'}>
                        {statusLabels[d.status] || d.status}
                      </Badge>
                    </div>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {formatDate(d.period_from)} ~ {formatDate(d.period_to)} | 작성자: {d.author_name}
                    </p>
                    <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">
                      {d.content.substring(0, 120)}
                    </p>
                  </div>
                  {d.reward > 0 && (
                    <span className="ml-2 shrink-0 text-sm font-medium text-primary">
                      {formatMoney(d.reward)}
                    </span>
                  )}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Review dialog */}
      <Dialog open={!!selected} onOpenChange={() => setSelected(null)}>
        <DialogContent className="max-h-[90vh] overflow-y-auto">
          {selected && (
            <>
              <DialogHeader>
                <DialogTitle className="flex items-center gap-2">
                  {selected.company_name} 공시 리뷰
                  <Badge variant={statusVariant[selected.status] || 'secondary'}>
                    {statusLabels[selected.status] || selected.status}
                  </Badge>
                </DialogTitle>
              </DialogHeader>
              <div className="space-y-4">
                <div className="text-sm text-muted-foreground">
                  기간: {formatDate(selected.period_from)} ~ {formatDate(selected.period_to)}
                  <br />
                  작성자: {selected.author_name}
                </div>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-sm">공시 내용</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <MarkdownContent content={selected.content} className="text-sm" />
                  </CardContent>
                </Card>

                {selected.status === 'pending' && (
                  <>
                    <div className="space-y-2">
                      <Label htmlFor="reward">수익금 (원)</Label>
                      <Input
                        id="reward"
                        type="number"
                        min={0}
                        value={reward}
                        onChange={(e) => setReward(e.target.value)}
                        placeholder="승인 시 법인 계좌에 입금할 금액"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="admin-note">관리자 코멘트</Label>
                      <Textarea
                        id="admin-note"
                        value={adminNote}
                        onChange={(e) => setAdminNote(e.target.value)}
                        placeholder="학생에게 전달할 코멘트"
                        rows={3}
                      />
                    </div>
                    <div className="flex justify-end gap-2">
                      <Button
                        variant="destructive"
                        onClick={handleReject}
                        disabled={actionLoading}
                      >
                        {actionLoading ? (
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        ) : (
                          <XCircle className="mr-2 h-4 w-4" />
                        )}
                        거절
                      </Button>
                      <Button onClick={handleApprove} disabled={actionLoading}>
                        {actionLoading ? (
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        ) : (
                          <CheckCircle2 className="mr-2 h-4 w-4" />
                        )}
                        승인 {Number(reward) > 0 && `(${formatMoney(Number(reward))})`}
                      </Button>
                    </div>
                  </>
                )}

                {selected.status !== 'pending' && (
                  <>
                    {selected.reward > 0 && (
                      <div className="rounded-lg bg-primary/5 p-3">
                        <span className="text-sm font-medium">
                          수익금: {formatMoney(selected.reward)}
                        </span>
                      </div>
                    )}
                    {selected.admin_note && (
                      <div className="rounded-lg bg-muted p-3">
                        <p className="text-xs font-medium text-muted-foreground mb-1">
                          관리자 코멘트
                        </p>
                        <p className="text-sm">{selected.admin_note}</p>
                      </div>
                    )}
                  </>
                )}
              </div>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
