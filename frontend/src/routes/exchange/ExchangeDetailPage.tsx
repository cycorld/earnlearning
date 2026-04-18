import { useEffect, useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { toast } from 'sonner'
import {
  ArrowLeft,
  TrendingUp,
  TrendingDown,
  ShoppingCart,
  X,
  RefreshCw,
} from 'lucide-react'
import { formatMoney } from '@/lib/utils'
import { Spinner } from '@/components/ui/spinner'

interface OrderbookEntry {
  price: number
  shares: number
  order_count: number
}

interface Orderbook {
  buy_orders: OrderbookEntry[]
  sell_orders: OrderbookEntry[]
}

interface ExchangeOrder {
  id: number
  company_id: number
  order_type: 'buy' | 'sell'
  shares: number
  price: number
  status: string
  created_at: string
}

export default function ExchangeDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const companyId = Number(id)

  const [orderbook, setOrderbook] = useState<Orderbook | null>(null)
  const [myOrders, setMyOrders] = useState<ExchangeOrder[]>([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)

  const [orderType, setOrderType] = useState<'buy' | 'sell'>('buy')
  const [shares, setShares] = useState('')
  const [price, setPrice] = useState('')

  const fetchData = useCallback(async () => {
    try {
      const [ob, orders] = await Promise.all([
        api.get<Orderbook>(`/exchange/orderbook/${companyId}`),
        api.get<ExchangeOrder[]>('/exchange/orders/mine'),
      ])
      setOrderbook(ob)
      setMyOrders((orders || []).filter((o) => o.company_id === companyId))
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [companyId])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!shares || !price) return

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

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="text-xl font-bold">호가창</h1>
        <div className="flex-1" />
        <Button variant="ghost" size="icon" onClick={fetchData}>
          <RefreshCw className="h-4 w-4" />
        </Button>
      </div>

      {orderbook && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-base">오더북</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-1">
              <p className="text-xs font-medium text-coral">매도 호가</p>
              {orderbook.sell_orders.length === 0 ? (
                <p className="py-2 text-center text-xs text-muted-foreground">
                  매도 주문 없음
                </p>
              ) : (
                [...orderbook.sell_orders].reverse().map((entry, i) => (
                  <div
                    key={`sell-${i}`}
                    className="flex items-center justify-between rounded px-2 py-1 text-sm"
                  >
                    <span className="text-muted-foreground">
                      {entry.shares}주
                      <span className="ml-1 text-xs">
                        ({entry.order_count}건)
                      </span>
                    </span>
                    <span className="font-medium text-coral">
                      {formatMoney(entry.price)}
                    </span>
                  </div>
                ))
              )}
            </div>

            <Separator className="my-2" />

            <div className="space-y-1">
              <p className="text-xs font-medium text-info">매수 호가</p>
              {orderbook.buy_orders.length === 0 ? (
                <p className="py-2 text-center text-xs text-muted-foreground">
                  매수 주문 없음
                </p>
              ) : (
                orderbook.buy_orders.map((entry, i) => (
                  <div
                    key={`buy-${i}`}
                    className="flex items-center justify-between rounded px-2 py-1 text-sm"
                  >
                    <span className="font-medium text-info">
                      {formatMoney(entry.price)}
                    </span>
                    <span className="text-muted-foreground">
                      {entry.shares}주
                      <span className="ml-1 text-xs">
                        ({entry.order_count}건)
                      </span>
                    </span>
                  </div>
                ))
              )}
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="flex items-center gap-2 text-base">
            <ShoppingCart className="h-4 w-4" />
            주문하기
          </CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label>주문 유형</Label>
              <Select
                value={orderType}
                onValueChange={(v) => setOrderType(v as 'buy' | 'sell')}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="buy">
                    <span className="flex items-center gap-1">
                      <TrendingUp className="h-3 w-3 text-info" /> 매수
                    </span>
                  </SelectItem>
                  <SelectItem value="sell">
                    <span className="flex items-center gap-1">
                      <TrendingDown className="h-3 w-3 text-coral" /> 매도
                    </span>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>수량 (주)</Label>
              <Input
                type="number"
                min="1"
                placeholder="주문 수량"
                value={shares}
                onChange={(e) => setShares(e.target.value)}
                required
              />
            </div>

            <div className="space-y-2">
              <Label>가격 (원/주)</Label>
              <Input
                type="number"
                min="1"
                placeholder="주당 가격"
                value={price}
                onChange={(e) => setPrice(e.target.value)}
                required
              />
            </div>

            {shares && price && (
              <p className="text-sm text-muted-foreground">
                총 금액:{' '}
                <span className="font-semibold text-foreground">
                  {formatMoney(Number(shares) * Number(price))}
                </span>
              </p>
            )}

            <Button
              type="submit"
              className="w-full"
              disabled={submitting}
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
                      {order.shares}주 x {formatMoney(order.price)}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {order.status === 'open' ? '대기' : order.status}
                    </p>
                  </div>
                </div>
                {order.status === 'open' && (
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
