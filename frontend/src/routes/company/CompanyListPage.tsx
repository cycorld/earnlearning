import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/lib/api'
import type { Company } from '@/types'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Plus } from 'lucide-react'

function formatMoney(amount: number): string {
  return new Intl.NumberFormat('ko-KR').format(amount) + '원'
}

export default function CompanyListPage() {
  const [companies, setCompanies] = useState<Company[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api
      .get<Company[]>('/companies/mine')
      .then((data) => setCompanies(data))
      .catch(() => setCompanies([]))
      .finally(() => setLoading(false))
  }, [])

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
        <h1 className="text-lg font-bold">내 회사</h1>
        <Button size="sm" asChild>
          <Link to="/company/new">
            <Plus className="mr-1 h-4 w-4" />
            회사 설립
          </Link>
        </Button>
      </div>

      {companies.length === 0 ? (
        <div className="py-8 text-center">
          <p className="text-muted-foreground">아직 설립한 회사가 없습니다.</p>
          <Button variant="link" asChild className="mt-2">
            <Link to="/company/new">첫 회사를 설립해보세요</Link>
          </Button>
        </div>
      ) : (
        <div className="space-y-3">
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
                      <span className="font-medium">{company.name}</span>
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
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      기업가치 {formatMoney(company.valuation)}
                    </p>
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
