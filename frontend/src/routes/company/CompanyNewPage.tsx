import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import type { Company } from '@/types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { toast } from 'sonner'
import { Loader2, Upload } from 'lucide-react'

function formatMoney(amount: number): string {
  return new Intl.NumberFormat('ko-KR').format(amount) + '원'
}

const MIN_CAPITAL = 1_000_000

export default function CompanyNewPage() {
  const navigate = useNavigate()
  const [form, setForm] = useState({
    name: '',
    description: '',
    initial_capital: '',
  })
  const [logoUrl, setLogoUrl] = useState('')
  const [logoPreview, setLogoPreview] = useState('')
  const [uploading, setUploading] = useState(false)
  const [loading, setLoading] = useState(false)

  async function handleLogoUpload(file: File) {
    setUploading(true)
    try {
      const formData = new FormData()
      formData.append('file', file)
      const result = await api.post<{ url: string }>('/upload', formData)
      setLogoUrl(result.url)
      setLogoPreview(URL.createObjectURL(file))
      toast.success('로고가 업로드되었습니다.')
    } catch {
      toast.error('로고 업로드에 실패했습니다.')
    } finally {
      setUploading(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    const capital = Number(form.initial_capital)
    if (!form.name.trim()) {
      toast.error('회사 이름을 입력해주세요.')
      return
    }
    if (capital < MIN_CAPITAL) {
      toast.error(`초기 자본금은 최소 ${formatMoney(MIN_CAPITAL)} 이상이어야 합니다.`)
      return
    }

    setLoading(true)
    try {
      const company = await api.post<Company>('/companies', {
        name: form.name.trim(),
        description: form.description.trim(),
        initial_capital: capital,
        logo_url: logoUrl,
      })
      toast.success('회사가 설립되었습니다!')
      navigate(`/company/${company.id}`)
    } catch (err) {
      const message = err instanceof Error ? err.message : '회사 설립에 실패했습니다.'
      toast.error(message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="mx-auto max-w-lg p-4">
      <Card>
        <CardHeader>
          <CardTitle>회사 설립</CardTitle>
        </CardHeader>
        <form onSubmit={handleSubmit}>
          <CardContent className="space-y-5">
            <div className="space-y-2">
              <Label>회사 로고</Label>
              <div className="flex items-center gap-4">
                {logoPreview ? (
                  <img
                    src={logoPreview}
                    alt="로고 미리보기"
                    className="h-16 w-16 rounded-lg border object-cover"
                  />
                ) : (
                  <div className="flex h-16 w-16 items-center justify-center rounded-lg border border-dashed bg-muted">
                    <Upload className="h-6 w-6 text-muted-foreground" />
                  </div>
                )}
                <div>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    disabled={uploading}
                    onClick={() => document.getElementById('logo-input')?.click()}
                  >
                    {uploading ? (
                      <>
                        <Loader2 className="mr-1 h-4 w-4 animate-spin" />
                        업로드 중
                      </>
                    ) : (
                      '이미지 선택'
                    )}
                  </Button>
                  <input
                    id="logo-input"
                    type="file"
                    accept="image/*"
                    className="hidden"
                    onChange={(e) => {
                      const file = e.target.files?.[0]
                      if (file) handleLogoUpload(file)
                    }}
                  />
                  <p className="mt-1 text-xs text-muted-foreground">
                    PNG, JPG (최대 2MB)
                  </p>
                </div>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="name">회사명 *</Label>
              <Input
                id="name"
                placeholder="회사 이름을 입력하세요"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                required
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">회사 소개</Label>
              <Textarea
                id="description"
                placeholder="회사에 대해 설명해 주세요"
                rows={3}
                value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="capital">초기 자본금 (원) *</Label>
              <Input
                id="capital"
                type="number"
                placeholder="1,000,000"
                value={form.initial_capital}
                onChange={(e) => setForm({ ...form, initial_capital: e.target.value })}
                required
                min={MIN_CAPITAL}
              />
              <p className="text-xs text-muted-foreground">
                최소 {formatMoney(MIN_CAPITAL)} (100만원) | 보유 현금에서 차감됩니다.
              </p>
            </div>

            <Button type="submit" className="w-full" disabled={loading}>
              {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              회사 설립하기
            </Button>
          </CardContent>
        </form>
      </Card>
    </div>
  )
}
