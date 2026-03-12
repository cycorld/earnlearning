import { useState, useEffect, useCallback } from 'react'
import { api } from '@/lib/api'
import type { Transaction, PaginatedData, Pagination } from '@/types'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ArrowDownLeft, ArrowUpRight, ChevronLeft, ChevronRight } from 'lucide-react'

function formatMoney(amount: number): string {
  return new Intl.NumberFormat('ko-KR').format(Math.abs(amount)) + '원'
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ko-KR', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export default function TransactionsPage() {
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [pagination, setPagination] = useState<Pagination | null>(null)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)

  const fetchTransactions = useCallback(async (p: number) => {
    setLoading(true)
    try {
      const data = await api.get<PaginatedData<Transaction>>(
        `/wallet/transactions?page=${p}&limit=20`,
      )
      setTransactions(data?.data ?? [])
      setPagination(data?.pagination ?? null)
    } catch {
      setTransactions([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchTransactions(page)
  }, [page, fetchTransactions])

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-lg space-y-3 p-4">
      <h1 className="text-lg font-bold">거래 내역</h1>
      {transactions.length === 0 ? (
        <p className="py-8 text-center text-muted-foreground">거래 내역이 없습니다.</p>
      ) : (
        <>
          {transactions.map((tx) => (
            <Card key={tx.id}>
              <CardContent className="flex items-center gap-3 p-4">
                <div
                  className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-full ${
                    tx.amount >= 0 ? 'bg-green-100' : 'bg-red-100'
                  }`}
                >
                  {tx.amount >= 0 ? (
                    <ArrowDownLeft className="h-4 w-4 text-green-600" />
                  ) : (
                    <ArrowUpRight className="h-4 w-4 text-red-600" />
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium">{tx.description}</p>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="text-xs">
                      {tx.tx_type}
                    </Badge>
                    <span className="text-xs text-muted-foreground">
                      {formatDate(tx.created_at)}
                    </span>
                  </div>
                </div>
                <div className="text-right">
                  <p
                    className={`text-sm font-semibold ${
                      tx.amount >= 0 ? 'text-green-600' : 'text-red-600'
                    }`}
                  >
                    {tx.amount >= 0 ? '+' : '-'}
                    {formatMoney(tx.amount)}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    잔액 {formatMoney(tx.balance_after)}
                  </p>
                </div>
              </CardContent>
            </Card>
          ))}

          {pagination && pagination.total_pages > 1 && (
            <div className="flex items-center justify-center gap-2 pt-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <span className="text-sm text-muted-foreground">
                {page} / {pagination.total_pages}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= pagination.total_pages}
                onClick={() => setPage((p) => p + 1)}
              >
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
