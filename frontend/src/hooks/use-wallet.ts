import { useState, useEffect, useCallback } from 'react'
import type { Wallet } from '@/types'
import { api } from '@/lib/api'

interface WalletApiResponse {
  wallet?: { id: number; user_id: number; balance: number }
  assets?: {
    cash: number
    stock_value: number
    company_equity: number
    total_debt: number
    total: number
  }
  // Also support flat format in case API changes
  balance?: number
  total_asset_value?: number
  asset_breakdown?: Wallet['asset_breakdown']
  rank?: number
  total_students?: number
}

function normalizeWallet(raw: WalletApiResponse): Wallet {
  // If response is already in flat format, use it directly with safe number casting
  if (raw.asset_breakdown !== undefined) {
    return {
      balance: Number(raw.balance) || 0,
      total_asset_value: Number(raw.total_asset_value) || 0,
      asset_breakdown: {
        cash: Number(raw.asset_breakdown?.cash) || 0,
        stock_value: Number(raw.asset_breakdown?.stock_value) || 0,
        company_equity: Number(raw.asset_breakdown?.company_equity) || 0,
        total_debt: Number(raw.asset_breakdown?.total_debt) || 0,
      },
      rank: Number(raw.rank) || 0,
      total_students: Number(raw.total_students) || 0,
    }
  }

  // Transform nested { wallet, assets } format to flat Wallet type
  const cash = Number(raw.assets?.cash ?? raw.wallet?.balance) || 0
  const stockValue = Number(raw.assets?.stock_value) || 0
  const companyEquity = Number(raw.assets?.company_equity) || 0
  const totalDebt = Number(raw.assets?.total_debt) || 0
  const total = Number(raw.assets?.total) || (cash + stockValue + companyEquity - totalDebt)

  return {
    balance: Number(raw.wallet?.balance) || 0,
    total_asset_value: total,
    asset_breakdown: {
      cash,
      stock_value: stockValue,
      company_equity: companyEquity,
      total_debt: totalDebt,
    },
    rank: Number(raw.rank) || 0,
    total_students: Number(raw.total_students) || 0,
  }
}

export function useWallet() {
  const [wallet, setWallet] = useState<Wallet | null>(null)
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api.get<WalletApiResponse>('/wallet')
      setWallet(normalizeWallet(data))
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
