import { useCallback, useEffect, useState } from 'react'
import {
  BarChart3,
  Check,
  ExternalLink,
  Globe,
  Loader2,
  Pencil,
  Plus,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'

import { api } from '@/lib/api'
import type { CompanyService } from '@/types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import { shortenUrl } from '@/lib/urls'

// #181: 검증 상태 → 뱃지
const validationLabels: Record<CompanyService['validation_status'], string> = {
  unvalidated: '미검증',
  valid: '검증됨',
  invalid: '검증 실패',
}

const validationVariants: Record<
  CompanyService['validation_status'],
  'secondary' | 'highlight' | 'destructive'
> = {
  unvalidated: 'secondary',
  valid: 'highlight',
  invalid: 'destructive',
}

// #181: Rybbit(애널리틱스) 연동 상태 → 뱃지
const rybbitLabels: Record<CompanyService['rybbit_status'], string> = {
  not_connected: '미연동',
  connected: '연동됨',
  needs_reconnect: '재연동 필요',
}

const rybbitVariants: Record<
  CompanyService['rybbit_status'],
  'secondary' | 'highlight' | 'coral'
> = {
  not_connected: 'secondary',
  connected: 'highlight',
  needs_reconnect: 'coral',
}

// #181: 서버가 내려주는 실패 사유 머신 토큰 → 학생이 읽을 한국어
const validationDetailLabels: Record<string, string> = {
  private_ip: '내부/사설 주소는 사용할 수 없어요',
  dns_error: '도메인을 찾을 수 없어요',
  timeout: '응답 시간 초과',
  connect_error: '접속할 수 없어요',
  too_many_redirects: '리다이렉트가 너무 많아요',
  redirect_not_https: '허용되지 않는 리다이렉트예요',
  redirect_blocked: '허용되지 않는 리다이렉트예요',
}

/** 모르는 토큰은 그대로 노출한다 (서버가 새 사유를 추가해도 화면이 비지 않도록). */
function validationDetailLabel(detail: string): string {
  if (!detail) return ''
  const known = validationDetailLabels[detail]
  if (known) return known
  const httpStatus = /^http_status_(\d+)$/.exec(detail)
  if (httpStatus) return `HTTP 오류 (${httpStatus[1]})`
  return detail
}

/** 이름이 비어 있으면 호스트로 대체. */
function serviceLabel(service: CompanyService): string {
  const name = service.name?.trim()
  if (name) return name
  try {
    return new URL(service.url).host
  } catch {
    return service.url
  }
}

type BusyKind = 'validate' | 'connect' | 'delete'

interface Props {
  companyId: number
  isOwner: boolean
}

/**
 * #181: 회사에 등록된 서비스 목록.
 * 서비스마다 자체 URL·검증 상태·Rybbit 연동 상태를 가진다.
 * 조회는 누구나, 추가/수정/검증/연동은 대표(소유자)만.
 */
export function CompanyServicesSection({ companyId, isOwner }: Props) {
  const [services, setServices] = useState<CompanyService[]>([])
  const [loading, setLoading] = useState(true)
  // 서비스별 진행 상태 — 스피너/비활성화가 행 단위로만 적용되도록 id → kind 로 관리한다.
  const [busy, setBusy] = useState<Record<number, BusyKind>>({})

  const [addForm, setAddForm] = useState({ name: '', url: '' })
  const [adding, setAdding] = useState(false)

  const [editing, setEditing] = useState<CompanyService | null>(null)
  const [editForm, setEditForm] = useState({ name: '', url: '' })
  const [editSaving, setEditSaving] = useState(false)

  const load = useCallback(async () => {
    try {
      const data = await api.get<CompanyService[]>(`/companies/${companyId}/services`)
      // 방어: 예상 밖 응답(객체/null)이 와도 목록 렌더가 깨지지 않게 한다.
      setServices(Array.isArray(data) ? data : [])
    } catch {
      setServices([])
    } finally {
      setLoading(false)
    }
  }, [companyId])

  useEffect(() => {
    load()
  }, [load])

  function startBusy(id: number, kind: BusyKind) {
    setBusy((prev) => ({ ...prev, [id]: kind }))
  }

  function endBusy(id: number) {
    setBusy((prev) => {
      const next = { ...prev }
      delete next[id]
      return next
    })
  }

  async function handleValidate(service: CompanyService) {
    startBusy(service.id, 'validate')
    try {
      // 검사가 끝나면 결과가 invalid 여도 200 이다 → 응답의 상태로 판단한다.
      const updated = await api.post<CompanyService>(
        `/companies/${companyId}/services/${service.id}/validate`,
      )
      if (updated?.validation_status === 'invalid') {
        toast.error(
          validationDetailLabel(updated.validation_detail) ||
            '서비스 주소를 확인할 수 없어요',
        )
      } else {
        toast.success('검증 완료')
      }
      await load()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '검증에 실패했습니다.')
    } finally {
      endBusy(service.id)
    }
  }

  async function handleConnect(service: CompanyService) {
    startBusy(service.id, 'connect')
    try {
      await api.post(`/companies/${companyId}/services/${service.id}/rybbit/connect`)
      toast.success('연동 완료')
      await load()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Rybbit 연동에 실패했습니다.')
    } finally {
      endBusy(service.id)
    }
  }

  async function handleDelete(service: CompanyService) {
    // 되돌릴 수 없는 작업이라 먼저 확인한다.
    if (
      !window.confirm('서비스를 삭제할까요? Rybbit 사이트는 함께 삭제되지 않아요.')
    ) {
      return
    }
    startBusy(service.id, 'delete')
    try {
      await api.del(`/companies/${companyId}/services/${service.id}`)
      toast.success('서비스를 삭제했어요')
      await load()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '서비스 삭제에 실패했습니다.')
    } finally {
      endBusy(service.id)
    }
  }

  async function handleAdd(e: React.FormEvent) {
    e.preventDefault()
    const name = addForm.name.trim()
    const url = addForm.url.trim()
    // 서버가 최종 판단하지만, 명백한 형식 오류는 왕복 없이 막는다.
    if (!url.startsWith('https://')) {
      toast.error('https:// URL만 등록할 수 있어요')
      return
    }
    setAdding(true)
    try {
      await api.post(`/companies/${companyId}/services`, { name, url })
      toast.success('서비스를 등록했어요.')
      setAddForm({ name: '', url: '' })
      await load()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '서비스 등록에 실패했습니다.')
    } finally {
      setAdding(false)
    }
  }

  function openEdit(service: CompanyService) {
    setEditForm({ name: service.name, url: service.url })
    setEditing(service)
  }

  async function handleEditSave(e: React.FormEvent) {
    e.preventDefault()
    if (!editing) return
    const name = editForm.name.trim()
    const url = editForm.url.trim()
    if (!url.startsWith('https://')) {
      toast.error('https:// URL만 등록할 수 있어요')
      return
    }
    setEditSaving(true)
    try {
      // URL 이 바뀌면 서버가 검증을 초기화하고 연동 상태를 재연동 필요로 바꾼다 → 다시 조회.
      await api.put(`/companies/${companyId}/services/${editing.id}`, { name, url })
      toast.success('서비스 정보를 수정했어요.')
      setEditing(null)
      await load()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '서비스 수정에 실패했습니다.')
    } finally {
      setEditSaving(false)
    }
  }

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Globe className="h-4 w-4 text-primary" />
            서비스
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {loading ? (
            <div className="flex justify-center py-4">
              <Spinner />
            </div>
          ) : (
            <>
              {services.length === 0 ? (
                <p className="py-2 text-center text-sm text-muted-foreground">
                  등록된 서비스가 없어요.
                </p>
              ) : (
                <div className="space-y-3">
                  {services.map((service) => (
                    <ServiceRow
                      key={service.id}
                      service={service}
                      isOwner={isOwner}
                      busy={busy[service.id] ?? null}
                      onValidate={() => handleValidate(service)}
                      onConnect={() => handleConnect(service)}
                      onEdit={() => openEdit(service)}
                      onDelete={() => handleDelete(service)}
                    />
                  ))}
                </div>
              )}

              {isOwner && (
                <form onSubmit={handleAdd} className="space-y-3 rounded-lg border p-3">
                  <div className="space-y-1.5">
                    <Label htmlFor="new-service-name">서비스 이름</Label>
                    <Input
                      id="new-service-name"
                      value={addForm.name}
                      onChange={(e) => setAddForm({ ...addForm, name: e.target.value })}
                      placeholder="내 서비스"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="new-service-url">서비스 주소</Label>
                    <Input
                      id="new-service-url"
                      value={addForm.url}
                      onChange={(e) => setAddForm({ ...addForm, url: e.target.value })}
                      placeholder="https://my-app.com"
                      autoCapitalize="none"
                      autoCorrect="off"
                      spellCheck={false}
                    />
                    <p className="text-xs text-muted-foreground">
                      https:// 로 시작하는 주소만 등록할 수 있어요.
                    </p>
                  </div>
                  <Button type="submit" size="sm" disabled={adding}>
                    {adding ? (
                      <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
                    ) : (
                      <Plus className="mr-1 h-3.5 w-3.5" />
                    )}
                    추가
                  </Button>
                </form>
              )}
            </>
          )}
        </CardContent>
      </Card>

      {/* 서비스 수정 다이얼로그 (대표 전용) */}
      <Dialog open={editing !== null} onOpenChange={(open) => !open && setEditing(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>서비스 수정</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleEditSave} className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="edit-service-name">서비스 이름</Label>
              <Input
                id="edit-service-name"
                value={editForm.name}
                onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
                placeholder="내 서비스"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="edit-service-url">서비스 주소</Label>
              <Input
                id="edit-service-url"
                value={editForm.url}
                onChange={(e) => setEditForm({ ...editForm, url: e.target.value })}
                placeholder="https://my-app.com"
                autoCapitalize="none"
                autoCorrect="off"
                spellCheck={false}
              />
              <p className="text-xs text-muted-foreground">
                주소를 바꾸면 검증과 Rybbit 연동을 다시 해야 해요.
              </p>
            </div>
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => setEditing(null)}>
                취소
              </Button>
              <Button type="submit" disabled={editSaving}>
                {editSaving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                저장
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </>
  )
}

