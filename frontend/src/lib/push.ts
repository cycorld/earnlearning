import { api } from './api'

export function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4)
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/')
  const rawData = atob(base64)
  const outputArray = new Uint8Array(rawData.length)
  for (let i = 0; i < rawData.length; i++) {
    outputArray[i] = rawData.charCodeAt(i)
  }
  return outputArray
}

export async function subscribeToPush(): Promise<boolean> {
  if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
    return false
  }

  try {
    const permission = await Notification.requestPermission()
    if (permission !== 'granted') {
      return false
    }

    const result = await api.get<{ vapid_public_key: string }>('/notifications/push/vapid-key')
    const vapidKey = result.vapid_public_key

    const registration = await navigator.serviceWorker.ready

    // 기존 구독이 있으면 완전히 해제 (endpoint 갱신을 위해)
    const existingSub = await registration.pushManager.getSubscription()
    if (existingSub) {
      await existingSub.unsubscribe()
      // iOS Safari에서 unsubscribe 직후 subscribe 시 hang 방지
      await new Promise((r) => setTimeout(r, 500))
    }

    const subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(vapidKey) as BufferSource,
    })

    const subJSON = subscription.toJSON()
    await api.post('/notifications/push/subscribe', {
      endpoint: subJSON.endpoint,
      p256dh: subJSON.keys?.p256dh,
      auth: subJSON.keys?.auth,
      user_agent: navigator.userAgent,
    })
    return true
  } catch (e) {
    console.error('Push subscribe error:', e)
    throw e
  }
}

export async function unsubscribeFromPush(): Promise<void> {
  if (!('serviceWorker' in navigator)) return

  try {
    const registration = await navigator.serviceWorker.ready
    const subscription = await registration.pushManager.getSubscription()
    if (subscription) {
      await api.del('/notifications/push/subscribe', { endpoint: subscription.endpoint })
      await subscription.unsubscribe()
    }
  } catch {
    // ignore errors during unsubscribe
  }
}
