import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import type { Company } from '@/types'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Plus, Building2 } from 'lucide-react'
import { formatMoney } from '@/lib/utils'
import { Spinner } from '@/components/ui/spinner'

export default function CompanyListPage() {
  const { user } = useAuth()
  const [myCompanies, setMyCompanies] = useState<Company[]>([])
  const [allCompanies, setAllCompanies] = useState<Company[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      api.get<Company[]>('/companies/mine').catch(() => [] as Company[]),
      api.get<Company[]>('/companies').catch(() => [] as Company[]),
    ])
      .then(([mine, all]) => {
        setMyCompanies(mine ?? [])
        setAllCompanies(all ?? [])
      })
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Spinner />
      </div>
    )
  }

  // 전체 목록에서 본인 회사 제외 → "다른 학생 회사" 만 별도 탭 표시
  const otherCompanies = user
    ? allCompanies.filter((c) => (c.owner_id ?? c.owner?.id) !== user.id)
    : allCompanies

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-bold">회사</h1>
        <Button size="sm" asChild>
          <Link to="/company/new">
            <Plus className="mr-1 h-4 w-4" />
            회사 설립
          </Link>
        </Button>
      </div>

      <Tabs defaultValue="mine" className="w-full">
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="mine">내 회사 ({myCompanies.length})</TabsTrigger>
          <TabsTrigger value="all">전체 기업 ({otherCompanies.length})</TabsTrigger>
        </TabsList>

        <TabsContent value="mine" className="mt-3">
          {myCompanies.length === 0 ? (
            <div className="py-8 text-center">
              <p className="text-muted-foreground">아직 설립한 회사가 없습니다.</p>
              <Button variant="link" asChild className="mt-2">
                <Link to="/company/new">첫 회사를 설립해보세요</Link>
              </Button>
            </div>
          ) : (
            <CompanyGrid companies={myCompanies} />
          )}
        </TabsContent>

        <TabsContent value="all" className="mt-3">
          {otherCompanies.length === 0 ? (
            <div className="py-8 text-center text-sm text-muted-foreground">
              <Building2 className="mx-auto mb-2 h-8 w-8 opacity-50" />
              아직 다른 학생이 만든 회사가 없어요
            </div>
          ) : (
            <CompanyGrid companies={otherCompanies} showOwner />
          )}
        </TabsContent>
      </Tabs>
    </div>
  )
}

function CompanyGrid({
  companies,
  showOwner = false,
}: {
  companies: Company[]
  showOwner?: boolean
}) {
  return (
    <div className="space-y-4">
      {companies.map((company) => (
        <Link key={company.id} to={`/company/${company.id}`}>
          <Card className="transition-colors hover:bg-accent/30">
            <CardContent className="flex items-center gap-4 p-4">
              <Avatar className="h-12 w-12">
                <AvatarImage src={company.logo_url} />
                <AvatarFallback className="bg-primary/10 text-primary">
                  {company.name.charAt(0)}
                </AvatarFallback>
              </Avatar>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="truncate font-medium">{company.name}</span>
                  {company.listed ? (
                    <Badge variant="default" className="text-xs">
                      상장
                    </Badge>
                  ) : (
                    <Badge variant="secondary" className="text-xs">
                      비상장
                    </Badge>
                  )}
                </div>
                <p className="mt-0.5 truncate text-xs text-muted-foreground">
                  {showOwner && company.owner_name && (
                    <span className="mr-1">대표 {company.owner_name} ·</span>
                  )}
                  기업가치 {formatMoney(company.valuation)}
                </p>
              </div>
            </CardContent>
          </Card>
        </Link>
      ))}
    </div>
  )
}
