import { useState, useEffect, useCallback } from 'react'
import { api } from '@/lib/api'

interface EmailPreference {
  user_id: number
  email_enabled: boolean
}

export function useEmailPreference() {
  const [preference, setPreference] = useState<EmailPreference | null>(null)
  const [loading, setLoading] = useState(true)
  const [updating, setUpdating] = useState(false)

  useEffect(() => {
    api
      .get<EmailPreference>('/notifications/email/preference')
      .then((res) => {
        if (res) {
          setPreference(res)
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const updatePreference = useCallback(
    async (emailEnabled: boolean) => {
      setUpdating(true)
      try {
        await api.put('/notifications/email/preference', {
          email_enabled: emailEnabled,
        })
        setPreference((prev) =>
          prev ? { ...prev, email_enabled: emailEnabled } : null
        )
      } finally {
        setUpdating(false)
      }
    },
    []
  )

  return {
    emailEnabled: preference?.email_enabled ?? true,
    loading,
    updating,
    updatePreference,
  }
}
