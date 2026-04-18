import { useState, useEffect, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/lib/api'
import { useWallet } from '@/hooks/use-wallet'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
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
  Wallet,
  TrendingUp,
  Building2,
  CreditCard,
  ArrowRight,
  Send,
  Search,
  Loader2,
  Check,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatMoney, displayName } from '@/lib/utils'

interface Recipient {
  id: number
  name: string
  student_id: string
  department: string
  avatar_url: string
  type: 'user' | 'company'
}

export default function WalletPage() {
  const { wallet, loading, refresh } = useWallet()

  // Transfer state
  const [transferOpen, setTransferOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [recipients, setRecipients] = useState<Recipient[]>([])
  const [searchLoading, setSearchLoading] = useState(false)
  const [selectedRecipient, setSelectedRecipient] = useState<Recipient | null>(null)
  const [amount, setAmount] = useState('')
  const [description, setDescription] = useState('')
  const [sending, setSending] = useState(false)

  // Search recipients
  const searchRecipients = useCallback(async (q: string) => {
    if (!q.trim()) {
      setRecipients([])
      return
    }
    setSearchLoading(true)
    try {
      const data = await api.get<Recipient[]>(`/wallet/recipients?q=${encodeURIComponent(q)}`)
      setRecipients(data ?? [])
    } catch {
      setRecipients([])
    } finally {
      setSearchLoading(false)
    }
  }, [])

  useEffect(() => {
    const timer = setTimeout(() => searchRecipients(searchQuery), 300)
    return () => clearTimeout(timer)
  }, [searchQuery, searchRecipients])

  const handleTransfer = async () => {
    if (!selectedRecipient || !amount || Number(amount) <= 0) return
    setSending(true)
    try {
      await api.post('/wallet/transfer', {
        target_user_id: selectedRecipient.id,
        target_type: selectedRecipient.type,
        amount: Number(amount),
        description: description.trim() || '개인 송금',
      })
      toast.success(`${displayName(selectedRecipient)}에게 ${formatMoney(Number(amount))} 송금 완료!`)
      setTransferOpen(false)
      setSelectedRecipient(null)
      setAmount('')
      setDescription('')
      setSearchQuery('')
      refresh()
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
    <div className="mx-auto max-w-lg space-y-5 p-4">
      <Card className="bg-gradient-to-br from-primary to-primary/80 text-primary-foreground">
        <CardContent className="p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm opacity-80">총 자산</p>
              <p className="text-2xl font-bold">{formatMoney(Number(wallet.total_asset_value) || 0)}</p>
            </div>
            <Badge variant="secondary" className="text-xs">
              {wallet.rank ?? 0}위 / {wallet.total_students ?? 0}명
            </Badge>
          </div>
        </CardContent>
      </Card>

      <div className="grid grid-cols-2 gap-4">
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10">
              <Wallet className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">보유 현금</p>
              <p className="text-sm font-semibold">{formatMoney(wallet.asset_breakdown?.cash ?? 0)}</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-info/15">
              <TrendingUp className="h-5 w-5 text-info" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">주식 가치</p>
              <p className="text-sm font-semibold">
                {formatMoney(wallet.asset_breakdown?.stock_value ?? 0)}
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-entity/15">
              <Building2 className="h-5 w-5 text-entity" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">회사 지분</p>
              <p className="text-sm font-semibold">
                {formatMoney(wallet.asset_breakdown?.company_equity ?? 0)}
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-coral/15">
              <CreditCard className="h-5 w-5 text-coral" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">총 부채</p>
              <p className="text-sm font-semibold text-coral">
                -{formatMoney(wallet.asset_breakdown?.total_debt ?? 0)}
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Transfer Button */}
      <Dialog open={transferOpen} onOpenChange={(open) => { setTransferOpen(open); if (!open) resetTransfer() }}>
        <DialogTrigger asChild>
          <Button className="w-full gap-2">
            <Send className="h-4 w-4" />
            송금하기
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>송금하기</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            {!selectedRecipient ? (
              <>
                {/* Recipient Search */}
                <div className="space-y-2">
                  <Label>받는 사람</Label>
                  <div className="relative">
                    <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                    <Input
                      placeholder="이름, 학번, 학과로 검색"
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
                  <p className="py-4 text-center text-sm text-muted-foreground">검색 결과가 없습니다.</p>
                ) : (
                  <p className="py-4 text-center text-sm text-muted-foreground">이름이나 학번으로 검색하세요.</p>
                )}
              </>
            ) : (
              <>
                {/* Selected Recipient + Amount */}
                <div className="space-y-2">
                  <Label>받는 사람</Label>
                  <div className="flex items-center gap-3 rounded-lg border bg-muted/50 p-3">
                    <Avatar className="h-8 w-8 shrink-0">
                      <AvatarImage src={selectedRecipient.avatar_url} />
                      <AvatarFallback className="text-xs">{selectedRecipient.name.charAt(0)}</AvatarFallback>
                    </Avatar>
                    <div className="flex-1">
                      <p className="text-sm font-medium">{displayName(selectedRecipient)}</p>
                      {selectedRecipient.type === 'company' && (
                        <p className="text-xs text-entity">법인 계좌로 송금</p>
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
                      {Number(amount) > (wallet.asset_breakdown?.cash ?? 0) && (
                        <span className="ml-2 text-destructive">잔액 부족</span>
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
                disabled={sending || !amount || Number(amount) <= 0 || Number(amount) > (wallet.asset_breakdown?.cash ?? 0)}
                className="w-full gap-2"
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
