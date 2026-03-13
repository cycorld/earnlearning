import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { toast } from 'sonner'
import {
  ArrowLeft,
  GraduationCap,
  Plus,
  Users,
  Wallet,
} from 'lucide-react'

interface Classroom {
  id: number
  name: string
  code: string
  initial_capital: number
  member_count?: number
  created_at: string
}

function formatMoney(amount: number): string {
  return new Intl.NumberFormat('ko-KR').format(amount) + '원'
}

export default function AdminClassroomPage() {
  const navigate = useNavigate()
  const [classrooms, setClassrooms] = useState<Classroom[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const [name, setName] = useState('')
  const [initialCapital, setInitialCapital] = useState('')

  const fetchClassrooms = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api.get<Classroom[]>('/classrooms')
      setClassrooms(data || [])
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchClassrooms()
  }, [fetchClassrooms])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim() || !initialCapital) return

    setSubmitting(true)
    try {
      await api.post('/classrooms', {
        name: name.trim(),
        initial_capital: Number(initialCapital),
      })
      toast.success('강의실이 생성되었습니다.')
      setName('')
      setInitialCapital('')
      setShowForm(false)
      fetchClassrooms()
    } catch (err: any) {
      toast.error(err.message || '강의실 생성에 실패했습니다.')
    } finally {
      setSubmitting(false)
    }
  }

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
            <GraduationCap className="h-5 w-5" />
            강의실 관리
          </h1>
        </div>
        <Button
          size="sm"
          onClick={() => setShowForm(!showForm)}
          variant={showForm ? 'outline' : 'default'}
        >
          <Plus className="mr-1 h-4 w-4" />
          {showForm ? '취소' : '새 강의실'}
        </Button>
      </div>

      {showForm && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-base">강의실 생성</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="classroom-name">강의실 이름</Label>
                <Input
                  id="classroom-name"
                  placeholder="예: 2026 봄학기 A반"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="initial-capital">초기 자본금 (원)</Label>
                <Input
                  id="initial-capital"
                  type="number"
                  min="0"
                  step="10000"
                  placeholder="학생에게 지급할 초기 자본금"
                  value={initialCapital}
                  onChange={(e) => setInitialCapital(e.target.value)}
                  required
                />
                {initialCapital && (
                  <p className="text-xs text-muted-foreground">
                    {formatMoney(Number(initialCapital))}
                  </p>
                )}
              </div>
              <Button type="submit" className="w-full" disabled={submitting}>
                {submitting ? '생성 중...' : '강의실 생성'}
              </Button>
            </form>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base">강의실 목록</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {classrooms.length === 0 ? (
            <p className="py-4 text-center text-sm text-muted-foreground">
              등록된 강의실이 없습니다.
            </p>
          ) : (
            classrooms.map((classroom) => (
              <Card key={classroom.id}>
                <CardContent className="flex items-center justify-between p-4">
                  <div>
                    <p className="font-medium">{classroom.name}</p>
                    <div className="mt-1 flex items-center gap-3 text-xs text-muted-foreground">
                      <span className="flex items-center gap-1">
                        <Users className="h-3 w-3" />
                        {classroom.member_count ?? 0}명
                      </span>
                      <span className="flex items-center gap-1">
                        <Wallet className="h-3 w-3" />
                        초기자본 {formatMoney(classroom.initial_capital)}
                      </span>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="text-xs text-muted-foreground">초대코드</p>
                    <Badge variant="secondary" className="font-mono text-sm tracking-wider">
                      {classroom.code}
                    </Badge>
                  </div>
                </CardContent>
              </Card>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  )
}
