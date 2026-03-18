import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/lib/api'
import type { InvestmentRound, Investment } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  TrendingUp,
  TrendingDown,
  Coins,
  Building2,
  Clock,
} from 'lucide-react'
import { formatMoney } from '@/lib/utils'

interface Dividend {
  id: number
  company: { id: number; name: string }
  amount: number
  shares: number
  per_share: number
  created_at: string
}

export default function InvestPage() {
  const [rounds, setRounds] = useState<InvestmentRound[]>([])
  const [portfolio, setPortfolio] = useState<Investment[]>([])
  const [dividends, setDividends] = useState<Dividend[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      api
        .get<{ rounds: InvestmentRound[]; total: number } | InvestmentRound[]>('/investment/rounds?status=active')
        .catch(() => [] as InvestmentRound[]),
      api
        .get<Investment[]>('/investment/portfolio')
        .catch(() => [] as Investment[]),
      api
        .get<Dividend[]>('/investment/dividends')
        .catch(() => [] as Dividend[]),
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

  const totalInvested = portfolio.reduce((sum, inv) => sum + inv.invested_amount, 0)
  const totalValue = portfolio.reduce((sum, inv) => sum + inv.current_value, 0)
  const totalProfit = totalValue - totalInvested
  const totalDividends = portfolio.reduce((sum, inv) => sum + inv.dividends_received, 0)

  return (
    <div className="mx-auto max-w-lg space-y-4 p-4">
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
                className={`text-sm font-bold ${totalProfit >= 0 ? 'text-green-600' : 'text-red-600'}`}
              >
                {totalProfit >= 0 ? '+' : ''}
                {formatMoney(totalProfit)}
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-3">
              <p className="text-xs text-muted-foreground">받은 배당금</p>
              <p className="text-sm font-bold text-blue-600">
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

        {/* Active Investment Rounds */}
        <TabsContent value="rounds" className="mt-4 space-y-3">
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
                              {round.company?.name}
                            </h3>
                            <Badge variant="default" className="text-xs">
                              모집 중
                            </Badge>
                          </div>
                          <p className="mt-0.5 text-xs text-muted-foreground">
                            지분 {round.offered_percent}% | 주당{' '}
                            {formatMoney(round.price_per_share)}
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

        {/* My Portfolio */}
        <TabsContent value="portfolio" className="mt-4 space-y-3">
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
                          <TrendingUp className="h-4 w-4 text-green-600" />
                        ) : (
                          <TrendingDown className="h-4 w-4 text-red-600" />
                        )}
                        <span
                          className={`text-sm font-bold ${inv.profit >= 0 ? 'text-green-600' : 'text-red-600'}`}
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
                          {inv.shares}주 ({inv.percentage}%)
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
                          className={`font-medium ${inv.profit >= 0 ? 'text-green-600' : 'text-red-600'}`}
                        >
                          {inv.profit >= 0 ? '+' : ''}
                          {formatMoney(inv.profit)}
                        </p>
                      </div>
                    </div>
                    {inv.dividends_received > 0 && (
                      <div className="mt-2 flex items-center gap-1 text-xs text-blue-600">
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

        {/* Dividends */}
        <TabsContent value="dividends" className="mt-4 space-y-3">
          {dividends.length === 0 ? (
            <p className="py-8 text-center text-muted-foreground">
              받은 배당금이 없습니다.
            </p>
          ) : (
            dividends.map((div) => (
              <Card key={div.id}>
                <CardContent className="flex items-center justify-between p-4">
                  <div>
                    <p className="text-sm font-medium">{div.company.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {div.shares}주 x {formatMoney(div.per_share)}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {new Date(div.created_at).toLocaleDateString('ko-KR')}
                    </p>
                  </div>
                  <div className="text-right">
                    <p className="font-bold text-blue-600">
                      {formatMoney(div.amount)}
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
