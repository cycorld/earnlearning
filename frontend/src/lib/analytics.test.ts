import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

// import.meta.env 를 stubbing 해야 하므로 동적 import 패턴 사용
async function loadModule(env: { PROD: boolean; VITE_GA_ID?: string }) {
  vi.stubEnv('PROD', env.PROD ? 'true' : '')
  if (env.VITE_GA_ID !== undefined) {
    vi.stubEnv('VITE_GA_ID', env.VITE_GA_ID)
  }
  vi.resetModules()
  return await import('./analytics')
}

describe('analytics', () => {
  beforeEach(() => {
    delete (window as unknown as { dataLayer?: unknown[] }).dataLayer
    delete (window as unknown as { gtag?: unknown }).gtag
    // 이전 테스트가 주입한 GA script 제거 — idempotent 검증 위해 격리 필요
    document
      .querySelectorAll('script[src*="googletagmanager.com/gtag/js"]')
      .forEach((s) => s.remove())
  })

  afterEach(() => {
    vi.unstubAllEnvs()
  })

  describe('dev 환경', () => {
    it('isAnalyticsEnabled 는 false', async () => {
      const m = await loadModule({ PROD: false })
      expect(m.isAnalyticsEnabled()).toBe(false)
    })

    it('initAnalytics 는 noop — script 도 dataLayer 도 안 만듦', async () => {
      const m = await loadModule({ PROD: false })
      m.initAnalytics()
      expect(window.dataLayer).toBeUndefined()
      expect(window.gtag).toBeUndefined()
      expect(document.querySelector('script[src*="googletagmanager"]')).toBeNull()
    })

    it('trackPageView · trackEvent 는 dev 에서 호출해도 throw X', async () => {
      const m = await loadModule({ PROD: false })
      expect(() => m.trackPageView('/foo')).not.toThrow()
      expect(() => m.trackEvent('signup_completed')).not.toThrow()
      expect(window.dataLayer).toBeUndefined()
    })
  })

  describe('production 환경 + 측정 ID 있음', () => {
    it('isAnalyticsEnabled 는 true', async () => {
      const m = await loadModule({ PROD: true, VITE_GA_ID: 'G-TEST12345' })
      expect(m.isAnalyticsEnabled()).toBe(true)
    })

    it('initAnalytics — gtag 스크립트 + dataLayer + js·config 이벤트 push', async () => {
      const m = await loadModule({ PROD: true, VITE_GA_ID: 'G-TEST12345' })
      m.__resetAnalyticsForTest()
      m.initAnalytics()

      const script = document.querySelector(
        'script[src*="googletagmanager.com/gtag/js"]',
      ) as HTMLScriptElement | null
      expect(script).not.toBeNull()
      expect(script?.src).toContain('id=G-TEST12345')

      expect(window.dataLayer).toBeDefined()
      expect(Array.isArray(window.dataLayer)).toBe(true)
      // 'js' + 'config' 두 개 이상은 push 됐어야
      expect(window.dataLayer.length).toBeGreaterThanOrEqual(2)
    })

    it('initAnalytics 두 번 호출해도 스크립트 1개만 (idempotent)', async () => {
      const m = await loadModule({ PROD: true, VITE_GA_ID: 'G-TEST12345' })
      m.__resetAnalyticsForTest()
      m.initAnalytics()
      m.initAnalytics()
      const scripts = document.querySelectorAll(
        'script[src*="googletagmanager.com/gtag/js"]',
      )
      expect(scripts.length).toBe(1)
    })

    it('trackPageView — page_view 이벤트가 dataLayer 에 push 됨', async () => {
      const m = await loadModule({ PROD: true, VITE_GA_ID: 'G-TEST12345' })
      m.__resetAnalyticsForTest()
      m.initAnalytics()
      const before = window.dataLayer.length
      m.trackPageView('/wallet/transactions')
      expect(window.dataLayer.length).toBe(before + 1)
      // last entry: ['event', 'page_view', { page_path: '/wallet/transactions', ... }]
      // gtag.apply 패턴으로 IArguments 객체가 push 되니, 인덱스 접근으로 검증
      const last = window.dataLayer[window.dataLayer.length - 1] as unknown[]
      expect(last[0]).toBe('event')
      expect(last[1]).toBe('page_view')
      expect((last[2] as { page_path: string }).page_path).toBe(
        '/wallet/transactions',
      )
    })

    it('trackEvent — 임의 이벤트가 dataLayer 에 push 됨', async () => {
      const m = await loadModule({ PROD: true, VITE_GA_ID: 'G-TEST12345' })
      m.__resetAnalyticsForTest()
      m.initAnalytics()
      const before = window.dataLayer.length
      m.trackEvent('signup_completed', { method: 'email' })
      expect(window.dataLayer.length).toBe(before + 1)
      const last = window.dataLayer[window.dataLayer.length - 1] as unknown[]
      expect(last[0]).toBe('event')
      expect(last[1]).toBe('signup_completed')
      expect((last[2] as { method: string }).method).toBe('email')
    })
  })

  describe('production 이지만 측정 ID 없음', () => {
    // 환경변수 미설정 + fallback 있는 케이스 — fallback 이 사용되므로 enabled.
    // (fallback 비우면 disabled. 테스트는 fallback 동작만 1회 확인.)
    it('VITE_GA_ID 비워도 fallback 으로 enabled (PROD 빌드)', async () => {
      const m = await loadModule({ PROD: true, VITE_GA_ID: '' })
      expect(m.isAnalyticsEnabled()).toBe(true)
    })
  })
})
