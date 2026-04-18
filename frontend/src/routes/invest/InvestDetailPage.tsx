import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api, ApiError } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
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
  Lightbulb,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import { formatMoney, displayName } from '@/lib/utils'

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

export default function InvestDetailPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const [round, setRound] = useState<InvestmentRound | null>(null)
  const [loading, setLoading] = useState(true)
  const [shares, setShares] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [ownerActionLoading, setOwnerActionLoading] = useState<
    'close' | 'cancel' | null
  >(null)

  const fetchRound = async () => {
    setLoading(true)
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

  // Derived values ---------------------------------------------------------
  const companyName = round.company?.name ?? round.company_name ?? '회사'
  const currentValuation = round.company?.valuation ?? 0
  // Post-money valuation that this round will set when fully funded.
  const postMoneyValuation =
    round.offered_percent > 0
      ? Math.round(round.target_amount / round.offered_percent)
      : 0
  const preMoneyValuation = postMoneyValuation - round.target_amount

  const remainingShares =
    round.remaining_shares ??
    Math.max(0, round.new_shares - (round.sold_shares ?? 0))
  const progress =
    round.target_amount > 0
      ? Math.min((round.current_amount / round.target_amount) * 100, 100)
      : 0
  const isActive = round.status === 'open'

  const sharesNum = Number(shares) || 0
  const isLastBuy =
    sharesNum > 0 && sharesNum === remainingShares && remainingShares > 0
  // Approximate cost preview. Backend collapses rounding for the last buy
  // so it pays exactly target - current. We mimic that in the UI.
  const computedCost = isLastBuy
    ? round.target_amount - round.current_amount
    : Math.round(sharesNum * round.price_per_share)

  // "If I buy N shares, what % of the company will I own?"
  //    = N / (company.total_shares + N)    (assuming no other partial buys)
  // We don't know company.total_shares directly here, but we know:
  //   post-funded total_shares = company.total_shares + round.new_shares
  // So: pre-funded total_shares ≈ postTotal - new_shares
  //   new_shares * price = target, valuation = target / offered_pct
  //   post total = new_shares / offered_pct  (pure math)
  const postTotalShares =
    round.offered_percent > 0
      ? Math.round(round.new_shares / round.offered_percent)
      : 0
  const existingShares = Math.max(0, postTotalShares - round.new_shares)
  const myEventualOwnership =
    sharesNum > 0 && existingShares + sharesNum > 0
      ? (sharesNum / (existingShares + sharesNum)) * 100
      : 0

  const isOwner = !!user && !!round.owner && user.id === round.owner.id
  const sharesSold = round.sold_shares ?? round.new_shares - remainingShares

  const handleCloseEarly = async () => {
    if (
      !window.confirm(
        `라운드를 조기 마감하시겠습니까?\n\n` +
          `지금까지 유치한 ${formatMoney(round.current_amount)}으로 확정됩니다. ` +
          `남은 주식(${remainingShares.toLocaleString('ko-KR')}주)은 발행되지 않습니다.\n\n` +
          `회사 가치는 주당 가격 기준으로 재평가돼요.`,
      )
    ) {
      return
    }
    setOwnerActionLoading('close')
    try {
      await api.post(`/investment/rounds/${id}/close`, {})
      toast.success('라운드를 조기 마감했습니다.')
      await fetchRound()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '조기 마감 실패')
    } finally {
      setOwnerActionLoading(null)
    }
  }

  const handleCancelRound = async () => {
    if (
      !window.confirm(
        `정말 라운드를 취소하시겠습니까?\n\n` +
          `투자자 전원에게 ${formatMoney(round.current_amount)}이 환불되고 ` +
          `발행된 주식 ${sharesSold.toLocaleString('ko-KR')}주가 모두 회수됩니다.\n\n` +
          `되돌릴 수 없습니다.`,
      )
    ) {
      return
    }
    setOwnerActionLoading('cancel')
    try {
      await api.post(`/investment/rounds/${id}/cancel`, {})
      toast.success('라운드를 취소했습니다. 환불이 완료됐어요.')
      await fetchRound()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '라운드 취소 실패')
    } finally {
      setOwnerActionLoading(null)
    }
  }

  const handleInvest = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (!sharesNum || sharesNum <= 0) {
      setError('1주 이상 입력해 주세요.')
      return
    }
    if (sharesNum > remainingShares) {
      setError(`남은 주식(${remainingShares}주)을 초과할 수 없습니다.`)
      return
    }
    setSubmitting(true)
    try {
      await api.post(`/investment/rounds/${id}/invest`, { shares: sharesNum })
      toast.success(
        `${sharesNum}주 매수 완료! (${formatMoney(computedCost)})`,
      )
      setShares('')
      await fetchRound()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '투자에 실패했습니다.')
    } finally {
      setSubmitting(false)
    }
  }

  const pctLabel = (round.offered_percent * 100).toFixed(1)

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      <div className="sticky top-14 z-40 -mx-4 bg-background px-4 py-1">
        <Button variant="ghost" size="sm" asChild>
          <Link to="/invest">
            <ArrowLeft className="mr-1 h-4 w-4" />
            투자 목록으로
          </Link>
        </Button>
      </div>

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
                <CardTitle className="text-lg">{companyName}</CardTitle>
                <Badge variant={statusVariant[round.status] || 'secondary'}>
                  {statusLabels[round.status] || round.status}
                </Badge>
              </div>
              {round.owner && (
                <p className="text-sm text-muted-foreground">
                  대표: {displayName(round.owner)}
                </p>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <p className="text-muted-foreground">목표 금액</p>
              <p className="font-semibold">{formatMoney(round.target_amount)}</p>
            </div>
            <div>
              <p className="text-muted-foreground">제공 지분</p>
              <p className="font-semibold">{pctLabel}%</p>
            </div>
            <div>
              <p className="text-muted-foreground">주당 가격</p>
              <p className="font-semibold">
                {formatMoney(Math.round(round.price_per_share))}
              </p>
            </div>
            <div>
              <p className="text-muted-foreground">발행 주식</p>
              <p className="font-semibold">
                {round.new_shares.toLocaleString('ko-KR')}주
              </p>
            </div>
          </div>

          <Separator />

          {/* Valuation breakdown */}
          <HelpBox title="가치평가 계산 보기" defaultOpen>
            <div className="grid grid-cols-2 gap-2 rounded-md bg-background/60 p-2 text-xs">
              <div>
                <p className="text-muted-foreground">현재 기업가치</p>
                <p className="font-semibold text-foreground">
                  {formatMoney(currentValuation)}
                </p>
              </div>
              <div>
                <p className="text-muted-foreground">프리머니 가치</p>
                <p className="font-semibold text-foreground">
                  {formatMoney(preMoneyValuation)}
                </p>
              </div>
              <div>
                <p className="text-muted-foreground">모집 금액(+)</p>
                <p className="font-semibold text-foreground">
                  +{formatMoney(round.target_amount)}
                </p>
              </div>
              <div>
                <p className="text-muted-foreground">포스트머니 가치</p>
                <p className="font-semibold text-primary">
                  {formatMoney(postMoneyValuation)}
                </p>
              </div>
            </div>
            <p>
              <strong>포스트머니</strong> = 목표금액 ÷ 제공 지분 ={' '}
              {formatMoney(round.target_amount)} ÷ {pctLabel}% ={' '}
              {formatMoney(postMoneyValuation)}
            </p>
            <p>
              <strong>프리머니</strong> = 포스트머니 − 모집 금액. "투자 들어오기
              전"의 회사 가치예요.
            </p>
            <p>
              주당 가격 = 목표금액 ÷ 발행 주식 ={' '}
              {formatMoney(round.target_amount)} ÷{' '}
              {round.new_shares.toLocaleString('ko-KR')}주 ≈{' '}
              {formatMoney(Math.round(round.price_per_share))}
            </p>
          </HelpBox>

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
            <p className="mt-2 text-xs text-muted-foreground">
              남은 주식: {remainingShares.toLocaleString('ko-KR')}주 /{' '}
              {round.new_shares.toLocaleString('ko-KR')}주
            </p>
          </div>

          {round.expires_at && (
            <div className="flex items-center gap-1 text-sm text-muted-foreground">
              <Clock className="h-4 w-4" />
              마감: {new Date(round.expires_at).toLocaleDateString('ko-KR')}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Owner-only controls */}
      {isActive && isOwner && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">대표자 도구</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <HelpBox title="조기 마감 vs 취소 — 언제 뭘 써야 할까?">
              <p>
                <strong>조기 마감</strong>: 투자자가 일부만 참여했지만 그 금액을{' '}
                <em>받아들이고</em> 라운드를 확정하고 싶을 때. 투자자들은 지분을
                유지하고 회사 가치는 주당 가격에 맞춰 재평가됩니다.
              </p>
              <p>
                <strong>취소</strong>: 라운드 자체를 <em>없던 일</em>로 되돌리고
                싶을 때. 모든 투자자에게 환불되고 지분이 회수됩니다. 회사 지갑
                잔액이 부족하면 취소할 수 없어요.
              </p>
              <p className="text-muted-foreground">
                둘 다 되돌릴 수 없으니 신중하게 선택하세요.
              </p>
            </HelpBox>
            <div className="grid grid-cols-2 gap-2">
              <Button
                variant="outline"
                disabled={
                  ownerActionLoading !== null || sharesSold <= 0
                }
                onClick={handleCloseEarly}
                title={
                  sharesSold <= 0
                    ? '투자자가 1명 이상 있어야 조기 마감 가능합니다'
                    : undefined
                }
              >
                {ownerActionLoading === 'close' ? (
                  <Loader2 className="mr-1 h-4 w-4 animate-spin" />
                ) : null}
                조기 마감
              </Button>
              <Button
                variant="destructive"
                disabled={ownerActionLoading !== null}
                onClick={handleCancelRound}
              >
                {ownerActionLoading === 'cancel' ? (
                  <Loader2 className="mr-1 h-4 w-4 animate-spin" />
                ) : null}
                라운드 취소 (환불)
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Invest form */}
      {isActive && !isOwner && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">투자하기</CardTitle>
          </CardHeader>
          <form onSubmit={handleInvest}>
            <CardContent className="space-y-4">
              <HelpBox title="투자 전에 알아두세요">
                <p>
                  투자는 <strong>원금 손실 가능한</strong> 활동입니다. 회사가
                  잘 되면 주가(기업가치)가 오르고, 배당금도 받습니다. 반대면
                  손실이 납니다.
                </p>
                <p>
                  한 라운드는 여러 명이 나눠서 살 수 있어요. 내가 사고 싶은
                  만큼 주식 수를 입력하고, 마지막 한 주까지 다 팔리면 라운드가
                  마감됩니다.
                </p>
                <p>
                  매수 버튼을 누르는 순간 입력한 금액이 바로 내 지갑에서 빠지고
                  회사 법인 계좌로 들어갑니다.
                </p>
              </HelpBox>

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
                <p className="text-xs text-muted-foreground">
                  잔여 주식: {remainingShares.toLocaleString('ko-KR')}주
                </p>
              </div>

              {sharesNum > 0 && (
                <div className="space-y-2 rounded-md border bg-muted/30 p-3 text-xs">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">매수 금액</span>
                    <span className="font-semibold">
                      {formatMoney(computedCost)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">취득 지분(예상)</span>
                    <span className="font-semibold">
                      {myEventualOwnership.toFixed(2)}%
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">투자 후 총 주식 수</span>
                    <span className="font-semibold">
                      {(existingShares + sharesNum).toLocaleString('ko-KR')}주
                    </span>
                  </div>
                  {isLastBuy && (
                    <p className="rounded bg-primary/10 p-2 text-primary">
                      🎯 마지막 주식까지 매수하는 거예요. 라운드가 바로
                      마감되고, 회사 가치가{' '}
                      <strong>{formatMoney(postMoneyValuation)}</strong>으로
                      재평가됩니다.
                    </p>
                  )}
                </div>
              )}

              <Separator />
              <Button
                type="submit"
                className="w-full"
                disabled={submitting || remainingShares <= 0}
              >
                {submitting ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    처리 중...
                  </>
                ) : (
                  <>
                    <TrendingUp className="mr-2 h-4 w-4" />
                    매수하기
                  </>
                )}
              </Button>
            </CardContent>
          </form>
        </Card>
      )}
    </div>
  )
}
