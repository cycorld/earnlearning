import { useEffect, useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { toast } from 'sonner'
import {
  ArrowLeft,
  TrendingUp,
  TrendingDown,
  X,
  RefreshCw,
} from 'lucide-react'
import { formatMoney } from '@/lib/utils'
import { Spinner } from '@/components/ui/spinner'

interface OrderbookEntry {
  price: number
  shares: number
  count: number
}

interface Orderbook {
  asks: OrderbookEntry[]
  bids: OrderbookEntry[]
}

interface ExchangeOrder {
  id: number
  company_id: number
  order_type: 'buy' | 'sell'
  shares: number
  price_per_share: number
  status: string
  created_at: string
}

interface StockTrade {
  id: number
  price_per_share: number
  shares: number
  total_amount: number
  created_at: string
}

interface ListedCompany {
  id: number
  name: string
  logo_url: string
  last_price: number
  change_percent: number
  volume_24h: number
}

interface Position {
  shares: number // 보유 주식
  available_shares: number // 매도 가능 (보유 − 미체결 매도)
  balance: number // 지갑 잔액
  available_cash: number // 매수 가능 (잔액 − 미체결 매수)
}

// 체결 시각을 HH:MM (브라우저 로컬, KST)로 표시
function timeHM(iso: string): string {
  return new Date(iso).toLocaleTimeString('ko-KR', {
    hour: '2-digit',
    minute: '2-digit',
  })
}

// 체결가 추이 라인+영역 차트. 차트 라이브러리 없이 SVG로 직접 그린다.
// trades 는 최신순으로 들어오므로 시간순(과거→현재)으로 뒤집어 그린다.
function PriceChart({ trades }: { trades: StockTrade[] }) {
  const points = [...trades].reverse()
  if (points.length < 2) {
    return (
      <div className="flex h-36 items-center justify-center text-sm text-muted-foreground">
        차트를 그릴 체결 데이터가 아직 부족합니다
      </div>
    )
  }

  const W = 320
  const H = 140
  const PAD = 10
  const prices = points.map((p) => p.price_per_share)
  const min = Math.min(...prices)
  const max = Math.max(...prices)
  const range = max - min || 1
  const stepX = (W - PAD * 2) / (points.length - 1)

  const coords = points.map((p, i) => {
    const x = PAD + i * stepX
    const y = PAD + (H - PAD * 2) * (1 - (p.price_per_share - min) / range)
    return [x, y] as const
  })

  const line = coords
    .map(([x, y], i) => `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`)
    .join(' ')
  const area = `${line} L${coords[coords.length - 1][0].toFixed(1)},${H - PAD} L${coords[0][0].toFixed(1)},${H - PAD} Z`

  // 한국식 색상: 상승=빨강(coral), 하락=파랑(info)
  const up = prices[prices.length - 1] >= prices[0]
  const color = up ? 'var(--coral)' : 'var(--info)'

  return (
    <div className="space-y-1">
      <svg
        viewBox={`0 0 ${W} ${H}`}
        className="h-36 w-full"
        preserveAspectRatio="none"
      >
        <defs>
          <linearGradient id="priceFill" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity="0.25" />
            <stop offset="100%" stopColor={color} stopOpacity="0" />
          </linearGradient>
        </defs>
        <path d={area} fill="url(#priceFill)" />
        <path
          d={line}
          fill="none"
          stroke={color}
          strokeWidth="2"
          strokeLinejoin="round"
          strokeLinecap="round"
          vectorEffect="non-scaling-stroke"
        />
      </svg>
      <div className="flex justify-between px-1 text-[10px] text-muted-foreground">
        <span>{timeHM(points[0].created_at)}</span>
        <span>최저 {formatMoney(min)} · 최고 {formatMoney(max)}</span>
        <span>{timeHM(points[points.length - 1].created_at)}</span>
      </div>
    </div>
  )
}

export default function ExchangeDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const companyId = Number(id)

  const [company, setCompany] = useState<ListedCompany | null>(null)
  const [orderbook, setOrderbook] = useState<Orderbook | null>(null)
  const [trades, setTrades] = useState<StockTrade[]>([])
  const [myOrders, setMyOrders] = useState<ExchangeOrder[]>([])
  const [position, setPosition] = useState<Position | null>(null)
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)

  const [orderType, setOrderType] = useState<'buy' | 'sell'>('buy')
  const [shares, setShares] = useState('')
  const [price, setPrice] = useState('')

  const fetchData = useCallback(async () => {
    try {
      const [companies, ob, tr, orders, pos] = await Promise.all([
        api.get<ListedCompany[]>('/exchange/companies'),
        api.get<Orderbook>(`/exchange/orderbook/${companyId}`),
        api.get<StockTrade[]>(`/exchange/trades/${companyId}?limit=50`),
        api.get<{ orders: ExchangeOrder[] } | ExchangeOrder[]>(
          '/exchange/orders/mine',
        ),
        api.get<Position>(`/exchange/position/${companyId}`),
      ])
      setCompany(
        (Array.isArray(companies) ? companies : []).find(
          (c) => c.id === companyId,
        ) ?? null,
      )
      setOrderbook(ob)
      setTrades(tr || [])
      const ordersArr = Array.isArray(orders) ? orders : (orders?.orders ?? [])
      setMyOrders(ordersArr.filter((o) => o.company_id === companyId))
      setPosition(pos)
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [companyId])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  // 호가를 누르면 주문 폼에 가격을 자동으로 채운다.
  // 매도호가(asks) 클릭 → 매수, 매수호가(bids) 클릭 → 매도 로 전환.
  const fillFromOrderbook = (side: 'buy' | 'sell', p: number) => {
    setOrderType(side)
    setPrice(String(p))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!shares || !price) return

    // 프론트 제약 (백엔드 검증과 동일 기준 — 매수=여유자금, 매도=매도가능 주식)
    const s = Number(shares)
    const p = Number(price)
    if (position) {
      if (orderType === 'buy' && p > 0 && s * p > position.available_cash) {
        toast.error('여유자금을 초과했습니다.')
        return
      }
      if (orderType === 'sell' && s > position.available_shares) {
        toast.error('매도 가능 수량을 초과했습니다.')
        return
      }
    }

    setSubmitting(true)
    try {
      await api.post('/exchange/orders', {
        company_id: companyId,
        order_type: orderType,
        shares: Number(shares),
        price: Number(price),
      })
      toast.success('주문이 등록되었습니다.')
      setShares('')
      setPrice('')
      fetchData()
    } catch (err: any) {
      toast.error(err.message || '주문 등록에 실패했습니다.')
    } finally {
      setSubmitting(false)
    }
  }

  const handleCancel = async (orderId: number) => {
    try {
      await api.del(`/exchange/orders/${orderId}`)
      toast.success('주문이 취소되었습니다.')
      fetchData()
    } catch (err: any) {
      toast.error(err.message || '주문 취소에 실패했습니다.')
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Spinner />
      </div>
    )
  }

  const lastPrice = trades[0]?.price_per_share ?? company?.last_price ?? 0
  const changePercent = company?.change_percent ?? 0
  const changeUp = changePercent >= 0
  const total = Number(shares) * Number(price)

  // 주문 제약 (매수=여유자금 한도, 매도=매도가능 주식 한도)
  const sharesNum = Number(shares) || 0
  const priceNum = Number(price) || 0
  const maxBuy = priceNum > 0 ? Math.floor((position?.available_cash ?? 0) / priceNum) : 0
  const maxShares = orderType === 'buy' ? maxBuy : (position?.available_shares ?? 0)
  const exceeds =
    position != null &&
    (orderType === 'buy'
      ? priceNum > 0 && sharesNum * priceNum > position.available_cash
      : sharesNum > position.available_shares)

  return (
    <div className="mx-auto max-w-lg space-y-4 p-4">
      {/* 헤더: 회사 + 현재가 + 등락률 */}
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        {company?.logo_url ? (
          <img
            src={company.logo_url}
            alt={company.name}
            className="h-8 w-8 rounded-full object-cover"
          />
        ) : (
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10">
            <TrendingUp className="h-4 w-4 text-primary" />
          </div>
        )}
        <h1 className="truncate text-lg font-bold">
          {company?.name ?? `기업 #${companyId}`}
        </h1>
        <div className="flex-1" />
        <Button variant="ghost" size="icon" onClick={fetchData}>
          <RefreshCw className="h-4 w-4" />
        </Button>
      </div>

      <Card>
        <CardContent className="space-y-3 p-4">
          <div className="flex items-end justify-between">
            <div>
              <p className="text-2xl font-bold">{formatMoney(lastPrice)}</p>
              <p className="text-xs text-muted-foreground">현재가 / 주</p>
            </div>
            <div
              className={`flex items-center gap-1 text-sm font-semibold ${
                changeUp ? 'text-coral' : 'text-info'
              }`}
            >
              {changeUp ? (
                <TrendingUp className="h-4 w-4" />
              ) : (
                <TrendingDown className="h-4 w-4" />
              )}
              {changeUp ? '+' : ''}
              {changePercent.toFixed(2)}%
            </div>
          </div>
          <Separator />
          <PriceChart trades={trades} />
        </CardContent>
      </Card>

      {/* 주문하기: 매수/매도 탭 */}
      <Card>
        <CardContent className="space-y-4 p-4">
          <Tabs
            value={orderType}
            onValueChange={(v) => setOrderType(v as 'buy' | 'sell')}
          >
            <TabsList className="w-full">
              <TabsTrigger value="buy" className="flex-1 gap-1">
                <TrendingUp className="h-3.5 w-3.5" /> 매수
              </TabsTrigger>
              <TabsTrigger value="sell" className="flex-1 gap-1">
                <TrendingDown className="h-3.5 w-3.5" /> 매도
              </TabsTrigger>
            </TabsList>
          </Tabs>

          {position && (
            <div className="flex items-center justify-between rounded-md bg-muted/50 px-3 py-2 text-xs">
              <span className="text-muted-foreground">
                보유{' '}
                <span className="font-medium text-foreground">
                  {position.shares.toLocaleString()}주
                </span>
                {position.available_shares !== position.shares && (
                  <span> · 매도가능 {position.available_shares.toLocaleString()}</span>
                )}
              </span>
              <span className="text-muted-foreground">
                여유자금{' '}
                <span className="font-medium text-foreground">
                  {formatMoney(position.available_cash)}
                </span>
              </span>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <div className="flex items-center justify-between">
                  <Label className="text-xs">수량 (주)</Label>
                  <button
                    type="button"
                    onClick={() => setShares(String(maxShares))}
                    disabled={maxShares <= 0}
                    className="text-[10px] font-medium text-primary disabled:opacity-40"
                  >
                    최대 {maxShares.toLocaleString()}
                  </button>
                </div>
                <Input
                  type="number"
                  min="1"
                  placeholder="0"
                  value={shares}
                  onChange={(e) => setShares(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs">가격 (원/주)</Label>
                <Input
                  type="number"
                  min="1"
                  placeholder="0"
                  value={price}
                  onChange={(e) => setPrice(e.target.value)}
                  required
                />
              </div>
            </div>

            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">총 금액</span>
              <span className="font-semibold">
                {total > 0 ? formatMoney(total) : '-'}
              </span>
            </div>

            {exceeds && (
              <p className="text-xs font-medium text-destructive">
                {orderType === 'buy'
                  ? '여유자금을 초과했습니다.'
                  : '매도 가능 수량을 초과했습니다.'}
              </p>
            )}

            <Button
              type="submit"
              className="w-full"
              disabled={submitting || exceeds || sharesNum <= 0 || priceNum <= 0}
              variant={orderType === 'buy' ? 'default' : 'destructive'}
            >
              {submitting
                ? '처리 중...'
                : orderType === 'buy'
                  ? '매수 주문'
                  : '매도 주문'}
            </Button>
          </form>
        </CardContent>
      </Card>

      {/* 호가창 + 체결 내역 */}
      <div className="grid grid-cols-2 gap-3">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">호가창</CardTitle>
            <p className="text-[10px] text-muted-foreground">눌러서 가격 입력</p>
          </CardHeader>
          <CardContent className="space-y-1 px-3 pb-3">
            {orderbook && orderbook.asks.length === 0 && orderbook.bids.length === 0 && (
              <p className="py-4 text-center text-xs text-muted-foreground">
                호가 없음
              </p>
            )}
            {[...(orderbook?.asks ?? [])].reverse().map((e, i) => (
              <button
                key={`ask-${i}`}
                type="button"
                onClick={() => fillFromOrderbook('buy', e.price)}
                className="flex w-full items-center justify-between rounded bg-coral/10 px-2 py-1 text-xs hover:bg-coral/20"
              >
                <span className="font-medium text-coral">
                  {formatMoney(e.price)}
                </span>
                <span className="text-muted-foreground">{e.shares}</span>
              </button>
            ))}
            {orderbook && (orderbook.asks.length > 0 || orderbook.bids.length > 0) && (
              <Separator className="my-1" />
            )}
            {(orderbook?.bids ?? []).map((e, i) => (
              <button
                key={`bid-${i}`}
                type="button"
                onClick={() => fillFromOrderbook('sell', e.price)}
                className="flex w-full items-center justify-between rounded bg-info/10 px-2 py-1 text-xs hover:bg-info/20"
              >
                <span className="font-medium text-info">
                  {formatMoney(e.price)}
                </span>
                <span className="text-muted-foreground">{e.shares}</span>
              </button>
            ))}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">체결 내역</CardTitle>
            <p className="text-[10px] text-muted-foreground">최근 거래이력</p>
          </CardHeader>
          <CardContent className="space-y-1 px-3 pb-3">
            {trades.length === 0 ? (
              <p className="py-4 text-center text-xs text-muted-foreground">
                체결 내역 없음
              </p>
            ) : (
              trades.map((t, i) => {
                const prev = trades[i + 1]?.price_per_share
                const up =
                  prev === undefined ? true : t.price_per_share >= prev
                return (
                  <div
                    key={t.id}
                    className="flex items-center justify-between px-1 py-1 text-xs"
                  >
                    <span className="text-muted-foreground">
                      {timeHM(t.created_at)}
                    </span>
                    <span
                      className={`font-medium ${up ? 'text-coral' : 'text-info'}`}
                    >
                      {formatMoney(t.price_per_share)}
                    </span>
                    <span className="text-muted-foreground">{t.shares}주</span>
                  </div>
                )
              })
            )}
          </CardContent>
        </Card>
      </div>

      {/* 내 주문 */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base">내 주문</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {myOrders.length === 0 ? (
            <p className="py-4 text-center text-sm text-muted-foreground">
              이 기업에 대한 주문이 없습니다.
            </p>
          ) : (
            myOrders.map((order) => (
              <div
                key={order.id}
                className="flex items-center justify-between rounded-lg border p-3"
              >
                <div className="flex items-center gap-2">
                  <Badge
                    variant={
                      order.order_type === 'buy' ? 'default' : 'destructive'
                    }
                    className="text-xs"
                  >
                    {order.order_type === 'buy' ? '매수' : '매도'}
                  </Badge>
                  <div>
                    <p className="text-sm">
                      {order.shares}주 x {formatMoney(order.price_per_share)}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {order.status === 'open'
                        ? '대기'
                        : order.status === 'partial'
                          ? '부분체결'
                          : order.status === 'filled'
                            ? '체결'
                            : order.status}
                    </p>
                  </div>
                </div>
                {(order.status === 'open' || order.status === 'partial') && (
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handleCancel(order.id)}
                  >
                    <X className="h-4 w-4 text-destructive" />
                  </Button>
                )}
              </div>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  )
}
