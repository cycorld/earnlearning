import { Outlet } from 'react-router-dom'
import Header from './Header'
import BottomNav from './BottomNav'

export default function MainLayout() {
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
