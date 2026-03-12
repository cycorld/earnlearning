import { useEffect } from 'react'
import { wsClient } from '@/lib/ws'

export function useWebSocket(
  event: string,
  callback: (data: unknown) => void,
): void {
  useEffect(() => {
    const unsubscribe = wsClient.on(event, callback)
    return unsubscribe
  }, [event, callback])
}
