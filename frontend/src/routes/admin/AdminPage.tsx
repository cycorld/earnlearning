import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  Users,
  GraduationCap,
  Landmark,
  BarChart3,
  ArrowRight,
  Clock,
  ShieldCheck,
} from 'lucide-react'
import { Badge } from '@/components/ui/badge'

interface AdminStats {
  pending_users: number
  active_loans: number
}

export default function AdminPage() {
  const [stats, setStats] = useState<AdminStats>({
    pending_users: 0,
    active_loans: 0,
  })
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const [pendingUsers, loans] = await Promise.all([
          api
            .get<any[]>('/admin/users/pending')
            .then((d) => d?.length ?? 0)
            .catch(() => 0),
          api
            .get<any[]>('/admin/loans')
            .then((d) => d?.filter((l: any) => l.status === 'active').length ?? 0)
            .catch(() => 0),
        ])
        setStats({ pending_users: pendingUsers, active_loans: loans })
      } catch {
        // ignore
      } finally {
        setLoading(false)
      }
    }
    fetchStats()
  }, [])

  const menuItems = [
    {
      title: '사용자 관리',
      description: '가입 승인 및 사용자 관리',
      icon: Users,
      href: '/admin/users',
      color: 'bg-blue-100 text-blue-600',
      badge: stats.pending_users > 0 ? `${stats.pending_users}명 대기` : null,
    },
    {
      title: '강의실 관리',
      description: '강의실 생성 및 관리',
      icon: GraduationCap,
      href: '/admin/classroom',
      color: 'bg-green-100 text-green-600',
      badge: null,
    },
    {
      title: '대출 관리',
      description: '대출 심사 및 이자 처리',
      icon: Landmark,
      href: '/admin/loans',
      color: 'bg-orange-100 text-orange-600',
      badge: stats.active_loans > 0 ? `${stats.active_loans}건 진행` : null,
    },
    {
      title: 'KPI 관리',
      description: 'KPI 규칙 및 배당 관리',
      icon: BarChart3,
      href: '/admin/kpi',
      color: 'bg-purple-100 text-purple-600',
      badge: null,
    },
  ]

  return (
    <div className="mx-auto max-w-lg space-y-4 p-4">
      <div className="flex items-center gap-2">
        <ShieldCheck className="h-5 w-5 text-primary" />
        <h1 className="text-xl font-bold">관리자</h1>
      </div>

      {loading ? (
        <div className="flex justify-center py-8">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        </div>
      ) : (
        <div className="grid gap-3">
          {menuItems.map((item) => (
            <Link key={item.href} to={item.href}>
              <Card className="transition-colors hover:bg-accent">
                <CardContent className="flex items-center justify-between p-4">
                  <div className="flex items-center gap-3">
                    <div
                      className={`flex h-10 w-10 items-center justify-center rounded-full ${item.color}`}
                    >
                      <item.icon className="h-5 w-5" />
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <p className="font-medium">{item.title}</p>
                        {item.badge && (
                          <Badge variant="secondary" className="text-xs">
                            {item.badge}
                          </Badge>
                        )}
                      </div>
                      <p className="text-xs text-muted-foreground">
                        {item.description}
                      </p>
                    </div>
                  </div>
                  <ArrowRight className="h-4 w-4 text-muted-foreground" />
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
