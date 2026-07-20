import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { toast } from 'sonner'
import {
  ArrowLeft,
  Building2,
  Check,
  Clock,
  Loader2,
  Mail,
  Plus,
  User,
  Users,
  UserPlus,
  X,
} from 'lucide-react'
import { Spinner } from '@/components/ui/spinner'

// ─── 계약 타입 ────────────────────────────────────────────────
interface MailAddressRequest {
  id: number
  user_id: number
  user_name: string
  user_email: string
  local_part: string
  status: string
  created_at: string
  owner_type: 'user' | 'company' | 'shared'
  owner_name: string
}

interface SharedGrant {
  user_id: number
  user_name: string
  revoked: boolean
}

interface SharedMailbox {
  address_id: number
  local_part: string
  display_name: string
  email: string
  grants: SharedGrant[]
}

interface SearchUser {
  id: number
  name: string
  department: string
  student_id: string
  avatar_url: string
}

const LOCAL_PART_RE = /^[a-z0-9][a-z0-9._-]{2,29}$/

function formatDate(s: string): string {
  return new Date(s).toLocaleString('ko-KR')
}

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return '방금'
  if (mins < 60) return `${mins}분 전`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}시간 전`
  const days = Math.floor(hours / 24)
  return `${days}일 전`
}

function OwnerBadge({ ownerType }: { ownerType: string }) {
  if (ownerType === 'company') {
    return (
      <Badge variant="outline" className="gap-1 text-xs">
        <Building2 className="h-3 w-3" />
        회사
      </Badge>
    )
  }
  if (ownerType === 'shared') {
    return (
      <Badge variant="outline" className="gap-1 text-xs">
        <Users className="h-3 w-3" />
        공용
      </Badge>
    )
  }
  return (
    <Badge variant="outline" className="gap-1 text-xs">
      <User className="h-3 w-3" />
      개인
    </Badge>
  )
}

function StatusBadge({ status }: { status: string }) {
  if (status === 'approved') {
    return <Badge className="text-xs">승인됨</Badge>
  }
  if (status === 'rejected') {
    return (
      <Badge variant="destructive" className="text-xs">
        반려됨
      </Badge>
    )
  }
  return (
    <Badge variant="secondary" className="gap-1 text-xs">
      <Clock className="h-3 w-3" />
      대기중
    </Badge>
  )
}

export default function AdminMailAddressesPage() {
  const navigate = useNavigate()
  const [pending, setPending] = useState<MailAddressRequest[]>([])
  const [allAccounts, setAllAccounts] = useState<MailAddressRequest[]>([])
  const [shared, setShared] = useState<SharedMailbox[]>([])
  const [loading, setLoading] = useState(true)
  // Tabs 를 제어형으로 — 액션 후 refetch(로딩 스피너 스왑)에도 활성 탭 유지
  const [tab, setTab] = useState('pending')

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const [pendingData, allData, sharedData] = await Promise.all([
        api
          .get<MailAddressRequest[]>('/admin/mail/addresses?status=pending')
          .catch(() => []),
        api
          .get<MailAddressRequest[]>('/admin/mail/addresses?status=all')
          .catch(() => []),
        api.get<SharedMailbox[]>('/admin/mail/shared').catch(() => []),
      ])
      setPending(Array.isArray(pendingData) ? pendingData : [])
      setAllAccounts(Array.isArray(allData) ? allData : [])
      setShared(Array.isArray(sharedData) ? sharedData : [])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  const handleApprove = async (id: number) => {
    try {
      await api.post(`/admin/mail/addresses/${id}/approve`)
      toast.success('승인되었습니다.')
      fetchData()
    } catch (err: any) {
      toast.error(err.message || '승인에 실패했습니다.')
    }
  }

  const handleReject = async (id: number) => {
    try {
      await api.post(`/admin/mail/addresses/${id}/reject`)
      toast.success('반려되었습니다.')
      fetchData()
    } catch (err: any) {
      toast.error(err.message || '반려에 실패했습니다.')
    }
  }

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
          <Mail className="h-5 w-5" />
          메일 주소 관리
        </h1>
      </div>

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList className="w-full">
          <TabsTrigger value="pending" className="flex-1">
            <Clock className="mr-1 h-3 w-3" />
            승인 대기 ({pending.length})
          </TabsTrigger>
          <TabsTrigger value="all" className="flex-1">
            <Mail className="mr-1 h-3 w-3" />
            전체 계정 ({allAccounts.length})
          </TabsTrigger>
          <TabsTrigger value="shared" className="flex-1">
            <Users className="mr-1 h-3 w-3" />
            공용 메일함
          </TabsTrigger>
        </TabsList>

        <TabsContent value="pending" className="mt-4">
          <Card>
            <CardContent className="space-y-2 p-4">
              {pending.length === 0 ? (
                <p className="py-4 text-center text-sm text-muted-foreground">
                  대기 중인 신청이 없습니다.
                </p>
              ) : (
                pending.map((req) => (
                  <div
                    key={req.id}
                    className="flex items-center justify-between rounded-lg border p-3"
                  >
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <p className="truncate text-sm font-medium">
                          {req.owner_name}
                        </p>
                        {req.owner_type === 'company' ? (
                          <Badge variant="outline" className="gap-1 text-xs">
                            <Building2 className="h-3 w-3" />
                            회사
                          </Badge>
                        ) : (
                          <Badge variant="outline" className="gap-1 text-xs">
                            <User className="h-3 w-3" />
                            개인
                          </Badge>
                        )}
                        <Badge variant="secondary" className="gap-1 text-xs">
                          <Clock className="h-3 w-3" />
                          대기
                        </Badge>
                      </div>
                      <p className="truncate text-xs text-muted-foreground">
                        {req.user_email}
                      </p>
                      <p className="truncate text-sm font-medium text-primary">
                        {req.local_part}@earnlearning.com
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {formatDate(req.created_at)}
                      </p>
                    </div>
                    <div className="flex shrink-0 items-center gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleApprove(req.id)}
                        title="승인"
                      >
                        <Check className="h-4 w-4 text-success" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleReject(req.id)}
                        title="반려"
                      >
                        <X className="h-4 w-4 text-coral" />
                      </Button>
                    </div>
                  </div>
                ))
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="all" className="mt-4">
          <Card>
            <CardContent className="space-y-2 p-4">
              {allAccounts.length === 0 ? (
                <p className="py-4 text-center text-sm text-muted-foreground">
                  등록된 메일 계정이 없습니다.
                </p>
              ) : (
                allAccounts.map((acc) => (
                  <div
                    key={acc.id}
                    className="flex items-center justify-between rounded-lg border p-3"
                  >
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <p className="truncate text-sm font-medium">
                          {acc.owner_name}
                        </p>
                        <OwnerBadge ownerType={acc.owner_type} />
                        <StatusBadge status={acc.status} />
                      </div>
                      <p className="truncate text-sm font-medium text-primary">
                        {acc.local_part}@earnlearning.com
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {timeAgo(acc.created_at)}
                      </p>
                    </div>
                    <div className="flex shrink-0 items-center gap-1">
                      {acc.status === 'pending' && (
                        <>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => handleApprove(acc.id)}
                            title="승인"
                          >
                            <Check className="h-4 w-4 text-success" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => handleReject(acc.id)}
                            title="반려"
                          >
                            <X className="h-4 w-4 text-coral" />
                          </Button>
                        </>
                      )}
                      {acc.status === 'rejected' && (
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleApprove(acc.id)}
                          title="승인"
                        >
                          <Check className="h-4 w-4 text-success" />
                        </Button>
                      )}
                    </div>
                  </div>
                ))
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="shared" className="mt-4 space-y-4">
          <CreateSharedForm onCreated={fetchData} />
          <Card>
            <CardContent className="space-y-3 p-4">
              {shared.length === 0 ? (
                <p className="py-4 text-center text-sm text-muted-foreground">
                  공용 메일함이 없습니다.
                </p>
              ) : (
                shared.map((mb) => (
                  <SharedMailboxCard
                    key={mb.address_id}
                    mailbox={mb}
                    onChanged={fetchData}
                  />
                ))
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}

// ─── 공용 메일함 생성 폼 ─────────────────────────────────────
function CreateSharedForm({ onCreated }: { onCreated: () => void }) {
  const [localPart, setLocalPart] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const trimmed = localPart.trim().toLowerCase()
  const validLocal = LOCAL_PART_RE.test(trimmed)
  const valid = validLocal && displayName.trim().length > 0

  const submit = async () => {
    if (!valid) return
    setSubmitting(true)
    try {
      await api.post('/admin/mail/shared', {
        local_part: trimmed,
        display_name: displayName.trim(),
      })
      toast.success('공용 메일함을 만들었습니다.')
      setLocalPart('')
      setDisplayName('')
      onCreated()
    } catch (err: any) {
      toast.error(err.message || '공용 메일함 생성에 실패했습니다.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Card>
      <CardContent className="space-y-3 p-4">
        <p className="flex items-center gap-1 text-sm font-medium">
          <Plus className="h-4 w-4" />새 공용 메일함
        </p>
        <div className="space-y-1.5">
          <Label htmlFor="shared-local-part">주소</Label>
          <div className="flex items-center gap-2">
            <Input
              id="shared-local-part"
              value={localPart}
              onChange={(e) => setLocalPart(e.target.value)}
              placeholder="hello"
              autoCapitalize="none"
              autoCorrect="off"
              spellCheck={false}
            />
            <span className="shrink-0 text-sm text-muted-foreground">
              @earnlearning.com
            </span>
          </div>
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="shared-display-name">표시 이름</Label>
          <Input
            id="shared-display-name"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            placeholder="고객지원"
          />
        </div>
        <Button className="w-full" onClick={submit} disabled={!valid || submitting}>
          {submitting ? (
            <Loader2 className="mr-1 h-4 w-4 animate-spin" />
          ) : (
            <Plus className="mr-1 h-4 w-4" />
          )}
          만들기
        </Button>
      </CardContent>
    </Card>
  )
}

// ─── 공용 메일함 카드 (권한 목록 + 부여/회수) ────────────────
function SharedMailboxCard({
  mailbox,
  onChanged,
}: {
  mailbox: SharedMailbox
  onChanged: () => void
}) {
  const revoke = async (userId: number) => {
    try {
      await api.post(
        `/admin/mail/shared/${mailbox.address_id}/grants/${userId}/revoke`,
      )
      toast.success('권한을 회수했습니다.')
      onChanged()
    } catch (err: any) {
      toast.error(err.message || '권한 회수에 실패했습니다.')
    }
  }

  const activeGrants = mailbox.grants.filter((g) => !g.revoked)

  return (
    <div className="space-y-3 rounded-lg border p-3">
      <div className="flex items-center gap-2">
        <Users className="h-4 w-4 shrink-0 text-muted-foreground" />
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium">{mailbox.display_name}</p>
          <p className="truncate text-xs text-muted-foreground">{mailbox.email}</p>
        </div>
      </div>

      <div className="space-y-1">
        <p className="text-xs font-medium text-muted-foreground">
          접근 권한 ({activeGrants.length})
        </p>
        {activeGrants.length === 0 ? (
          <p className="text-xs text-muted-foreground">
            아직 권한을 받은 사용자가 없습니다.
          </p>
        ) : (
          activeGrants.map((g) => (
            <div
              key={g.user_id}
              className="flex items-center justify-between rounded-md bg-muted/40 px-2 py-1 text-sm"
            >
              <span className="truncate">{g.user_name}</span>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={() => revoke(g.user_id)}
                title="권한 회수"
              >
                <X className="h-3.5 w-3.5 text-coral" />
              </Button>
            </div>
          ))
        )}
      </div>

      <GrantAdder addressId={mailbox.address_id} onGranted={onChanged} />
    </div>
  )
}

// ─── 사용자 검색 후 권한 부여 ────────────────────────────────
function GrantAdder({
  addressId,
  onGranted,
}: {
  addressId: number
  onGranted: () => void
}) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchUser[]>([])
  const [granting, setGranting] = useState(false)

  useEffect(() => {
    const q = query.trim()
    if (q.length === 0) {
      setResults([])
      return
    }
    const t = setTimeout(async () => {
      try {
        const res = await api.get<SearchUser[]>(
          `/users/search?q=${encodeURIComponent(q)}`,
        )
        setResults(res ?? [])
      } catch {
        setResults([])
      }
    }, 200)
    return () => clearTimeout(t)
  }, [query])

  const grant = async (user: SearchUser) => {
    setGranting(true)
    try {
      await api.post(`/admin/mail/shared/${addressId}/grants`, {
        user_id: user.id,
      })
      toast.success(`${user.name}에게 권한을 부여했습니다.`)
      setQuery('')
      setResults([])
      onGranted()
    } catch (err: any) {
      toast.error(err.message || '권한 부여에 실패했습니다.')
    } finally {
      setGranting(false)
    }
  }

  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-1.5">
        <UserPlus className="h-3.5 w-3.5 text-muted-foreground" />
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="사용자 이름·학번으로 검색"
          className="h-8 text-sm"
          disabled={granting}
        />
      </div>
      {results.length > 0 && (
        <div className="space-y-1 rounded-md border p-1">
          {results.map((u) => (
            <button
              key={u.id}
              type="button"
              disabled={granting}
              onClick={() => grant(u)}
              className="flex w-full items-center gap-2 rounded px-2 py-1 text-left text-sm hover:bg-accent disabled:opacity-50"
            >
              <span className="font-medium">{u.name}</span>
              <span className="text-xs text-muted-foreground">
                {u.department} · {u.student_id}
              </span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
