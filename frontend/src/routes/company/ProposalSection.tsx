import { useState, useEffect, useCallback } from 'react'
import { api, ApiError } from '@/lib/api'
import type { Proposal, ProposalType, VoteChoice } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { toast } from 'sonner'
import { Vote, Loader2, Plus, CheckCircle2, XCircle, Hammer } from 'lucide-react'

const statusLabels: Record<string, string> = {
  active: '진행 중',
  passed: '가결',
  rejected: '부결',
  cancelled: '취소',
  executed: '집행 완료',
}

const statusVariant: Record<
  string,
  'default' | 'secondary' | 'destructive' | 'outline'
> = {
  active: 'secondary',
  passed: 'default',
  rejected: 'destructive',
  cancelled: 'outline',
  executed: 'default',
}

const typeLabels: Record<ProposalType, string> = {
  general: '일반 안건',
  liquidation: '회사 청산',
}

interface Props {
  companyId: number
  isShareholder: boolean
  onCompanyChanged?: () => void
}

export function ProposalSection({
  companyId,
  isShareholder,
  onCompanyChanged,
}: Props) {
  const [proposals, setProposals] = useState<Proposal[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [createLoading, setCreateLoading] = useState(false)
  const [voteLoading, setVoteLoading] = useState<number | null>(null)
  const [executeLoading, setExecuteLoading] = useState<number | null>(null)
  const [form, setForm] = useState<{
    proposal_type: ProposalType
    title: string
    description: string
    pass_threshold: string
    duration_days: string
  }>({
    proposal_type: 'general',
    title: '',
    description: '',
    pass_threshold: '',
    duration_days: '7',
  })

  const fetchProposals = useCallback(async () => {
    try {
      const data = await api.get<Proposal[]>(
        `/companies/${companyId}/proposals`,
      )
      setProposals(data ?? [])
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [companyId])

  useEffect(() => {
    fetchProposals()
  }, [fetchProposals])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!form.title.trim()) {
      toast.error('안건 제목을 입력해주세요.')
      return
    }
    setCreateLoading(true)
    try {
      const payload: Record<string, unknown> = {
        proposal_type: form.proposal_type,
        title: form.title.trim(),
        description: form.description.trim(),
      }
      if (form.pass_threshold) {
        payload.pass_threshold = parseInt(form.pass_threshold, 10)
      }
      if (form.duration_days) {
        payload.duration_days = parseInt(form.duration_days, 10)
      }
      await api.post(`/companies/${companyId}/proposals`, payload)
      toast.success('안건이 상정되었습니다.')
      setCreateOpen(false)
      setForm({
        proposal_type: 'general',
        title: '',
        description: '',
        pass_threshold: '',
        duration_days: '7',
      })
      await fetchProposals()
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : '안건 상정에 실패했습니다.',
      )
    } finally {
      setCreateLoading(false)
    }
  }

  const handleVote = async (proposalId: number, choice: VoteChoice) => {
    setVoteLoading(proposalId)
    try {
      await api.post(`/proposals/${proposalId}/vote`, { choice })
      toast.success(choice === 'yes' ? '찬성 투표 완료' : '반대 투표 완료')
      await fetchProposals()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '투표 실패')
    } finally {
      setVoteLoading(null)
    }
  }

  const handleExecuteLiquidation = async (proposalId: number) => {
    if (
      !window.confirm(
        '정말 청산을 집행하시겠어요?\n세금 20%를 제외한 잔액이 주주별 지분율에 따라 분배되고, 회사는 영구 정지됩니다.',
      )
    ) {
      return
    }
    setExecuteLoading(proposalId)
    try {
      const result = await api.post<{
        total_balance: number
        tax: number
        distributable: number
        payouts: { user_name: string; amount: number }[]
      }>(`/proposals/${proposalId}/execute`, {})
      toast.success(
        `청산 완료! 세금 ${result.tax.toLocaleString()}원, 분배 ${result.distributable.toLocaleString()}원`,
      )
      await fetchProposals()
      onCompanyChanged?.()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '청산 집행 실패')
    } finally {
      setExecuteLoading(null)
    }
  }

  if (loading) {
    return (
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center gap-2 text-base">
            <Vote className="h-4 w-4" />
            주주총회
          </CardTitle>
        </CardHeader>
        <CardContent className="flex justify-center py-4">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <Vote className="h-4 w-4" />
          주주총회
        </CardTitle>
        {isShareholder && (
          <Button
            size="sm"
            variant="outline"
            className="h-7 gap-1 text-xs"
            onClick={() => setCreateOpen(true)}
          >
            <Plus className="h-3 w-3" /> 안건 상정
          </Button>
        )}
      </CardHeader>
      <CardContent className="space-y-3">
        {proposals.length === 0 ? (
          <p className="py-4 text-center text-sm text-muted-foreground">
            아직 상정된 안건이 없어요.
          </p>
        ) : (
          proposals.map((p) => (
            <ProposalCard
              key={p.id}
              proposal={p}
              isShareholder={isShareholder}
              voteLoading={voteLoading === p.id}
              executeLoading={executeLoading === p.id}
              onVote={(choice) => handleVote(p.id, choice)}
              onExecute={() => handleExecuteLiquidation(p.id)}
            />
          ))
        )}
      </CardContent>

      {/* Create proposal dialog */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>새 주주총회 안건 상정</DialogTitle>
            <DialogDescription>
              주주가 투표할 안건을 상정합니다. 안건 종류에 따라 기본 가결 기준이
              다릅니다 (일반 50%, 청산 70%).
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleCreate} className="space-y-3">
            <div className="space-y-1">
              <Label>안건 종류</Label>
              <Select
                value={form.proposal_type}
                onValueChange={(v) =>
                  setForm({ ...form, proposal_type: v as ProposalType })
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="general">일반 안건</SelectItem>
                  <SelectItem value="liquidation">회사 청산</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {form.proposal_type === 'liquidation' && (
              <div className="rounded-md border border-destructive/30 bg-destructive/5 p-3 text-xs text-destructive-foreground">
                <p className="font-medium text-destructive">⚠️ 청산 안내</p>
                <ul className="mt-1 list-disc space-y-0.5 pl-4 text-muted-foreground">
                  <li>가결 시 회사 자산의 <strong>20%는 세금</strong>으로 납부됩니다.</li>
                  <li>나머지 80%가 <strong>주주별 지분율에 따라 자동 분배</strong>됩니다.</li>
                  <li>가결 즉시 집행되며, <strong>회사는 영구 정지</strong>됩니다.</li>
                </ul>
              </div>
            )}
            <div className="space-y-1">
              <Label>제목</Label>
              <Input
                value={form.title}
                onChange={(e) => setForm({ ...form, title: e.target.value })}
                placeholder="예: 배당 지급 승인 건"
                required
              />
            </div>
            <div className="space-y-1">
              <Label>설명</Label>
              <Textarea
                value={form.description}
                onChange={(e) =>
                  setForm({ ...form, description: e.target.value })
                }
                placeholder="안건 배경, 근거, 기대 효과 등"
                rows={4}
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label>가결 기준(%)</Label>
                <Input
                  type="number"
                  min={1}
                  max={100}
                  value={form.pass_threshold}
                  onChange={(e) =>
                    setForm({ ...form, pass_threshold: e.target.value })
                  }
                  placeholder={form.proposal_type === 'liquidation' ? '70' : '50'}
                />
              </div>
              <div className="space-y-1">
                <Label>투표 기간(일)</Label>
                <Input
                  type="number"
                  min={1}
                  max={30}
                  value={form.duration_days}
                  onChange={(e) =>
                    setForm({ ...form, duration_days: e.target.value })
                  }
                />
              </div>
            </div>
            <div className="flex justify-end gap-2 pt-2">
              <Button
                type="button"
                variant="ghost"
                onClick={() => setCreateOpen(false)}
              >
                취소
              </Button>
              <Button type="submit" disabled={createLoading}>
                {createLoading && (
                  <Loader2 className="mr-1 h-4 w-4 animate-spin" />
                )}
                상정
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </Card>
  )
}

function ProposalCard({
  proposal,
  isShareholder,
  voteLoading,
  executeLoading,
  onVote,
  onExecute,
}: {
  proposal: Proposal
  isShareholder: boolean
  voteLoading: boolean
  executeLoading: boolean
  onVote: (choice: VoteChoice) => void
  onExecute: () => void
}) {
  const tally = proposal.tally
  const yesPct = tally?.yes_percent ?? 0
  const noPct = tally?.no_percent ?? 0
  const remainPct = Math.max(0, 100 - yesPct - noPct)
  const canVote =
    isShareholder && proposal.status === 'active' && !proposal.my_vote
  const canExecuteLiquidation =
    isShareholder &&
    proposal.proposal_type === 'liquidation' &&
    proposal.status === 'passed'

  return (
    <div className="rounded-md border p-3">
      <div className="mb-2 flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <Badge variant={statusVariant[proposal.status] ?? 'secondary'}>
              {statusLabels[proposal.status] ?? proposal.status}
            </Badge>
            <span className="text-xs text-muted-foreground">
              {typeLabels[proposal.proposal_type]}
            </span>
            <span className="text-xs text-muted-foreground">
              가결 {proposal.pass_threshold}%
            </span>
          </div>
          <p className="mt-1 truncate text-sm font-medium">{proposal.title}</p>
          {proposal.description && (
            <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">
              {proposal.description}
            </p>
          )}
        </div>
      </div>

      {/* Tally bar */}
      <div className="mb-2 flex h-2 w-full overflow-hidden rounded-full bg-muted">
        <div
          className="bg-success"
          style={{ width: `${yesPct}%` }}
          title={`찬성 ${yesPct.toFixed(1)}%`}
        />
        <div
          className="bg-coral"
          style={{ width: `${noPct}%` }}
          title={`반대 ${noPct.toFixed(1)}%`}
        />
        <div
          className="bg-gray-300 dark:bg-gray-700"
          style={{ width: `${remainPct}%` }}
          title={`미투표 ${remainPct.toFixed(1)}%`}
        />
      </div>
      <div className="mb-2 flex justify-between text-xs text-muted-foreground">
        <span className="flex items-center gap-1">
          <CheckCircle2 className="h-3 w-3 text-success" />
          찬성 {yesPct.toFixed(1)}% ({tally?.yes_shares ?? 0}주)
        </span>
        <span className="flex items-center gap-1">
          <XCircle className="h-3 w-3 text-coral" />
          반대 {noPct.toFixed(1)}% ({tally?.no_shares ?? 0}주)
        </span>
      </div>

      {proposal.result_note && (
        <p className="mb-2 rounded bg-muted/30 p-2 text-xs text-muted-foreground">
          {proposal.result_note}
        </p>
      )}

      {proposal.my_vote && (
        <p className="mb-2 text-xs text-muted-foreground">
          내 투표:{' '}
          <span
            className={
              proposal.my_vote.choice === 'yes'
                ? 'font-medium text-success'
                : 'font-medium text-coral'
            }
          >
            {proposal.my_vote.choice === 'yes' ? '찬성' : '반대'}
          </span>{' '}
          ({proposal.my_vote.shares_at_vote}주)
        </p>
      )}

      {canVote && (
        <div className="flex gap-2">
          <Button
            size="sm"
            variant="outline"
            className="flex-1 border-success text-success hover:bg-success/10"
            disabled={voteLoading}
            onClick={() => onVote('yes')}
          >
            {voteLoading ? (
              <Loader2 className="h-3 w-3 animate-spin" />
            ) : (
              <>
                <CheckCircle2 className="mr-1 h-3 w-3" /> 찬성
              </>
            )}
          </Button>
          <Button
            size="sm"
            variant="outline"
            className="flex-1 border-coral text-coral hover:bg-coral/10"
            disabled={voteLoading}
            onClick={() => onVote('no')}
          >
            {voteLoading ? (
              <Loader2 className="h-3 w-3 animate-spin" />
            ) : (
              <>
                <XCircle className="mr-1 h-3 w-3" /> 반대
              </>
            )}
          </Button>
        </div>
      )}

      {canExecuteLiquidation && (
        <Button
          size="sm"
          variant="destructive"
          className="w-full"
          disabled={executeLoading}
          onClick={onExecute}
        >
          {executeLoading ? (
            <Loader2 className="mr-1 h-3 w-3 animate-spin" />
          ) : (
            <Hammer className="mr-1 h-3 w-3" />
          )}
          청산 집행 (세금 20% + 주주 분배)
        </Button>
      )}

      <p className="mt-2 text-[10px] text-muted-foreground">
        상정: {proposal.proposer_name ?? '알 수 없음'} ·{' '}
        {new Date(proposal.start_date).toLocaleDateString('ko-KR')} ~{' '}
        {new Date(proposal.end_date).toLocaleDateString('ko-KR')}
      </p>
    </div>
  )
}
