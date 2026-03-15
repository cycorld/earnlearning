import { useState, useEffect } from 'react'
import { X, Download, Bell } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { usePush } from '@/hooks/use-push'

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>
}

export function PWAPrompt() {
  const [installPrompt, setInstallPrompt] = useState<BeforeInstallPromptEvent | null>(null)
  const [showInstallBanner, setShowInstallBanner] = useState(false)
  const [showPushBanner, setShowPushBanner] = useState(false)
  const { isSupported: pushSupported, isSubscribed, subscribe } = usePush()

  useEffect(() => {
    // Check if already installed (standalone mode)
    const isInstalled = window.matchMedia('(display-mode: standalone)').matches
      || (navigator as { standalone?: boolean }).standalone === true

    const handleBeforeInstall = (e: Event) => {
      e.preventDefault()
      setInstallPrompt(e as BeforeInstallPromptEvent)
      // Show banner if not dismissed recently
      const dismissed = localStorage.getItem('pwa-install-dismissed')
      if (!dismissed || Date.now() - Number(dismissed) > 7 * 24 * 60 * 60 * 1000) {
        setShowInstallBanner(true)
      }
    }

    if (!isInstalled) {
      window.addEventListener('beforeinstallprompt', handleBeforeInstall)
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

  const handlePushSubscribe = async () => {
    await subscribe()
    setShowPushBanner(false)
  }

  const dismissInstall = () => {
    setShowInstallBanner(false)
    localStorage.setItem('pwa-install-dismissed', String(Date.now()))
  }

  const dismissPush = () => {
    setShowPushBanner(false)
    localStorage.setItem('pwa-push-dismissed', String(Date.now()))
  }

  if (!showInstallBanner && !showPushBanner) return null

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

      {showPushBanner && !showInstallBanner && (
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
              <div className="mt-3 flex gap-2">
                <Button size="sm" onClick={handlePushSubscribe}>
                  알림 켜기
                </Button>
                <Button size="sm" variant="ghost" onClick={dismissPush}>
                  나중에
                </Button>
              </div>
            </div>
            <button onClick={dismissPush} className="text-muted-foreground hover:text-foreground">
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
