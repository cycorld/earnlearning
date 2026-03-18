import { Link } from 'react-router-dom'
import { useAuth } from '@/hooks/use-auth'
import { useWallet } from '@/hooks/use-wallet'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import {
  LogOut,
  Wallet,
  Settings,
  ChevronRight,
  Building2,
  GraduationCap,
  Bell,
  BellOff,
  Loader2,
} from 'lucide-react'
import { usePush } from '@/hooks/use-push'
import { useEmailPreference } from '@/hooks/use-email-preference'
import { Mail, MailX } from 'lucide-react'
import { formatMoney } from '@/lib/utils'

export default function ProfilePage() {
  const { user, isLoading, logout } = useAuth()
  const { wallet, loading: walletLoading } = useWallet()
  const { isSupported: pushSupported, isSubscribed, loading: pushLoading, error: pushError, subscribe, unsubscribe } = usePush()
  const { emailEnabled, loading: emailLoading, updating: emailUpdating, updatePreference } = useEmailPreference()

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    )
  }

  if (!user) return null

  return (
    <div className="mx-auto max-w-lg space-y-4 p-4">
      {/* User info card */}
      <Card>
        <CardContent className="p-6">
          <div className="flex items-center gap-4">
            <Avatar className="h-16 w-16">
              <AvatarImage src={user.avatar_url} />
              <AvatarFallback className="text-lg">
                {user.name.charAt(0)}
              </AvatarFallback>
            </Avatar>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <h2 className="text-lg font-semibold">{user.name}</h2>
                <Badge
                  variant={user.role === 'admin' ? 'default' : 'secondary'}
                  className="text-xs"
                >
                  {user.role === 'admin' ? '관리자' : '학생'}
                </Badge>
              </div>
              <p className="text-sm text-muted-foreground">{user.email}</p>
              <div className="mt-1 flex items-center gap-3 text-xs text-muted-foreground">
                <span className="flex items-center gap-1">
                  <GraduationCap className="h-3 w-3" />
                  {user.department}
                </span>
                <span>{user.student_id}</span>
              </div>
            </div>
          </div>
          {user.bio && (
            <p className="mt-3 text-sm text-muted-foreground">{user.bio}</p>
          )}
        </CardContent>
      </Card>

      {/* Wallet summary */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="flex items-center gap-2 text-base">
            <Wallet className="h-4 w-4" />
            자산 요약
          </CardTitle>
          <Button variant="ghost" size="sm" asChild>
            <Link to="/wallet" className="flex items-center gap-1 text-xs">
              상세보기 <ChevronRight className="h-4 w-4" />
            </Link>
          </Button>
        </CardHeader>
        <CardContent>
          {walletLoading ? (
            <div className="flex justify-center py-4">
              <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
            </div>
          ) : wallet ? (
            <div className="grid grid-cols-2 gap-3">
              <div>
                <p className="text-xs text-muted-foreground">총 자산</p>
                <p className="text-sm font-semibold">
                  {formatMoney(Number(wallet.total_asset_value) || 0)}
                </p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">보유 현금</p>
                <p className="text-sm font-semibold">
                  {formatMoney(Number(wallet.balance) || 0)}
                </p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">순위</p>
                <p className="text-sm font-semibold">
                  {wallet.rank ?? 0}위 / {wallet.total_students ?? 0}명
                </p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">회사 지분</p>
                <p className="text-sm font-semibold">
                  {formatMoney(wallet.asset_breakdown?.company_equity ?? 0)}
                </p>
              </div>
            </div>
          ) : (
            <p className="py-4 text-center text-sm text-muted-foreground">
              자산 정보를 불러올 수 없습니다.
            </p>
          )}
        </CardContent>
      </Card>

      {/* Navigation links */}
      <Card>
        <CardContent className="p-2">
          <Link
            to="/company"
            className="flex items-center gap-3 rounded-md px-3 py-3 transition-colors hover:bg-accent"
          >
            <Building2 className="h-5 w-5 text-muted-foreground" />
            <span className="flex-1 text-sm">내 회사</span>
            <ChevronRight className="h-4 w-4 text-muted-foreground" />
          </Link>
          <Separator />
          <Link
            to="/wallet/transactions"
            className="flex items-center gap-3 rounded-md px-3 py-3 transition-colors hover:bg-accent"
          >
            <Wallet className="h-5 w-5 text-muted-foreground" />
            <span className="flex-1 text-sm">거래 내역</span>
            <ChevronRight className="h-4 w-4 text-muted-foreground" />
          </Link>
          {pushSupported && (
            <>
              <Separator />
              <button
                onClick={isSubscribed ? unsubscribe : subscribe}
                disabled={pushLoading}
                className="flex w-full items-center gap-3 rounded-md px-3 py-3 transition-colors hover:bg-accent disabled:opacity-50"
              >
                {pushLoading ? (
                  <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                ) : isSubscribed ? (
                  <Bell className="h-5 w-5 text-primary" />
                ) : (
                  <BellOff className="h-5 w-5 text-muted-foreground" />
                )}
                <span className="flex-1 text-left text-sm">
                  {pushLoading
                    ? '처리 중...'
                    : `푸시 알림 ${isSubscribed ? '켜짐' : '꺼짐'}`}
                </span>
                {!pushLoading && (
                  <span className="text-xs text-muted-foreground">
                    {isSubscribed ? '끄기' : '켜기'}
                  </span>
                )}
              </button>
              {pushError && (
                <p className="px-3 pb-2 text-xs text-destructive">{pushError}</p>
              )}
            </>
          )}
          <Separator />
          <button
            onClick={() => updatePreference(!emailEnabled)}
            disabled={emailLoading || emailUpdating}
            className="flex w-full items-center gap-3 rounded-md px-3 py-3 transition-colors hover:bg-accent disabled:opacity-50"
          >
            {emailLoading || emailUpdating ? (
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            ) : emailEnabled ? (
              <Mail className="h-5 w-5 text-primary" />
            ) : (
              <MailX className="h-5 w-5 text-muted-foreground" />
            )}
            <span className="flex-1 text-left text-sm">
              {emailLoading
                ? '로딩 중...'
                : emailUpdating
                  ? '변경 중...'
                  : `이메일 알림 ${emailEnabled ? '켜짐' : '꺼짐'}`}
            </span>
            {!emailLoading && !emailUpdating && (
              <span className="text-xs text-muted-foreground">
                {emailEnabled ? '끄기' : '켜기'}
              </span>
            )}
          </button>
          {user.role === 'admin' && (
            <>
              <Separator />
              <Link
                to="/admin"
                className="flex items-center gap-3 rounded-md px-3 py-3 transition-colors hover:bg-accent"
              >
                <Settings className="h-5 w-5 text-muted-foreground" />
                <span className="flex-1 text-sm">관리자 페이지</span>
                <ChevronRight className="h-4 w-4 text-muted-foreground" />
              </Link>
            </>
          )}
        </CardContent>
      </Card>

      {/* Logout */}
      <Button
        variant="outline"
        className="w-full gap-2 text-destructive hover:bg-destructive/10 hover:text-destructive"
        onClick={logout}
      >
        <LogOut className="h-4 w-4" />
        로그아웃
      </Button>
    </div>
  )
}
