import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import type { Loan } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { toast } from 'sonner'
import {
  ArrowLeft,
  Landmark,
  Check,
  X,
  Clock,
  PlayCircle,
  CheckCircle,
  Calculator,
} from 'lucide-react'

interface AdminLoan extends Loan {
  borrower_id: number
  borrower_name?: string
}

function formatMoney(amount: number): string {
  return new Intl.NumberFormat('ko-KR').format(amount) + '원'
}

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

export default function AdminLoansPage() {
  const navigate = useNavigate()
  const [loans, setLoans] = useState<AdminLoan[]>([])
  const [loading, setLoading] = useState(true)

  const [approveDialogOpen, setApproveDialogOpen] = useState(false)
  const [selectedLoan, setSelectedLoan] = useState<AdminLoan | null>(null)
  const [interestRate, setInterestRate] = useState('')
  const [processing, setProcessing] = useState(false)

  const fetchLoans = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api.get<{ loans: AdminLoan[]; total: number } | AdminLoan[]>('/admin/loans')
      const loansArr = Array.isArray(data) ? data : (data?.loans ?? [])
      setLoans(loansArr)
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchLoans()
  }, [fetchLoans])

  const openApproveDialog = (loan: AdminLoan) => {
    setSelectedLoan(loan)
    setInterestRate('')
    setApproveDialogOpen(true)
  }

  const handleApprove = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedLoan || !interestRate) return

    setProcessing(true)
    try {
      await api.put(`/admin/loans/${selectedLoan.id}/approve`, {
        interest_rate: Number(interestRate),
      })
      toast.success('대출이 승인되었습니다.')
      setApproveDialogOpen(false)
      fetchLoans()
    } catch (err: any) {
      toast.error(err.message || '승인에 실패했습니다.')
    } finally {
      setProcessing(false)
    }
  }

  const handleReject = async (loanId: number) => {
    try {
      await api.put(`/admin/loans/${loanId}/reject`)
      toast.success('대출이 거절되었습니다.')
      fetchLoans()
    } catch (err: any) {
      toast.error(err.message || '거절에 실패했습니다.')
    }
  }

  const handleWeeklyInterest = async () => {
    if (!confirm('주간 이자를 처리하시겠습니까?')) return

    setProcessing(true)
    try {
      await api.post('/admin/loans/weekly-interest')
      toast.success('주간 이자가 처리되었습니다.')
      fetchLoans()
    } catch (err: any) {
      toast.error(err.message || '이자 처리에 실패했습니다.')
    } finally {
      setProcessing(false)
    }
  }

  const pendingLoans = loans.filter((l) => l.status === 'pending')
  const activeLoans = loans.filter((l) => l.status === 'active')
  const completedLoans = loans.filter(
    (l) => l.status === 'completed' || l.status === 'rejected',
  )

  const renderLoanCard = (loan: AdminLoan) => (
    <div
      key={loan.id}
      className="space-y-2 rounded-lg border p-3"
    >
      <div className="flex items-start justify-between">
        <div>
          <p className="text-sm font-medium">
            {loan.borrower_name || `사용자 #${loan.borrower_id}`}
          </p>
          <p className="text-xs text-muted-foreground">{loan.purpose}</p>
        </div>
        <Badge variant={loanStatusVariant(loan.status)}>
          {loanStatusLabel(loan.status)}
        </Badge>
      </div>

      <div className="grid grid-cols-3 gap-2 text-xs">
        <div>
          <p className="text-muted-foreground">대출금</p>
          <p className="font-medium">{formatMoney(loan.amount)}</p>
        </div>
        <div>
          <p className="text-muted-foreground">잔액</p>
          <p className="font-medium">{formatMoney(loan.remaining)}</p>
        </div>
        <div>
          <p className="text-muted-foreground">이자율</p>
          <p className="font-medium">{loan.interest_rate}%</p>
        </div>
      </div>

      {loan.status === 'pending' && (
        <div className="flex items-center gap-2 pt-1">
          <Button
            size="sm"
            variant="default"
            className="flex-1"
            onClick={() => openApproveDialog(loan)}
          >
            <Check className="mr-1 h-3 w-3" /> 승인
          </Button>
          <Button
            size="sm"
            variant="destructive"
            className="flex-1"
            onClick={() => handleReject(loan.id)}
          >
            <X className="mr-1 h-3 w-3" /> 거절
          </Button>
        </div>
      )}
    </div>
  )

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-lg space-y-4 p-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <h1 className="flex items-center gap-2 text-xl font-bold">
            <Landmark className="h-5 w-5" />
            대출 관리
          </h1>
        </div>
        <Button
          size="sm"
          variant="outline"
          onClick={handleWeeklyInterest}
          disabled={processing}
        >
          <Calculator className="mr-1 h-4 w-4" />
          주간 이자
        </Button>
      </div>

      <Tabs defaultValue="pending">
        <TabsList className="w-full">
          <TabsTrigger value="pending" className="flex-1">
            <Clock className="mr-1 h-3 w-3" />
            대기 ({pendingLoans.length})
          </TabsTrigger>
          <TabsTrigger value="active" className="flex-1">
            <PlayCircle className="mr-1 h-3 w-3" />
            진행 ({activeLoans.length})
          </TabsTrigger>
          <TabsTrigger value="completed" className="flex-1">
            <CheckCircle className="mr-1 h-3 w-3" />
            완료 ({completedLoans.length})
          </TabsTrigger>
        </TabsList>

        <TabsContent value="pending" className="mt-4">
          <Card>
            <CardContent className="space-y-3 p-4">
              {pendingLoans.length === 0 ? (
                <p className="py-4 text-center text-sm text-muted-foreground">
                  대기 중인 대출이 없습니다.
                </p>
              ) : (
                pendingLoans.map(renderLoanCard)
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="active" className="mt-4">
          <Card>
            <CardContent className="space-y-3 p-4">
              {activeLoans.length === 0 ? (
                <p className="py-4 text-center text-sm text-muted-foreground">
                  진행 중인 대출이 없습니다.
                </p>
              ) : (
                activeLoans.map(renderLoanCard)
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="completed" className="mt-4">
          <Card>
            <CardContent className="space-y-3 p-4">
              {completedLoans.length === 0 ? (
                <p className="py-4 text-center text-sm text-muted-foreground">
                  완료된 대출이 없습니다.
                </p>
              ) : (
                completedLoans.map(renderLoanCard)
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <Dialog open={approveDialogOpen} onOpenChange={setApproveDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>대출 승인</DialogTitle>
          </DialogHeader>
          {selectedLoan && (
            <form onSubmit={handleApprove} className="space-y-4">
              <div className="rounded-lg bg-muted p-3 text-sm">
                <p>
                  <span className="text-muted-foreground">신청자: </span>
                  {selectedLoan.borrower_name || `#${selectedLoan.borrower_id}`}
                </p>
                <p>
                  <span className="text-muted-foreground">금액: </span>
                  {formatMoney(selectedLoan.amount)}
                </p>
                <p>
                  <span className="text-muted-foreground">목적: </span>
                  {selectedLoan.purpose}
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="interest-rate">이자율 (%)</Label>
                <Input
                  id="interest-rate"
                  type="number"
                  min="0"
                  max="100"
                  step="0.1"
                  placeholder="주간 이자율"
                  value={interestRate}
                  onChange={(e) => setInterestRate(e.target.value)}
                  required
                />
              </div>
              <Button type="submit" className="w-full" disabled={processing}>
                {processing ? '처리 중...' : '승인하기'}
              </Button>
            </form>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
