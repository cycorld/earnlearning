import { useState, useEffect, useCallback } from 'react'
import { api } from '@/lib/api'
import type { Disclosure } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { MarkdownEditor } from '@/components/MarkdownEditor'
import { MarkdownContent } from '@/components/MarkdownContent'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { toast } from 'sonner'
import { FileText, Loader2, Plus } from 'lucide-react'
import { formatMoney, formatDate } from '@/lib/utils'

const statusLabels: Record<string, string> = {
  pending: '심사 중',
  approved: '승인',
  rejected: '거절',
}

const statusVariant: Record<string, 'default' | 'secondary' | 'destructive'> = {
  pending: 'secondary',
  approved: 'default',
  rejected: 'destructive',
}

interface Props {
  companyId: number
  isOwner: boolean
}

export function DisclosureSection({ companyId, isOwner }: Props) {
  const [disclosures, setDisclosures] = useState<Disclosure[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [createLoading, setCreateLoading] = useState(false)
  const [form, setForm] = useState({ content: '', period_from: '', period_to: '' })
  const [selectedDisclosure, setSelectedDisclosure] = useState<Disclosure | null>(null)

  const fetchDisclosures = useCallback(async () => {
    try {
      const data = await api.get<Disclosure[]>(`/companies/${companyId}/disclosures`)
      setDisclosures(data ?? [])
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [companyId])

  useEffect(() => {
    fetchDisclosures()
  }, [fetchDisclosures])

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    if (!form.content.trim() || !form.period_from || !form.period_to) {
      toast.error('모든 필드를 입력해주세요.')
      return
    }
    setCreateLoading(true)
    try {
      await api.post(`/companies/${companyId}/disclosures`, {
        content: form.content.trim(),
        period_from: form.period_from,
        period_to: form.period_to,
      })
      toast.success('공시가 등록되었습니다.')
      setCreateOpen(false)
      setForm({ content: '', period_from: '', period_to: '' })
      await fetchDisclosures()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '공시 등록에 실패했습니다.')
    } finally {
      setCreateLoading(false)
    }
  }

  if (loading) return null

  return (
    <>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-base">공시</CardTitle>
          {isOwner && (
            <Button size="sm" variant="outline" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-1 h-3 w-3" />
              작성
            </Button>
          )}
        </CardHeader>
        <CardContent>
          {disclosures.length === 0 ? (
            <p className="text-center text-sm text-muted-foreground py-4">
              아직 공시가 없습니다.
            </p>
          ) : (
            <div className="space-y-3">
              {disclosures.map((d) => (
                <div
                  key={d.id}
                  className="flex cursor-pointer items-start justify-between rounded-lg border p-3 hover:bg-muted/50"
                  onClick={() => setSelectedDisclosure(d)}
                >
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <FileText className="h-4 w-4 text-muted-foreground shrink-0" />
                      <span className="text-sm font-medium">
                        {formatDate(d.period_from)} ~ {formatDate(d.period_to)}
                      </span>
                      <Badge variant={statusVariant[d.status] || 'secondary'}>
                        {statusLabels[d.status] || d.status}
                      </Badge>
                    </div>
                    <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">
                      {d.content.substring(0, 100)}
                    </p>
                  </div>
                  {d.reward > 0 && (
                    <span className="ml-2 shrink-0 text-sm font-medium text-primary">
                      +{formatMoney(d.reward)}
                    </span>
                  )}
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create disclosure dialog */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>공시 작성</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleCreate} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="period-from">기간 시작</Label>
                <Input
                  id="period-from"
                  type="date"
                  value={form.period_from}
                  onChange={(e) => setForm({ ...form, period_from: e.target.value })}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="period-to">기간 종료</Label>
                <Input
                  id="period-to"
                  type="date"
                  value={form.period_to}
                  onChange={(e) => setForm({ ...form, period_to: e.target.value })}
                  required
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="disc-content">성과 내용</Label>
              <MarkdownEditor
                value={form.content}
                onChange={(v) => setForm({ ...form, content: v })}
                placeholder="이번 주 성과를 작성해주세요 (마크다운 지원)"
                rows={8}
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => setCreateOpen(false)}>
                취소
              </Button>
              <Button type="submit" disabled={createLoading}>
                {createLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                등록
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>

      {/* View disclosure detail dialog */}
      <Dialog open={!!selectedDisclosure} onOpenChange={() => setSelectedDisclosure(null)}>
        <DialogContent className="max-h-[90vh] overflow-y-auto">
          {selectedDisclosure && (
            <>
              <DialogHeader>
                <DialogTitle className="flex items-center gap-2">
                  공시 상세
                  <Badge variant={statusVariant[selectedDisclosure.status] || 'secondary'}>
                    {statusLabels[selectedDisclosure.status] || selectedDisclosure.status}
                  </Badge>
                </DialogTitle>
              </DialogHeader>
              <div className="space-y-4">
                <div className="text-sm text-muted-foreground">
                  {formatDate(selectedDisclosure.period_from)} ~ {formatDate(selectedDisclosure.period_to)}
                </div>
                <MarkdownContent content={selectedDisclosure.content} className="text-sm" />
                {selectedDisclosure.reward > 0 && (
                  <div className="rounded-lg bg-primary/5 p-3">
                    <span className="text-sm font-medium">
                      수익금: {formatMoney(selectedDisclosure.reward)}
                    </span>
                  </div>
                )}
                {selectedDisclosure.admin_note && (
                  <div className="rounded-lg bg-muted p-3">
                    <p className="text-xs font-medium text-muted-foreground mb-1">관리자 코멘트</p>
                    <p className="text-sm">{selectedDisclosure.admin_note}</p>
                  </div>
                )}
              </div>
            </>
          )}
        </DialogContent>
      </Dialog>
    </>
  )
}
