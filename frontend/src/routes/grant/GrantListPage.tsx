import { useState, useEffect, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/lib/api'
import type { Grant, PaginatedData } from '@/types'
import { useAuth } from '@/hooks/use-auth'
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
import { Plus, Users, CheckCircle } from 'lucide-react'
import { formatMoney } from '@/lib/utils'

const statusLabels: Record<string, string> = {
  open: '모집 중',
  closed: '종료',
}

const statusOptions = [
  { value: 'all', label: '전체' },
  { value: 'open', label: '모집 중' },
  { value: 'closed', label: '종료' },
]

export default function GrantListPage() {
  const { user } = useAuth()
  const [grants, setGrants] = useState<Grant[]>([])
  const [loading, setLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState('all')
  const [page, setPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)

  const fetchGrants = useCallback(async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams({ page: String(page), limit: '20' })
      if (statusFilter !== 'all') params.set('status', statusFilter)
      const data = await api.get<PaginatedData<Grant>>(`/grants?${params.toString()}`)
      setGrants(data.data ?? [])
      setTotalPages(data.pagination?.total_pages || 1)
    } catch {
      setGrants([])
    } finally {
      setLoading(false)
    }
  }, [page, statusFilter])

  useEffect(() => {
    fetchGrants()
  }, [fetchGrants])

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-bold">정부과제</h1>
        {user?.role === 'admin' && (
          <Button size="sm" asChild>
            <Link to="/grant/new">
              <Plus className="mr-1 h-4 w-4" />
              과제 등록
            </Link>
          </Button>
        )}
      </div>

      <div className="flex gap-2">
        <Select value={statusFilter} onValueChange={(v) => { setStatusFilter(v); setPage(1) }}>
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
      </div>

      {loading ? (
        <div className="flex justify-center py-8">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        </div>
      ) : grants.length === 0 ? (
        <p className="py-8 text-center text-muted-foreground">등록된 과제가 없습니다.</p>
      ) : (
        <>
          <div className="space-y-3">
            {grants.map((grant) => (
              <Link key={grant.id} to={`/grant/${grant.id}`}>
                <Card className="transition-colors hover:bg-accent/30">
                  <CardContent className="p-4">
                    <div className="flex items-start justify-between">
                      <div className="min-w-0 flex-1">
                        <h3 className="font-medium">{grant.title}</h3>
                        <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">
                          {grant.description}
                        </p>
                      </div>
                      <Badge
                        variant={grant.status === 'open' ? 'default' : 'secondary'}
                        className="ml-2 shrink-0"
                      >
                        {statusLabels[grant.status] || grant.status}
                      </Badge>
                    </div>
                    <div className="mt-3 flex items-center gap-4 text-xs text-muted-foreground">
                      <span className="font-medium text-foreground">
                        보상 {formatMoney(grant.reward)}
                      </span>
                      {grant.max_applicants > 0 && (
                        <span>정원 {grant.max_applicants}명</span>
                      )}
                      <span className="flex items-center gap-1">
                        <Users className="h-3 w-3" />
                        지원 {grant.application_count ?? 0}명
                      </span>
                      {(grant.approved_count ?? 0) > 0 && (
                        <span className="flex items-center gap-1 text-green-600">
                          <CheckCircle className="h-3 w-3" />
                          승인 {grant.approved_count}명
                        </span>
                      )}
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
