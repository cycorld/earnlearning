import { useState, useEffect, useCallback } from 'react'
import { subscribeToPush, unsubscribeFromPush } from '@/lib/push'

export function usePush() {
  const [isSupported, setIsSupported] = useState(false)
  const [isSubscribed, setIsSubscribed] = useState(false)

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
    const success = await subscribeToPush()
    setIsSubscribed(success)
    return success
  }, [])

  const unsubscribe = useCallback(async () => {
    await unsubscribeFromPush()
    setIsSubscribed(false)
  }, [])

  return { isSupported, isSubscribed, subscribe, unsubscribe }
}
