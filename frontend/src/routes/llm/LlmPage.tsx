import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { AlertTriangle, Check, Copy, Key, RefreshCw, Sparkles } from 'lucide-react'

import { api, ApiError } from '@/lib/api'
import { formatMoney } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'

interface UserKey {
  id: number
  proxy_student_id: number
  proxy_key_id: number
  prefix: string
  label: string
  issued_at: string
  revoked_at?: string | null
  plaintext?: string // 발급 직후 1회만 존재
}

interface DailyUsageRow {
  id: number
  usage_date: string
  prompt_tokens: number
  completion_tokens: number
  cache_hits: number
  requests: number
  cost_krw: number
  debited_krw: number
  debt_krw: number
  billed_at: string
}

interface Summary {
  cumulative_cost_krw: number
  cumulative_debt_krw: number
  last_week_cost_krw: number
}

interface UsageResponse {
  daily: DailyUsageRow[]
  summary: Summary
}

const PROXY_BASE = 'https://llm.cycorld.com'

export default function LlmPage() {
  const [key, setKey] = useState<UserKey | null>(null)
  const [usage, setUsage] = useState<UsageResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [rotating, setRotating] = useState(false)
  const [copied, setCopied] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [k, u] = await Promise.all([
        api.get<UserKey>('/llm/me'),
        api.get<UsageResponse>('/llm/me/usage?days=30'),
      ])
      setKey(k)
      setUsage(u)
    } catch (err) {
      if (err instanceof ApiError && err.code === 'PROXY_DOWN') {
        toast.error('LLM 서비스가 응답하지 않습니다. 잠시 후 다시 시도해주세요.')
      } else {
        toast.error(err instanceof Error ? err.message : 'LLM 정보를 불러오지 못했습니다.')
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const handleRotate = async () => {
    if (!window.confirm('기존 키가 즉시 폐기되고 새 키가 발급됩니다. 진행할까요?')) return
    setRotating(true)
    try {
      const k = await api.post<UserKey>('/llm/me/rotate')
      setKey(k)
      toast.success('새 키가 발급되었습니다. 이 창을 닫기 전에 반드시 복사해두세요.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '재발급에 실패했습니다.')
    } finally {
      setRotating(false)
    }
  }

  const copy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
      toast.success('클립보드에 복사되었습니다.')
    } catch {
      toast.error('복사에 실패했습니다. 수동으로 선택해서 복사해주세요.')
    }
  }

  if (loading) {
    return (
      <div className="flex min-h-[50vh] items-center justify-center">
        <Spinner />
      </div>
    )
  }

  return (
    <div className="container mx-auto max-w-3xl space-y-6 px-4 py-6">
      <div className="flex items-center gap-2">
        <Sparkles className="h-5 w-5 text-highlight" />
        <h1>LLM API 키</h1>
      </div>
      <p className="text-sm text-muted-foreground">
        <strong className="text-foreground">{PROXY_BASE}</strong> 에 연결되는 개인 API 키.
        Claude Code / Cursor / curl 같은 도구에서 <code className="rounded bg-muted px-1 py-0.5 text-xs">Authorization: Bearer &lt;키&gt;</code>
        {' '}로 사용할 수 있습니다. 사용한 만큼 매일 새벽 03:33 KST 에 자동으로 지갑에서 차감됩니다.
      </p>

      <KeyCard
        k={key}
        copied={copied}
        rotating={rotating}
        onRotate={handleRotate}
        onCopy={copy}
      />

      <PricingCard />

      <UsageCard usage={usage} />
    </div>
  )
}

function KeyCard({
  k,
  copied,
  rotating,
  onRotate,
  onCopy,
}: {
  k: UserKey | null
  copied: boolean
  rotating: boolean
  onRotate: () => void
  onCopy: (text: string) => void
}) {
  if (!k) return null

  const hasPlaintext = !!k.plaintext
  const issuedDate = new Date(k.issued_at).toLocaleString('ko-KR')

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Key className="h-4 w-4 text-primary" />
          내 키
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {hasPlaintext && (
          <div className="rounded-lg border border-highlight/40 bg-highlight/10 p-4 text-sm">
            <div className="flex items-start gap-2 text-highlight">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
              <p className="font-semibold">이 키는 지금 이 화면에서만 볼 수 있습니다.</p>
            </div>
            <p className="mt-1 pl-6 text-foreground/80">
              창을 닫으면 복구할 수 없고, 잃어버리면 재발급만 가능합니다.
              안전한 곳(비밀번호 관리자 등)에 먼저 저장해주세요.
            </p>
            <div className="mt-3 flex items-center gap-2 pl-6">
              <code className="flex-1 break-all rounded bg-background px-3 py-2 text-xs">
                {k.plaintext}
              </code>
              <Button variant="outline" size="sm" onClick={() => onCopy(k.plaintext!)}>
                {copied ? <Check className="h-4 w-4 text-success" /> : <Copy className="h-4 w-4" />}
              </Button>
            </div>
          </div>
        )}

        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
          <dt className="text-muted-foreground">Prefix</dt>
          <dd className="font-mono">{k.prefix}…</dd>
          <dt className="text-muted-foreground">발급일</dt>
          <dd>{issuedDate}</dd>
          <dt className="text-muted-foreground">Label</dt>
          <dd>{k.label}</dd>
        </dl>

        <div className="flex flex-wrap gap-2 pt-2">
          <Button variant="outline" size="sm" onClick={onRotate} disabled={rotating}>
            <RefreshCw className={`h-4 w-4 ${rotating ? 'animate-spin' : ''}`} />
            재발급
          </Button>
          <a
            href={`${PROXY_BASE}/admin/docs`}
            target="_blank"
            rel="noreferrer"
            className="text-sm text-muted-foreground underline-offset-2 hover:underline"
          >
            API 문서 보기 ↗
          </a>
        </div>
      </CardContent>
    </Card>
  )
}

