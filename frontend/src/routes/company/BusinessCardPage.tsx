import { useState, useEffect, useRef } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '@/lib/api'
import type { Company } from '@/types'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { toast } from 'sonner'
import { Download, Share2, ArrowLeft, ChevronLeft, ChevronRight } from 'lucide-react'
import { toPng } from 'html-to-image'

interface BusinessCard {
  company: Company
}

const TEMPLATES = [
  { id: 'classic', name: 'Classic' },
  { id: 'modern', name: 'Modern' },
  { id: 'minimal', name: 'Minimal' },
  { id: 'bold', name: 'Bold' },
  { id: 'elegant', name: 'Elegant' },
]

function CardTemplate({
  template,
  company,
}: {
  template: string
  company: Company
}) {
  const owner = company.owner?.name || '대표'
  const email = (company.owner as { email?: string } | undefined)?.email || ''
  const name = company.name
  const desc = company.description || ''
  const logo = company.logo_url

  const logoEl = logo ? (
    <img src={logo} alt={name} className="h-10 w-10 rounded-lg object-cover" />
  ) : (
    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-white/20 text-lg font-bold">
      {name.charAt(0)}
    </div>
  )

  switch (template) {
    case 'classic':
      return (
        <div className="flex h-[220px] w-[400px] flex-col justify-between rounded-2xl bg-gradient-to-br from-slate-900 to-slate-700 p-8 text-white shadow-2xl">
          <div className="flex items-start justify-between">
            {logoEl}
            <p className="text-right text-[10px] uppercase tracking-[3px] text-slate-400">Business Card</p>
          </div>
          <div>
            <h2 className="text-xl font-bold tracking-tight">{owner}</h2>
            <p className="text-xs text-slate-300">CEO / Founder</p>
          </div>
          <div className="space-y-0.5 border-t border-slate-600 pt-3">
            <p className="text-sm font-semibold">{name}</p>
            <p className="line-clamp-1 text-[10px] text-slate-400">{desc}</p>
            <p className="text-[10px] text-slate-400">{email}</p>
          </div>
        </div>
      )

    case 'modern':
      return (
        <div className="flex h-[220px] w-[400px] overflow-hidden rounded-2xl bg-white shadow-2xl">
          <div className="flex w-2/5 flex-col justify-center bg-gradient-to-b from-teal-500 to-emerald-600 p-6 text-white">
            <div className="mb-4">
              {logo ? (
                <img src={logo} alt={name} className="h-12 w-12 rounded-xl bg-white/20 object-cover" />
              ) : (
                <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-white/20 text-xl font-bold">{name.charAt(0)}</div>
              )}
            </div>
            <p className="text-lg font-bold leading-tight">{name}</p>
            <p className="mt-1 line-clamp-2 text-[9px] leading-relaxed text-white/70">{desc}</p>
          </div>
          <div className="flex w-3/5 flex-col justify-center p-6">
            <h2 className="text-xl font-bold text-slate-800">{owner}</h2>
            <p className="text-xs font-medium text-teal-600">CEO / Founder</p>
            <div className="mt-4 space-y-1.5">
              <div className="flex items-center gap-2">
                <div className="h-1 w-1 rounded-full bg-teal-500" />
                <p className="text-[11px] text-slate-500">{email}</p>
              </div>
              <div className="flex items-center gap-2">
                <div className="h-1 w-1 rounded-full bg-teal-500" />
                <p className="text-[11px] text-slate-500">earnlearning.com</p>
              </div>
            </div>
          </div>
        </div>
      )

    case 'minimal':
      return (
        <div className="flex h-[220px] w-[400px] flex-col justify-between rounded-2xl border-2 border-slate-200 bg-white p-8 shadow-lg">
          <div className="flex items-center gap-3">
            {logo ? (
              <img src={logo} alt={name} className="h-8 w-8 rounded-md object-cover" />
            ) : (
              <div className="flex h-8 w-8 items-center justify-center rounded-md bg-slate-100 text-sm font-bold text-slate-600">{name.charAt(0)}</div>
            )}
            <span className="text-sm font-semibold text-slate-800">{name}</span>
          </div>
          <div>
            <h2 className="text-2xl font-light text-slate-900">{owner}</h2>
            <p className="mt-0.5 text-xs text-slate-400">CEO / Founder</p>
          </div>
          <div className="flex items-end justify-between">
            <p className="line-clamp-1 max-w-[60%] text-[10px] text-slate-400">{desc}</p>
            <p className="text-[10px] text-slate-400">{email}</p>
          </div>
        </div>
      )

    case 'bold':
      return (
        <div className="flex h-[220px] w-[400px] flex-col justify-between rounded-2xl bg-gradient-to-br from-violet-600 via-purple-600 to-fuchsia-600 p-8 text-white shadow-2xl">
          <div className="flex items-start justify-between">
            {logo ? (
              <img src={logo} alt={name} className="h-12 w-12 rounded-2xl bg-white/10 object-cover ring-2 ring-white/20" />
            ) : (
              <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-white/10 text-xl font-black ring-2 ring-white/20">{name.charAt(0)}</div>
            )}
            <div className="text-right">
              <p className="text-lg font-black tracking-tight">{name}</p>
              <p className="line-clamp-1 text-[9px] text-white/60">{desc}</p>
            </div>
          </div>
          <div>
            <h2 className="text-3xl font-black tracking-tight">{owner}</h2>
            <div className="mt-2 flex items-center gap-3">
              <span className="rounded-full bg-white/20 px-3 py-0.5 text-[10px] font-medium">CEO</span>
              <span className="text-[10px] text-white/70">{email}</span>
            </div>
          </div>
        </div>
      )

    case 'elegant':
      return (
        <div className="relative flex h-[220px] w-[400px] flex-col justify-between overflow-hidden rounded-2xl bg-gradient-to-br from-warning/10 to-highlight/10 p-8 shadow-2xl">
          <div className="absolute -right-8 -top-8 h-32 w-32 rounded-full bg-warning/25/30" />
          <div className="absolute -bottom-4 -left-4 h-20 w-20 rounded-full bg-highlight/25/30" />
          <div className="relative flex items-center gap-3">
            {logo ? (
              <img src={logo} alt={name} className="h-10 w-10 rounded-xl object-cover shadow-md" />
            ) : (
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-warning text-lg font-bold text-white shadow-md">{name.charAt(0)}</div>
            )}
            <div>
              <p className="text-sm font-bold text-warning">{name}</p>
              <p className="line-clamp-1 text-[9px] text-warning/60">{desc}</p>
            </div>
          </div>
          <div className="relative">
            <div className="mb-2 h-px bg-gradient-to-r from-warning/40 to-transparent" />
            <h2 className="text-xl font-bold text-warning">{owner}</h2>
            <p className="text-[10px] font-medium text-warning">CEO / Founder</p>
          </div>
          <p className="relative text-[10px] text-warning/60">{email}</p>
        </div>
      )

    default:
      return null
  }
}

