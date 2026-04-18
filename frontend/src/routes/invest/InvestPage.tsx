import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/lib/api'
import type { InvestmentRound, Investment } from '@/types'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  TrendingUp,
  TrendingDown,
  Coins,
  Building2,
  Clock,
  Lightbulb,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import { formatMoney } from '@/lib/utils'

// DividendPayment matches backend investment.DividendPayment JSON shape.
interface DividendPayment {
  id: number
  dividend_id: number
  user_id: number
  shares: number
  amount: number
  created_at: string
  user_name?: string
  company_name?: string
}

// A collapsible education block. Explains one concept in student-friendly
// language. Default closed so the page stays tidy.
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

export default function InvestPage() {
  const [rounds, setRounds] = useState<InvestmentRound[]>([])
  const [portfolio, setPortfolio] = useState<Investment[]>([])
  const [dividends, setDividends] = useState<DividendPayment[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      // Bug A fix: backend status enum is open/funded/failed/cancelled.
      api
        .get<{ rounds: InvestmentRound[]; total: number } | InvestmentRound[]>(
          '/investment/rounds?status=open',
        )
        .catch(() => [] as InvestmentRound[]),
      api
        .get<Investment[]>('/investment/portfolio')
        .catch(() => [] as Investment[]),
      api
        .get<DividendPayment[]>('/investment/dividends')
        .catch(() => [] as DividendPayment[]),
    ]).then(([r, p, d]) => {
      const roundsArr = Array.isArray(r) ? r : (r?.rounds ?? [])
      const portfolioArr = Array.isArray(p) ? p : []
      const dividendsArr = Array.isArray(d) ? d : []
      setRounds(roundsArr)
      setPortfolio(portfolioArr)
      setDividends(dividendsArr)
      setLoading(false)
    })
  }, [])

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  const totalInvested = portfolio.reduce((s, i) => s + i.invested_amount, 0)
  const totalValue = portfolio.reduce((s, i) => s + i.current_value, 0)
  const totalProfit = totalValue - totalInvested
  const totalDividends = portfolio.reduce(
    (s, i) => s + i.dividends_received,
    0,
  )

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      <h1 className="text-lg font-bold">투자</h1>

      {/* Summary Cards */}
      {portfolio.length > 0 && (
        <div className="grid grid-cols-2 gap-2">
          <Card>
            <CardContent className="p-3">
              <p className="text-xs text-muted-foreground">총 투자금</p>
              <p className="text-sm font-bold">{formatMoney(totalInvested)}</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-3">
              <p className="text-xs text-muted-foreground">현재 가치</p>
              <p className="text-sm font-bold">{formatMoney(totalValue)}</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-3">
              <p className="text-xs text-muted-foreground">수익/손실</p>
              <p
                className={`text-sm font-bold ${totalProfit >= 0 ? 'text-success' : 'text-coral'}`}
              >
                {totalProfit >= 0 ? '+' : ''}
                {formatMoney(totalProfit)}
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-3">
              <p className="text-xs text-muted-foreground">받은 배당금</p>
              <p className="text-sm font-bold text-info">
                {formatMoney(totalDividends)}
              </p>
            </CardContent>
          </Card>
        </div>
      )}

      <Tabs defaultValue="rounds">
        <TabsList className="w-full">
          <TabsTrigger value="rounds" className="flex-1">
            투자 라운드
          </TabsTrigger>
          <TabsTrigger value="portfolio" className="flex-1">
            내 포트폴리오
          </TabsTrigger>
          <TabsTrigger value="dividends" className="flex-1">
            배당금
          </TabsTrigger>
        </TabsList>

        {/* --- Rounds tab --- */}
        <TabsContent value="rounds" className="mt-4 space-y-3">
          <HelpBox title="투자 라운드란?" defaultOpen={rounds.length > 0}>
            <p>
              <strong>투자 라운드</strong>는 회사가 자금을 모으기 위해 새
              주식을 발행해 파는 이벤트입니다. 회사는 "얼마를 모을지(목표
              금액)"와 "얼마만큼의 지분을 넘길지(제안 지분)"를 정하고, 이를
              보고 투자자들이 참여합니다.
            </p>
            <p>
              예) 목표 100만원 · 제안 지분 20%이면, 투자자들 전체가 100만원을
              내면 회사 지분 20%를 나눠 갖습니다. 지금 회사의 가치는{' '}
              <strong>100만원 ÷ 20% = 500만원</strong>으로 평가됩니다
              (포스트머니 가치).
            </p>
            <p>
              라운드는 여러 명이 나눠서 투자할 수 있어요. 마지막 한 주까지
              팔리면 라운드가 마감됩니다.
            </p>
          </HelpBox>

          {rounds.length === 0 ? (
            <p className="py-8 text-center text-muted-foreground">
              현재 진행 중인 투자 라운드가 없습니다.
            </p>
          ) : (
            rounds.map((round) => {
              const progress =
                round.target_amount > 0
                  ? Math.min(
                      (round.current_amount / round.target_amount) * 100,
                      100,
                    )
                  : 0
              const pctLabel = (round.offered_percent * 100).toFixed(1)
              return (
                <Link key={round.id} to={`/invest/${round.id}`}>
                  <Card className="transition-colors hover:bg-accent/30">
                    <CardContent className="p-4">
                      <div className="flex items-center gap-3">
                        <Avatar className="h-10 w-10">
                          <AvatarImage src={round.company?.logo_url} />
                          <AvatarFallback className="bg-primary/10 text-primary">
                            <Building2 className="h-5 w-5" />
                          </AvatarFallback>
                        </Avatar>
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center justify-between">
                            <h3 className="font-medium">
                              {round.company?.name ?? round.company_name}
                            </h3>
                            <Badge variant="default" className="text-xs">
                              모집 중
                            </Badge>
                          </div>
                          <p className="mt-0.5 text-xs text-muted-foreground">
                            지분 {pctLabel}% · 주당{' '}
                            {formatMoney(Math.round(round.price_per_share))}
                          </p>
                        </div>
                      </div>

                      {/* Progress bar */}
                      <div className="mt-3">
                        <div className="flex justify-between text-xs text-muted-foreground">
                          <span>{formatMoney(round.current_amount)}</span>
                          <span>{formatMoney(round.target_amount)}</span>
                        </div>
                        <div className="mt-1 h-2 w-full overflow-hidden rounded-full bg-secondary">
                          <div
                            className="h-full rounded-full bg-primary transition-all"
                            style={{ width: `${progress}%` }}
                          />
                        </div>
                        <p className="mt-1 text-right text-xs text-muted-foreground">
                          {progress.toFixed(0)}% 달성
                        </p>
                      </div>

                      {round.expires_at && (
                        <div className="mt-2 flex items-center gap-1 text-xs text-muted-foreground">
                          <Clock className="h-3 w-3" />
                          마감{' '}
                          {new Date(round.expires_at).toLocaleDateString(
                            'ko-KR',
                          )}
                        </div>
                      )}
                    </CardContent>
                  </Card>
                </Link>
              )
            })
          )}
        </TabsContent>

        {/* --- Portfolio tab --- */}
        <TabsContent value="portfolio" className="mt-4 space-y-3">
          <HelpBox title="포트폴리오 읽는 법">
            <p>
              <strong>보유 주식</strong>은 라운드에서 사들인 주식 수,{' '}
              <strong>지분율</strong>은 전체 주식 중 내 몫의 비율입니다.
            </p>
            <p>
              <strong>현재 가치</strong> = 회사 가치 × 내 지분율. 회사 가치는
              마지막 투자 라운드의 포스트머니 가치로 평가합니다.
            </p>
            <p>
              <strong>수익/손실</strong> = 현재 가치 − 투자금. 회사 가치가
              아직 바뀌지 않았으면 0원이 자연스럽습니다.
            </p>
          </HelpBox>

          {portfolio.length === 0 ? (
            <p className="py-8 text-center text-muted-foreground">
              보유한 투자가 없습니다.
            </p>
          ) : (
            portfolio.map((inv) => {
              const profitPercent =
                inv.invested_amount > 0
                  ? ((inv.profit / inv.invested_amount) * 100).toFixed(1)
                  : '0.0'
              return (
                <Card key={inv.company.id}>
                  <CardContent className="p-4">
                    <div className="flex items-center justify-between">
                      <h3 className="font-medium">{inv.company.name}</h3>
                      <div className="flex items-center gap-1">
                        {inv.profit >= 0 ? (
                          <TrendingUp className="h-4 w-4 text-success" />
                        ) : (
                          <TrendingDown className="h-4 w-4 text-coral" />
                        )}
                        <span
                          className={`text-sm font-bold ${inv.profit >= 0 ? 'text-success' : 'text-coral'}`}
                        >
                          {inv.profit >= 0 ? '+' : ''}
                          {profitPercent}%
                        </span>
                      </div>
                    </div>
                    <div className="mt-2 grid grid-cols-2 gap-2 text-sm">
                      <div>
                        <p className="text-xs text-muted-foreground">
                          보유 주식
                        </p>
                        <p className="font-medium">
                          {inv.shares}주 ({inv.percentage.toFixed(1)}%)
                        </p>
                      </div>
                      <div>
                        <p className="text-xs text-muted-foreground">투자금</p>
                        <p className="font-medium">
                          {formatMoney(inv.invested_amount)}
                        </p>
                      </div>
                      <div>
                        <p className="text-xs text-muted-foreground">
                          현재 가치
                        </p>
                        <p className="font-medium">
                          {formatMoney(inv.current_value)}
                        </p>
                      </div>
                      <div>
                        <p className="text-xs text-muted-foreground">
                          수익/손실
                        </p>
                        <p
                          className={`font-medium ${inv.profit >= 0 ? 'text-success' : 'text-coral'}`}
                        >
                          {inv.profit >= 0 ? '+' : ''}
                          {formatMoney(inv.profit)}
                        </p>
                      </div>
                    </div>
                    {inv.dividends_received > 0 && (
                      <div className="mt-2 flex items-center gap-1 text-xs text-info">
                        <Coins className="h-3 w-3" />
                        배당금 수령: {formatMoney(inv.dividends_received)}
                      </div>
                    )}
                  </CardContent>
                </Card>
              )
            })
          )}
        </TabsContent>

        {/* --- Dividends tab --- */}
        <TabsContent value="dividends" className="mt-4 space-y-3">
          <HelpBox title="배당금이란?">
            <p>
              <strong>배당금(dividend)</strong>은 회사가 번 돈의 일부를 주주에게
              나눠주는 것입니다. 대표가 "총 얼마를 배당할지" 정하면, 각 주주는
              본인의 지분율만큼 받아갑니다.
            </p>
            <p>
              예) 회사가 100만원을 배당하고 내 지분이 20%라면 내 계좌에 20만원이
              자동 입금됩니다.
            </p>
          </HelpBox>

          {dividends.length === 0 ? (
            <p className="py-8 text-center text-muted-foreground">
              받은 배당금이 없습니다.
            </p>
          ) : (
            dividends.map((d) => (
              <Card key={d.id}>
                <CardContent className="flex items-center justify-between p-4">
                  <div>
                    <p className="text-sm font-medium">
                      {d.company_name ?? '-'}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {d.shares.toLocaleString('ko-KR')}주 기준 배당
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {new Date(d.created_at).toLocaleDateString('ko-KR')}
                    </p>
                  </div>
                  <div className="text-right">
                    <p className="font-bold text-info">
                      {formatMoney(d.amount)}
                    </p>
                  </div>
                </CardContent>
              </Card>
            ))
          )}
        </TabsContent>
      </Tabs>
    </div>
  )
}
