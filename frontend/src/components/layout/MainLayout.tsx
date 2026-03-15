import { Outlet } from 'react-router-dom'
import Header from './Header'
import BottomNav from './BottomNav'
import { useVersionCheck } from '@/hooks/use-version-check'

export default function MainLayout() {
  useVersionCheck()

  return (
    <div className="min-h-screen bg-background">
      <Header />
      <main className="pb-16">
        <Outlet />
      </main>
      <BottomNav />
    </div>
  )
}
