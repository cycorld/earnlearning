import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { MarkdownEditor } from '@/components/MarkdownEditor'
import { toast } from 'sonner'
import { ArrowLeft, Landmark, Info } from 'lucide-react'
import { formatMoney } from '@/lib/utils'

export default function LoanApplyPage() {
  const navigate = useNavigate()
  const [amount, setAmount] = useState('')
  const [purpose, setPurpose] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!amount || !purpose.trim()) return

    setSubmitting(true)
    try {
      await api.post('/loans', {
        amount: Number(amount),
        purpose: purpose.trim(),
      })
      toast.success('대출 신청이 완료되었습니다.')
      navigate('/bank')
    } catch (err: any) {
      toast.error(err.message || '대출 신청에 실패했습니다.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="flex items-center gap-2 text-xl font-bold">
          <Landmark className="h-5 w-5" />
          대출 신청
        </h1>
      </div>

      <Card className="border-blue-200 bg-blue-50/50">
        <CardContent className="flex items-start gap-3 p-4">
          <Info className="mt-0.5 h-5 w-5 shrink-0 text-blue-500" />
          <div className="space-y-1 text-sm text-blue-700">
            <p className="font-medium">대출 안내</p>
            <ul className="list-inside list-disc space-y-0.5 text-xs">
              <li>대출 신청 후 관리자 승인이 필요합니다.</li>
              <li>승인 시 이자율이 결정됩니다.</li>
              <li>매주 이자가 자동으로 부과됩니다.</li>
              <li>연체 시 추가 이자가 발생할 수 있습니다.</li>
            </ul>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base">신청서 작성</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="amount">대출 금액 (원)</Label>
              <Input
                id="amount"
                type="number"
                min="1000"
                step="1000"
                placeholder="대출 희망 금액"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                required
              />
              {amount && (
                <p className="text-xs text-muted-foreground">
                  {formatMoney(Number(amount))}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="purpose">대출 목적</Label>
              <MarkdownEditor
                value={purpose}
                onChange={setPurpose}
                placeholder="대출 목적을 상세히 작성해주세요. (예: 회사 운영 자금, 주식 투자 등)"
                rows={8}
              />
            </div>

            <Button type="submit" className="w-full" disabled={submitting}>
              {submitting ? '신청 중...' : '대출 신청'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
