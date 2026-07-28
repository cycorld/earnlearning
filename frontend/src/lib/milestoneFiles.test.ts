import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { openMilestoneFile, openPrivateFile } from './milestoneFiles'

// #176 — HTML 첨부는 절대 새 탭(blob: same-origin)으로 열리면 안 된다.

// openPrivateFile 이 api.ts(apiFetch) 를 거치므로 auth 모듈 export 를 모두 채워야 한다.
vi.mock('./auth', () => ({
  getToken: () => 'test-token',
  setToken: vi.fn(),
  removeToken: vi.fn(),
}))
vi.mock('sonner', () => ({ toast: { error: vi.fn() } }))

function mockFetch(body: string, contentType: string) {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue({
      ok: true,
      blob: async () => new Blob([body], { type: contentType }),
      headers: { get: (k: string) => (k.toLowerCase() === 'content-type' ? contentType : null) },
    }),
  )
}

describe('openMilestoneFile', () => {
  let openSpy: ReturnType<typeof vi.fn>
  let clickSpy: ReturnType<typeof vi.fn>
  let fakeWin: { location: { replace: ReturnType<typeof vi.fn> }; close: ReturnType<typeof vi.fn> }

  beforeEach(() => {
    // 팝업 차단을 피하려면 fetch 전에 탭을 먼저 열어야 한다 → 진짜 창 객체를 흉내낸다.
    fakeWin = { location: { replace: vi.fn() }, close: vi.fn() }
    openSpy = vi.fn(() => fakeWin)
    vi.stubGlobal('open', openSpy)
    URL.createObjectURL = vi.fn(() => 'blob:mock')
    URL.revokeObjectURL = vi.fn()
    clickSpy = vi.fn()
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(clickSpy)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('downloads html instead of opening a tab', async () => {
    mockFetch('<html><script>alert(1)</script></html>', 'application/octet-stream')
    await openMilestoneFile({ id: 1, filename: 'plan.html' } as never)
    expect(openSpy).not.toHaveBeenCalled()
    expect(clickSpy).toHaveBeenCalled()
  })

  it('downloads html even if the server mislabels it as text/html', async () => {
    mockFetch('<html><script>alert(1)</script></html>', 'text/html')
    await openMilestoneFile({ id: 2, filename: 'plan.htm' } as never)
    expect(openSpy).not.toHaveBeenCalled()
    expect(clickSpy).toHaveBeenCalled()
  })

  it('still opens pdf inline', async () => {
    mockFetch('%PDF-1.4', 'application/pdf')
    await openMilestoneFile({ id: 3, filename: 'plan.pdf' } as never)
    // fetch 전에 미리 연 탭을 blob URL 로 이동시킨다(사용자 제스처 유지 → 팝업 차단 회피).
    expect(openSpy).toHaveBeenCalledWith('', '_blank')
    expect(fakeWin.location.replace).toHaveBeenCalledWith('blob:mock')
    expect(clickSpy).not.toHaveBeenCalled()
  })

  it('팝업이 차단되면(window.open → null) 다운로드로 대체한다', async () => {
    openSpy.mockReturnValue(null as never)
    mockFetch('%PDF-1.4', 'application/pdf')
    await openMilestoneFile({ id: 4, filename: 'plan.pdf' } as never)
    expect(clickSpy).toHaveBeenCalled()
  })

  it('inline 불가 타입이면 미리 연 탭을 닫는다', async () => {
    mockFetch('%PDF-1.4', 'application/octet-stream')
    await openMilestoneFile({ id: 5, filename: 'plan.pdf' } as never)
    expect(fakeWin.location.replace).not.toHaveBeenCalled()
    expect(fakeWin.close).toHaveBeenCalled()
    expect(clickSpy).toHaveBeenCalled()
  })

  // #184 DM 첨부도 같은 헬퍼를 쓴다 — 임의 URL 버전.
  it('openPrivateFile: inline 불가 타입은 octet-stream 으로 강제 후 다운로드한다', async () => {
    mockFetch('PK', 'application/zip')
    const createSpy = URL.createObjectURL as unknown as ReturnType<typeof vi.fn>
    await openPrivateFile('/api/dm/attachments/7', 'bundle.zip')
    expect(openSpy).not.toHaveBeenCalled()
    expect(clickSpy).toHaveBeenCalled()
    expect((createSpy.mock.calls[0]![0] as Blob).type).toBe('application/octet-stream')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      '/api/dm/attachments/7',
      expect.objectContaining({ headers: { Authorization: 'Bearer test-token' } }),
    )
  })
})
