import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { Activity, AlertTriangle, Check, Copy, Key, RefreshCw, Sparkles } from 'lucide-react'

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
  cache_tokens: number
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

interface ProxyStatus {
  service: string
  version: string
  uptime_seconds: number
  upstream_status: string
  model: string
  latency_ms?: number
  context_window?: number
  slots_total?: number
  slots_idle?: number
  slots_processing?: number
}

const PROXY_BASE = 'https://llm.cycorld.com'

export default function LlmPage() {
  const [key, setKey] = useState<UserKey | null>(null)
  const [usage, setUsage] = useState<UsageResponse | null>(null)
  const [status, setStatus] = useState<ProxyStatus | null>(null)
  const [statusError, setStatusError] = useState<string | null>(null)
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
    // 상태는 실패해도 페이지 자체는 뜨도록 별도 처리
    try {
      const s = await api.get<ProxyStatus>('/llm/status')
      setStatus(s)
      setStatusError(null)
    } catch (err) {
      setStatus(null)
      setStatusError(err instanceof Error ? err.message : '상태 조회 실패')
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

      <StatusCard status={status} error={statusError} />

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

function StatusCard({ status, error }: { status: ProxyStatus | null; error: string | null }) {
  const ok = status?.upstream_status === 'ok'
  const color = ok ? 'text-success' : error || status?.upstream_status ? 'text-coral' : 'text-muted-foreground'
  const dot = ok ? 'bg-success' : error || status?.upstream_status ? 'bg-coral' : 'bg-muted-foreground'

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Activity className="h-4 w-4 text-primary" />
          서비스 상태
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex items-center gap-2">
          <span className={`inline-block h-2.5 w-2.5 rounded-full ${dot} ${ok ? 'animate-pulse' : ''}`} />
          <span className={`text-sm font-semibold ${color}`}>
            {ok
              ? '정상 작동 중'
              : error
                ? '상태 조회 실패'
                : `상태 ${status?.upstream_status ?? 'unknown'}`}
          </span>
          {status?.latency_ms != null && (
            <span className="text-xs text-muted-foreground">· 지연 {status.latency_ms.toFixed(1)}ms</span>
          )}
        </div>

        {status && (
          <dl className="mt-3 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-xs">
            <dt className="text-muted-foreground">모델</dt>
            <dd className="font-mono break-all">{status.model || '—'}</dd>
            {status.context_window ? (
              <>
                <dt className="text-muted-foreground">컨텍스트</dt>
                <dd className="tabular-nums">{status.context_window.toLocaleString()} tok</dd>
              </>
            ) : null}
            {status.slots_total != null && status.slots_total > 0 ? (
              <>
                <dt className="text-muted-foreground">슬롯</dt>
                <dd className="tabular-nums">
                  <span className={status.slots_processing && status.slots_processing === status.slots_total ? 'text-coral' : ''}>
                    {status.slots_processing ?? 0} / {status.slots_total} 처리 중
                  </span>
                  <span className="ml-2 text-muted-foreground">
                    (여유 {status.slots_idle ?? 0})
                  </span>
                </dd>
              </>
            ) : null}
            <dt className="text-muted-foreground">가동 시간</dt>
            <dd className="tabular-nums">{formatUptime(status.uptime_seconds)}</dd>
            <dt className="text-muted-foreground">버전</dt>
            <dd className="font-mono text-muted-foreground">{status.service} {status.version}</dd>
          </dl>
        )}

        {error && !status && (
          <p className="mt-2 text-xs text-muted-foreground">
            {error}. 키 발급은 계속 가능하지만, 실제 API 호출은 복구될 때까지 실패할 수 있습니다.
          </p>
        )}
      </CardContent>
    </Card>
  )
}

function formatUptime(sec: number): string {
  if (sec < 60) return `${sec}초`
  const m = Math.floor(sec / 60)
  if (m < 60) return `${m}분`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}시간 ${m % 60}분`
  const d = Math.floor(h / 24)
  return `${d}일 ${h % 24}시간`
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
            href={PROXY_BASE}
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
          실제 서비스 모델은 <strong className="text-foreground">Qwen3.6-35B-A3B</strong> (무료,
          강의 GPU 서버 운영). 요금은 <strong className="text-foreground">Anthropic Claude Opus 4.7</strong> 공식 가격을 환율 1 USD = 1,400원 으로 환산해 학습용 과금:
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
                  <th className="py-2 pr-3 text-right">캐시 재사용</th>
                  <th className="py-2 pr-3 text-right">출력 tok</th>
                  <th className="py-2 pr-3 text-right">요청 수</th>
                  <th className="py-2 pr-3 text-right">청구</th>
                  <th className="py-2 text-right">부채</th>
                </tr>
              </thead>
              <tbody>
                {daily.map((d) => {
                  const cacheRatio = d.prompt_tokens > 0 ? d.cache_tokens / d.prompt_tokens : 0
                  return (
                    <tr key={d.id} className="border-b last:border-none">
                      <td className="py-2 pr-3 font-medium">{d.usage_date.slice(0, 10)}</td>
                      <td className="py-2 pr-3 text-right tabular-nums">{d.prompt_tokens.toLocaleString()}</td>
                      <td className="py-2 pr-3 text-right tabular-nums">
                        <span className={cacheRatio > 0 ? 'text-success' : 'text-muted-foreground'}>
                          {d.cache_tokens.toLocaleString()}
                        </span>
                        {cacheRatio > 0 && (
                          <span className="ml-1 text-[11px] text-muted-foreground">
                            ({Math.round(cacheRatio * 100)}%)
                          </span>
                        )}
                      </td>
                      <td className="py-2 pr-3 text-right tabular-nums">{d.completion_tokens.toLocaleString()}</td>
                      <td className="py-2 pr-3 text-right tabular-nums text-muted-foreground">{d.requests}</td>
                      <td className="py-2 pr-3 text-right tabular-nums font-semibold">
                        {formatMoney(d.cost_krw)}
                      </td>
                      <td className={`py-2 text-right tabular-nums ${d.debt_krw > 0 ? 'text-coral' : 'text-muted-foreground'}`}>
                        {d.debt_krw > 0 ? formatMoney(d.debt_krw) : '—'}
                      </td>
                    </tr>
                  )
                })}
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
