import { useEffect, useState } from 'react'
import { ChevronDown, GraduationCap, Check, Plus } from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { api, ApiError } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import type { Classroom } from '@/types'

// #159 활성 강의실 스위처 — 내가 속한 강의실 목록에서 현재 강의실을 전환한다.
// 전환 시 지갑·회사·거래소 등 모든 데이터가 해당 강의실 기준으로 바뀌므로 전체 리로드.
// 목록 맨 아래 "초대 코드로 입장" 으로 새 학기 등 두 번째 강의실에도 참여할 수 있다.
export default function ClassroomSwitcher() {
  const { user } = useAuth()
  const [classrooms, setClassrooms] = useState<Classroom[]>([])
  const [switching, setSwitching] = useState(false)

  // #159 초대 코드 입장 다이얼로그 상태 (JoinClassroomGate 와 동일한 조인 흐름)
  const [joinOpen, setJoinOpen] = useState(false)
  const [joinCode, setJoinCode] = useState('')
  const [joinError, setJoinError] = useState('')
  const [joining, setJoining] = useState(false)

  useEffect(() => {
    if (!user) return
    api
      .get<Classroom[]>('/classrooms')
      .then((list) => setClassrooms(list ?? []))
      .catch(() => setClassrooms([]))
  }, [user])

  if (!user || classrooms.length === 0) return null

  const active = classrooms.find((c) => c.id === user.active_classroom_id)

  const handleSwitch = async (id: number) => {
    if (id === user.active_classroom_id || switching) return
    setSwitching(true)
    try {
      await api.post(`/classrooms/${id}/activate`)
      window.location.reload()
    } catch {
      setSwitching(false)
    }
  }

  // 초대 코드로 새 강의실 참여 → 성공 시 전체 리로드로 새 강의실 컨텍스트 진입
  const handleJoin = async (e: { preventDefault: () => void }) => {
    e.preventDefault()
    const trimmed = joinCode.trim().toUpperCase()
    if (!trimmed) return
    setJoining(true)
    setJoinError('')
    try {
      await api.post('/classrooms/join', { code: trimmed })
      window.location.reload()
    } catch (err) {
      setJoinError(
        err instanceof ApiError ? err.message : '참여에 실패했습니다. 코드를 확인해주세요.',
      )
      setJoining(false)
    }
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            className="h-8 min-w-0 gap-1 px-2 text-xs text-muted-foreground"
            disabled={switching}
          >
            <GraduationCap className="h-3.5 w-3.5 shrink-0" />
            <span className="max-w-24 truncate sm:max-w-44">
              {active ? active.name : '강의실 선택'}
            </span>
            <ChevronDown className="h-3 w-3 shrink-0" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-64">
          {classrooms.map((c) => (
            <DropdownMenuItem
              key={c.id}
              onClick={() => handleSwitch(c.id)}
              className="flex items-center justify-between"
            >
              <span className="truncate">{c.name}</span>
              {c.id === user.active_classroom_id && (
                <Check className="h-4 w-4 shrink-0 text-primary" />
              )}
            </DropdownMenuItem>
          ))}
          <DropdownMenuSeparator />
          {/* #159 새 강의실 참여 진입점 — 이미 강의실이 있는 학생도 두 번째 강의실 입장 가능 */}
          <DropdownMenuItem
            onSelect={() => {
              setJoinError('')
              setJoinCode('')
              setJoinOpen(true)
            }}
            className="flex items-center gap-2 text-muted-foreground"
          >
            <Plus className="h-4 w-4 shrink-0" />
            <span>초대 코드로 입장</span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <Dialog open={joinOpen} onOpenChange={setJoinOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>강의실 입장</DialogTitle>
            <DialogDescription>
              교수님께 받은 초대 코드를 입력하면 새 강의실에 입장합니다.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleJoin} className="space-y-3">
            <Input
              value={joinCode}
              onChange={(e) => setJoinCode(e.target.value)}
              placeholder="초대 코드 (예: ABC123)"
              className="text-center font-mono uppercase tracking-widest"
              maxLength={6}
              autoFocus
            />
            {joinError && <p className="text-center text-sm text-destructive">{joinError}</p>}
            <Button type="submit" className="w-full" disabled={joining || !joinCode.trim()}>
              {joining ? '입장 중…' : '강의실 입장'}
            </Button>
          </form>
        </DialogContent>
      </Dialog>
    </>
  )
}
