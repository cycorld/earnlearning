// Hard reload: SW 캐시/등록 모두 삭제 후 캐시버스팅 쿼리로 재진입.
// 버전 체크 훅(#018/#028) 과 force-reload 브로드캐스트(#027) 에서 공통 사용.
export async function forceRefresh(): Promise<void> {
  // 1. SW 캐시 모두 삭제
  if ('caches' in window) {
    try {
      const cacheNames = await caches.keys()
      await Promise.all(cacheNames.map((name) => caches.delete(name)))
    } catch {
      // ignore
    }
  }

  // 2. SW unregister (다음 로드에서 새로 등록됨)
  if ('serviceWorker' in navigator) {
    try {
      const registrations = await navigator.serviceWorker.getRegistrations()
      await Promise.all(registrations.map((r) => r.unregister()))
    } catch {
      // ignore
    }
  }

  // 3. 캐시 버스팅 쿼리 + hard reload
  const url = new URL(window.location.href)
  url.searchParams.set('_v', Date.now().toString())
  window.location.replace(url.toString())
}
