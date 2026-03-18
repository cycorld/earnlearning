import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { api } from '@/lib/api'
import { formatMoney } from '@/lib/utils'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  ArrowLeft,
  Users,
  Wallet,
  Building2,
  TrendingUp,
  Search,
  ArrowUpDown,
  FileText,
  Banknote,
} from 'lucide-react'

interface MemberDashboard {
  user_id: number
  name: string
  email: string
  student_id: string
  department: string
  avatar_url: string
  status: string
  joined_at: string
  balance: number
  total_asset: number
  company_count: number
  loan_count: number
  total_debt: number
  post_count: number
  company_names: string
}

interface Classroom {
  id: number
  name: string
  code: string
  initial_capital: number
  created_at: string
}

type SortKey = 'name' | 'total_asset' | 'balance' | 'company_count' | 'post_count' | 'total_debt'

export default function AdminClassroomDetailPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [classroom, setClassroom] = useState<Classroom | null>(null)
  const [members, setMembers] = useState<MemberDashboard[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [sortKey, setSortKey] = useState<SortKey>('total_asset')
  const [sortAsc, setSortAsc] = useState(false)

  const fetchDashboard = useCallback(async () => {
    try {
      const data = await api.get<{ classroom: Classroom; members: MemberDashboard[] }>(
        `/admin/classrooms/${id}/dashboard`,
      )
      setClassroom(data.classroom)
      setMembers(data.members ?? [])
    } catch {
      setClassroom(null)
      setMembers([])
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    fetchDashboard()
  }, [fetchDashboard])

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortAsc(!sortAsc)
    } else {
      setSortKey(key)
      setSortAsc(key === 'name')
    }
  }

  const filtered = members
    .filter((m) => {
      if (!search) return true
      const q = search.toLowerCase()
      return (
        m.name.toLowerCase().includes(q) ||
        m.student_id.toLowerCase().includes(q) ||
        m.department.toLowerCase().includes(q)
      )
    })
    .sort((a, b) => {
      const mul = sortAsc ? 1 : -1
      if (sortKey === 'name') return mul * a.name.localeCompare(b.name)
      return mul * ((a[sortKey] as number) - (b[sortKey] as number))
    })

  // Summary stats
  const totalStudents = members.length
  const avgAsset = totalStudents > 0 ? Math.round(members.reduce((s, m) => s + m.total_asset, 0) / totalStudents) : 0
  const totalCompanies = members.reduce((s, m) => s + m.company_count, 0)
  const totalDebt = members.reduce((s, m) => s + m.total_debt, 0)

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  if (!classroom) {
    return <div className="p-4 text-center text-muted-foreground">강의실을 찾을 수 없습니다.</div>
  }

  return (
    <div className="mx-auto max-w-2xl space-y-4 p-4">
      {/* Header */}
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="icon" onClick={() => navigate('/admin/classroom')}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="flex-1">
          <h1 className="text-xl font-bold">{classroom.name}</h1>
          <p className="text-sm text-muted-foreground">
            초대코드 <Badge variant="secondary" className="ml-1 font-mono">{classroom.code}</Badge>
          </p>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Card>
          <CardContent className="p-3 text-center">
            <Users className="mx-auto mb-1 h-5 w-5 text-muted-foreground" />
            <p className="text-lg font-bold">{totalStudents}</p>
            <p className="text-xs text-muted-foreground">학생 수</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-3 text-center">
            <TrendingUp className="mx-auto mb-1 h-5 w-5 text-muted-foreground" />
            <p className="text-lg font-bold">{formatMoney(avgAsset)}</p>
            <p className="text-xs text-muted-foreground">평균 자산</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-3 text-center">
            <Building2 className="mx-auto mb-1 h-5 w-5 text-muted-foreground" />
            <p className="text-lg font-bold">{totalCompanies}</p>
            <p className="text-xs text-muted-foreground">총 회사 수</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-3 text-center">
            <Banknote className="mx-auto mb-1 h-5 w-5 text-muted-foreground" />
            <p className="text-lg font-bold">{formatMoney(totalDebt)}</p>
            <p className="text-xs text-muted-foreground">총 부채</p>
          </CardContent>
        </Card>
      </div>

      {/* Search + Sort */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="이름, 학번, 학과 검색"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>

      {/* Sort buttons */}
      <div className="flex flex-wrap gap-1">
        {([
          ['total_asset', '총 자산'],
          ['balance', '현금'],
          ['company_count', '회사'],
          ['post_count', '게시글'],
          ['total_debt', '부채'],
          ['name', '이름'],
        ] as [SortKey, string][]).map(([key, label]) => (
          <Button
            key={key}
            variant={sortKey === key ? 'default' : 'outline'}
            size="sm"
            className="h-7 gap-1 text-xs"
            onClick={() => handleSort(key)}
          >
            {label}
            {sortKey === key && <ArrowUpDown className="h-3 w-3" />}
          </Button>
        ))}
      </div>

      {/* Student List */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base">
            학생 현황 ({filtered.length}명)
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {filtered.length === 0 ? (
            <p className="py-4 text-center text-sm text-muted-foreground">
              {search ? '검색 결과가 없습니다.' : '등록된 학생이 없습니다.'}
            </p>
          ) : (
            filtered.map((m, idx) => (
              <Link key={m.user_id} to={`/profile/${m.user_id}`}>
                <div className="flex items-center gap-3 rounded-lg border p-3 transition-colors hover:bg-accent/30">
                  {/* Rank */}
                  <span className="w-6 text-center text-sm font-bold text-muted-foreground">
                    {idx + 1}
                  </span>

                  {/* Avatar */}
                  <Avatar className="h-9 w-9 shrink-0">
                    <AvatarImage src={m.avatar_url} />
                    <AvatarFallback>{m.name.charAt(0)}</AvatarFallback>
                  </Avatar>

                  {/* Info */}
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{m.name}</span>
                      {m.student_id && (
                        <span className="text-xs text-muted-foreground">{m.student_id}</span>
                      )}
                    </div>
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-muted-foreground">
                      <span className="flex items-center gap-0.5">
                        <TrendingUp className="h-3 w-3" />
                        {formatMoney(m.total_asset)}
                      </span>
                      <span className="flex items-center gap-0.5">
                        <Wallet className="h-3 w-3" />
                        {formatMoney(m.balance)}
                      </span>
                      {m.company_count > 0 && (
                        <span className="flex items-center gap-0.5" title={m.company_names}>
                          <Building2 className="h-3 w-3" />
                          {m.company_count}개
                          {m.company_names && (
                            <span className="max-w-[120px] truncate text-muted-foreground">
                              ({m.company_names})
                            </span>
                          )}
                        </span>
                      )}
                      {m.total_debt > 0 && (
                        <span className="flex items-center gap-0.5 text-red-500">
                          <Banknote className="h-3 w-3" />
                          -{formatMoney(m.total_debt)}
                        </span>
                      )}
                      <span className="flex items-center gap-0.5">
                        <FileText className="h-3 w-3" />
                        글 {m.post_count}
                      </span>
                    </div>
                  </div>
                </div>
              </Link>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  )
}
