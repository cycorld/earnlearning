import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import type { Company } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { toast } from 'sonner'
import {
  ArrowLeft,
  BarChart3,
  Target,
  DollarSign,
  Gift,
} from 'lucide-react'
import { formatMoney } from '@/lib/utils'
import { Spinner } from '@/components/ui/spinner'

export default function AdminKpiPage() {
  const navigate = useNavigate()
  const [companies, setCompanies] = useState<Company[]>([])
  const [loading, setLoading] = useState(true)

  // KPI Rule form
  const [kpiCompanyId, setKpiCompanyId] = useState('')
  const [metricName, setMetricName] = useState('')
  const [targetValue, setTargetValue] = useState('')
  const [rewardPerUnit, setRewardPerUnit] = useState('')
  const [kpiSubmitting, setKpiSubmitting] = useState(false)

  // KPI Revenue form
  const [revCompanyId, setRevCompanyId] = useState('')
  const [period, setPeriod] = useState('')
  const [revenue, setRevenue] = useState('')
  const [revSubmitting, setRevSubmitting] = useState(false)

  // Dividend form
  const [divCompanyId, setDivCompanyId] = useState('')
  const [amountPerShare, setAmountPerShare] = useState('')
  const [divSubmitting, setDivSubmitting] = useState(false)

  useEffect(() => {
    const fetchCompanies = async () => {
      try {
        const data = await api.get<Company[]>('/admin/companies')
        setCompanies(data || [])
      } catch {
        // ignore
      } finally {
        setLoading(false)
      }
    }
    fetchCompanies()
  }, [])

  const handleCreateKpiRule = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!kpiCompanyId || !metricName || !targetValue || !rewardPerUnit) return

    setKpiSubmitting(true)
    try {
      await api.post('/investment/kpi-rules', {
        company_id: Number(kpiCompanyId),
        metric_name: metricName.trim(),
        target_value: Number(targetValue),
        reward_per_unit: Number(rewardPerUnit),
      })
      toast.success('KPI 규칙이 생성되었습니다.')
      setMetricName('')
      setTargetValue('')
      setRewardPerUnit('')
    } catch (err: any) {
      toast.error(err.message || 'KPI 규칙 생성에 실패했습니다.')
    } finally {
      setKpiSubmitting(false)
    }
  }

  const handleAddRevenue = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!revCompanyId || !period || !revenue) return

    setRevSubmitting(true)
    try {
      await api.post('/investment/kpi-revenue', {
        company_id: Number(revCompanyId),
        period: period.trim(),
        revenue: Number(revenue),
      })
      toast.success('매출이 등록되었습니다.')
      setPeriod('')
      setRevenue('')
    } catch (err: any) {
      toast.error(err.message || '매출 등록에 실패했습니다.')
    } finally {
      setRevSubmitting(false)
    }
  }

  const handleDividend = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!divCompanyId || !amountPerShare) return

    if (!confirm('배당을 실행하시겠습니까? 이 작업은 되돌릴 수 없습니다.'))
      return

    setDivSubmitting(true)
    try {
      await api.post('/investment/dividends', {
        company_id: Number(divCompanyId),
        amount_per_share: Number(amountPerShare),
      })
      toast.success('배당이 실행되었습니다.')
      setAmountPerShare('')
    } catch (err: any) {
      toast.error(err.message || '배당 실행에 실패했습니다.')
    } finally {
      setDivSubmitting(false)
    }
  }

  const companySelect = (
    value: string,
    onChange: (v: string) => void,
    id: string,
  ) => (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger id={id}>
        <SelectValue placeholder="기업 선택" />
      </SelectTrigger>
      <SelectContent>
        {companies.map((c) => (
          <SelectItem key={c.id} value={String(c.id)}>
            {c.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Spinner />
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="flex items-center gap-2 text-xl font-bold">
          <BarChart3 className="h-5 w-5" />
          KPI 관리
        </h1>
      </div>

      {companies.length === 0 ? (
        <Card>
          <CardContent className="py-6 text-center text-sm text-muted-foreground">
            등록된 기업이 없습니다. 먼저 기업을 생성해주세요.
          </CardContent>
        </Card>
      ) : (
        <>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="flex items-center gap-2 text-base">
                <Target className="h-4 w-4" />
                KPI 규칙 생성
              </CardTitle>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleCreateKpiRule} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="kpi-company">기업</Label>
                  {companySelect(kpiCompanyId, setKpiCompanyId, 'kpi-company')}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="metric-name">지표명</Label>
                  <Input
                    id="metric-name"
                    placeholder="예: 주간매출, DAU, 고객수"
                    value={metricName}
                    onChange={(e) => setMetricName(e.target.value)}
                    required
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="target-value">목표값</Label>
                    <Input
                      id="target-value"
                      type="number"
                      min="1"
                      placeholder="목표"
                      value={targetValue}
                      onChange={(e) => setTargetValue(e.target.value)}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="reward-unit">단위당 보상 (원)</Label>
                    <Input
                      id="reward-unit"
                      type="number"
                      min="1"
                      placeholder="보상액"
                      value={rewardPerUnit}
                      onChange={(e) => setRewardPerUnit(e.target.value)}
                      required
                    />
                  </div>
                </div>
                <Button
                  type="submit"
                  className="w-full"
                  disabled={kpiSubmitting}
                >
                  {kpiSubmitting ? '생성 중...' : 'KPI 규칙 생성'}
                </Button>
              </form>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="flex items-center gap-2 text-base">
                <DollarSign className="h-4 w-4" />
                KPI 매출 등록
              </CardTitle>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleAddRevenue} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="rev-company">기업</Label>
                  {companySelect(revCompanyId, setRevCompanyId, 'rev-company')}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="period">기간</Label>
                  <Input
                    id="period"
                    placeholder="예: 2026-W10, 3월 2주차"
                    value={period}
                    onChange={(e) => setPeriod(e.target.value)}
                    required
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="revenue">매출액 (원)</Label>
                  <Input
                    id="revenue"
                    type="number"
                    min="0"
                    placeholder="매출액"
                    value={revenue}
                    onChange={(e) => setRevenue(e.target.value)}
                    required
                  />
                  {revenue && (
                    <p className="text-xs text-muted-foreground">
                      {formatMoney(Number(revenue))}
                    </p>
                  )}
                </div>
                <Button
                  type="submit"
                  className="w-full"
                  disabled={revSubmitting}
                >
                  {revSubmitting ? '등록 중...' : '매출 등록'}
                </Button>
              </form>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="flex items-center gap-2 text-base">
                <Gift className="h-4 w-4" />
                배당 실행
              </CardTitle>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleDividend} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="div-company">기업</Label>
                  {companySelect(divCompanyId, setDivCompanyId, 'div-company')}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="amount-per-share">주당 배당금 (원)</Label>
                  <Input
                    id="amount-per-share"
                    type="number"
                    min="1"
                    placeholder="주당 배당금"
                    value={amountPerShare}
                    onChange={(e) => setAmountPerShare(e.target.value)}
                    required
                  />
                  {amountPerShare && (
                    <p className="text-xs text-muted-foreground">
                      주당 {formatMoney(Number(amountPerShare))} 배당
                    </p>
                  )}
                </div>
                <Button
                  type="submit"
                  className="w-full"
                  variant="destructive"
                  disabled={divSubmitting}
                >
                  {divSubmitting ? '실행 중...' : '배당 실행'}
                </Button>
              </form>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}
