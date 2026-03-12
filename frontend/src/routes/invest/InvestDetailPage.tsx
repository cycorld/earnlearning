import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '@/lib/api'
import type { InvestmentRound } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { toast } from 'sonner'
import {
  ArrowLeft,
  Building2,
  Clock,
  Loader2,
  TrendingUp,
} from 'lucide-react'

function formatMoney(amount: number): string {
  return new Intl.NumberFormat('ko-KR').format(amount) + '원'
}

const statusLabels: Record<string, string> = {
  active: '모집 중',
  open: '모집 중',
  closed: '마감',
  completed: '완료',
}

const statusVariant: Record<
  string,
  'default' | 'secondary' | 'destructive' | 'outline'
> = {
  active: 'default',
  open: 'default',
  closed: 'secondary',
  completed: 'outline',
}

export default function InvestDetailPage() {
  const { id } = useParams()
  const [round, setRound] = useState<InvestmentRound | null>(null)
  const [loading, setLoading] = useState(true)
  const [shares, setShares] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const fetchRound = async () => {
    try {
      const data = await api.get<InvestmentRound>(
        `/investment/rounds/${id}`,
      )
      setRound(data)
    } catch {
      setRound(null)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchRound()
  }, [id])

  const computedCost =
    shares && round ? Number(shares) * round.price_per_share : 0
  const remainingShares = round
    ? round.new_shares -
      (round.target_amount > 0
        ? Math.floor(round.current_amount / round.price_per_share)
        : 0)
    : 0

  const handleInvest = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (!shares || Number(shares) <= 0) {
      setError('1주 이상 입력해 주세요.')
      return
    }
    setSubmitting(true)
    try {
      await api.post(`/investment/rounds/${id}/invest`, {
        shares: Number(shares),
      })
      toast.success(
        `${shares}주 투자 완료! (${formatMoney(computedCost)})`,
      )
      setShares('')
      await fetchRound()
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : '투자에 실패했습니다.',
      )
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  if (!round) {
    return (
      <div className="p-4 text-center text-muted-foreground">
        투자 라운드를 찾을 수 없습니다.
      </div>
    )
  }

  const progress =
    round.target_amount > 0
      ? Math.min(
          (round.current_amount / round.target_amount) * 100,
          100,
        )
      : 0
  const isActive = round.status === 'active' || round.status === 'open'

  return (
    <div className="mx-auto max-w-lg space-y-4 p-4">
      <Button variant="ghost" size="sm" asChild>
        <Link to="/invest">
          <ArrowLeft className="mr-1 h-4 w-4" />
          투자 목록으로
        </Link>
      </Button>

      {/* Round Detail */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <Avatar className="h-12 w-12">
              <AvatarImage src={round.company?.logo_url} />
              <AvatarFallback className="bg-primary/10 text-primary">
                <Building2 className="h-6 w-6" />
              </AvatarFallback>
            </Avatar>
            <div className="flex-1">
              <div className="flex items-center justify-between">
                <CardTitle className="text-lg">
                  {round.company?.name}
                </CardTitle>
                <Badge
                  variant={statusVariant[round.status] || 'secondary'}
                >
                  {statusLabels[round.status] || round.status}
                </Badge>
              </div>
              {round.owner && (
                <p className="text-sm text-muted-foreground">
                  대표: {round.owner.name}
                </p>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <p className="text-muted-foreground">목표 금액</p>
              <p className="font-semibold">
                {formatMoney(round.target_amount)}
              </p>
            </div>
            <div>
              <p className="text-muted-foreground">제공 지분</p>
              <p className="font-semibold">{round.offered_percent}%</p>
            </div>
            <div>
              <p className="text-muted-foreground">주당 가격</p>
              <p className="font-semibold">
                {formatMoney(round.price_per_share)}
              </p>
            </div>
            <div>
              <p className="text-muted-foreground">발행 주식</p>
              <p className="font-semibold">{round.new_shares}주</p>
            </div>
          </div>

          {round.company?.valuation != null && (
            <>
              <Separator />
              <div className="flex items-center gap-2 text-sm">
                <TrendingUp className="h-4 w-4 text-primary" />
                <span className="text-muted-foreground">기업가치</span>
                <span className="font-semibold">
                  {formatMoney(round.company.valuation)}
                </span>
              </div>
            </>
          )}

          <Separator />

          {/* Progress */}
          <div>
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">모집 현황</span>
              <span className="font-medium">{progress.toFixed(0)}%</span>
            </div>
            <div className="mt-2 flex justify-between text-xs text-muted-foreground">
              <span>{formatMoney(round.current_amount)}</span>
              <span>{formatMoney(round.target_amount)}</span>
            </div>
            <div className="mt-1 h-3 w-full overflow-hidden rounded-full bg-secondary">
              <div
                className="h-full rounded-full bg-primary transition-all"
                style={{ width: `${progress}%` }}
              />
            </div>
          </div>

          {round.expires_at && (
            <div className="flex items-center gap-1 text-sm text-muted-foreground">
              <Clock className="h-4 w-4" />
              마감:{' '}
              {new Date(round.expires_at).toLocaleDateString('ko-KR')}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Invest Form */}
      {isActive && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">투자하기</CardTitle>
          </CardHeader>
          <form onSubmit={handleInvest}>
            <CardContent className="space-y-4">
              {error && (
                <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                  {error}
                </div>
              )}
              <div className="space-y-2">
                <Label htmlFor="shares">매수 주식 수</Label>
                <Input
                  id="shares"
                  type="number"
                  placeholder="매수할 주식 수를 입력하세요"
                  value={shares}
                  onChange={(e) => setShares(e.target.value)}
                  required
                  min={1}
                  max={remainingShares > 0 ? remainingShares : undefined}
                />
                {remainingShares > 0 && (
                  <p className="text-xs text-muted-foreground">
                    잔여 주식: {remainingShares}주
                  </p>
                )}
              </div>

              <Separator />

              <div className="rounded-lg bg-muted p-3">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">주당 가격</span>
                  <span>{formatMoney(round.price_per_share)}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">매수 수량</span>
                  <span>{shares || 0}주</span>
                </div>
                <Separator className="my-2" />
                <div className="flex justify-between font-bold">
                  <span>총 투자 금액</span>
                  <span className="text-primary">
                    {formatMoney(computedCost)}
                  </span>
                </div>
              </div>

              <Button
                type="submit"
                className="w-full"
                disabled={submitting || !shares || Number(shares) <= 0}
              >
                {submitting && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                투자하기
              </Button>
            </CardContent>
          </form>
        </Card>
      )}
    </div>
  )
}
