/**
 * GA4 통합 — production 에서만 동작.
 *
 * 사용:
 *   - main.tsx 에서 `initAnalytics()` 1회 호출
 *   - 라우트 변경마다 `useGAPageView` hook (자동)
 *   - 커스텀 이벤트는 `trackEvent('signup_completed', { method: 'email' })`
 *
 * 측정 ID 는 빌드 타임에 `VITE_GA_ID` env 로 주입. 없으면 GA 로딩 자체 skip.
 * 측정 ID 는 비밀이 아님 — 프론트엔드 노출 OK (#111 강의노트 참조).
 */

declare global {
  interface Window {
    dataLayer: unknown[]
    gtag: (...args: unknown[]) => void
  }
}

/**
 * 측정 ID. 측정 ID 는 비밀이 아니므로 (브라우저 소스에 어차피 노출됨) 하드코딩 OK.
 * staging 등에서 다른 ID 쓰고 싶으면 빌드 타임에 `VITE_GA_ID` 로 override.
 */
const GA_ID_FALLBACK = 'G-T4KX9MKVL0'
const GA_ID = (import.meta.env.VITE_GA_ID as string | undefined) || GA_ID_FALLBACK

/**
 * GA 가 실제로 동작하는 환경인가?
 * - production 빌드일 때 (`import.meta.env.PROD`)
 * - 측정 ID 가 비어있지 않을 때
 */
export function isAnalyticsEnabled(): boolean {
  return Boolean(import.meta.env.PROD && GA_ID)
}

let initialized = false

/** 테스트 격리용 — production 코드에선 호출 X */
export function __resetAnalyticsForTest(): void {
  initialized = false
  if (typeof window !== 'undefined') {
    delete (window as unknown as { dataLayer?: unknown[] }).dataLayer
    delete (window as unknown as { gtag?: unknown }).gtag
  }
}

/**
 * GA 스크립트 주입 + dataLayer/gtag 초기화.
 * SPA 라서 자동 page_view 는 끄고, 라우트 hook 에서 수동 발사.
 *
 * ⚠️ Consent Mode v2 (#112) — `consent default` 를 `js`/`config` 보다 **먼저** 호출 필수.
 * 안 하면 implicit denied 모드로 들어가서 g/collect 비콘이 차단됨 (Realtime 0).
 * LMS 는 광고 없음 + 로그인 게이트 + ToS 에 분석 사용 명시 → analytics 만 granted.
 */
export function initAnalytics(): void {
  if (!isAnalyticsEnabled() || initialized) return
  if (typeof window === 'undefined' || typeof document === 'undefined') return
  initialized = true

  // dataLayer + gtag shim 을 먼저 정의 — 스크립트 로드 전에 큐잉 가능하도록.
  // ⚠️ #112: `dataLayer.push(arguments)` (IArguments) 필수. rest-param `(...args)` 로 push 하면
  // 진짜 Array 가 되어 gtag.js 가 gtag 명령으로 인식 못 함 (data layer push 로 간주).
  // 표준 Google 스니펫 패턴 그대로 — 변경 금지.
  window.dataLayer = window.dataLayer ?? []
  // eslint-disable-next-line @typescript-eslint/no-explicit-any, prefer-rest-params
  window.gtag = function () {
    // eslint-disable-next-line prefer-rest-params
    window.dataLayer.push(arguments as unknown as unknown[])
  }

  // ★ 반드시 js/config 이전에 — Consent Mode v2 default
  window.gtag('consent', 'default', {
    ad_storage: 'denied',
    ad_user_data: 'denied',
    ad_personalization: 'denied',
    analytics_storage: 'granted',
    functionality_storage: 'granted',
    security_storage: 'granted',
  })

  window.gtag('js', new Date())
  window.gtag('config', GA_ID, {
    send_page_view: false, // SPA 라 수동 발사
  })

  // gtag.js 스크립트는 dataLayer 큐잉 후 비동기 로드 — 큐가 안전하게 처리됨
  const script = document.createElement('script')
  script.async = true
  script.src = `https://www.googletagmanager.com/gtag/js?id=${GA_ID}`
  document.head.appendChild(script)
}

/** 라우트 이동마다 호출. dev/no-id 환경에선 noop. */
export function trackPageView(path: string, title?: string): void {
  if (!isAnalyticsEnabled() || typeof window === 'undefined' || !window.gtag) return
  window.gtag('event', 'page_view', {
    page_path: path,
    page_title: title ?? document.title,
    page_location: window.location.href,
  })
}

/** 커스텀 이벤트. dev/no-id 환경에선 noop. */
export function trackEvent(name: string, params: Record<string, unknown> = {}): void {
  if (!isAnalyticsEnabled() || typeof window === 'undefined' || !window.gtag) return
  window.gtag('event', name, params)
}
