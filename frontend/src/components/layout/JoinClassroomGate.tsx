import { useState } from 'react'
import { GraduationCap } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { api, ApiError } from '@/lib/api'

// #159 온보딩 게이트 — 승인된 학생이 아직 어떤 강의실에도 속하지 않았을 때
// 초대 코드 입력 화면을 먼저 보여준다. 조인 성공 시 전체 리로드로 앱 진입.
export default function JoinClassroomGate() {
  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const handleJoin = async (e: { preventDefault: () => void }) => {
    e.preventDefault()
    const trimmed = code.trim().toUpperCase()
    if (!trimmed) return
    setSubmitting(true)
    setError('')
    try {
      await api.post('/classrooms/join', { code: trimmed })
      window.location.reload()
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : '참여에 실패했습니다. 코드를 확인해주세요.',
      )
      setSubmitting(false)
    }
  }

  return (
    <div className="flex min-h-[70vh] items-center justify-center px-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="items-center text-center">
          <div className="mb-2 flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
            <GraduationCap className="h-6 w-6 text-primary" />
          </div>
          <CardTitle className="text-lg">강의실 입장</CardTitle>
          <p className="text-sm text-muted-foreground">
            교수님께 받은 초대 코드를 입력하면 강의실에 입장합니다.
          </p>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleJoin} className="space-y-3">
            <Input
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder="초대 코드 (예: ABC123)"
              className="text-center font-mono uppercase tracking-widest"
              maxLength={6}
              autoFocus
            />
            {error && <p className="text-center text-sm text-destructive">{error}</p>}
            <Button type="submit" className="w-full" disabled={submitting || !code.trim()}>
              {submitting ? '입장 중…' : '강의실 입장'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
