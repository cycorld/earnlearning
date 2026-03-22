import { useState, useEffect } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Shield, Check, X, Loader2, AlertTriangle } from 'lucide-react'

interface AuthorizeInfo {
  client_name: string
  scopes: string[]
  redirect_uri: string
}

interface AuthorizeResult {
  code: string
  redirect_uri: string
  state: string
}

const SCOPE_LABELS: Record<string, { label: string; desc: string }> = {
  'read:profile': { label: '프로필 조회', desc: '이름, 이메일, 학과 등 기본 정보' },
  'write:profile': { label: '프로필 수정', desc: '자기소개, 프로필 사진 변경' },
  'read:wallet': { label: '지갑 조회', desc: '잔액, 거래 내역 확인' },
  'write:wallet': { label: '송금', desc: '다른 사용자에게 송금' },
  'read:posts': { label: '게시물 조회', desc: '피드, 댓글 읽기' },
  'write:posts': { label: '게시물 작성', desc: '글 작성, 댓글, 좋아요' },
  'read:company': { label: '회사 조회', desc: '회사 정보 확인' },
  'write:company': { label: '회사 수정', desc: '회사 정보 변경' },
  'read:market': { label: '마켓 조회', desc: '프리랜서, 거래소, 투자 정보' },
  'write:market': { label: '마켓 활동', desc: '프리랜서 등록, 주문, 투자' },
  'read:notifications': { label: '알림 조회', desc: '알림 목록 확인' },
}

export default function AuthorizePage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { user } = useAuth()

  const clientId = searchParams.get('client_id') || ''
  const redirectUri = searchParams.get('redirect_uri') || ''
  const scope = searchParams.get('scope') || ''
  const state = searchParams.get('state') || ''
  const codeChallenge = searchParams.get('code_challenge') || ''
  const codeChallengeMethod = searchParams.get('code_challenge_method') || ''
  const responseType = searchParams.get('response_type') || 'code'

  const scopes = scope.split(' ').filter(Boolean)

  const [info, setInfo] = useState<AuthorizeInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [authorizing, setAuthorizing] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!clientId || !redirectUri) {
      setError('client_id와 redirect_uri는 필수입니다.')
      setLoading(false)
      return
    }

    api
      .get<AuthorizeInfo>(
        `/oauth/authorize?client_id=${encodeURIComponent(clientId)}&redirect_uri=${encodeURIComponent(redirectUri)}&scope=${encodeURIComponent(scope)}`,
      )
      .then(setInfo)
      .catch((e: any) => setError(e.message || '앱 정보를 불러올 수 없습니다.'))
      .finally(() => setLoading(false))
  }, [clientId, redirectUri, scope])

  const handleAuthorize = async () => {
    setAuthorizing(true)
    try {
      const result = await api.post<AuthorizeResult>('/oauth/authorize', {
        client_id: clientId,
        redirect_uri: redirectUri,
        scopes,
        state,
        code_challenge: codeChallenge,
        code_challenge_method: codeChallengeMethod,
      })

      // Redirect back to the app with the authorization code
      const url = new URL(result.redirect_uri)
      url.searchParams.set('code', result.code)
      if (result.state) url.searchParams.set('state', result.state)
      window.location.href = url.toString()
    } catch (e: any) {
      setError(e.message || '인가에 실패했습니다.')
      setAuthorizing(false)
    }
  }

  const handleDeny = () => {
    if (redirectUri) {
      const url = new URL(redirectUri)
      url.searchParams.set('error', 'access_denied')
      url.searchParams.set('error_description', 'User denied the request')
      if (state) url.searchParams.set('state', state)
      window.location.href = url.toString()
    } else {
      navigate('/feed')
    }
  }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background p-4">
        <Card className="w-full max-w-sm">
          <CardContent className="py-8 text-center">
            <AlertTriangle className="mx-auto mb-3 h-10 w-10 text-destructive" />
            <p className="text-sm font-medium">인가 요청 오류</p>
            <p className="mt-2 text-xs text-muted-foreground">{error}</p>
            <Button variant="outline" className="mt-4" onClick={() => navigate('/feed')}>
              홈으로 돌아가기
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  const hasWriteScope = scopes.some((s) => s.startsWith('write:'))

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-sm">
        <CardContent className="p-6">
          {/* App info */}
          <div className="text-center">
            <div className="mx-auto mb-3 flex h-14 w-14 items-center justify-center rounded-full bg-primary/10">
              <Shield className="h-7 w-7 text-primary" />
            </div>
            <h1 className="text-lg font-semibold">{info?.client_name}</h1>
            <p className="mt-1 text-xs text-muted-foreground">
              이 앱이 아래 권한을 요청합니다
            </p>
          </div>

          <Separator className="my-4" />

          {/* User */}
          <div className="mb-4 rounded-lg bg-muted/50 p-3">
            <p className="text-xs text-muted-foreground">로그인된 계정</p>
            <p className="text-sm font-medium">{user?.name} ({user?.email})</p>
          </div>

          {/* Scopes */}
          <div className="space-y-2">
            <p className="text-xs font-medium text-muted-foreground">요청 권한</p>
            {scopes.map((s) => {
              const scopeInfo = SCOPE_LABELS[s]
              const isWrite = s.startsWith('write:')
              return (
                <div
                  key={s}
                  className={`flex items-start gap-3 rounded-lg border p-3 ${
                    isWrite ? 'border-amber-200 bg-amber-50/50 dark:border-amber-800 dark:bg-amber-950/30' : ''
                  }`}
                >
                  <Check className={`mt-0.5 h-4 w-4 shrink-0 ${isWrite ? 'text-amber-600' : 'text-primary'}`} />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">
                        {scopeInfo?.label || s}
                      </span>
                      {isWrite && (
                        <Badge variant="outline" className="border-amber-300 text-[10px] text-amber-700">
                          쓰기
                        </Badge>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground">
                      {scopeInfo?.desc || s}
                    </p>
                  </div>
                </div>
              )
            })}
          </div>

          {hasWriteScope && (
            <div className="mt-3 rounded-lg border border-amber-200 bg-amber-50 p-2 text-xs text-amber-800 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200">
              <AlertTriangle className="mb-0.5 inline h-3 w-3" /> 이 앱에 쓰기 권한이 포함되어 있습니다. 신뢰할 수 있는 앱인지 확인하세요.
            </div>
          )}

          <Separator className="my-4" />

          {/* Actions */}
          <div className="flex gap-3">
            <Button variant="outline" className="flex-1 gap-1" onClick={handleDeny}>
              <X className="h-4 w-4" />
              거부
            </Button>
            <Button className="flex-1 gap-1" onClick={handleAuthorize} disabled={authorizing}>
              {authorizing ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Check className="h-4 w-4" />
              )}
              허용
            </Button>
          </div>

          <p className="mt-3 text-center text-[10px] text-muted-foreground">
            허용하면 {info?.client_name}이(가) 위 권한으로 EarnLearning 데이터에 접근할 수 있습니다.
            <br />
            설정 → 개발자에서 언제든 앱 접근을 해제할 수 있습니다.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
