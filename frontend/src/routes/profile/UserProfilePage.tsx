import { useState, useEffect } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import type { User, Company } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  GraduationCap,
  Building2,
  ChevronRight,
  ArrowLeft,
  MessageSquare,
  FileText,
  Briefcase,
  FileCheck,
} from 'lucide-react'
import { formatMoney } from '@/lib/utils'
import { useAuth } from '@/hooks/use-auth'
import { MarkdownContent } from '@/components/MarkdownContent'

interface UserProfile extends User {
  companies?: Company[]
}

interface ActivityPost {
  id: number
  content: string
  post_type: string
  channel: string
  like_count: number
  created_at: string
}

interface ActivityFreelanceJob {
  id: number
  title: string
  budget: number
  status: string
  created_at: string
}

interface ActivityGrantApp {
  id: number
  grant_id: number
  grant_title: string
  status: string
  proposal: string
  created_at: string
}

interface UserActivity {
  posts: ActivityPost[] | null
  freelance_jobs: ActivityFreelanceJob[] | null
  grant_apps: ActivityGrantApp[] | null
}

const grantStatusLabels: Record<string, string> = {
  pending: '심사 중',
  approved: '승인됨',
  rejected: '거절됨',
}

const jobStatusLabels: Record<string, string> = {
  open: '모집 중',
  in_progress: '진행 중',
  completed: '완료',
  cancelled: '취소',
}

