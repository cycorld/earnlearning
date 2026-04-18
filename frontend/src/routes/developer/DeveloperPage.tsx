import { useState, useEffect, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  ArrowLeft,
  Plus,
  Trash2,
  Copy,
  Check,
  Key,
  Globe,
  Shield,
  Loader2,
  AlertCircle,
  Sparkles,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'

interface OAuthClient {
  id: string
  name: string
  description: string
  redirect_uris: string[]
  scopes: string[]
  status: string
  created_at: string
}

interface RegisterResult {
  client_id: string
  client_secret: string
  name: string
}

const AVAILABLE_SCOPES = [
  { value: 'read:profile', label: '프로필 조회' },
  { value: 'write:profile', label: '프로필 수정' },
  { value: 'read:wallet', label: '지갑 조회' },
  { value: 'write:wallet', label: '송금' },
  { value: 'read:posts', label: '게시물 조회' },
  { value: 'write:posts', label: '게시물 작성' },
  { value: 'read:company', label: '회사 조회' },
  { value: 'write:company', label: '회사 수정' },
  { value: 'read:market', label: '마켓 조회' },
  { value: 'write:market', label: '마켓 활동' },
  { value: 'read:notifications', label: '알림 조회' },
]

export default function DeveloperPage() {
  const [clients, setClients] = useState<OAuthClient[]>([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [creating, setCreating] = useState(false)
  const [newClient, setNewClient] = useState<RegisterResult | null>(null)
  const [copiedField, setCopiedField] = useState<string | null>(null)
  const [error, setError] = useState('')

  // Form state
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [redirectUris, setRedirectUris] = useState('')
  const [selectedScopes, setSelectedScopes] = useState<string[]>(['read:profile'])

  const fetchClients = useCallback(async () => {
    try {
      const data = await api.get<OAuthClient[]>('/oauth/clients')
      setClients(data || [])
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchClients()
  }, [fetchClients])

  const handleCreate = async () => {
    setError('')
    if (!name.trim()) {
      setError('앱 이름을 입력해주세요.')
      return
    }
    const uris = redirectUris
      .split('\n')
      .map((u) => u.trim())
      .filter(Boolean)
    if (uris.length === 0) {
      setError('Redirect URI를 1개 이상 입력해주세요.')
      return
    }
    if (selectedScopes.length === 0) {
      setError('스코프를 1개 이상 선택해주세요.')
      return
    }

    setCreating(true)
    try {
      const result = await api.post<RegisterResult>('/oauth/clients', {
        name: name.trim(),
        description: description.trim(),
        redirect_uris: uris,
        scopes: selectedScopes,
      })
      setNewClient(result)
      fetchClients()
    } catch (e: any) {
      setError(e.message || '앱 등록에 실패했습니다.')
    } finally {
      setCreating(false)
    }
  }

  const handleDelete = async (clientId: string, clientName: string) => {
    if (!confirm(`"${clientName}" 앱을 삭제하시겠습니까?\n발급된 모든 토큰이 폐기됩니다.`)) return
    try {
      await api.del(`/oauth/clients/${clientId}`)
      setClients((prev) => prev.filter((c) => c.id !== clientId))
    } catch {
      // ignore
    }
  }

  const copyToClipboard = (text: string, field: string) => {
    navigator.clipboard.writeText(text)
    setCopiedField(field)
    setTimeout(() => setCopiedField(null), 2000)
  }

  const resetForm = () => {
    setName('')
    setDescription('')
    setRedirectUris('')
    setSelectedScopes(['read:profile'])
    setNewClient(null)
    setError('')
  }

  const toggleScope = (scope: string) => {
    setSelectedScopes((prev) =>
      prev.includes(scope) ? prev.filter((s) => s !== scope) : [...prev, scope],
    )
  }

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="sm" asChild>
          <Link to="/profile">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div className="flex-1">
          <h1 className="text-lg font-semibold">개발자 설정</h1>
          <p className="text-xs text-muted-foreground">OAuth 앱 관리</p>
        </div>
        <Dialog
          open={dialogOpen}
          onOpenChange={(open) => {
            setDialogOpen(open)
            if (!open) resetForm()
          }}
        >
          <DialogTrigger asChild>
            <Button size="sm" className="gap-1">
              <Plus className="h-4 w-4" />앱 등록
            </Button>
          </DialogTrigger>
          <DialogContent className="max-h-[90vh] overflow-y-auto">
            {newClient ? (
              <>
                <DialogHeader>
                  <DialogTitle>앱 등록 완료</DialogTitle>
                </DialogHeader>
                <div className="space-y-4">
                  <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200">
                    <AlertCircle className="mb-1 inline h-4 w-4" /> client_secret은 이 화면에서만 확인할 수 있습니다. 안전한 곳에 저장해주세요.
                  </div>
                  <div>
                    <Label className="text-xs text-muted-foreground">Client ID</Label>
                    <div className="flex items-center gap-2">
                      <code className="flex-1 rounded bg-muted px-2 py-1 text-xs break-all">
                        {newClient.client_id}
                      </code>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => copyToClipboard(newClient.client_id, 'id')}
                      >
                        {copiedField === 'id' ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
                      </Button>
                    </div>
                  </div>
                  <div>
                    <Label className="text-xs text-muted-foreground">Client Secret</Label>
                    <div className="flex items-center gap-2">
                      <code className="flex-1 rounded bg-muted px-2 py-1 text-xs break-all">
                        {newClient.client_secret}
                      </code>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => copyToClipboard(newClient.client_secret, 'secret')}
                      >
                        {copiedField === 'secret' ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
                      </Button>
                    </div>
                  </div>
                </div>
                <DialogFooter>
                  <Button onClick={() => setDialogOpen(false)}>확인</Button>
                </DialogFooter>
              </>
            ) : (
              <>
                <DialogHeader>
                  <DialogTitle>새 앱 등록</DialogTitle>
                </DialogHeader>
                <div className="space-y-4">
                  <div>
                    <Label htmlFor="app-name">앱 이름 *</Label>
                    <Input
                      id="app-name"
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      placeholder="내 앱 이름"
                    />
                  </div>
                  <div>
                    <Label htmlFor="app-desc">설명</Label>
                    <Textarea
                      id="app-desc"
                      value={description}
                      onChange={(e) => setDescription(e.target.value)}
                      placeholder="앱 설명 (선택)"
                      rows={2}
                    />
                  </div>
                  <div>
                    <Label htmlFor="app-uris">Redirect URI * (줄바꿈 구분)</Label>
                    <Textarea
                      id="app-uris"
                      value={redirectUris}
                      onChange={(e) => setRedirectUris(e.target.value)}
                      placeholder="http://localhost:3000/callback"
                      rows={3}
                    />
                  </div>
                  <div>
                    <Label>요청 스코프 *</Label>
                    <div className="mt-2 flex flex-wrap gap-2">
                      {AVAILABLE_SCOPES.map((s) => (
                        <button
                          key={s.value}
                          type="button"
                          onClick={() => toggleScope(s.value)}
                          className={`rounded-full border px-3 py-1 text-xs transition-colors ${
                            selectedScopes.includes(s.value)
                              ? 'border-primary bg-primary/10 text-primary'
                              : 'border-border text-muted-foreground hover:border-primary/50'
                          }`}
                        >
                          {s.label}
                        </button>
                      ))}
                    </div>
                  </div>
                  {error && (
                    <p className="text-sm text-destructive">{error}</p>
                  )}
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={() => setDialogOpen(false)}>
                    취소
                  </Button>
                  <Button onClick={handleCreate} disabled={creating}>
                    {creating && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                    등록
                  </Button>
                </DialogFooter>
              </>
            )}
          </DialogContent>
        </Dialog>
      </div>

      {/* Client list */}
      {loading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      ) : clients.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <Key className="mx-auto mb-3 h-10 w-10 text-muted-foreground/50" />
            <p className="text-sm text-muted-foreground">등록된 앱이 없습니다.</p>
            <p className="mt-1 text-xs text-muted-foreground">
              "앱 등록" 버튼을 눌러 OAuth 앱을 만들어보세요.
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {clients.map((client) => (
            <Card key={client.id}>
              <CardHeader className="flex flex-row items-start justify-between pb-2">
                <div className="min-w-0 flex-1">
                  <CardTitle className="text-sm">{client.name}</CardTitle>
                  {client.description && (
                    <p className="mt-1 text-xs text-muted-foreground">{client.description}</p>
                  )}
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-destructive hover:text-destructive"
                  onClick={() => handleDelete(client.id, client.name)}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </CardHeader>
              <CardContent className="space-y-3">
                {/* Client ID */}
                <div className="flex items-center gap-2">
                  <Key className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                  <code className="min-w-0 flex-1 truncate text-xs text-muted-foreground">
                    {client.id}
                  </code>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-6 w-6 p-0"
                    onClick={() => copyToClipboard(client.id, client.id)}
                  >
                    {copiedField === client.id ? (
                      <Check className="h-3 w-3 text-green-600" />
                    ) : (
                      <Copy className="h-3 w-3" />
                    )}
                  </Button>
                </div>

                {/* Redirect URIs */}
                <div className="flex items-start gap-2">
                  <Globe className="mt-0.5 h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                  <div className="min-w-0 flex-1">
                    {client.redirect_uris?.map((uri, i) => (
                      <p key={i} className="truncate text-xs text-muted-foreground">
                        {uri}
                      </p>
                    ))}
                  </div>
                </div>

                {/* Scopes */}
                <div className="flex items-start gap-2">
                  <Shield className="mt-0.5 h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                  <div className="flex flex-wrap gap-1">
                    {client.scopes?.map((scope) => (
                      <Badge key={scope} variant="secondary" className="text-[10px]">
                        {scope}
                      </Badge>
                    ))}
                  </div>
                </div>

                <Separator />
                <p className="text-[10px] text-muted-foreground">
                  생성: {new Date(client.created_at).toLocaleDateString('ko-KR')}
                </p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* AI Prompt Card */}
      <AiPromptCard clients={clients} copiedField={copiedField} onCopy={copyToClipboard} />

      {/* Info */}
      <Card>
        <CardContent className="p-4">
          <h3 className="mb-2 text-sm font-medium">OAuth 연동 가이드</h3>
          <ol className="space-y-1 text-xs text-muted-foreground">
            <li>1. 위에서 앱을 등록하고 client_id를 받으세요.</li>
            <li>2. 사용자를 <code className="rounded bg-muted px-1">/oauth/authorize</code> 페이지로 리다이렉트하세요.</li>
            <li>3. 사용자가 허용하면 redirect_uri로 인가 코드가 전달됩니다.</li>
            <li>4. 인가 코드로 <code className="rounded bg-muted px-1">POST /api/oauth/token</code> 을 호출해 토큰을 받으세요.</li>
          </ol>
          <div className="mt-3">
            <a
              href="/docs"
              target="_blank"
              className="text-xs text-primary hover:underline"
            >
              API 문서 보기 →
            </a>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

// ============================================================
// AI 프롬프트 카드 — 등록된 앱 정보를 자동으로 반영
// ============================================================

function AiPromptCard({
  clients,
  copiedField,
  onCopy,
}: {
  clients: OAuthClient[]
  copiedField: string | null
  onCopy: (text: string, field: string) => void
}) {
  const [expanded, setExpanded] = useState(false)
  const [selectedClientIdx, setSelectedClientIdx] = useState(0)

  const client = clients[selectedClientIdx]

  const baseUrl = window.location.origin

  const prompt = client
    ? `내 웹 서비스에 EarnLearning OAuth 로그인을 연동해줘.

## EarnLearning OAuth 정보

- **API 서버**: ${baseUrl}
- **API 문서 (OpenAPI JSON)**: ${baseUrl}/docs/openapi.json
- **OAuth 방식**: Authorization Code + PKCE (S256)

### 내 앱 설정
- **client_id**: ${client.id}
- **redirect_uri**: ${client.redirect_uris?.[0] || 'http://localhost:3000/callback'}
- **요청 스코프**: ${client.scopes?.join(' ') || 'read:profile'}

## OAuth 플로우

### 1단계: 인가 요청
사용자를 아래 URL로 리다이렉트:
\`\`\`
${baseUrl}/oauth/authorize?client_id=${client.id}&redirect_uri=${encodeURIComponent(client.redirect_uris?.[0] || '')}&scope=${encodeURIComponent(client.scopes?.join(' ') || 'read:profile')}&response_type=code&state={랜덤문자열}&code_challenge={SHA256(code_verifier)의 Base64URL}&code_challenge_method=S256
\`\`\`

### 2단계: 토큰 교환
사용자가 허용하면 redirect_uri로 \`?code=xxx&state=xxx\` 가 전달됨.
이 code로 토큰 교환:
\`\`\`
POST ${baseUrl}/api/oauth/token
Content-Type: application/json

{
  "grant_type": "authorization_code",
  "code": "받은_인가코드",
  "client_id": "${client.id}",
  "redirect_uri": "${client.redirect_uris?.[0] || ''}",
  "code_verifier": "1단계에서_생성한_code_verifier"
}
\`\`\`
→ access_token, refresh_token 반환

### 3단계: API 호출
\`\`\`
GET ${baseUrl}/api/oauth/userinfo
Authorization: Bearer {access_token}
\`\`\`
→ 사용자 id, email, name, department 반환

### 4단계: 토큰 갱신
\`\`\`
POST ${baseUrl}/api/oauth/token
{ "grant_type": "refresh_token", "refresh_token": "...", "client_id": "${client.id}", "client_secret": "앱등록시_받은_시크릿" }
\`\`\`

## 사용 가능한 API (스코프별)
${client.scopes?.includes('read:profile') ? '- GET /api/oauth/userinfo — 프로필 조회\n' : ''}${client.scopes?.includes('read:wallet') ? '- GET /api/wallet — 지갑 잔액 조회\n- GET /api/wallet/transactions — 거래 내역\n' : ''}${client.scopes?.includes('read:posts') ? '- GET /api/posts — 게시물 목록\n- GET /api/posts/:id/comments — 댓글 조회\n' : ''}${client.scopes?.includes('write:posts') ? '- POST /api/channels/:channelId/posts — 게시물 작성\n- POST /api/posts/:id/comments — 댓글 작성\n- POST /api/posts/:id/like — 좋아요\n' : ''}${client.scopes?.includes('read:company') ? '- GET /api/companies/:id — 회사 정보\n' : ''}${client.scopes?.includes('read:market') ? '- GET /api/freelance/jobs — 프리랜서 잡 목록\n- GET /api/exchange/companies — 거래소 상장 회사\n- GET /api/investment/rounds — 투자 라운드\n' : ''}${client.scopes?.includes('read:notifications') ? '- GET /api/notifications — 알림 목록\n' : ''}
## 구현 요구사항
- PKCE (S256) 필수 — code_verifier는 최소 43자 랜덤 문자열, code_challenge = Base64URL(SHA256(code_verifier))
- access_token 유효기간: 1시간, refresh_token: 30일
- 모든 API 응답 형식: { "success": bool, "data": ..., "error": { "code": "...", "message": "..." } }
- 전체 API 스펙은 ${baseUrl}/docs/openapi.json 에서 확인 가능

프론트엔드에 "EarnLearning으로 로그인" 버튼을 만들고, 위 플로우를 구현해줘.`
    : '먼저 위에서 앱을 등록해주세요.'

  return (
    <Card>
      <CardContent className="p-4">
        <button
          onClick={() => setExpanded(!expanded)}
          className="flex w-full items-center gap-2"
        >
          <Sparkles className="h-4 w-4 text-violet-500" />
          <h3 className="flex-1 text-left text-sm font-medium">
            AI에게 시킬 연동 프롬프트
          </h3>
          {expanded ? (
            <ChevronUp className="h-4 w-4 text-muted-foreground" />
          ) : (
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          )}
        </button>

        {expanded && (
          <div className="mt-3 space-y-3">
            <p className="text-xs text-muted-foreground">
              아래 프롬프트를 복사해서 Claude, ChatGPT 등 AI에 붙여넣으면,
              내 앱에 EarnLearning OAuth 로그인을 자동으로 구현해줍니다.
            </p>

            {clients.length > 1 && (
              <div className="flex gap-2">
                {clients.map((c, i) => (
                  <button
                    key={c.id}
                    onClick={() => setSelectedClientIdx(i)}
                    className={`rounded-full border px-3 py-1 text-xs transition-colors ${
                      i === selectedClientIdx
                        ? 'border-primary bg-primary/10 text-primary'
                        : 'border-border text-muted-foreground'
                    }`}
                  >
                    {c.name}
                  </button>
                ))}
              </div>
            )}

            <div className="relative">
              <pre className="max-h-64 overflow-auto rounded-lg bg-muted p-3 text-xs leading-relaxed whitespace-pre-wrap">
                {prompt}
              </pre>
              {client && (
                <Button
                  variant="secondary"
                  size="sm"
                  className="absolute right-2 top-2 gap-1"
                  onClick={() => onCopy(prompt, 'prompt')}
                >
                  {copiedField === 'prompt' ? (
                    <>
                      <Check className="h-3 w-3 text-green-600" />
                      복사됨
                    </>
                  ) : (
                    <>
                      <Copy className="h-3 w-3" />
                      복사
                    </>
                  )}
                </Button>
              )}
            </div>

            {client && (
              <p className="text-[10px] text-muted-foreground">
                * client_secret은 보안상 프롬프트에 포함되지 않습니다. 서버 사이드 코드에서만 사용하세요.
              </p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