function ServiceRow({
  service,
  isOwner,
  busy,
  onValidate,
  onConnect,
  onEdit,
  onDelete,
}: {
  service: CompanyService
  isOwner: boolean
  busy: BusyKind | null
  onValidate: () => void
  onConnect: () => void
  onEdit: () => void
  onDelete: () => void
}) {
  const rowBusy = busy !== null
  const label = serviceLabel(service)
  const detail =
    service.validation_status === 'invalid'
      ? validationDetailLabel(service.validation_detail)
      : ''

  const connectLabel =
    service.rybbit_status === 'connected'
      ? '연동 완료'
      : service.rybbit_status === 'needs_reconnect'
        ? '재연동'
        : '연동하기'

  // connect_ready 는 서버 계산값 — 클라이언트가 상태로 활성화를 추측하지 않는다.
  const connectDisabled =
    service.rybbit_status === 'connected' || !service.connect_ready || rowBusy

  return (
    <div className="space-y-2 rounded-lg border p-3">
      <div className="min-w-0">
        <p className="truncate text-sm font-medium">{label}</p>
        <a
          href={service.url}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
        >
          <ExternalLink className="h-3 w-3" />
          {shortenUrl(service.url)}
        </a>
      </div>

      <div className="flex flex-wrap items-center gap-1.5">
        <Badge variant={validationVariants[service.validation_status] || 'secondary'}>
          {validationLabels[service.validation_status] || service.validation_status}
        </Badge>
        <Badge variant={rybbitVariants[service.rybbit_status] || 'secondary'}>
          {rybbitLabels[service.rybbit_status] || service.rybbit_status}
        </Badge>
      </div>

      {detail && <p className="text-xs text-destructive">{detail}</p>}

      {isOwner && (
        <div className="flex flex-wrap items-center gap-2 pt-1">
          <Button
            variant="outline"
            size="sm"
            aria-label={`${label} 검증하기`}
            disabled={rowBusy}
            onClick={onValidate}
          >
            {busy === 'validate' ? (
              <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
            ) : (
              <Check className="mr-1 h-3.5 w-3.5" />
            )}
            검증하기
          </Button>
          <Button
            size="sm"
            aria-label={`${label} ${connectLabel}`}
            disabled={connectDisabled}
            onClick={onConnect}
          >
            {busy === 'connect' ? (
              <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
            ) : (
              <BarChart3 className="mr-1 h-3.5 w-3.5" />
            )}
            {connectLabel}
          </Button>
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={`${label} 수정`}
            disabled={rowBusy}
            onClick={onEdit}
          >
            <Pencil className="h-3.5 w-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon-sm"
            className="text-destructive hover:text-destructive"
            aria-label={`${label} 삭제`}
            disabled={rowBusy}
            onClick={onDelete}
          >
            {busy === 'delete' ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Trash2 className="h-3.5 w-3.5" />
            )}
          </Button>
        </div>
      )}
    </div>
  )
}
