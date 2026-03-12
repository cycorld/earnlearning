import { useState, useEffect, useRef } from 'react'
import { useParams } from 'react-router-dom'
import { api } from '@/lib/api'
import type { Company } from '@/types'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { toast } from 'sonner'
import { Download, Share2 } from 'lucide-react'

function formatMoney(amount: number): string {
  return new Intl.NumberFormat('ko-KR').format(amount) + '원'
}

interface BusinessCard {
  html?: string
  image_url?: string
  company: Company
}

export default function BusinessCardPage() {
  const { id } = useParams()
  const [card, setCard] = useState<BusinessCard | null>(null)
  const [loading, setLoading] = useState(true)
  const cardRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    api
      .get<BusinessCard>(`/companies/${id}/business-card`)
      .then(setCard)
      .catch(() => setCard(null))
      .finally(() => setLoading(false))
  }, [id])

  async function handleShare() {
    const company = card?.company
    if (!company) return

    const shareData = {
      title: `${company.name} 명함`,
      text: `${company.name} - ${company.description || ''}`,
      url: window.location.href,
    }

    if (navigator.share) {
      try {
        await navigator.share(shareData)
      } catch {
        // User cancelled share
      }
    } else {
      await navigator.clipboard.writeText(window.location.href)
      toast.success('링크가 복사되었습니다.')
    }
  }

  async function handleDownload() {
    if (card?.image_url) {
      const link = document.createElement('a')
      link.href = card.image_url
      link.download = `${card.company.name}_명함.png`
      link.click()
      return
    }

    // Fallback: copy link
    await navigator.clipboard.writeText(window.location.href)
    toast.success('링크가 복사되었습니다.')
  }

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  if (!card) {
    return (
      <div className="p-4 text-center text-muted-foreground">
        명함을 찾을 수 없습니다.
      </div>
    )
  }

  const company = card.company

  return (
    <div className="mx-auto max-w-lg space-y-4 p-4">
      <h1 className="text-lg font-bold text-center">명함</h1>

      <div className="flex justify-center">
        <div ref={cardRef}>
          {card.html ? (
            <div
              className="w-full max-w-sm overflow-hidden rounded-xl shadow-lg"
              dangerouslySetInnerHTML={{ __html: card.html }}
            />
          ) : (
            <Card className="w-full max-w-sm overflow-hidden bg-gradient-to-br from-primary/5 to-background shadow-lg">
              <CardContent className="space-y-4 p-8 text-center">
                <Avatar className="mx-auto h-20 w-20">
                  <AvatarImage src={company.logo_url} />
                  <AvatarFallback className="bg-primary/10 text-2xl text-primary">
                    {company.name.charAt(0)}
                  </AvatarFallback>
                </Avatar>
                <div>
                  <h2 className="text-xl font-bold">{company.name}</h2>
                  {company.listed && <Badge className="mt-1">상장기업</Badge>}
                </div>
                {company.description && (
                  <p className="text-sm text-muted-foreground">{company.description}</p>
                )}
                <div className="rounded-lg bg-muted/50 p-3">
                  <div className="grid grid-cols-2 gap-2 text-sm">
                    <div>
                      <p className="text-xs text-muted-foreground">기업가치</p>
                      <p className="font-semibold">{formatMoney(company.valuation)}</p>
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">자본금</p>
                      <p className="font-semibold">{formatMoney(company.total_capital)}</p>
                    </div>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground">
                  대표: {company.owner?.name || '-'}
                </p>
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      <div className="flex gap-2">
        <Button variant="outline" className="flex-1" onClick={handleShare}>
          <Share2 className="mr-2 h-4 w-4" />
          공유하기
        </Button>
        <Button variant="outline" className="flex-1" onClick={handleDownload}>
          <Download className="mr-2 h-4 w-4" />
          다운로드
        </Button>
      </div>
    </div>
  )
}
