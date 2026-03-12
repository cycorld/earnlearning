import { Link } from 'react-router-dom'
import { useWallet } from '@/hooks/use-wallet'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Wallet, TrendingUp, Building2, CreditCard, ArrowRight } from 'lucide-react'

function formatMoney(amount: number): string {
  return new Intl.NumberFormat('ko-KR').format(amount) + '원'
}

export default function WalletPage() {
  const { wallet, loading, refresh } = useWallet()

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  if (!wallet) {
    return (
      <div className="p-4 text-center text-muted-foreground">
        지갑 정보를 불러올 수 없습니다.
        <Button variant="link" onClick={refresh}>
          다시 시도
        </Button>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-lg space-y-4 p-4">
      <Card className="bg-gradient-to-br from-primary to-primary/80 text-primary-foreground">
        <CardContent className="p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm opacity-80">총 자산</p>
              <p className="text-2xl font-bold">{formatMoney(wallet.total_asset_value)}</p>
            </div>
            <Badge variant="secondary" className="text-xs">
              {wallet.rank}위 / {wallet.total_students}명
            </Badge>
          </div>
        </CardContent>
      </Card>

      <div className="grid grid-cols-2 gap-3">
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-green-100">
              <Wallet className="h-5 w-5 text-green-600" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">보유 현금</p>
              <p className="text-sm font-semibold">{formatMoney(wallet.asset_breakdown.cash)}</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-blue-100">
              <TrendingUp className="h-5 w-5 text-blue-600" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">주식 가치</p>
              <p className="text-sm font-semibold">
                {formatMoney(wallet.asset_breakdown.stock_value)}
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-purple-100">
              <Building2 className="h-5 w-5 text-purple-600" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">회사 지분</p>
              <p className="text-sm font-semibold">
                {formatMoney(wallet.asset_breakdown.company_equity)}
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-red-100">
              <CreditCard className="h-5 w-5 text-red-600" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">총 부채</p>
              <p className="text-sm font-semibold text-red-600">
                -{formatMoney(wallet.asset_breakdown.total_debt)}
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-base">거래 내역</CardTitle>
          <Button variant="ghost" size="sm" asChild>
            <Link to="/wallet/transactions" className="flex items-center gap-1">
              전체보기 <ArrowRight className="h-4 w-4" />
            </Link>
          </Button>
        </CardHeader>
        <CardContent>
          <p className="text-center text-sm text-muted-foreground py-4">
            거래 내역은 전체보기에서 확인하세요.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
