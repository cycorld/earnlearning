import { useState, useEffect, useCallback } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  Building2,
  Send,
  Search,
  Loader2,
  Check,
  ArrowDownLeft,
  ArrowUpRight,
  ChevronLeft,
  ChevronRight,
  ArrowLeft,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatMoney, displayName } from '@/lib/utils'
import type { Transaction, Pagination } from '@/types'

interface CompanyWalletData {
  company: {
    id: number
    owner_id: number
    name: string
    logo_url: string
    status: string
  }
  wallet: {
    id: number
    company_id: number
    balance: number
  }
  owner_name: string
  account_name: string // "회사명(대표자명)" 컨벤션
}

interface Recipient {
  id: number
  name: string
  student_id: string
  department: string
  avatar_url: string
  type: 'user' | 'company'
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ko-KR', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export default function CompanyWalletPage() {
  const { id } = useParams()
  const { user } = useAuth()

  const [data, setData] = useState<CompanyWalletData | null>(null)
  const [loading, setLoading] = useState(true)

  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [pagination, setPagination] = useState<Pagination | null>(null)
  const [txPage, setTxPage] = useState(1)
  const [txLoading, setTxLoading] = useState(false)

  const [transferOpen, setTransferOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [recipients, setRecipients] = useState<Recipient[]>([])
  const [searchLoading, setSearchLoading] = useState(false)
  const [selectedRecipient, setSelectedRecipient] = useState<Recipient | null>(null)
  const [amount, setAmount] = useState('')
  const [description, setDescription] = useState('')
  const [sending, setSending] = useState(false)

  const fetchWallet = useCallback(async () => {
    try {
      const res = await api.get<CompanyWalletData>(`/companies/${id}/wallet`)
      setData(res)
    } catch {
      setData(null)
    } finally {
      setLoading(false)
    }
  }, [id])

  const fetchTransactions = useCallback(
    async (p: number) => {
      setTxLoading(true)
      try {
        const res = await api.get<{ data: Transaction[]; pagination: Pagination }>(
          `/companies/${id}/transactions?page=${p}&limit=20`,
        )
        setTransactions(res?.data ?? [])
        setPagination(res?.pagination ?? null)
      } catch {
        setTransactions([])
      } finally {
        setTxLoading(false)
      }
    },
    [id],
  )

  useEffect(() => {
    fetchWallet()
  }, [fetchWallet])

  useEffect(() => {
    fetchTransactions(txPage)
  }, [txPage, fetchTransactions])

  // Recipient search
  useEffect(() => {
    const timer = setTimeout(async () => {
      if (!searchQuery.trim()) {
        setRecipients([])
        return
      }
      setSearchLoading(true)
      try {
        const res = await api.get<Recipient[]>(
          `/wallet/recipients?q=${encodeURIComponent(searchQuery)}`,
        )
        // 자기 자신(법인)은 받는 사람 목록에서 제외
        const filtered = (res ?? []).filter(
          (r) => !(r.type === 'company' && r.id === Number(id)),
        )
        setRecipients(filtered)
      } catch {
        setRecipients([])
      } finally {
        setSearchLoading(false)
      }
    }, 300)
    return () => clearTimeout(timer)
  }, [searchQuery, id])

  const handleTransfer = async () => {
    if (!selectedRecipient || !amount || Number(amount) <= 0) return
    setSending(true)
    try {
      await api.post(`/companies/${id}/transfer`, {
        target_id: selectedRecipient.id,
        target_type: selectedRecipient.type,
        amount: Number(amount),
        description: description.trim() || '법인 송금',
      })
      toast.success(`${displayName(selectedRecipient)}에게 ${formatMoney(Number(amount))} 송금 완료!`)
      setTransferOpen(false)
      setSelectedRecipient(null)
      setAmount('')
      setDescription('')
      setSearchQuery('')
      fetchWallet()
      fetchTransactions(1)
      setTxPage(1)
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : '송금에 실패했습니다.')
    } finally {
      setSending(false)
    }
  }

