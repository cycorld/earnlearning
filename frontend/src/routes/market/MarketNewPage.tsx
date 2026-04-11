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
      <div className="sticky top-14 z-40 -mx-4 mb-4 bg-background px-4 py-1">
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
              <Label htmlFor="budget">예산 (원)</Label>
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
