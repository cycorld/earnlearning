import { useState, useEffect, useCallback } from 'react'
import type { Wallet } from '@/types'
import { api } from '@/lib/api'

export function useWallet() {
  const [wallet, setWallet] = useState<Wallet | null>(null)
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api.get<Wallet>('/wallet')
      setWallet(data)
    } catch {
      setWallet(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  return { wallet, loading, refresh }
}
