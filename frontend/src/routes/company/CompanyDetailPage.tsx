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
import { MarkdownEditor } from '@/components/MarkdownEditor'
import { MarkdownContent } from '@/components/MarkdownContent'
import { Separator } from '@/components/ui/separator'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { toast } from 'sonner'
import { CreditCard, ExternalLink, Pencil, Loader2, Plus, Upload, Wallet } from 'lucide-react'
import { formatMoney, displayName } from '@/lib/utils'
import { DisclosureSection } from './DisclosureSection'
import { ProposalSection } from './ProposalSection'
import { InvestmentRoundSection } from './InvestmentRoundSection'

export default function CompanyDetailPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const [company, setCompany] = useState<Company | null>(null)
  const [loading, setLoading] = useState(true)

  const [editOpen, setEditOpen] = useState(false)
  const [editForm, setEditForm] = useState({ name: '', description: '', logo_url: '', service_url: '' })
  const [editLoading, setEditLoading] = useState(false)
  const [logoUploading, setLogoUploading] = useState(false)

  const [cardLoading, setCardLoading] = useState(false)

  const isOwner = user && company?.owner?.id === user.id
  const isShareholder =
    !!user &&
    !!company?.shareholders?.some(
      (sh) => sh.user_id === user.id && sh.shares > 0,
    )

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
    setEditForm({
      name: company.name,
      description: company.description,
      logo_url: company.logo_url || '',
      service_url: company.service_url || '',
    })
    setEditOpen(true)
  }

  async function handleLogoUpload(file: File) {
    setLogoUploading(true)
    try {
      const formData = new FormData()
      formData.append('file', file)
      const result = await api.post<{ url: string }>('/upload', formData)
      setEditForm((prev) => ({ ...prev, logo_url: result.url }))
      toast.success('로고가 업로드되었습니다.')
    } catch {
      toast.error('로고 업로드에 실패했습니다.')
    } finally {
      setLogoUploading(false)
    }
  }

  async function handleEdit(e: React.FormEvent) {
    e.preventDefault()
    if (!editForm.name.trim()) {
      toast.error('회사 이름을 입력해주세요.')
      return
    }

    setEditLoading(true)
    try {
      await api.put(`/companies/${id}`, {
        name: editForm.name.trim(),
        description: editForm.description.trim(),
        logo_url: editForm.logo_url,
        service_url: editForm.service_url.trim(),
      })
      // 백엔드 PUT 응답이 {message: ...} 라 객체 갱신 못함 → 다시 조회
      await fetchCompany()
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
    <div className="mx-auto max-w-lg space-y-5 p-4">
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
              {company.status === 'dissolved' ? (
                <Badge variant="destructive">청산됨</Badge>
              ) : company.listed ? (
                <Badge>상장</Badge>
              ) : (
                <Badge variant="secondary">비상장</Badge>
              )}
            </div>
            <p className="text-sm text-muted-foreground">
              대표: {displayName(company.owner) || '-'}
            </p>
            {company.service_url && (
              <a
                href={company.service_url}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
              >
                <ExternalLink className="h-3 w-3" />
                {company.service_url.replace(/^https?:\/\//, '')}
              </a>
            )}
          </div>
          {isOwner && (
            <Button variant="ghost" size="icon" onClick={openEditDialog}>
              <Pencil className="h-4 w-4" />
            </Button>
          )}
        </CardContent>
      </Card>

      {/* Key metrics */}
      <div className="grid grid-cols-2 gap-4">
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
            <MarkdownContent content={company.description} maxLines={8} className="text-sm" />
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

      {/* Disclosures */}
      <DisclosureSection companyId={Number(id)} isOwner={!!isOwner} />

      {/* Shareholder proposals (주주총회) */}
      <ProposalSection
        companyId={Number(id)}
        isShareholder={isShareholder}
        onCompanyChanged={fetchCompany}
      />

      {/* Investment rounds (투자 유치) */}
      <InvestmentRoundSection
        companyId={Number(id)}
        companyValuation={company.valuation}
        isOwner={!!isOwner}
        onRoundCreated={fetchCompany}
      />

      <Separator />

      {/* Actions */}
      <div className="space-y-2">
        <Button variant="outline" className="w-full border-entity/25 text-entity hover:bg-entity/10 hover:text-entity" asChild>
          <Link to={`/company/${id}/wallet`}>
            <Wallet className="mr-2 h-4 w-4" />
            법인 계좌 관리
            {company.wallet_balance !== undefined && (
              <span className="ml-auto text-xs font-semibold">
                {formatMoney(company.wallet_balance)}
              </span>
            )}
          </Link>
        </Button>
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
            {/* 로고 업로드 */}
            <div className="space-y-2">
              <Label>회사 로고</Label>
              <div className="flex items-center gap-4">
                {editForm.logo_url ? (
                  <img
                    src={editForm.logo_url}
                    alt="로고 미리보기"
                    className="h-16 w-16 rounded-lg border object-cover"
                  />
                ) : (
                  <div className="flex h-16 w-16 items-center justify-center rounded-lg border border-dashed bg-muted">
                    <Upload className="h-6 w-6 text-muted-foreground" />
                  </div>
                )}
                <div className="flex flex-col gap-1">
                  <div className="flex gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      disabled={logoUploading}
                      onClick={() => document.getElementById('edit-logo-input')?.click()}
                    >
                      {logoUploading ? (
                        <>
                          <Loader2 className="mr-1 h-4 w-4 animate-spin" />
                          업로드 중
                        </>
                      ) : (
                        editForm.logo_url ? '로고 변경' : '이미지 선택'
                      )}
                    </Button>
                    {editForm.logo_url && (
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => setEditForm({ ...editForm, logo_url: '' })}
                      >
                        제거
                      </Button>
                    )}
                  </div>
                  <input
                    id="edit-logo-input"
                    type="file"
                    accept="image/*"
                    className="hidden"
                    onChange={(e) => {
                      const file = e.target.files?.[0]
                      if (file) handleLogoUpload(file)
                      e.target.value = ''
                    }}
                  />
                  <p className="text-xs text-muted-foreground">PNG, JPG (최대 2MB)</p>
                </div>
              </div>
            </div>

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
              <Label htmlFor="edit-service-url">서비스 URL</Label>
              <Input
                id="edit-service-url"
                value={editForm.service_url}
                onChange={(e) => setEditForm({ ...editForm, service_url: e.target.value })}
                placeholder="https://my-app.example.com"
                type="url"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-desc">회사 소개</Label>
              <MarkdownEditor
                value={editForm.description}
                onChange={(v) => setEditForm({ ...editForm, description: v })}
                placeholder="회사에 대해 설명해 주세요"
                rows={8}
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => setEditOpen(false)}>
                취소
              </Button>
              <Button type="submit" disabled={editLoading || logoUploading}>
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
