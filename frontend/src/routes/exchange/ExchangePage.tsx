import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/lib/api'
import type { Company } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import {
  TrendingUp,
  ArrowRight,
  ShoppingCart,
  Clock,
  RefreshCw,
} from 'lucide-react'
import { formatMoney } from '@/lib/utils'
import { Spinner } from '@/components/ui/spinner'

interface ExchangeOrder {
  id: number
  company_id: number
  company_name?: string
  order_type: 'buy' | 'sell'
  shares: number
  price: number
  status: string
  created_at: string
}

export default function ExchangePage() {
  const [companies, setCompanies] = useState<Company[]>([])
  const [myOrders, setMyOrders] = useState<ExchangeOrder[]>([])
  const [loading, setLoading] = useState(true)

  const fetchData = async () => {
    setLoading(true)
    try {
      const [companiesData, ordersData] = await Promise.all([
        api.get<Company[]>('/exchange/companies'),
        api.get<{ orders: ExchangeOrder[]; total: number } | ExchangeOrder[]>('/exchange/orders/mine'),
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
        <CardContent className="space-y-2">
          {companies.length === 0 ? (
            <p className="py-4 text-center text-sm text-muted-foreground">
              상장된 기업이 없습니다.
            </p>
          ) : (
            companies.map((company) => {
              const price =
                company.total_shares > 0
                  ? Math.round(company.valuation / company.total_shares)
                  : 0
              return (
                <Link key={company.id} to={`/exchange/${company.id}`}>
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
        <CardContent className="space-y-2">
          {myOrders.length === 0 ? (
            <p className="py-4 text-center text-sm text-muted-foreground">
              주문 내역이 없습니다.
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
                    <p className="text-sm font-medium">
                      {order.company_name || `기업 #${order.company_id}`}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {order.shares}주 x {formatMoney(order.price)}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-1 text-xs text-muted-foreground">
                  <Clock className="h-3 w-3" />
                  <Badge variant="outline" className="text-xs">
                    {order.status === 'open' ? '대기' : order.status}
                  </Badge>
                </div>
              </div>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  )
}
