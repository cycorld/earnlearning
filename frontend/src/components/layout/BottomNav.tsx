import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  Home,
  Wallet,
  Store,
  Building2,
  MoreHorizontal,
  TrendingUp,
  BarChart3,
  Landmark,
  User,
  Bell,
  Settings,
  BookOpen,
} from 'lucide-react'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { useAuth } from '@/hooks/use-auth'

interface NavItem {
  label: string
  icon: React.ReactNode
  path: string
}

const mainTabs: NavItem[] = [
  { label: '홈', icon: <Home className="h-5 w-5" />, path: '/feed' },
  { label: '자산', icon: <Wallet className="h-5 w-5" />, path: '/wallet' },
  { label: '마켓', icon: <Store className="h-5 w-5" />, path: '/market' },
  { label: '회사', icon: <Building2 className="h-5 w-5" />, path: '/company' },
]

export default function BottomNav() {
  const navigate = useNavigate()
  const location = useLocation()
  const { user } = useAuth()
  const [sheetOpen, setSheetOpen] = useState(false)

  const moreItems: NavItem[] = [
    { label: '투자', icon: <TrendingUp className="h-5 w-5" />, path: '/invest' },
    { label: '거래소', icon: <BarChart3 className="h-5 w-5" />, path: '/exchange' },
    { label: '은행', icon: <Landmark className="h-5 w-5" />, path: '/bank' },
    { label: '프로필', icon: <User className="h-5 w-5" />, path: '/profile' },
    { label: '알림', icon: <Bell className="h-5 w-5" />, path: '/notifications' },
    { label: '개발일지', icon: <BookOpen className="h-5 w-5" />, path: '/changelog' },
    ...(user?.role === 'admin'
      ? [{ label: '관리자', icon: <Settings className="h-5 w-5" />, path: '/admin' }]
      : []),
  ]

  const isActive = (path: string) => location.pathname.startsWith(path)

  const isMoreActive =
    !mainTabs.some((tab) => isActive(tab.path)) && !location.pathname.startsWith('/login')

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-50 border-t bg-background">
      <div className="flex h-16 items-stretch">
        {mainTabs.map((tab) => (
          <button
            key={tab.path}
            onClick={() => navigate(tab.path)}
            className={`flex flex-1 flex-col items-center justify-center gap-0.5 text-xs transition-colors ${
              isActive(tab.path)
                ? 'text-primary'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            {tab.icon}
            <span>{tab.label}</span>
          </button>
        ))}

        <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
          <SheetTrigger asChild>
            <button
              className={`flex flex-1 flex-col items-center justify-center gap-0.5 text-xs transition-colors ${
                isMoreActive
                  ? 'text-primary'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              <MoreHorizontal className="h-5 w-5" />
              <span>더보기</span>
            </button>
          </SheetTrigger>
          <SheetContent side="bottom" className="rounded-t-2xl">
            <SheetHeader>
              <SheetTitle>더보기</SheetTitle>
            </SheetHeader>
            <div className="grid grid-cols-3 gap-4 py-4">
              {moreItems.map((item) => (
                <button
                  key={item.path}
                  onClick={() => {
                    setSheetOpen(false)
                    navigate(item.path)
                  }}
                  className={`flex flex-col items-center gap-2 rounded-lg p-3 transition-colors hover:bg-accent ${
                    isActive(item.path) ? 'text-primary' : 'text-foreground'
                  }`}
                >
                  {item.icon}
                  <span className="text-xs">{item.label}</span>
                </button>
              ))}
            </div>
          </SheetContent>
        </Sheet>
      </div>
    </nav>
  )
}
