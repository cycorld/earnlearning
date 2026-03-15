import { useEffect, useRef } from 'react'
import { useLocation } from 'react-router-dom'

let knownVersion: string | null = null

export function useVersionCheck() {
  const location = useLocation()
  const checkCount = useRef(0)

  useEffect(() => {
    // Skip first render (initial page load already has latest)
    checkCount.current++
    if (checkCount.current <= 1) {
      // Fetch initial version on first mount
      fetchVersion().then((v) => {
        knownVersion = v
      })
      return
    }

    fetchVersion().then((serverVersion) => {
      if (!serverVersion || !knownVersion) {
        if (serverVersion) knownVersion = serverVersion
        return
      }

      if (serverVersion !== knownVersion) {
        knownVersion = serverVersion
        window.location.reload()
      }
    })
  }, [location.pathname])
}

async function fetchVersion(): Promise<string | null> {
  try {
    const res = await fetch('/api/version')
    if (!res.ok) return null
    const data = await res.json()
    const { build_number, commit_sha } = data.data
    return `${build_number}-${commit_sha}`
  } catch {
    return null
  }
}
