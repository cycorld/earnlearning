import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { toast } from 'sonner'
import { TrendingUp, ArrowRight, ShoppingCart, RefreshCw, X } from 'lucide-react'
import { formatMoney } from '@/lib/utils'
import { Spinner } from '@/components/ui/spinner'

interface ExchangeOrder {
  id: number
  company_id: number
  order_type: 'buy' | 'sell'
  shares: number
  remaining_shares: number
  price_per_share: number
  status: string
  created_at: string
}

// 경과 시간 (분/시간/일 전)
function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return '방금'
  if (mins < 60) return `${mins}분 전`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}시간 전`
  return `${Math.floor(hours / 24)}일 전`
}

// 주문 상태 → 한글 라벨
const STATUS_LABEL: Record<string, string> = {
  open: '대기',
  partial: '부분체결',
  filled: '체결',
  cancelled: '취소',
}

// 탭 정의: 진행중(open+partial) / 체결 / 취소 / 전체
const ORDER_TABS: { key: string; label: string; match: (s: string) => boolean }[] = [
  { key: 'active', label: '진행중', match: (s) => s === 'open' || s === 'partial' },
  { key: 'filled', label: '체결', match: (s) => s === 'filled' },
  { key: 'cancelled', label: '취소', match: (s) => s === 'cancelled' },
  { key: 'all', label: '전체', match: () => true },
]

// GET /exchange/companies 응답 (백엔드 ListedCompany)
interface ListedCompany {
  id: number
  name: string
  logo_url: string
  total_shares: number
  last_price: number // 시가 (마지막 체결가, 거래 없으면 마지막 라운드 가격)
  change_percent: number
  volume_24h: number
  market_cap: number
}

export default function ExchangePage() {
  const [companies, setCompanies] = useState<ListedCompany[]>([])
  const [myOrders, setMyOrders] = useState<ExchangeOrder[]>([])
  const [activeTab, setActiveTab] = useState('active')
  const [loading, setLoading] = useState(true)

  const fetchData = async () => {
    setLoading(true)
    try {
      const [companiesData, ordersData] = await Promise.all([
        api.get<ListedCompany[]>('/exchange/companies'),
        // limit=100: 탭 개수·필터를 위해 전량 조회 (교실 규모)
        api.get<{ orders: ExchangeOrder[]; total: number } | ExchangeOrder[]>(
          '/exchange/orders/mine?limit=100',
        ),
      ])
      setCompanies(Array.isArray(companiesData) ? companiesData : [])
      const ordersArr = Array.isArray(ordersData) ? ordersData : (ordersData?.orders ?? [])
      setMyOrders(ordersArr)
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  const handleCancel = async (orderId: number) => {
    try {
      await api.del(`/exchange/orders/${orderId}`)
      toast.success('주문이 취소되었습니다.')
      fetchData()
    } catch (err: any) {
      toast.error(err.message || '주문 취소에 실패했습니다.')
    }
  }

  // company_id → 회사 메타 (이름·로고). 주문은 회사명을 안 주므로 매핑.
  const companyMap = new Map(companies.map((c) => [c.id, c]))
  const countFor = (key: string) =>
    myOrders.filter((o) => ORDER_TABS.find((t) => t.key === key)!.match(o.status)).length
  const visibleOrders = myOrders.filter(
    (o) => ORDER_TABS.find((t) => t.key === activeTab)!.match(o.status),
  )

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Spinner />
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">거래소</h1>
        <Button variant="ghost" size="icon" onClick={fetchData}>
          <RefreshCw className="h-4 w-4" />
        </Button>
      </div>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="flex items-center gap-2 text-base">
            <TrendingUp className="h-4 w-4" />
            상장 기업
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {companies.length === 0 ? (
            <p className="py-4 text-center text-sm text-muted-foreground">
              상장된 기업이 없습니다.
            </p>
          ) : (
            companies.map((company) => {
              const price = company.last_price
              return (
                <Link
                  key={company.id}
                  to={`/exchange/${company.id}`}
                  className="block"
                >
                  <Card className="transition-colors hover:bg-accent">
                    <CardContent className="flex items-center justify-between p-4">
                      <div className="flex items-center gap-3">
                        {company.logo_url ? (
                          <img
                            src={company.logo_url}
                            alt={company.name}
                            className="h-10 w-10 rounded-full object-cover"
                          />
                        ) : (
                          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10">
                            <TrendingUp className="h-5 w-5 text-primary" />
                          </div>
                        )}
                        <div>
                          <p className="font-medium">{company.name}</p>
                          <p className="text-xs text-muted-foreground">
                            총 {company.total_shares.toLocaleString()}주
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <div className="text-right">
                          <p className="text-sm font-semibold">
                            {formatMoney(price)}
                          </p>
                          <p className="text-xs text-muted-foreground">/주</p>
                        </div>
                        <ArrowRight className="h-4 w-4 text-muted-foreground" />
                      </div>
                    </CardContent>
                  </Card>
                </Link>
              )
            })
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="flex items-center gap-2 text-base">
            <ShoppingCart className="h-4 w-4" />
            내 주문
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <Tabs value={activeTab} onValueChange={setActiveTab}>
            <TabsList className="w-full">
              {ORDER_TABS.map((t) => {
                const n = countFor(t.key)
                return (
                  <TabsTrigger key={t.key} value={t.key} className="flex-1 gap-1 text-xs">
                    {t.label}
                    {n > 0 && (
                      <span className="rounded-full bg-muted px-1.5 text-[10px] text-muted-foreground">
                        {n}
                      </span>
                    )}
                  </TabsTrigger>
                )
              })}
            </TabsList>
          </Tabs>

          {visibleOrders.length === 0 ? (
            <p className="py-4 text-center text-sm text-muted-foreground">
              {activeTab === 'active'
                ? '진행중인 주문이 없습니다.'
                : '해당 주문이 없습니다.'}
            </p>
          ) : (
            visibleOrders.map((order) => {
              const co = companyMap.get(order.company_id)
              const isActive = order.status === 'open' || order.status === 'partial'
              return (
                <div
                  key={order.id}
                  className="flex items-center justify-between gap-2 rounded-lg border p-3"
                >
                  <div className="flex min-w-0 items-center gap-2">
                    <Badge
                      variant={order.order_type === 'buy' ? 'default' : 'destructive'}
                      className="shrink-0 text-xs"
                    >
                      {order.order_type === 'buy' ? '매수' : '매도'}
                    </Badge>
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium">
                        {co?.name || `기업 #${order.company_id}`}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {order.shares}주 × {formatMoney(order.price_per_share)}
                        {order.status === 'partial' && (
                          <span className="text-coral">
                            {' '}
                            · 잔여 {order.remaining_shares}주
                          </span>
                        )}
                      </p>
                      <p className="text-[10px] text-muted-foreground">
                        {timeAgo(order.created_at)}
                      </p>
                    </div>
                  </div>
                  <div className="flex shrink-0 items-center gap-1">
                    <Badge variant="outline" className="text-xs">
                      {STATUS_LABEL[order.status] ?? order.status}
                    </Badge>
                    {isActive && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7"
                        onClick={() => handleCancel(order.id)}
                      >
                        <X className="h-4 w-4 text-destructive" />
                      </Button>
                    )}
                  </div>
                </div>
              )
            })
          )}
        </CardContent>
      </Card>
    </div>
  )
}
