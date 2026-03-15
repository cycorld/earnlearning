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
        forceRefresh()
      }
    })
  }, [location.pathname])
}

async function fetchVersion(): Promise<string | null> {
  try {
    // cache: 'no-store' prevents browser/CDN from caching the response
    const res = await fetch('/api/version', { cache: 'no-store' })
    if (!res.ok) return null
    const data = await res.json()
    const { build_number, commit_sha } = data.data
    return `${build_number}-${commit_sha}`
  } catch {
    return null
  }
}

async function forceRefresh(): Promise<void> {
  // 1. Clear all Service Worker caches
  if ('caches' in window) {
    const cacheNames = await caches.keys()
    await Promise.all(cacheNames.map((name) => caches.delete(name)))
  }

  // 2. Unregister service workers so they don't serve stale content
  if ('serviceWorker' in navigator) {
    const registrations = await navigator.serviceWorker.getRegistrations()
    await Promise.all(registrations.map((r) => r.unregister()))
  }

  // 3. Hard reload — bypass browser cache by navigating with a cache-bust param
  // The param is stripped on the server side (SPA always serves index.html)
  const url = new URL(window.location.href)
  url.searchParams.set('_v', Date.now().toString())
  window.location.replace(url.toString())
}
