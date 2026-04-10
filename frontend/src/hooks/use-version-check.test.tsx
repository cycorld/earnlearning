import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { createElement } from 'react'

// 빌드 define 흉내. 테스트에서는 commit_sha 'embedded' 로 고정.
// (vite.config.ts 의 define 은 vitest 빌드에는 적용 안 되므로 globalThis 로 주입)
vi.stubGlobal('__BUILD_NUMBER__', '100')
vi.stubGlobal('__COMMIT_SHA__', 'embedded')

// vi.mock 은 hoisted 되므로 외부 변수 참조 불가 → 팩토리 안에서 직접 정의 후 export
vi.mock('sonner', () => {
  const toastFn = vi.fn()
  return {
    toast: Object.assign(toastFn, {
      success: vi.fn(),
      error: vi.fn(),
      info: vi.fn(),
    }),
  }
})

import { toast as mockedToast } from 'sonner'
const toastMock = mockedToast as unknown as ReturnType<typeof vi.fn>

import { useVersionCheck } from './use-version-check'

function TestComponent() {
  useVersionCheck()
  return null
}

function renderHook() {
  return render(createElement(MemoryRouter, null, createElement(TestComponent)))
}

describe('useVersionCheck', () => {
  let fetchMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    toastMock.mockClear()

    // 매 테스트마다 module-level toastShown flag 도 리셋해야 함
    // (아쉽게도 module 내부 변수라 외부에서 직접 못 건드림 → toast 의 onDismiss 호출)
    // 가장 깔끔한 건 vi.resetModules + dynamic import 인데 코드량 늘어남.
    // 지금은 매 테스트가 새 fetch mock 으로 다른 결과를 받게만 보장.
    fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    vi.stubGlobal('__BUILD_NUMBER__', '100')
    vi.stubGlobal('__COMMIT_SHA__', 'embedded')
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('서버 버전이 임베드된 버전과 같으면 토스트가 뜨지 않는다', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        data: { build_number: '100', commit_sha: 'embedded' },
      }),
    })

    renderHook()

    // 초기 fetch 후 micro-task drain
    await new Promise((r) => setTimeout(r, 50))
    expect(toastMock).not.toHaveBeenCalled()
  })

  it('서버 버전이 다르면 토스트가 노출된다', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({
        data: { build_number: '101', commit_sha: 'newhash' },
      }),
    })

    renderHook()
    await new Promise((r) => setTimeout(r, 50))

    expect(toastMock).toHaveBeenCalledTimes(1)
    const [message, opts] = toastMock.mock.calls[0]
    expect(message).toContain('새 버전')
    expect(opts.action).toBeDefined()
    expect(opts.action.label).toContain('새로고침')
  })

  it('fetch 가 실패하면 토스트는 안 뜬다 (조용한 실패)', async () => {
    fetchMock.mockRejectedValue(new Error('network'))
    renderHook()
    await new Promise((r) => setTimeout(r, 50))
    expect(toastMock).not.toHaveBeenCalled()
  })
})
