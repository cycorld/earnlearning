import { useEffect, useState } from 'react'
import { Outlet } from 'react-router-dom'
import Header from './Header'
import BottomNav from './BottomNav'
import JoinClassroomGate from './JoinClassroomGate'
import { PWAPrompt } from '@/components/PWAPrompt'
import ChatDock from '@/components/chat/ChatDock'
import { PullToRefresh } from '@/components/PullToRefresh'
import { useVersionCheck } from '@/hooks/use-version-check'
import { useForceReload } from '@/hooks/use-force-reload'
import { useAuth } from '@/hooks/use-auth'
import { api } from '@/lib/api'
import type { Classroom } from '@/types'

export default function MainLayout() {
  useVersionCheck()
  useForceReload()

  // #159 온보딩 게이트: 승인된 학생이 강의실 미소속이면 초대 코드 입력 화면
  const { user } = useAuth()
  const [needsJoin, setNeedsJoin] = useState(false)

  useEffect(() => {
    if (!user || user.role !== 'student' || user.status !== 'approved') {
      setNeedsJoin(false)
      return
    }
    api
      .get<Classroom[]>('/classrooms')
      .then((list) => setNeedsJoin((list ?? []).length === 0))
      .catch(() => setNeedsJoin(false))
  }, [user])

  return (
    <PullToRefresh>
      <div className="min-h-screen bg-background">
        <Header />
        <main className="pb-16">{needsJoin ? <JoinClassroomGate /> : <Outlet />}</main>
        <BottomNav />
        <PWAPrompt />
        <ChatDock />
      </div>
    </PullToRefresh>
  )
}
