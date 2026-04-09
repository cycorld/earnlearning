import { useEffect, useState } from 'react'
import { api, ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Database,
  Plus,
  Copy,
  Check,
  RefreshCw,
  Trash2,
  Loader2,
  Eye,
  EyeOff,
  AlertTriangle,
} from 'lucide-react'
import { toast } from 'sonner'

// --- Types (backend response shape) ---

type UserDatabase = {
  id: number
  user_id: number
  project_name: string
  db_name: string
  pg_username: string
  host: string
  port: number
  created_at: string
  last_rotated?: string | null
}

type UserDatabaseWithCreds = UserDatabase & {
  password: string
  url: string
}

// --- Main section ---

export function UserDatabasesSection() {
  const [items, setItems] = useState<UserDatabase[]>([])
  const [loading, setLoading] = useState(true)
  const [showNew, setShowNew] = useState(false)
  const [justCreated, setJustCreated] = useState<UserDatabaseWithCreds | null>(null)

  async function refresh() {
    setLoading(true)
    try {
      const list = await api.get<UserDatabase[]>('/users/me/databases')
      setItems(list ?? [])
    } catch (err) {
      toast.error('DB 목록을 불러오지 못했습니다')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void refresh()
  }, [])

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="flex items-center gap-2 text-base">
          <Database className="h-4 w-4" />내 데이터베이스
        </CardTitle>
        <Button
          variant="outline"
          size="sm"
          className="h-7 gap-1 text-xs"
          onClick={() => setShowNew(true)}
        >
          <Plus className="h-3 w-3" />새 DB
        </Button>
      </CardHeader>
      <CardContent className="space-y-2">
        {loading ? (
          <div className="flex justify-center py-4">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        ) : items.length === 0 ? (
          <div className="space-y-2 rounded-md border border-dashed py-6 text-center">
            <p className="text-sm text-muted-foreground">
              아직 만든 데이터베이스가 없어요
            </p>
            <p className="text-xs text-muted-foreground">
              바이브코딩 프로젝트용 PostgreSQL DB 를 받아보세요
            </p>
          </div>
        ) : (
          items.map(db => (
            <DatabaseCard
              key={db.id}
              db={db}
              onDeleted={refresh}
              onRotated={cred => setJustCreated(cred)}
            />
          ))
        )}
      </CardContent>

      {showNew && (
        <NewDatabaseDialog
          onClose={() => setShowNew(false)}
          onCreated={cred => {
            setJustCreated(cred)
            setShowNew(false)
            void refresh()
          }}
        />
      )}
      {justCreated && (
        <CredentialsDialog
          cred={justCreated}
          onClose={() => setJustCreated(null)}
        />
      )}
    </Card>
  )
}

// --- Single DB card ---

