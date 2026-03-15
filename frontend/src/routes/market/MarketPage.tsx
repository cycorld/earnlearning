import { useState, useEffect, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/lib/api'
import type { FreelanceJob, PaginatedData } from '@/types'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Plus, Clock, Users, Search } from 'lucide-react'
import { Input } from '@/components/ui/input'

function formatMoney(amount: number): string {
  return new Intl.NumberFormat('ko-KR').format(amount) + '원'
}

const statusLabels: Record<string, string> = {
  open: '모집 중',
  in_progress: '진행 중',
  completed: '완료',
  cancelled: '취소됨',
}

const statusOptions = [
  { value: 'all', label: '전체' },
  { value: 'open', label: '모집 중' },
  { value: 'in_progress', label: '진행 중' },
  { value: 'completed', label: '완료' },
]

export default function MarketPage() {
  const [jobs, setJobs] = useState<FreelanceJob[]>([])
  const [loading, setLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState('all')
  const [skillFilter, setSkillFilter] = useState('')
  const [page, setPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)

  const fetchJobs = useCallback(async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams({ page: String(page), limit: '20' })
      if (statusFilter !== 'all') params.set('status', statusFilter)
      if (skillFilter.trim()) params.set('skills', skillFilter.trim())
      const data = await api.get<PaginatedData<FreelanceJob>>(
        `/freelance/jobs?${params.toString()}`,
      )
      setJobs(data.data ?? [])
      setTotalPages(data.pagination?.total_pages || 1)
    } catch {
      setJobs([])
    } finally {
      setLoading(false)
    }
  }, [page, statusFilter, skillFilter])

  useEffect(() => {
    fetchJobs()
  }, [fetchJobs])

  const handleStatusChange = (value: string) => {
    setStatusFilter(value)
    setPage(1)
  }

  const handleSkillSearch = (e: React.FormEvent) => {
    e.preventDefault()
    setPage(1)
    fetchJobs()
  }

  return (
    <div className="mx-auto max-w-lg space-y-4 p-4">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-bold">프리랜서 마켓</h1>
        <Button size="sm" asChild>
          <Link to="/market/new">
            <Plus className="mr-1 h-4 w-4" />
            의뢰 등록
          </Link>
        </Button>
      </div>

      {/* Filters */}
      <div className="flex gap-2">
        <Select value={statusFilter} onValueChange={handleStatusChange}>
          <SelectTrigger className="w-28">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {statusOptions.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <form onSubmit={handleSkillSearch} className="flex flex-1 gap-1">
          <Input
            placeholder="기술 검색"
            value={skillFilter}
            onChange={(e) => setSkillFilter(e.target.value)}
            className="flex-1"
          />
          <Button type="submit" size="icon" variant="ghost">
            <Search className="h-4 w-4" />
          </Button>
        </form>
      </div>

      {loading ? (
        <div className="flex justify-center py-8">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        </div>
      ) : jobs.length === 0 ? (
        <p className="py-8 text-center text-muted-foreground">등록된 의뢰가 없습니다.</p>
      ) : (
        <>
          <div className="space-y-3">
            {jobs.map((job) => (
              <Link key={job.id} to={`/market/${job.id}`}>
                <Card className="transition-colors hover:bg-accent/30">
                  <CardContent className="p-4">
                    <div className="flex items-start justify-between">
                      <div className="min-w-0 flex-1">
                        <h3 className="font-medium">{job.title}</h3>
                        <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">
                          {job.description}
                        </p>
                      </div>
                      <Badge
                        variant={job.status === 'open' ? 'default' : 'secondary'}
                        className="ml-2 shrink-0"
                      >
                        {statusLabels[job.status] || job.status}
                      </Badge>
                    </div>
                    {job.required_skills?.length > 0 && (
                      <div className="mt-3 flex flex-wrap gap-1">
                        {job.required_skills.map((skill) => (
                          <Badge key={skill} variant="outline" className="text-xs">
                            {skill}
                          </Badge>
                        ))}
                      </div>
                    )}
                    <div className="mt-3 flex items-center gap-4 text-xs text-muted-foreground">
                      <span className="font-medium text-foreground">
                        {job.price_type === 'fixed' ? '고정' : '예산'} {formatMoney(job.budget)}
                      </span>
                      {job.deadline && (
                        <span className="flex items-center gap-1">
                          <Clock className="h-3 w-3" />
                          {new Date(job.deadline).toLocaleDateString('ko-KR')}
                        </span>
                      )}
                      <span className="flex items-center gap-1">
                        <Users className="h-3 w-3" />
                        지원 {job.application_count ?? 0}명
                      </span>
                    </div>
                  </CardContent>
                </Card>
              </Link>
            ))}
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 pt-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
              >
                이전
              </Button>
              <span className="text-sm text-muted-foreground">
                {page} / {totalPages}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
              >
                다음
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