function PricingCard() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>요금 기준</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2 text-sm text-muted-foreground">
        <p>
          Anthropic Claude <strong className="text-foreground">Opus 4.7</strong> 공식 가격을
          환율 1 USD = 1,400원 으로 환산:
        </p>
        <ul className="ml-4 list-disc space-y-1">
          <li>입력 토큰: <strong className="text-foreground">0.021원/토큰</strong> (21원/1k)</li>
          <li>출력 토큰: <strong className="text-foreground">0.105원/토큰</strong> (105원/1k)</li>
          <li>캐시 적중 입력: <strong className="text-foreground">90% 할인</strong> (0.0021원/토큰)</li>
        </ul>
        <p className="pt-1">
          매일 새벽 03:33 KST 에 전날 사용량이 정산되어 지갑에서 자동 차감됩니다.
          잔액이 부족하면 차감 가능한 만큼만 차감되고, 나머지는 부채로 누적되어
          다음 과금 때 지갑이 충전되면 우선 차감됩니다.
        </p>
      </CardContent>
    </Card>
  )
}

function UsageCard({ usage }: { usage: UsageResponse | null }) {
  if (!usage) return null
  const { daily, summary } = usage

  return (
    <Card>
      <CardHeader>
        <CardTitle>사용량 · 청구 내역</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-3 gap-3 text-center">
          <Stat label="누적 청구" value={formatMoney(summary.cumulative_cost_krw)} />
          <Stat label="최근 7일" value={formatMoney(summary.last_week_cost_krw)} />
          <Stat
            label="미차감 부채"
            value={formatMoney(summary.cumulative_debt_krw)}
            tone={summary.cumulative_debt_krw > 0 ? 'warning' : undefined}
          />
        </div>

        {daily.length === 0 ? (
          <p className="py-8 text-center text-sm text-muted-foreground">
            아직 사용 기록이 없습니다. 키를 복사해서 Claude Code / Cursor / curl 에 붙여보세요.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-left text-xs uppercase text-muted-foreground">
                  <th className="py-2 pr-3">일자</th>
                  <th className="py-2 pr-3 text-right">입력 tok</th>
                  <th className="py-2 pr-3 text-right">출력 tok</th>
                  <th className="py-2 pr-3 text-right">캐시/요청</th>
                  <th className="py-2 pr-3 text-right">청구</th>
                  <th className="py-2 text-right">부채</th>
                </tr>
              </thead>
              <tbody>
                {daily.map((d) => (
                  <tr key={d.id} className="border-b last:border-none">
                    <td className="py-2 pr-3 font-medium">{d.usage_date.slice(0, 10)}</td>
                    <td className="py-2 pr-3 text-right tabular-nums">{d.prompt_tokens.toLocaleString()}</td>
                    <td className="py-2 pr-3 text-right tabular-nums">{d.completion_tokens.toLocaleString()}</td>
                    <td className="py-2 pr-3 text-right tabular-nums text-muted-foreground">
                      {d.cache_hits}/{d.requests}
                    </td>
                    <td className="py-2 pr-3 text-right tabular-nums font-semibold">
                      {formatMoney(d.cost_krw)}
                    </td>
                    <td className={`py-2 text-right tabular-nums ${d.debt_krw > 0 ? 'text-coral' : 'text-muted-foreground'}`}>
                      {d.debt_krw > 0 ? formatMoney(d.debt_krw) : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function Stat({ label, value, tone }: { label: string; value: string; tone?: 'warning' }) {
  return (
    <div className={`rounded-lg border p-3 ${tone === 'warning' ? 'border-coral/30 bg-coral/10' : 'bg-muted/30'}`}>
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className={`mt-1 text-base font-semibold tabular-nums ${tone === 'warning' ? 'text-coral' : ''}`}>
        {value}
      </div>
    </div>
  )
}
