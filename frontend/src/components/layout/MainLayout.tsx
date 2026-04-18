import { Outlet } from 'react-router-dom'
import Header from './Header'
import BottomNav from './BottomNav'
import { PWAPrompt } from '@/components/PWAPrompt'
import ChatDock from '@/components/chat/ChatDock'
import { useVersionCheck } from '@/hooks/use-version-check'
import { useForceReload } from '@/hooks/use-force-reload'

export default function MainLayout() {
  useVersionCheck()
  useForceReload()

  return (
    <div className="min-h-screen bg-background">
      <Header />
      <main className="pb-16">
        <Outlet />
      </main>
      <BottomNav />
      <PWAPrompt />
      <ChatDock />
    </div>
  )
}
