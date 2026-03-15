import { useState, useEffect } from 'react'
import { X, Download, Bell, Share, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { usePush } from '@/hooks/use-push'

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>
}

function isIOS(): boolean {
  return /iPad|iPhone|iPod/.test(navigator.userAgent) && !(window as any).MSStream
}

function isInStandaloneMode(): boolean {
  return window.matchMedia('(display-mode: standalone)').matches
    || (navigator as { standalone?: boolean }).standalone === true
}

export function PWAPrompt() {
  const [installPrompt, setInstallPrompt] = useState<BeforeInstallPromptEvent | null>(null)
  const [showInstallBanner, setShowInstallBanner] = useState(false)
  const [showIOSInstallBanner, setShowIOSInstallBanner] = useState(false)
  const [showPushBanner, setShowPushBanner] = useState(false)
  const { isSupported: pushSupported, isSubscribed, subscribe } = usePush()

  useEffect(() => {
    const isInstalled = isInStandaloneMode()

    const handleBeforeInstall = (e: Event) => {
      e.preventDefault()
      setInstallPrompt(e as BeforeInstallPromptEvent)
      const dismissed = localStorage.getItem('pwa-install-dismissed')
      if (!dismissed || Date.now() - Number(dismissed) > 7 * 24 * 60 * 60 * 1000) {
        setShowInstallBanner(true)
      }
    }

    if (!isInstalled) {
      // Android/Chrome: beforeinstallprompt 이벤트 사용
      window.addEventListener('beforeinstallprompt', handleBeforeInstall)

      // iOS Safari: beforeinstallprompt가 없으므로 수동 안내
      if (isIOS()) {
        const dismissed = localStorage.getItem('pwa-ios-install-dismissed')
        if (!dismissed || Date.now() - Number(dismissed) > 7 * 24 * 60 * 60 * 1000) {
          setTimeout(() => setShowIOSInstallBanner(true), 3000)
        }
      }
    }

    // Show push banner after install or if already installed
    if (isInstalled && pushSupported && !isSubscribed) {
      const pushDismissed = localStorage.getItem('pwa-push-dismissed')
      if (!pushDismissed || Date.now() - Number(pushDismissed) > 7 * 24 * 60 * 60 * 1000) {
        setTimeout(() => setShowPushBanner(true), 2000)
      }
    }

    return () => window.removeEventListener('beforeinstallprompt', handleBeforeInstall)
  }, [pushSupported, isSubscribed])

  const handleInstall = async () => {
    if (!installPrompt) return
    await installPrompt.prompt()
    const result = await installPrompt.userChoice
    if (result.outcome === 'accepted') {
      setShowInstallBanner(false)
      // Show push banner after install
      if (pushSupported && !isSubscribed) {
        setTimeout(() => setShowPushBanner(true), 1000)
      }
    }
    setInstallPrompt(null)
  }

  const [pushLoading, setPushLoading] = useState(false)
  const [pushError, setPushError] = useState<string | null>(null)

  const handlePushSubscribe = async () => {
    setPushLoading(true)
    setPushError(null)
    try {
      const timeoutPromise = new Promise<never>((_, reject) =>
        setTimeout(() => reject(new Error('시간 초과. 다시 시도해주세요.')), 15000)
      )
      const success = await Promise.race([subscribe(), timeoutPromise])
      if (success) {
        setShowPushBanner(false)
      } else {
        setPushError('알림 권한이 거부되었습니다. 설정에서 허용해주세요.')
      }
    } catch (e: any) {
      setPushError(e.message || '알림 구독에 실패했습니다.')
    } finally {
      setPushLoading(false)
    }
  }

  const dismissInstall = () => {
    setShowInstallBanner(false)
    localStorage.setItem('pwa-install-dismissed', String(Date.now()))
  }

  const dismissIOSInstall = () => {
    setShowIOSInstallBanner(false)
    localStorage.setItem('pwa-ios-install-dismissed', String(Date.now()))
  }

  const dismissPush = () => {
    setShowPushBanner(false)
    localStorage.setItem('pwa-push-dismissed', String(Date.now()))
  }

  if (!showInstallBanner && !showIOSInstallBanner && !showPushBanner) return null

  return (
    <div className="fixed bottom-20 left-4 right-4 z-50 md:left-auto md:right-4 md:max-w-sm">
      {showInstallBanner && (
        <div className="rounded-xl border bg-card p-4 shadow-lg">
          <div className="flex items-start gap-3">
            <div className="rounded-lg bg-primary/10 p-2">
              <Download className="h-5 w-5 text-primary" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium">홈 화면에 추가</p>
              <p className="mt-0.5 text-xs text-muted-foreground">
                앱처럼 빠르게 접속할 수 있어요
              </p>
              <div className="mt-3 flex gap-2">
                <Button size="sm" onClick={handleInstall}>
                  설치하기
                </Button>
                <Button size="sm" variant="ghost" onClick={dismissInstall}>
                  나중에
                </Button>
              </div>
            </div>
            <button onClick={dismissInstall} className="text-muted-foreground hover:text-foreground">
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}

      {showIOSInstallBanner && !showInstallBanner && (
        <div className="rounded-xl border bg-card p-4 shadow-lg">
          <div className="flex items-start gap-3">
            <div className="rounded-lg bg-primary/10 p-2">
              <Share className="h-5 w-5 text-primary" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium">홈 화면에 추가하기</p>
              <div className="mt-1.5 space-y-1.5">
                <p className="text-xs text-muted-foreground">
                  <span className="font-medium">1.</span> 하단의 <span className="inline-flex items-center"><Share className="mx-0.5 inline h-3 w-3" /></span> 공유 버튼 탭
                </p>
                <p className="text-xs text-muted-foreground">
                  <span className="font-medium">2.</span> <span className="font-medium">"홈 화면에 추가"</span> 선택
                </p>
                <p className="text-xs text-muted-foreground">
                  <span className="font-medium">3.</span> 우측 상단 <span className="font-medium">"추가"</span> 탭
                </p>
              </div>
              <p className="mt-2 text-xs text-muted-foreground">
                앱처럼 사용하고, 푸시 알림도 받을 수 있어요!
              </p>
              <div className="mt-3">
                <Button size="sm" variant="ghost" onClick={dismissIOSInstall}>
                  확인
                </Button>
              </div>
            </div>
            <button onClick={dismissIOSInstall} className="text-muted-foreground hover:text-foreground">
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}

      {showPushBanner && !showInstallBanner && !showIOSInstallBanner && (
        <div className="rounded-xl border bg-card p-4 shadow-lg">
          <div className="flex items-start gap-3">
            <div className="rounded-lg bg-primary/10 p-2">
              <Bell className="h-5 w-5 text-primary" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium">알림 받기</p>
              <p className="mt-0.5 text-xs text-muted-foreground">
                과제 승인, 외주 진행 등 중요 알림을 받아보세요
              </p>
              {pushError && (
                <p className="mt-1 text-xs text-destructive">{pushError}</p>
              )}
              <div className="mt-3 flex gap-2">
                <Button size="sm" onClick={handlePushSubscribe} disabled={pushLoading}>
                  {pushLoading ? (
                    <><Loader2 className="mr-1 h-3 w-3 animate-spin" /> 처리 중...</>
                  ) : '알림 켜기'}
                </Button>
                <Button size="sm" variant="ghost" onClick={dismissPush} disabled={pushLoading}>
                  나중에
                </Button>
              </div>
            </div>
            <button onClick={dismissPush} className="text-muted-foreground hover:text-foreground" disabled={pushLoading}>
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
