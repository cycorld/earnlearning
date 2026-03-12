import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { useAuth } from '@/hooks/use-auth'
import { Loader2 } from 'lucide-react'

interface FormErrors {
  email?: string
  password?: string
  name?: string
  department?: string
  student_id?: string
}

export default function RegisterPage() {
  const navigate = useNavigate()
  const { register } = useAuth()
  const [form, setForm] = useState({
    email: '',
    password: '',
    name: '',
    department: '',
    student_id: '',
  })
  const [errors, setErrors] = useState<FormErrors>({})
  const [apiError, setApiError] = useState('')
  const [loading, setLoading] = useState(false)
  const [showDialog, setShowDialog] = useState(false)

  const validate = (): boolean => {
    const newErrors: FormErrors = {}

    if (!form.email) {
      newErrors.email = '이메일을 입력하세요.'
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) {
      newErrors.email = '올바른 이메일 형식이 아닙니다.'
    }

    if (!form.password) {
      newErrors.password = '비밀번호를 입력하세요.'
    } else if (form.password.length < 8) {
      newErrors.password = '비밀번호는 8자 이상이어야 합니다.'
    }

    if (!form.name) {
      newErrors.name = '이름을 입력하세요.'
    }

    if (!form.department) {
      newErrors.department = '학과를 입력하세요.'
    }

    if (!form.student_id) {
      newErrors.student_id = '학번을 입력하세요.'
    } else if (!/^\d{7,10}$/.test(form.student_id)) {
      newErrors.student_id = '학번은 7~10자리 숫자여야 합니다.'
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleChange = (field: string, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }))
    if (errors[field as keyof FormErrors]) {
      setErrors((prev) => ({ ...prev, [field]: undefined }))
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setApiError('')

    if (!validate()) return

    setLoading(true)
    try {
      await register({
        email: form.email,
        password: form.password,
        name: form.name,
        department: form.department,
        student_id: form.student_id,
      })
      setShowDialog(true)
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : '회원가입에 실패했습니다.'
      setApiError(message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="text-center text-xl">회원가입</CardTitle>
        </CardHeader>
        <form onSubmit={handleSubmit}>
          <CardContent className="space-y-4">
            {apiError && (
              <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                {apiError}
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="email">이메일</Label>
              <Input
                id="email"
                type="email"
                placeholder="이메일을 입력하세요"
                value={form.email}
                onChange={(e) => handleChange('email', e.target.value)}
                autoComplete="email"
              />
              {errors.email && (
                <p className="text-xs text-destructive">{errors.email}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">비밀번호</Label>
              <Input
                id="password"
                type="password"
                placeholder="8자 이상 입력하세요"
                value={form.password}
                onChange={(e) => handleChange('password', e.target.value)}
                autoComplete="new-password"
              />
              {errors.password && (
                <p className="text-xs text-destructive">{errors.password}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="name">이름</Label>
              <Input
                id="name"
                type="text"
                placeholder="이름을 입력하세요"
                value={form.name}
                onChange={(e) => handleChange('name', e.target.value)}
                autoComplete="name"
              />
              {errors.name && (
                <p className="text-xs text-destructive">{errors.name}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="department">학과</Label>
              <Input
                id="department"
                type="text"
                placeholder="학과를 입력하세요"
                value={form.department}
                onChange={(e) => handleChange('department', e.target.value)}
              />
              {errors.department && (
                <p className="text-xs text-destructive">{errors.department}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="student_id">학번</Label>
              <Input
                id="student_id"
                type="text"
                inputMode="numeric"
                placeholder="학번을 입력하세요 (7~10자리)"
                value={form.student_id}
                onChange={(e) => handleChange('student_id', e.target.value)}
              />
              {errors.student_id && (
                <p className="text-xs text-destructive">{errors.student_id}</p>
              )}
            </div>
          </CardContent>
          <CardFooter className="flex flex-col gap-3">
            <Button type="submit" className="w-full" disabled={loading}>
              {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              회원가입
            </Button>
            <p className="text-center text-sm text-muted-foreground">
              이미 계정이 있으신가요?{' '}
              <Link to="/login" className="text-primary hover:underline">
                로그인
              </Link>
            </p>
          </CardFooter>
        </form>
      </Card>

      <Dialog open={showDialog} onOpenChange={setShowDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>회원가입 완료</DialogTitle>
            <DialogDescription>
              관리자 승인을 기다리고 있습니다. 승인이 완료되면 로그인할 수 있습니다.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              onClick={() => {
                setShowDialog(false)
                navigate('/pending', { replace: true })
              }}
              className="w-full"
            >
              확인
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