export default function UserProfilePage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user: currentUser } = useAuth()
  const [profile, setProfile] = useState<UserProfile | null>(null)
  const [activity, setActivity] = useState<UserActivity | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!id) return
    setLoading(true)
    setError('')
    Promise.all([
      api.get<UserProfile>(`/users/${id}/profile`),
      api.get<UserActivity>(`/users/${id}/activity`),
    ])
      .then(([p, a]) => {
        setProfile(p)
        setActivity(a)
      })
      .catch((err: unknown) => {
        setProfile(null)
        const message =
          err instanceof Error ? err.message : '프로필을 불러올 수 없습니다.'
        setError(message)
      })
      .finally(() => setLoading(false))
  }, [id])

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  if (error || !profile) {
    return (
      <div className="mx-auto max-w-lg p-4 text-center">
        <p className="text-muted-foreground">
          {error || '프로필을 찾을 수 없습니다.'}
        </p>
        <Button variant="link" asChild className="mt-2">
          <Link to="/feed">
            <ArrowLeft className="mr-1 h-4 w-4" />
            돌아가기
          </Link>
        </Button>
      </div>
    )
  }

  const posts = activity?.posts ?? []
  const freelanceJobs = activity?.freelance_jobs ?? []
  const grantApps = activity?.grant_apps ?? []

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      {/* User info */}
      <Card>
        <CardContent className="p-6">
          <div className="flex flex-col items-center text-center">
            <Avatar className="h-20 w-20">
              <AvatarImage src={profile.avatar_url} />
              <AvatarFallback className="text-xl">
                {profile.name.charAt(0)}
              </AvatarFallback>
            </Avatar>
            <div className="mt-3">
              <div className="flex items-center justify-center gap-2">
                <h2 className="text-lg font-semibold">{profile.name}</h2>
                <Badge
                  variant={profile.role === 'admin' ? 'default' : 'secondary'}
                  className="text-xs"
                >
                  {profile.role === 'admin' ? '관리자' : '학생'}
                </Badge>
              </div>
              <div className="mt-1 flex items-center justify-center gap-3 text-xs text-muted-foreground">
                <span className="flex items-center gap-1">
                  <GraduationCap className="h-3 w-3" />
                  {profile.department}
                </span>
                <span>{profile.student_id}</span>
              </div>
            </div>
            {profile.bio && (
              <p className="mt-3 text-sm text-muted-foreground">
                {profile.bio}
              </p>
            )}
            {currentUser?.id !== profile.id && (
              <Button
                variant="outline"
                size="sm"
                className="mt-3 gap-2"
                onClick={() => navigate(`/messages/${profile.id}`)}
              >
                <MessageSquare className="h-4 w-4" />
                메시지 보내기
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Stats */}
      {(profile.wallet_balance !== undefined ||
        profile.company_count !== undefined) && (
        <div className="grid grid-cols-3 gap-3">
          {profile.total_asset_value !== undefined && (
            <Card>
              <CardContent className="p-3 text-center">
                <p className="text-xs text-muted-foreground">총 자산</p>
                <p className="text-sm font-semibold">
                  {formatMoney(profile.total_asset_value)}
                </p>
              </CardContent>
            </Card>
          )}
          {profile.wallet_balance !== undefined && (
            <Card>
              <CardContent className="p-3 text-center">
                <p className="text-xs text-muted-foreground">보유 현금</p>
                <p className="text-sm font-semibold">
                  {formatMoney(profile.wallet_balance)}
                </p>
              </CardContent>
            </Card>
          )}
          {profile.company_count !== undefined && (
            <Card>
              <CardContent className="p-3 text-center">
                <p className="text-xs text-muted-foreground">회사</p>
                <p className="text-sm font-semibold">
                  {profile.company_count}개
                </p>
              </CardContent>
            </Card>
          )}
        </div>
      )}

      {/* Companies */}
      {profile.companies && profile.companies.length > 0 && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-base">
              <Building2 className="h-4 w-4" />
              운영 회사
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {profile.companies.map((company) => (
              <Link
                key={company.id}
                to={`/company/${company.id}`}
                className="flex items-center gap-3 rounded-md p-2 transition-colors hover:bg-accent"
              >
                <Avatar className="h-10 w-10">
                  <AvatarImage src={company.logo_url} />
                  <AvatarFallback>
                    {company.name.charAt(0)}
                  </AvatarFallback>
                </Avatar>
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium">{company.name}</p>
                  <p className="truncate text-xs text-muted-foreground">
                    {company.description}
                  </p>
                </div>
                <div className="text-right">
                  <p className="text-xs text-muted-foreground">기업가치</p>
                  <p className="text-xs font-medium">
                    {formatMoney(company.valuation)}
                  </p>
                </div>
                <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
              </Link>
            ))}
          </CardContent>
        </Card>
      )}

      {/* Posts */}
      {posts.length > 0 && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-base">
              <FileText className="h-4 w-4" />
              게시글 ({posts.length})
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {posts.map((post) => (
              <div key={post.id} className="rounded-md border p-3">
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  {post.channel && <Badge variant="secondary" className="text-[10px]">{post.channel}</Badge>}
                  <span>{new Date(post.created_at).toLocaleDateString('ko-KR')}</span>
                  {post.like_count > 0 && <span>♥ {post.like_count}</span>}
                </div>
                <MarkdownContent
                  content={post.content}
                  maxLines={3}
                  className="mt-1 text-sm"
                />
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      {/* Freelance Jobs */}
      {freelanceJobs.length > 0 && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-base">
              <Briefcase className="h-4 w-4" />
              마켓 포스팅 ({freelanceJobs.length})
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {freelanceJobs.map((job) => (
              <Link
                key={job.id}
                to={`/market/${job.id}`}
                className="flex items-center justify-between rounded-md border p-3 transition-colors hover:bg-accent"
              >
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium">{job.title}</p>
                  <p className="text-xs text-muted-foreground">
                    {formatMoney(job.budget)} · {new Date(job.created_at).toLocaleDateString('ko-KR')}
                  </p>
                </div>
                <Badge
                  variant={job.status === 'open' ? 'default' : 'secondary'}
                  className="text-xs"
                >
                  {jobStatusLabels[job.status] || job.status}
                </Badge>
              </Link>
            ))}
          </CardContent>
        </Card>
      )}

      {/* Grant Applications */}
      {grantApps.length > 0 && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-base">
              <FileCheck className="h-4 w-4" />
              정부과제 지원 ({grantApps.length})
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {grantApps.map((app) => (
              <Link
                key={app.id}
                to={`/grant/${app.grant_id}`}
                className="block rounded-md border p-3 transition-colors hover:bg-accent"
              >
                <div className="flex items-center justify-between">
                  <p className="text-sm font-medium">{app.grant_title}</p>
                  <Badge
                    variant={app.status === 'approved' ? 'default' : 'secondary'}
                    className="text-xs"
                  >
                    {grantStatusLabels[app.status] || app.status}
                  </Badge>
                </div>
                {app.proposal && (
                  <MarkdownContent
                    content={app.proposal}
                    maxLines={2}
                    className="mt-1 text-xs text-muted-foreground"
                  />
                )}
              </Link>
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