export default function BusinessCardPage() {
  const { id } = useParams()
  const [card, setCard] = useState<BusinessCard | null>(null)
  const [loading, setLoading] = useState(true)
  const [templateIdx, setTemplateIdx] = useState(0)
  const [downloading, setDownloading] = useState(false)
  const cardRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    api
      .get<BusinessCard>(`/companies/${id}/business-card`)
      .then(setCard)
      .catch(() => setCard(null))
      .finally(() => setLoading(false))
  }, [id])

  const handleDownload = async () => {
    if (!cardRef.current) return
    setDownloading(true)
    try {
      const dataUrl = await toPng(cardRef.current, { pixelRatio: 3 })
      const link = document.createElement('a')
      link.download = `${card?.company.name || 'card'}_명함.png`
      link.href = dataUrl
      link.click()
      toast.success('명함이 다운로드되었습니다.')
    } catch {
      toast.error('다운로드에 실패했습니다.')
    } finally {
      setDownloading(false)
    }
  }

  const handleShare = async () => {
    const shareData = {
      title: `${card?.company.name} 명함`,
      text: `${card?.company.name} - ${card?.company.description || ''}`,
      url: window.location.href,
    }
    if (navigator.share) {
      try { await navigator.share(shareData) } catch { /* cancelled */ }
    } else {
      await navigator.clipboard.writeText(window.location.href)
      toast.success('링크가 복사되었습니다.')
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Spinner />
      </div>
    )
  }

  if (!card) {
    return (
      <div className="p-4 text-center text-muted-foreground">명함을 찾을 수 없습니다.</div>
    )
  }

  const template = TEMPLATES[templateIdx]

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      <div className="sticky top-14 z-40 -mx-4 flex items-center gap-2 bg-background px-4 py-2">
        <Button variant="ghost" size="icon" asChild>
          <Link to={`/company/${id}`}>
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <h1 className="text-lg font-bold">명함</h1>
      </div>

      {/* Card preview */}
      <div className="flex justify-center overflow-x-auto py-2">
        <div ref={cardRef}>
          <CardTemplate template={template.id} company={card.company} />
        </div>
      </div>

      {/* Template selector */}
      <div className="flex items-center justify-center gap-3">
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8"
          onClick={() => setTemplateIdx((i) => (i - 1 + TEMPLATES.length) % TEMPLATES.length)}
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>
        <div className="flex gap-2">
          {TEMPLATES.map((t, i) => (
            <button
              key={t.id}
              onClick={() => setTemplateIdx(i)}
              className={`rounded-full px-3 py-1 text-xs transition-colors ${
                i === templateIdx
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-muted text-muted-foreground hover:bg-muted/80'
              }`}
            >
              {t.name}
            </button>
          ))}
        </div>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8"
          onClick={() => setTemplateIdx((i) => (i + 1) % TEMPLATES.length)}
        >
          <ChevronRight className="h-4 w-4" />
        </Button>
      </div>

      {/* Actions */}
      <div className="flex gap-2">
        <Button variant="outline" className="flex-1" onClick={handleShare}>
          <Share2 className="mr-2 h-4 w-4" />
          공유
        </Button>
        <Button className="flex-1" onClick={handleDownload} disabled={downloading}>
          <Download className="mr-2 h-4 w-4" />
          {downloading ? '저장 중...' : 'PNG 다운로드'}
        </Button>
      </div>
    </div>
  )
}
