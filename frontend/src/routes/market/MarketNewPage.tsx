import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { MarkdownEditor } from '@/components/MarkdownEditor'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ArrowLeft, Loader2 } from 'lucide-react'
import { Link } from 'react-router-dom'

export default function MarketNewPage() {
  const navigate = useNavigate()
  const [form, setForm] = useState({
    title: '',
    description: '',
    budget: '',
    deadline: '',
    required_skills: '',
    max_workers: '1',
    auto_approve_application: false,
    price_type: 'negotiable' as 'fixed' | 'negotiable',
  })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const job = await api.post<{ id: number }>('/freelance/jobs', {
        title: form.title,
        description: form.description,
        budget: Number(form.budget),
        deadline: form.deadline || undefined,
        required_skills: form.required_skills
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean),
        max_workers: Number(form.max_workers),
        auto_approve_application: form.auto_approve_application,
        price_type: form.price_type,
      })
      navigate(`/market/${job.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : '등록에 실패했습니다.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="mx-auto max-w-lg p-4">
      <div className="mb-4">
        <Button variant="ghost" size="sm" asChild>
          <Link to="/market">
            <ArrowLeft className="mr-1 h-4 w-4" />
            마켓으로 돌아가기
          </Link>
        </Button>
      </div>
      <Card>
        <CardHeader>
          <CardTitle>의뢰 등록</CardTitle>
        </CardHeader>
        <form onSubmit={handleSubmit}>
          <CardContent className="space-y-4">
            {error && (
              <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                {error}
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="title">제목</Label>
              <Input
                id="title"
                placeholder="의뢰 제목을 입력하세요"
                value={form.title}
                onChange={(e) => setForm({ ...form, title: e.target.value })}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">상세 설명</Label>
              <MarkdownEditor
                value={form.description}
                onChange={(v) => setForm({ ...form, description: v })}
                placeholder="의뢰 내용을 자세히 설명해 주세요 (마크다운 지원, 파일 첨부 가능)"
                rows={10}
              />
            </div>
            <div className="space-y-2">
              <Label>금액 방식</Label>
              <div className="flex gap-2">
                <Button
                  type="button"
                  variant={form.price_type === 'negotiable' ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setForm({ ...form, price_type: 'negotiable' })}
                  className="flex-1"
                >
                  협의 가능
                </Button>
                <Button
                  type="button"
                  variant={form.price_type === 'fixed' ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setForm({ ...form, price_type: 'fixed' })}
                  className="flex-1"
                >
                  금액 고정
                </Button>
              </div>
              <p className="text-xs text-muted-foreground">
                {form.price_type === 'fixed'
                  ? '지원자는 설정한 금액으로만 지원할 수 있습니다.'
                  : '지원자가 희망 금액을 자유롭게 제안할 수 있습니다.'}
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="budget">{form.price_type === 'fixed' ? '금액 (원)' : '예산 (원)'}</Label>
              <Input
                id="budget"
                type="number"
                placeholder="10000"
                value={form.budget}
                onChange={(e) => setForm({ ...form, budget: e.target.value })}
                required
                min={1}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="deadline">마감일 (선택)</Label>
              <Input
                id="deadline"
                type="date"
                value={form.deadline}
                onChange={(e) => setForm({ ...form, deadline: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="skills">필요 기술 (쉼표로 구분)</Label>
              <Input
                id="skills"
                placeholder="React, TypeScript, 디자인"
                value={form.required_skills}
                onChange={(e) => setForm({ ...form, required_skills: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="max_workers">최대 작업자 수</Label>
              <Input
                id="max_workers"
                type="number"
                placeholder="1 (기본), 0 = 무제한"
                value={form.max_workers}
                onChange={(e) => setForm({ ...form, max_workers: e.target.value })}
                min={0}
              />
              <p className="text-xs text-muted-foreground">
                0 = 무제한, 1 = 기존 방식 (1명만), 2+ = 해당 인원까지 허용
              </p>
            </div>
            <div className="flex items-center gap-2">
              <input
                id="auto_approve"
                type="checkbox"
                checked={form.auto_approve_application}
                onChange={(e) =>
                  setForm({ ...form, auto_approve_application: e.target.checked })
                }
                className="h-4 w-4 rounded border-gray-300"
              />
              <Label htmlFor="auto_approve" className="cursor-pointer">
                지원 즉시 자동 승인 (과제 모드)
              </Label>
            </div>
            <Button type="submit" className="w-full" disabled={loading}>
              {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              등록하기
            </Button>
          </CardContent>
        </form>
      </Card>
    </div>
  )
}
