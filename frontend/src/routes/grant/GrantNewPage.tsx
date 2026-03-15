import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { MarkdownEditor } from '@/components/MarkdownEditor'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ArrowLeft, Loader2 } from 'lucide-react'

export default function GrantNewPage() {
  const navigate = useNavigate()
  const [form, setForm] = useState({
    title: '',
    description: '',
    reward: '',
    max_applicants: '0',
  })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const grant = await api.post<{ id: number }>('/admin/grants', {
        title: form.title,
        description: form.description,
        reward: Number(form.reward),
        max_applicants: Number(form.max_applicants),
      })
      navigate(`/grant/${grant.id}`)
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
          <Link to="/grant">
            <ArrowLeft className="mr-1 h-4 w-4" />
            과제 목록으로
          </Link>
        </Button>
      </div>
      <Card>
        <CardHeader>
          <CardTitle>정부과제 등록</CardTitle>
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
                placeholder="과제 제목을 입력하세요"
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
                placeholder="과제 내용을 자세히 설명해 주세요 (마크다운 지원)"
                rows={8}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="reward">보상 금액 (원)</Label>
              <Input
                id="reward"
                type="number"
                placeholder="5000"
                value={form.reward}
                onChange={(e) => setForm({ ...form, reward: e.target.value })}
                required
                min={1}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="max_applicants">최대 지원자 수 (0 = 무제한)</Label>
              <Input
                id="max_applicants"
                type="number"
                placeholder="0"
                value={form.max_applicants}
                onChange={(e) => setForm({ ...form, max_applicants: e.target.value })}
                min={0}
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
