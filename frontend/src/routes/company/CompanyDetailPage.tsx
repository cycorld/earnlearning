import { useState, useEffect, useCallback } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import type { Company } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Separator } from '@/components/ui/separator'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { toast } from 'sonner'
import { CreditCard, Pencil, Loader2, Plus } from 'lucide-react'

function formatMoney(amount: number): string {
  return new Intl.NumberFormat('ko-KR').format(amount) + '원'
}

export default function CompanyDetailPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const [company, setCompany] = useState<Company | null>(null)
  const [loading, setLoading] = useState(true)

  const [editOpen, setEditOpen] = useState(false)
  const [editForm, setEditForm] = useState({ name: '', description: '' })
  const [editLoading, setEditLoading] = useState(false)

  const [cardLoading, setCardLoading] = useState(false)

  const isOwner = user && company?.owner?.id === user.id

  const fetchCompany = useCallback(async () => {
    try {
      const data = await api.get<Company>(`/companies/${id}`)
      setCompany(data)
    } catch {
      setCompany(null)
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    fetchCompany()
  }, [fetchCompany])

  function openEditDialog() {
    if (!company) return
    setEditForm({ name: company.name, description: company.description })
    setEditOpen(true)
  }

  async function handleEdit(e: React.FormEvent) {
    e.preventDefault()
    if (!editForm.name.trim()) {
      toast.error('회사 이름을 입력해주세요.')
      return
    }

    setEditLoading(true)
    try {
      const updated = await api.put<Company>(`/companies/${id}`, {
        name: editForm.name.trim(),
        description: editForm.description.trim(),
      })
      setCompany(updated)
      setEditOpen(false)
      toast.success('회사 정보가 수정되었습니다.')
    } catch (err) {
      const message = err instanceof Error ? err.message : '수정에 실패했습니다.'
      toast.error(message)
    } finally {
      setEditLoading(false)
    }
  }

  async function handleCreateBusinessCard() {
    setCardLoading(true)
    try {
      await api.post(`/companies/${id}/business-card`)
      toast.success('명함이 생성되었습니다.')
      await fetchCompany()
    } catch (err) {
      const message = err instanceof Error ? err.message : '명함 생성에 실패했습니다.'
      toast.error(message)
    } finally {
      setCardLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  if (!company) {
    return <div className="p-4 text-center text-muted-foreground">회사를 찾을 수 없습니다.</div>
  }

  return (
    <div className="mx-auto max-w-lg space-y-4 p-4">
      {/* Company header */}
      <Card>
        <CardContent className="flex items-center gap-4 p-6">
          <Avatar className="h-16 w-16">
            <AvatarImage src={company.logo_url} />
            <AvatarFallback className="bg-primary/10 text-lg text-primary">
              {company.name.charAt(0)}
            </AvatarFallback>
          </Avatar>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <h1 className="text-xl font-bold">{company.name}</h1>
              {company.listed ? (
                <Badge>상장</Badge>
              ) : (
                <Badge variant="secondary">비상장</Badge>
              )}
            </div>
            <p className="text-sm text-muted-foreground">
              대표: {company.owner?.name || '-'}
            </p>
          </div>
          {isOwner && (
            <Button variant="ghost" size="icon" onClick={openEditDialog}>
              <Pencil className="h-4 w-4" />
            </Button>
          )}
        </CardContent>
      </Card>

      {/* Key metrics */}
      <div className="grid grid-cols-2 gap-3">
        <Card>
          <CardContent className="p-4 text-center">
            <p className="text-xs text-muted-foreground">기업가치</p>
            <p className="text-sm font-bold">{formatMoney(company.valuation)}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 text-center">
            <p className="text-xs text-muted-foreground">총 자본금</p>
            <p className="text-sm font-bold">{formatMoney(company.total_capital)}</p>
          </CardContent>
        </Card>
      </div>

      {/* My shares */}
      {company.my_shares !== undefined && company.my_shares > 0 && (
        <Card>
          <CardContent className="flex items-center justify-between p-4">
            <div>
              <p className="text-xs text-muted-foreground">내 지분</p>
              <p className="text-sm font-bold">
                {company.my_shares.toLocaleString('ko-KR')}주
              </p>
            </div>
            <div className="text-right">
              <p className="text-xs text-muted-foreground">지분율</p>
              <p className="text-sm font-bold text-primary">
                {company.my_percentage?.toFixed(1)}%
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Description */}
      {company.description && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">회사 소개</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="whitespace-pre-wrap text-sm">{company.description}</p>
          </CardContent>
        </Card>
      )}

      {/* Shareholders */}
      {company.shareholders && company.shareholders.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">주주 현황</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {company.shareholders.map((sh) => (
              <div key={sh.user_id} className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2">
                  <span className="font-medium">{sh.name}</span>
                  <Badge variant="outline" className="text-xs">
                    {sh.acquisition_type}
                  </Badge>
                </div>
                <span className="text-muted-foreground">
                  {sh.shares.toLocaleString('ko-KR')}주 ({sh.percentage.toFixed(1)}%)
                </span>
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      <Separator />

      {/* Actions */}
      <div className="space-y-2">
        {company.business_card ? (
          <Button variant="outline" className="w-full" asChild>
            <Link to={`/company/${id}/card`}>
              <CreditCard className="mr-2 h-4 w-4" />
              명함 보기
            </Link>
          </Button>
        ) : (
          isOwner && (
            <Button
              variant="outline"
              className="w-full"
              disabled={cardLoading}
              onClick={handleCreateBusinessCard}
            >
              {cardLoading ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Plus className="mr-2 h-4 w-4" />
              )}
              명함 생성
            </Button>
          )
        )}
      </div>

      {/* Edit dialog */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>회사 정보 수정</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleEdit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="edit-name">회사명</Label>
              <Input
                id="edit-name"
                value={editForm.name}
                onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-desc">회사 소개</Label>
              <Textarea
                id="edit-desc"
                value={editForm.description}
                onChange={(e) => setEditForm({ ...editForm, description: e.target.value })}
                rows={4}
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => setEditOpen(false)}>
                취소
              </Button>
              <Button type="submit" disabled={editLoading}>
                {editLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                저장
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