function DatabaseCard({
  db,
  onDeleted,
  onRotated,
}: {
  db: UserDatabase
  onDeleted: () => void
  onRotated: (cred: UserDatabaseWithCreds) => void
}) {
  const [showDetails, setShowDetails] = useState(false)
  const [busy, setBusy] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [deleteTypo, setDeleteTypo] = useState('')

  async function handleRotate() {
    setBusy(true)
    try {
      const cred = await api.post<UserDatabaseWithCreds>(
        `/users/me/databases/${db.id}/rotate`,
      )
      onRotated(cred)
      toast.success('비밀번호가 재발급되었습니다')
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '재발급 실패')
    } finally {
      setBusy(false)
    }
  }

  async function handleDelete() {
    if (deleteTypo !== db.db_name) {
      toast.error(`확인을 위해 "${db.db_name}" 을 정확히 입력해주세요`)
      return
    }
    setBusy(true)
    try {
      await api.del(`/users/me/databases/${db.id}`)
      toast.success('DB 가 삭제되었습니다')
      onDeleted()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '삭제 실패')
    } finally {
      setBusy(false)
      setConfirmDelete(false)
    }
  }

  return (
    <div className="rounded-md border p-3">
      <div className="flex items-start justify-between">
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium">{db.project_name}</p>
          <p className="truncate font-mono text-xs text-muted-foreground">
            {db.db_name}
          </p>
        </div>
        <div className="flex gap-1">
          <Button
            variant="ghost"
            size="sm"
            className="h-7 w-7 p-0"
            onClick={() => setShowDetails(v => !v)}
            title="접속 정보"
          >
            {showDetails ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-7 w-7 p-0"
            onClick={handleRotate}
            disabled={busy}
            title="비밀번호 재발급"
          >
            {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-7 w-7 p-0 text-destructive hover:text-destructive"
            onClick={() => setConfirmDelete(true)}
            disabled={busy}
            title="삭제"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {showDetails && (
        <div className="mt-2 space-y-1 rounded-sm bg-muted/30 p-2 text-xs">
          <KV label="host" value={db.host} />
          <KV label="port" value={String(db.port)} />
          <KV label="database" value={db.db_name} />
          <KV label="username" value={db.pg_username} />
          <div className="pt-1 text-muted-foreground">
            비밀번호는 생성/재발급 시 한 번만 표시돼요. 잊어버렸으면{' '}
            <RefreshCw className="inline h-3 w-3" /> 버튼을 눌러 재발급하세요.
          </div>
        </div>
      )}

      {confirmDelete && (
        <Dialog open onOpenChange={() => setConfirmDelete(false)}>
          <DialogContent className="max-w-md">
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <AlertTriangle className="h-4 w-4 text-destructive" />
                정말 삭제할까요?
              </DialogTitle>
              <DialogDescription>
                <span className="font-mono">{db.db_name}</span> 안의 모든 테이블과
                데이터가 <strong>복구 불가능</strong>하게 삭제돼요. 계속하려면
                아래에 DB 이름을 정확히 입력해주세요.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-2">
              <Label htmlFor={`del-${db.id}`}>{db.db_name} 입력</Label>
              <Input
                id={`del-${db.id}`}
                value={deleteTypo}
                onChange={e => setDeleteTypo(e.target.value)}
                placeholder={db.db_name}
                autoFocus
              />
            </div>
            <DialogFooter>
              <Button variant="ghost" onClick={() => setConfirmDelete(false)}>
                취소
              </Button>
              <Button
                variant="destructive"
                onClick={handleDelete}
                disabled={busy || deleteTypo !== db.db_name}
              >
                {busy && <Loader2 className="mr-1 h-4 w-4 animate-spin" />}
                영구 삭제
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </div>
  )
}

function KV({ label, value }: { label: string; value: string }) {
  const [copied, setCopied] = useState(false)
  return (
    <div className="flex items-center gap-2">
      <span className="w-16 shrink-0 text-muted-foreground">{label}:</span>
      <code className="flex-1 truncate rounded bg-background px-1 font-mono">
        {value}
      </code>
      <button
        className="text-muted-foreground hover:text-foreground"
        onClick={() => {
          navigator.clipboard.writeText(value)
          setCopied(true)
          setTimeout(() => setCopied(false), 1500)
        }}
      >
        {copied ? <Check className="h-3 w-3 text-green-600" /> : <Copy className="h-3 w-3" />}
      </button>
    </div>
  )
}

// --- New DB dialog ---

function NewDatabaseDialog({
  onClose,
  onCreated,
}: {
  onClose: () => void
  onCreated: (cred: UserDatabaseWithCreds) => void
}) {
  const [projectName, setProjectName] = useState('')
  const [busy, setBusy] = useState(false)

  const nameValid = /^[a-z][a-z0-9_]{2,31}$/.test(projectName)

  async function handleSubmit() {
    if (!nameValid) {
      toast.error('이름 형식이 올바르지 않습니다')
      return
    }
    setBusy(true)
    try {
      const cred = await api.post<UserDatabaseWithCreds>(
        '/users/me/databases',
        { project_name: projectName },
      )
      onCreated(cred)
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : 'DB 생성 실패')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog open onOpenChange={onClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>새 데이터베이스 만들기</DialogTitle>
          <DialogDescription>
            바이브코딩 프로젝트용 PostgreSQL DB 를 만들어요. 프로젝트명만 정하면 돼요.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <Label htmlFor="project-name">프로젝트명</Label>
          <Input
            id="project-name"
            value={projectName}
            onChange={e => setProjectName(e.target.value.toLowerCase())}
            placeholder="todoapp"
            autoFocus
          />
          <p className="text-xs text-muted-foreground">
            소문자/숫자/밑줄, 3~32자, 소문자로 시작 (예: <code>todoapp</code>, <code>portfolio_v2</code>)
          </p>
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={onClose}>
            취소
          </Button>
          <Button onClick={handleSubmit} disabled={!nameValid || busy}>
            {busy && <Loader2 className="mr-1 h-4 w-4 animate-spin" />}
            생성
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// --- Credentials dialog (shows password once) ---

function CredentialsDialog({
  cred,
  onClose,
}: {
  cred: UserDatabaseWithCreds
  onClose: () => void
}) {
  const [ack, setAck] = useState(false)

  const psqlCmd = `PGPASSWORD='${cred.password}' psql -h ${cred.host} -p ${cred.port} -U ${cred.pg_username} ${cred.db_name}`
  const envLine = `DATABASE_URL=${cred.url}`

  return (
    <Dialog open onOpenChange={() => ack && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>접속 정보</DialogTitle>
          <DialogDescription className="flex items-start gap-2 rounded bg-yellow-50 p-2 text-xs text-yellow-900 dark:bg-yellow-950/40 dark:text-yellow-200">
            <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
            <span>
              비밀번호는 <strong>지금만</strong> 볼 수 있어요. 안전한 곳에 복사하세요.
              잊어버리면 재발급만 가능합니다.
            </span>
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2 rounded-md border bg-muted/30 p-3 font-mono text-xs">
          <KV label="host" value={cred.host} />
          <KV label="port" value={String(cred.port)} />
          <KV label="database" value={cred.db_name} />
          <KV label="username" value={cred.pg_username} />
          <KV label="password" value={cred.password} />
        </div>
        <div className="space-y-2">
          <div>
            <Label className="text-xs">.env 파일</Label>
            <CopyBlock value={envLine} />
          </div>
          <div>
            <Label className="text-xs">psql 명령어</Label>
            <CopyBlock value={psqlCmd} />
          </div>
        </div>
        <label className="flex items-center gap-2 text-xs text-muted-foreground">
          <input
            type="checkbox"
            checked={ack}
            onChange={e => setAck(e.target.checked)}
          />
          비밀번호를 안전한 곳에 저장했어요
        </label>
        <DialogFooter>
          <Button onClick={onClose} disabled={!ack}>
            닫기
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function CopyBlock({ value }: { value: string }) {
  const [copied, setCopied] = useState(false)
  return (
    <div className="flex items-center gap-1 rounded border bg-background p-2">
      <code className="flex-1 overflow-x-auto whitespace-nowrap font-mono text-[10px]">
        {value}
      </code>
      <Button
        variant="ghost"
        size="sm"
        className="h-6 w-6 shrink-0 p-0"
        onClick={() => {
          navigator.clipboard.writeText(value)
          setCopied(true)
          setTimeout(() => setCopied(false), 1500)
        }}
      >
        {copied ? <Check className="h-3 w-3 text-green-600" /> : <Copy className="h-3 w-3" />}
      </Button>
    </div>
  )
}
