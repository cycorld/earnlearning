import { useEffect, useState } from 'react'
import { ChevronDown, GraduationCap, Check } from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import { api } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import type { Classroom } from '@/types'

// #159 활성 강의실 스위처 — 내가 속한 강의실 목록에서 현재 강의실을 전환한다.
// 전환 시 지갑·회사·거래소 등 모든 데이터가 해당 강의실 기준으로 바뀌므로 전체 리로드.
export default function ClassroomSwitcher() {
  const { user } = useAuth()
  const [classrooms, setClassrooms] = useState<Classroom[]>([])
  const [switching, setSwitching] = useState(false)

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

  // 강의실이 1개면 이름만 표시 (전환 UI 불필요)
  if (classrooms.length === 1) {
    return (
      <span className="flex min-w-0 items-center gap-1 text-xs text-muted-foreground">
        <GraduationCap className="h-3.5 w-3.5 shrink-0" />
        <span className="max-w-28 truncate sm:max-w-48">{classrooms[0].name}</span>
      </span>
    )
  }

  return (
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
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
