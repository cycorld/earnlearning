import { useState, useEffect, useCallback } from 'react'
import { subscribeToPush, unsubscribeFromPush } from '@/lib/push'

export function usePush() {
  const [isSupported, setIsSupported] = useState(false)
  const [isSubscribed, setIsSubscribed] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const supported =
      'serviceWorker' in navigator &&
      'PushManager' in window &&
      'Notification' in window

    setIsSupported(supported)

    if (supported) {
      navigator.serviceWorker.ready.then((registration) => {
        registration.pushManager.getSubscription().then((subscription) => {
          setIsSubscribed(subscription !== null)
        })
      })
    }
  }, [])

  const subscribe = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const timeoutPromise = new Promise<never>((_, reject) =>
        setTimeout(() => reject(new Error('구독 요청 시간 초과 (30초). 앱을 삭제 후 재설치해주세요.')), 30000)
      )
      const success = await Promise.race([subscribeToPush(), timeoutPromise])
      setIsSubscribed(success)
      if (!success) {
        setError('알림 권한이 거부되었습니다. 설정에서 알림을 허용해주세요.')
      }
      return success
    } catch (e: any) {
      setError(e.message || '푸시 구독에 실패했습니다.')
      return false
    } finally {
      setLoading(false)
    }
  }, [])

  const unsubscribe = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      await unsubscribeFromPush()
      setIsSubscribed(false)
    } catch (e: any) {
      setError(e.message || '푸시 해제에 실패했습니다.')
    } finally {
      setLoading(false)
    }
  }, [])

  return { isSupported, isSubscribed, loading, error, subscribe, unsubscribe }
}