  const resetTransfer = () => {
    setSelectedRecipient(null)
    setAmount('')
    setDescription('')
    setSearchQuery('')
    setRecipients([])
  }

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  if (!data) {
    return (
      <div className="p-4 text-center text-muted-foreground">
        법인 지갑 정보를 불러올 수 없습니다.
      </div>
    )
  }

  const isOwner = user?.id === data.company.owner_id
  const canTransfer = isOwner && data.company.status !== 'dissolved'

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      {/* Back link */}
      <Button variant="ghost" size="sm" asChild className="gap-1 px-2">
        <Link to={`/company/${id}`}>
          <ArrowLeft className="h-4 w-4" />
          {data.account_name}
        </Link>
      </Button>

      {/* Balance card — purple/indigo 톤으로 개인 지갑(primary gradient)과 구분 */}
      <Card className="border-purple-200 bg-gradient-to-br from-purple-600 to-indigo-700 text-white">
        <CardContent className="p-6">
          <div className="mb-3 flex items-center gap-2">
            <Avatar className="h-8 w-8 border border-white/30">
              <AvatarImage src={data.company.logo_url} />
              <AvatarFallback className="bg-white/20 text-sm text-white">
                {data.company.name.charAt(0)}
              </AvatarFallback>
            </Avatar>
            <div>
              <Badge variant="secondary" className="gap-1 bg-white/20 text-white hover:bg-white/20">
                <Building2 className="h-3 w-3" />
                법인 계좌
              </Badge>
            </div>
          </div>
          <p className="text-sm opacity-80">{data.account_name} 법인 잔액</p>
          <p className="text-3xl font-bold">{formatMoney(data.wallet.balance)}</p>
          <p className="mt-2 text-xs opacity-70">
            ⚠️ 이 금액은 개인 재산과 분리되어 있습니다. 법인의 자산으로 법인 운영에 사용됩니다.
          </p>
        </CardContent>
      </Card>

