import { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { useAuth } from '@/hooks/use-auth'
import { Loader2 } from 'lucide-react'

const SAVED_EMAIL_KEY = 'el_saved_email'
const REMEMBER_EMAIL_KEY = 'el_remember_email'
const REMEMBER_ME_KEY = 'el_remember_me'

export default function LoginPage() {
  const navigate = useNavigate()
  const { login } = useAuth()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [rememberEmail, setRememberEmail] = useState(true)
  const [rememberMe, setRememberMe] = useState(false)

  useEffect(() => {
    const savedRememberEmail = localStorage.getItem(REMEMBER_EMAIL_KEY)
    // Default to true if not set
    const shouldRememberEmail = savedRememberEmail === null || savedRememberEmail === 'true'
    setRememberEmail(shouldRememberEmail)

    if (shouldRememberEmail) {
      const savedEmail = localStorage.getItem(SAVED_EMAIL_KEY)
      if (savedEmail) setEmail(savedEmail)
    }

    const savedRememberMe = localStorage.getItem(REMEMBER_ME_KEY)
    if (savedRememberMe === 'true') setRememberMe(true)
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      // Save/clear email preference
      localStorage.setItem(REMEMBER_EMAIL_KEY, String(rememberEmail))
      if (rememberEmail) {
        localStorage.setItem(SAVED_EMAIL_KEY, email)
      } else {
        localStorage.removeItem(SAVED_EMAIL_KEY)
      }

      // Save remember me preference
      localStorage.setItem(REMEMBER_ME_KEY, String(rememberMe))

      await login(email, password, rememberMe)
      navigate('/feed', { replace: true })
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : '로그인에 실패했습니다.'
      setError(message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-center text-xl">로그인</CardTitle>
      </CardHeader>
      <form onSubmit={handleSubmit}>
        <CardContent className="space-y-4">
          {error && (
            <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
              {error}
            </div>
          )}
          <div className="space-y-2">
            <Label htmlFor="email">이메일</Label>
            <Input
              id="email"
              type="email"
              placeholder="이메일을 입력하세요"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoComplete="email"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="password">비밀번호</Label>
            <Input
              id="password"
              type="password"
              placeholder="비밀번호를 입력하세요"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              autoComplete="current-password"
            />
          </div>
          <div className="flex items-center justify-between text-sm">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={rememberEmail}
                onChange={(e) => setRememberEmail(e.target.checked)}
                className="h-4 w-4 rounded border-gray-300 accent-primary"
              />
              <span className="text-muted-foreground">아이디 저장</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={rememberMe}
                onChange={(e) => setRememberMe(e.target.checked)}
                className="h-4 w-4 rounded border-gray-300 accent-primary"
              />
              <span className="text-muted-foreground">로그인 유지</span>
            </label>
          </div>
        </CardContent>
        <CardFooter className="flex flex-col gap-3">
          <Button type="submit" className="w-full" disabled={loading}>
            {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            로그인
          </Button>
          <p className="text-center text-sm text-muted-foreground">
            계정이 없으신가요?{' '}
            <Link to="/register" className="text-primary hover:underline">
              회원가입
            </Link>
          </p>
        </CardFooter>
      </form>
    </Card>
  )
}
