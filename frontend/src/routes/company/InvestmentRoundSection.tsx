import { useState, useEffect, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { api, ApiError } from '@/lib/api'
import type { InvestmentRound } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { toast } from 'sonner'
import {
  TrendingUp,
  Plus,
  Loader2,
  Lightbulb,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import { formatMoney } from '@/lib/utils'

const statusLabels: Record<string, string> = {
  open: '모집 중',
  funded: '모집 완료',
  failed: '실패',
  cancelled: '취소',
}

const statusVariant: Record<
  string,
  'default' | 'secondary' | 'destructive' | 'outline'
> = {
  open: 'default',
  funded: 'secondary',
  failed: 'destructive',
  cancelled: 'outline',
}

function HelpBox({
  title,
  children,
  defaultOpen = false,
}: {
  title: string
  children: React.ReactNode
  defaultOpen?: boolean
}) {
  const [open, setOpen] = useState(defaultOpen)
  return (
    <Card className="border-primary/30 bg-primary/5">
      <CardContent className="p-3">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="flex w-full items-center gap-2 text-sm font-medium text-primary"
        >
          <Lightbulb className="h-4 w-4 shrink-0" />
          <span className="flex-1 text-left">{title}</span>
          {open ? (
            <ChevronUp className="h-4 w-4" />
          ) : (
            <ChevronDown className="h-4 w-4" />
          )}
        </button>
        {open && (
          <div className="mt-2 space-y-2 text-xs leading-relaxed text-muted-foreground">
            {children}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

interface Props {
  companyId: number
  companyValuation: number
  isOwner: boolean
  onRoundCreated?: () => void
}

export function InvestmentRoundSection({
  companyId,
  companyValuation,
  isOwner,
  onRoundCreated,
}: Props) {
  const [rounds, setRounds] = useState<InvestmentRound[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)

  const fetchRounds = useCallback(async () => {
    try {
      const resp = await api.get<
        { rounds: InvestmentRound[]; total: number } | InvestmentRound[]
      >(`/investment/rounds?company_id=${companyId}`)
      const arr = Array.isArray(resp) ? resp : (resp?.rounds ?? [])
      setRounds(arr)
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [companyId])

  useEffect(() => {
    fetchRounds()
  }, [fetchRounds])

  const hasOpenRound = rounds.some((r) => r.status === 'open')

  if (loading) {
    return (
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center gap-2 text-base">
            <TrendingUp className="h-4 w-4" />
            투자 유치
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
          <TrendingUp className="h-4 w-4" />
          투자 유치
        </CardTitle>
        {isOwner && !hasOpenRound && (
          <Button
            size="sm"
            variant="outline"
            className="h-7 gap-1 text-xs"
            onClick={() => setCreateOpen(true)}
          >
            <Plus className="h-3 w-3" /> 라운드 개설
          </Button>
        )}
      </CardHeader>
      <CardContent className="space-y-3">
        {rounds.length === 0 ? (
          <p className="py-4 text-center text-sm text-muted-foreground">
            아직 개설된 투자 라운드가 없어요.
          </p>
        ) : (
          rounds.map((r) => {
            const pctLabel = (r.offered_percent * 100).toFixed(1)
            const progress =
              r.target_amount > 0
                ? Math.min((r.current_amount / r.target_amount) * 100, 100)
                : 0
            return (
              <Link
                key={r.id}
                to={`/invest/${r.id}`}
                className="block rounded-md border p-3 transition-colors hover:bg-accent/30"
              >
                <div className="flex items-center justify-between">
                  <Badge variant={statusVariant[r.status] ?? 'secondary'}>
                    {statusLabels[r.status] ?? r.status}
                  </Badge>
                  <span className="text-xs text-muted-foreground">
                    지분 {pctLabel}% · 주당{' '}
                    {formatMoney(Math.round(r.price_per_share))}
                  </span>
                </div>
                <p className="mt-1 text-sm font-medium">
                  {formatMoney(r.current_amount)} /{' '}
                  {formatMoney(r.target_amount)}
                </p>
                <div className="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-secondary">
                  <div
                    className="h-full bg-primary"
                    style={{ width: `${progress}%` }}
                  />
                </div>
              </Link>
            )
          })
        )}
      </CardContent>

      {createOpen && (
        <CreateRoundDialog
          companyId={companyId}
          companyValuation={companyValuation}
          onClose={() => setCreateOpen(false)}
          onCreated={() => {
            setCreateOpen(false)
            fetchRounds()
            onRoundCreated?.()
          }}
        />
      )}
    </Card>
  )
}

function CreateRoundDialog({
  companyId,
  companyValuation,
  onClose,
  onCreated,
}: {
  companyId: number
  companyValuation: number
  onClose: () => void
  onCreated: () => void
}) {
  const [targetAmount, setTargetAmount] = useState('')
  const [offeredPercent, setOfferedPercent] = useState('10')
  const [loading, setLoading] = useState(false)

  const targetNum = Number(targetAmount) || 0
  const pctNum = Number(offeredPercent) || 0
  const pctFraction = pctNum / 100

  // Live valuation preview
  const postMoney =
    pctFraction > 0 && pctFraction < 1
      ? Math.round(targetNum / pctFraction)
      : 0
  const preMoney = postMoney > 0 ? postMoney - targetNum : 0
  const valid =
    targetNum > 0 && pctNum > 0 && pctNum < 100 && pctFraction < 0.99

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!valid) return
    setLoading(true)
    try {
      await api.post('/investment/rounds', {
        company_id: companyId,
        target_amount: targetNum,
        offered_percent: pctFraction,
      })
      toast.success('투자 라운드가 개설되었습니다.')
      onCreated()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '라운드 개설 실패')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>투자 라운드 개설</DialogTitle>
          <DialogDescription>
            회사 지분 일부를 투자자에게 팔아 자본을 모읍니다. 모집이 완료되면
            라운드가 자동 마감되고 회사의 법인 계좌에 투자금이 입금됩니다.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-3">
          <HelpBox title="투자 라운드 기초 지식" defaultOpen>
            <p>
              <strong>목표 금액</strong>: 이번 라운드로 모을 총 자금. 이
              금액이 전부 모이면 라운드가 마감됩니다.
            </p>
            <p>
              <strong>제공 지분</strong>: 회사의 몇 %를 투자자들에게 넘길지
              정하는 값(1~99%). 이 지분만큼의 새 주식이 발행되어 기존 주주의
              지분율이 <strong>희석</strong>됩니다.
            </p>
            <p>
              <strong>가치평가</strong>: 모집 금액 ÷ 제공 지분 = 투자 후
              회사 가치(포스트머니). 예) 100만원을 10%에 팔면 회사 가치는
              1000만원으로 평가받습니다.
            </p>
            <p>
              <strong>주당 가격</strong>은 목표 금액을 새로 발행되는 주식 수로
              나눠 자동 계산됩니다. 투자자들이 이 가격에 원하는 만큼 주식을
              살 수 있어요 (여러 명이 나눠서 살 수 있음).
            </p>
            <p>
              ⚠️ 동시에 여러 라운드는 못 열어요. 지금 라운드가 끝나야 다음
              라운드를 열 수 있습니다.
            </p>
          </HelpBox>

          <div className="space-y-1">
            <Label htmlFor="target">목표 금액 (원)</Label>
            <Input
              id="target"
              type="number"
              min={10000}
              step={10000}
              value={targetAmount}
              onChange={(e) => setTargetAmount(e.target.value)}
              placeholder="예: 1000000"
              required
            />
          </div>

          <div className="space-y-1">
            <Label htmlFor="pct">제공 지분 (%)</Label>
            <Input
              id="pct"
              type="number"
              min={1}
              max={99}
              step={0.1}
              value={offeredPercent}
              onChange={(e) => setOfferedPercent(e.target.value)}
              required
            />
            <p className="text-xs text-muted-foreground">
              1 ~ 99 사이의 값. 너무 높으면 경영권이 흔들릴 수 있어요.
            </p>
          </div>

          {/* Live preview */}
          {targetNum > 0 && pctNum > 0 && pctNum < 100 && (
            <div className="space-y-2 rounded-md border bg-muted/30 p-3 text-xs">
              <p className="font-medium">가치평가 미리보기</p>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <p className="text-muted-foreground">현재 기업가치</p>
                  <p className="font-semibold">
                    {formatMoney(companyValuation)}
                  </p>
                </div>
                <div>
                  <p className="text-muted-foreground">프리머니 가치</p>
                  <p className="font-semibold">{formatMoney(preMoney)}</p>
                </div>
                <div>
                  <p className="text-muted-foreground">모집 금액(+)</p>
                  <p className="font-semibold">
                    +{formatMoney(targetNum)}
                  </p>
                </div>
                <div>
                  <p className="text-muted-foreground">포스트머니 가치</p>
                  <p className="font-semibold text-primary">
                    {formatMoney(postMoney)}
                  </p>
                </div>
              </div>
              {preMoney < companyValuation && companyValuation > 0 && (
                <p className="rounded bg-warning/15 p-2 text-warning">
                  ⚠️ 프리머니({formatMoney(preMoney)})가 현재 기업가치(
                  {formatMoney(companyValuation)})보다 낮습니다. "다운라운드"
                  (가치 하락)로 기존 주주에게 불리할 수 있어요.
                </p>
              )}
            </div>
          )}

          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="ghost" onClick={onClose}>
              취소
            </Button>
            <Button type="submit" disabled={!valid || loading}>
              {loading && <Loader2 className="mr-1 h-4 w-4 animate-spin" />}
              개설
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