      {/* Transfer button — owner only */}
      {canTransfer && (
        <Dialog
          open={transferOpen}
          onOpenChange={(open) => {
            setTransferOpen(open)
            if (!open) resetTransfer()
          }}
        >
          <DialogTrigger asChild>
            <Button className="w-full gap-2 bg-purple-600 hover:bg-purple-700">
              <Send className="h-4 w-4" />
              법인에서 송금하기
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>법인 송금 — {data.account_name}</DialogTitle>
            </DialogHeader>
            <div className="space-y-4">
              {!selectedRecipient ? (
                <>
                  <div className="space-y-2">
                    <Label>받는 사람 / 법인</Label>
                    <div className="relative">
                      <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                      <Input
                        placeholder="이름, 학번, 법인명으로 검색"
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="pl-9"
                        autoFocus
                      />
                    </div>
                  </div>
                  {searchLoading ? (
                    <div className="flex justify-center py-4">
                      <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                    </div>
                  ) : recipients.length > 0 ? (
                    <div className="max-h-60 space-y-1 overflow-y-auto">
                      {recipients.map((r) => (
                        <button
                          key={`${r.type}-${r.id}`}
                          onClick={() => setSelectedRecipient(r)}
                          className="flex w-full items-center gap-3 rounded-lg border p-3 text-left transition-colors hover:bg-accent"
                        >
                          <Avatar className="h-8 w-8 shrink-0">
                            <AvatarImage src={r.avatar_url} />
                            <AvatarFallback className="text-xs">{r.name.charAt(0)}</AvatarFallback>
                          </Avatar>
                          <div className="min-w-0 flex-1">
                            <p className="text-sm font-medium">{displayName(r)}</p>
                          </div>
                          {r.type === 'company' && (
                            <Badge variant="secondary" className="gap-1 text-xs">
                              <Building2 className="h-3 w-3" />
                              법인
                            </Badge>
                          )}
                        </button>
                      ))}
                    </div>
                  ) : searchQuery.trim() ? (
                    <p className="py-4 text-center text-sm text-muted-foreground">
                      검색 결과가 없습니다.
                    </p>
                  ) : (
                    <p className="py-4 text-center text-sm text-muted-foreground">
                      개인은 이름/학번, 법인은 법인명으로 검색하세요.
                    </p>
                  )}
                </>
              ) : (
                <>
                  <div className="space-y-2">
                    <Label>받는 사람</Label>
                    <div className="flex items-center gap-3 rounded-lg border bg-muted/50 p-3">
                      <Avatar className="h-8 w-8 shrink-0">
                        <AvatarImage src={selectedRecipient.avatar_url} />
                        <AvatarFallback className="text-xs">
                          {selectedRecipient.name.charAt(0)}
                        </AvatarFallback>
                      </Avatar>
                      <div className="flex-1">
                        <p className="text-sm font-medium">{displayName(selectedRecipient)}</p>
                        {selectedRecipient.type === 'company' && (
                          <p className="text-xs text-muted-foreground">법인 계좌로 송금</p>
                        )}
                      </div>
                      <Button variant="ghost" size="sm" onClick={() => setSelectedRecipient(null)}>
                        변경
                      </Button>
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label>금액 (원)</Label>
                    <Input
                      type="number"
                      min="1"
                      placeholder="송금할 금액"
                      value={amount}
                      onChange={(e) => setAmount(e.target.value)}
                      autoFocus
                    />
                    {amount && Number(amount) > 0 && (
                      <p className="text-xs text-muted-foreground">
                        {formatMoney(Number(amount))}
                        {Number(amount) > data.wallet.balance && (
                          <span className="ml-2 text-red-500">법인 잔액 부족</span>
                        )}
                      </p>
                    )}
                  </div>
                  <div className="space-y-2">
                    <Label>메모 (선택)</Label>
                    <Input
                      placeholder="송금 사유를 입력하세요"
                      value={description}
                      onChange={(e) => setDescription(e.target.value)}
                    />
                  </div>
                </>
              )}
            </div>
            {selectedRecipient && (
              <DialogFooter>
                <Button
                  onClick={handleTransfer}
                  disabled={
                    sending ||
                    !amount ||
                    Number(amount) <= 0 ||
                    Number(amount) > data.wallet.balance
                  }
                  className="w-full gap-2 bg-purple-600 hover:bg-purple-700"
                >
                  {sending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Check className="h-4 w-4" />
                  )}
                  {amount && Number(amount) > 0 ? `${formatMoney(Number(amount))} 송금` : '송금'}
                </Button>
              </DialogFooter>
            )}
          </DialogContent>
        </Dialog>
      )}

      {!isOwner && (
        <p className="rounded-lg border bg-muted/40 p-3 text-center text-xs text-muted-foreground">
          법인 송금은 대표만 할 수 있습니다.
        </p>
      )}

      {/* Transactions */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base">법인 거래 내역</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {txLoading ? (
            <div className="flex justify-center py-4">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : transactions.length === 0 ? (
            <p className="py-6 text-center text-sm text-muted-foreground">
              아직 거래 내역이 없습니다.
            </p>
          ) : (
            transactions.map((tx) => (
              <div key={tx.id} className="flex items-center gap-3 rounded-lg border p-3">
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
                    {formatMoney(Math.abs(tx.amount))}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    잔액 {formatMoney(tx.balance_after)}
                  </p>
                </div>
              </div>
            ))
          )}

          {pagination && pagination.total_pages > 1 && (
            <div className="flex items-center justify-center gap-2 pt-2">
              <Button
                variant="outline"
                size="sm"
                disabled={txPage <= 1}
                onClick={() => setTxPage((p) => p - 1)}
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <span className="text-sm text-muted-foreground">
                {txPage} / {pagination.total_pages}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={txPage >= pagination.total_pages}
                onClick={() => setTxPage((p) => p + 1)}
              >
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
