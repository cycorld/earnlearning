import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { openMilestoneFile } from './milestoneFiles'

// #176 — HTML 첨부는 절대 새 탭(blob: same-origin)으로 열리면 안 된다.

vi.mock('./auth', () => ({ getToken: () => 'test-token' }))
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

  beforeEach(() => {
    openSpy = vi.fn()
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
    expect(openSpy).toHaveBeenCalled()
  })
})
