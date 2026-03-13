import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import type { User } from '@/types'
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
  DialogTrigger,
} from '@/components/ui/dialog'
import { Separator } from '@/components/ui/separator'
import { toast } from 'sonner'
import {
  ArrowLeft,
  Users,
  Check,
  X,
  Send,
  UserCheck,
  Clock,
} from 'lucide-react'

function formatMoney(amount: number): string {
  return new Intl.NumberFormat('ko-KR').format(amount) + '원'
}

export default function AdminUsersPage() {
  const navigate = useNavigate()
  const [allUsers, setAllUsers] = useState<User[]>([])
  const [pendingUsers, setPendingUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)

  const [transferOpen, setTransferOpen] = useState(false)
  const [transferUserId, setTransferUserId] = useState<number | null>(null)
  const [transferAmount, setTransferAmount] = useState('')
  const [transferDesc, setTransferDesc] = useState('')
  const [transferring, setTransferring] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const [allData, pending] = await Promise.all([
        api.get<{ users: User[]; total: number } | User[]>('/admin/users?page=1&limit=1000'),
        api.get<User[]>('/admin/users/pending'),
      ])
      const allArr = Array.isArray(allData) ? allData : (allData?.users ?? [])
      const pendingArr = Array.isArray(pending) ? pending : []
      setAllUsers(allArr)
      setPendingUsers(pendingArr)
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  const handleApprove = async (userId: number) => {
    try {
      await api.put(`/admin/users/${userId}/approve`)
      toast.success('승인되었습니다.')
      fetchData()
    } catch (err: any) {
      toast.error(err.message || '승인에 실패했습니다.')
    }
  }

  const handleReject = async (userId: number) => {
    try {
      await api.put(`/admin/users/${userId}/reject`)
      toast.success('거절되었습니다.')
      fetchData()
    } catch (err: any) {
      toast.error(err.message || '거절에 실패했습니다.')
    }
  }

  const handleTransfer = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!transferUserId || !transferAmount) return

    setTransferring(true)
    try {
      await api.post('/admin/wallet/transfer', {
        user_id: transferUserId,
        amount: Number(transferAmount),
        description: transferDesc.trim() || '관리자 송금',
      })
      toast.success('송금이 완료되었습니다.')
      setTransferOpen(false)
      setTransferAmount('')
      setTransferDesc('')
      setTransferUserId(null)
    } catch (err: any) {
      toast.error(err.message || '송금에 실패했습니다.')
    } finally {
      setTransferring(false)
    }
  }

  const openTransfer = (userId: number) => {
    setTransferUserId(userId)
    setTransferAmount('')
    setTransferDesc('')
    setTransferOpen(true)
  }

  const statusLabel = (status: string) => {
    switch (status) {
      case 'approved':
        return '승인됨'
      case 'pending':
        return '대기'
      case 'rejected':
        return '거절됨'
      default:
        return status
    }
  }

  const statusVariant = (
    status: string,
  ): 'default' | 'secondary' | 'destructive' | 'outline' => {
    switch (status) {
      case 'approved':
        return 'default'
      case 'pending':
        return 'secondary'
      case 'rejected':
        return 'destructive'
      default:
        return 'outline'
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  const renderUserCard = (user: User, showActions: boolean) => (
    <div
      key={user.id}
      className="flex items-center justify-between rounded-lg border p-3"
    >
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="truncate text-sm font-medium">{user.name}</p>
          <Badge variant={statusVariant(user.status)} className="text-xs">
            {statusLabel(user.status)}
          </Badge>
          {user.role === 'admin' && (
            <Badge variant="outline" className="text-xs">
              관리자
            </Badge>
          )}
        </div>
        <p className="truncate text-xs text-muted-foreground">
          {user.email} | {user.department} | {user.student_id}
        </p>
        {user.wallet_balance != null && (
          <p className="text-xs text-muted-foreground">
            잔액: {formatMoney(user.wallet_balance)}
          </p>
        )}
      </div>
      <div className="flex shrink-0 items-center gap-1">
        {showActions && user.status === 'pending' && (
          <>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => handleApprove(user.id)}
              title="승인"
            >
              <Check className="h-4 w-4 text-green-600" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => handleReject(user.id)}
              title="거절"
            >
              <X className="h-4 w-4 text-red-600" />
            </Button>
          </>
        )}
        {user.status === 'approved' && (
          <Button
            variant="ghost"
            size="icon"
            onClick={() => openTransfer(user.id)}
            title="송금"
          >
            <Send className="h-4 w-4" />
          </Button>
        )}
      </div>
    </div>
  )

  return (
    <div className="mx-auto max-w-lg space-y-4 p-4">
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="flex items-center gap-2 text-xl font-bold">
          <Users className="h-5 w-5" />
          사용자 관리
        </h1>
      </div>

      <Tabs defaultValue="pending">
        <TabsList className="w-full">
          <TabsTrigger value="pending" className="flex-1">
            <Clock className="mr-1 h-3 w-3" />
            대기 ({pendingUsers.length})
          </TabsTrigger>
          <TabsTrigger value="all" className="flex-1">
            <UserCheck className="mr-1 h-3 w-3" />
            전체 ({allUsers.length})
          </TabsTrigger>
        </TabsList>

        <TabsContent value="pending" className="mt-4">
          <Card>
            <CardContent className="space-y-2 p-4">
              {pendingUsers.length === 0 ? (
                <p className="py-4 text-center text-sm text-muted-foreground">
                  대기 중인 사용자가 없습니다.
                </p>
              ) : (
                pendingUsers.map((user) => renderUserCard(user, true))
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="all" className="mt-4">
          <Card>
            <CardContent className="space-y-2 p-4">
              {allUsers.length === 0 ? (
                <p className="py-4 text-center text-sm text-muted-foreground">
                  등록된 사용자가 없습니다.
                </p>
              ) : (
                allUsers.map((user) => renderUserCard(user, true))
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <Dialog open={transferOpen} onOpenChange={setTransferOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>송금하기</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleTransfer} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="transfer-amount">금액 (원)</Label>
              <Input
                id="transfer-amount"
                type="number"
                min="1"
                placeholder="송금 금액"
                value={transferAmount}
                onChange={(e) => setTransferAmount(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="transfer-desc">설명</Label>
              <Input
                id="transfer-desc"
                placeholder="송금 사유 (선택)"
                value={transferDesc}
                onChange={(e) => setTransferDesc(e.target.value)}
              />
            </div>
            <Button type="submit" className="w-full" disabled={transferring}>
              {transferring ? '처리 중...' : '송금'}
            </Button>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
