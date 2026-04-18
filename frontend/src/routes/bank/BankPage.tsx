import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/lib/api'
import type { Loan } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Landmark,
  Plus,
  Calendar,
  Percent,
  RefreshCw,
} from 'lucide-react'
import { formatMoney } from '@/lib/utils'

function loanStatusLabel(status: string): string {
  switch (status) {
    case 'pending':
      return '심사 중'
    case 'active':
      return '상환 중'
    case 'completed':
      return '상환 완료'
    case 'rejected':
      return '거절됨'
    case 'defaulted':
      return '연체'
    default:
      return status
  }
}

function loanStatusVariant(
  status: string,
): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (status) {
    case 'pending':
      return 'secondary'
    case 'active':
      return 'default'
    case 'completed':
      return 'outline'
    case 'rejected':
    case 'defaulted':
      return 'destructive'
    default:
      return 'outline'
  }
}

export default function BankPage() {
  const [loans, setLoans] = useState<Loan[]>([])
  const [loading, setLoading] = useState(true)

  const fetchLoans = async () => {
    setLoading(true)
    try {
      const data = await api.get<Loan[]>('/loans/mine')
      setLoans(data || [])
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchLoans()
  }, [])

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      <div className="flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-xl font-bold">
          <Landmark className="h-5 w-5" />
          은행
        </h1>
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="icon" onClick={fetchLoans}>
            <RefreshCw className="h-4 w-4" />
          </Button>
          <Button size="sm" asChild>
            <Link to="/bank/apply" className="flex items-center gap-1">
              <Plus className="h-4 w-4" /> 대출 신청
            </Link>
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base">내 대출 현황</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {loans.length === 0 ? (
            <div className="py-6 text-center">
              <p className="text-sm text-muted-foreground">
                대출 내역이 없습니다.
              </p>
              <Button variant="link" asChild className="mt-2">
                <Link to="/bank/apply">대출 신청하기</Link>
              </Button>
            </div>
          ) : (
            loans.map((loan) => (
              <Card key={loan.id}>
                <CardContent className="space-y-3 p-4">
                  <div className="flex items-start justify-between">
                    <div>
                      <p className="text-sm font-medium">{loan.purpose}</p>
                      <p className="text-xs text-muted-foreground">
                        대출 #{loan.id}
                      </p>
                    </div>
                    <Badge variant={loanStatusVariant(loan.status)}>
                      {loanStatusLabel(loan.status)}
                    </Badge>
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <p className="text-xs text-muted-foreground">대출금</p>
                      <p className="text-sm font-semibold">
                        {formatMoney(loan.amount)}
                      </p>
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">잔액</p>
                      <p className="text-sm font-semibold text-coral">
                        {formatMoney(loan.remaining)}
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center gap-4 text-xs text-muted-foreground">
                    <span className="flex items-center gap-1">
                      <Percent className="h-3 w-3" />
                      이자율 {loan.interest_rate}%
                    </span>
                    {loan.next_payment && (
                      <span className="flex items-center gap-1">
                        <Calendar className="h-3 w-3" />
                        다음 납부:{' '}
                        {new Date(loan.next_payment).toLocaleDateString(
                          'ko-KR',
                        )}
                      </span>
                    )}
                  </div>

                  {loan.weekly_interest != null && loan.status === 'active' && (
                    <p className="text-xs text-muted-foreground">
                      주간 이자: {formatMoney(loan.weekly_interest)}
                    </p>
                  )}
                </CardContent>
              </Card>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  )
}
