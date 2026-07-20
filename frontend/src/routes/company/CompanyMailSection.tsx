import { useCallback, useEffect, useState } from 'react'
import { Check, Clock, Copy, Loader2, Mail } from 'lucide-react'
import { toast } from 'sonner'

import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'

// GET /api/mail/mailboxes 계약과 1:1 (회사 항목만 사용)
type AddressStatus = null | 'pending' | 'rejected' | 'approved'

interface Mailbox {
  address_id: number
  kind: 'user' | 'company' | 'shared'
  company_id: number | null
  name: string
  local_part: string | null
  email: string | null
  status: AddressStatus
}

const LOCAL_PART_RE = /^[a-z0-9][a-z0-9._-]{2,29}$/

/**
 * 회사 이메일 섹션 — 대표(소유자)에게만 노출한다.
 * 미신청: 등록 폼 / 승인대기·반려: 상태 + 재신청 / 승인: 읽기전용 주소 + 복사.
 */
export function CompanyMailSection({ companyId }: { companyId: number }) {
  const [mailbox, setMailbox] = useState<Mailbox | null>(null)
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api.get<{ mailboxes: Mailbox[] }>('/mail/mailboxes')
      const found =
        data?.mailboxes?.find(
          (m) => m.kind === 'company' && m.company_id === companyId,
        ) ?? null
      setMailbox(found)
    } catch {
      setMailbox(null)
    } finally {
      setLoading(false)
    }
  }, [companyId])

  useEffect(() => {
    load()
  }, [load])

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <Mail className="h-4 w-4 text-primary" />
          회사 이메일
        </CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="flex justify-center py-4">
            <Spinner />
          </div>
        ) : mailbox?.status === 'approved' ? (
          <ApprovedAddress email={mailbox.email ?? ''} />
        ) : (
          <ClaimForm
            companyId={companyId}
            status={mailbox?.status ?? null}
            initialLocalPart={mailbox?.local_part ?? ''}
            onClaimed={load}
          />
        )}
      </CardContent>
    </Card>
  )
}

function ApprovedAddress({ email }: { email: string }) {
  const [copied, setCopied] = useState(false)

  const copy = async () => {
    try {
      await navigator.clipboard?.writeText(email)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      toast.error('복사에 실패했습니다.')
    }
  }

  return (
    <div className="flex items-center gap-2 rounded-md border bg-muted/40 px-3 py-2">
      <Mail className="h-4 w-4 shrink-0 text-muted-foreground" />
      <span className="min-w-0 flex-1 truncate text-sm font-medium">{email}</span>
      <Button
        variant="ghost"
        size="sm"
        className="h-7 gap-1 px-2 text-xs"
        onClick={copy}
        aria-label="주소 복사"
      >
        {copied ? (
          <Check className="h-3.5 w-3.5 text-emerald-600" />
        ) : (
          <Copy className="h-3.5 w-3.5" />
        )}
        {copied ? '복사됨' : '복사'}
      </Button>
    </div>
  )
}

function ClaimForm({
  companyId,
  status,
  initialLocalPart,
  onClaimed,
}: {
  companyId: number
  status: AddressStatus
  initialLocalPart: string
  onClaimed: () => void
}) {
  const [localPart, setLocalPart] = useState(initialLocalPart)
  const [submitting, setSubmitting] = useState(false)

  const trimmed = localPart.trim().toLowerCase()
  const valid = LOCAL_PART_RE.test(trimmed)
  const showError = trimmed.length > 0 && !valid

  const submit = async () => {
    if (!valid) return
    setSubmitting(true)
    try {
      await api.post(`/companies/${companyId}/mail-address`, {
        local_part: trimmed,
      })
      toast.success('회사 이메일을 신청했습니다. 관리자 승인 후 사용할 수 있어요.')
      onClaimed()
    } catch (e) {
      const msg = e instanceof Error ? e.message : '회사 이메일 신청에 실패했습니다.'
      toast.error(msg)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="space-y-4">
      {status === 'pending' && (
        <div className="flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800">
          <Clock className="mt-0.5 h-4 w-4 shrink-0" />
          <span>관리자 승인 대기 중입니다. 아래에서 주소를 다시 신청할 수 있어요.</span>
        </div>
      )}
      {status === 'rejected' && (
        <div className="rounded-md border border-red-200 bg-red-50 p-3 text-xs text-red-700">
          이전 회사 이메일 신청이 반려되었습니다. 다른 주소로 다시 신청해주세요.
        </div>
      )}

      <p className="text-sm text-muted-foreground">
        회사 이메일 주소를 신청하면 관리자 승인 후 회사 이름으로 메일을 주고받을 수
        있어요.
      </p>

      <div className="space-y-1.5">
        <Label htmlFor="company-local-part">주소</Label>
        <div className="flex items-center gap-2">
          <Input
            id="company-local-part"
            value={localPart}
            onChange={(e) => setLocalPart(e.target.value)}
            placeholder="acompany"
            autoCapitalize="none"
            autoCorrect="off"
            spellCheck={false}
          />
          <span className="shrink-0 text-sm text-muted-foreground">
            @earnlearning.com
          </span>
        </div>
        {showError ? (
          <p className="text-xs text-red-600">
            3~30자, 영소문자·숫자로 시작하고 영소문자·숫자·.·_·- 만 쓸 수 있어요.
          </p>
        ) : (
          <p className="text-xs text-muted-foreground">
            미리보기:{' '}
            <span className="font-medium text-foreground">
              {trimmed || 'acompany'}@earnlearning.com
            </span>
          </p>
        )}
      </div>

      <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800">
        한 번 승인되면 변경할 수 없습니다. 신중히 정해주세요.
      </div>

      <Button className="w-full" onClick={submit} disabled={!valid || submitting}>
        {submitting ? (
          <Loader2 className="mr-1 h-4 w-4 animate-spin" />
        ) : (
          <Check className="mr-1 h-4 w-4" />
        )}
        이 주소로 신청하기
      </Button>
    </div>
  )
}
