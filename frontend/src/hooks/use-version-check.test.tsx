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

import { useVersionCheck, __testing } from './use-version-check'

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

// #028 — 자동 refresh 로직. checkAndNotify 를 직접 N회 호출해 모듈 상태 전이를 검증.
// 마운트 라이프사이클(useEffect) 은 위쪽 describe 가 이미 커버.
describe('useVersionCheck — 자동 refresh (#028)', () => {
  let fetchMock: ReturnType<typeof vi.fn>
  let replaceMock: ReturnType<typeof vi.fn>
  let originalLocation: Location

  beforeEach(() => {
    toastMock.mockClear()
    __testing.resetState()

    fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    vi.stubGlobal('__BUILD_NUMBER__', '100')
    vi.stubGlobal('__COMMIT_SHA__', 'embedded')

    // forceRefresh 가 호출하는 window.location.replace 를 스파이로.
    originalLocation = window.location
    replaceMock = vi.fn()
    Object.defineProperty(window, 'location', {
      writable: true,
      configurable: true,
      value: {
        href: 'http://localhost/',
        replace: replaceMock,
      },
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    Object.defineProperty(window, 'location', {
      writable: true,
      configurable: true,
      value: originalLocation,
    })
  })

  function mockVersion(build: string, sha: string) {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ data: { build_number: build, commit_sha: sha } }),
    })
  }

  it('같은 신버전을 3회 연속 감지하면 자동으로 새로고침한다', async () => {
    mockVersion('101', 'newhash')

    await __testing.checkAndNotify() // count=1 → 토스트
    await __testing.checkAndNotify() // count=2 → 토스트 이미 있음
    await __testing.checkAndNotify() // count=3 → 자동 refresh

    expect(replaceMock).toHaveBeenCalledTimes(1)
  })

  it('2회 감지까지는 자동 새로고침 안 한다', async () => {
    mockVersion('101', 'newhash')

    await __testing.checkAndNotify()
    await __testing.checkAndNotify()

    expect(replaceMock).not.toHaveBeenCalled()
    expect(toastMock).toHaveBeenCalledTimes(1) // 토스트는 1번 노출 (중복 방지)
  })

  it('input 에 포커스가 있으면 3회 연속이어도 자동 새로고침 안 한다', async () => {
    const input = document.createElement('input')
    document.body.appendChild(input)
    input.focus()

    mockVersion('101', 'newhash')

    await __testing.checkAndNotify()
    await __testing.checkAndNotify()
    await __testing.checkAndNotify()

    expect(replaceMock).not.toHaveBeenCalled()
    document.body.removeChild(input)
  })

  it('열려있는 dialog 가 있으면 자동 새로고침 안 한다', async () => {
    const dialog = document.createElement('div')
    dialog.setAttribute('role', 'dialog')
    dialog.setAttribute('data-state', 'open')
    document.body.appendChild(dialog)

    mockVersion('101', 'newhash')

    await __testing.checkAndNotify()
    await __testing.checkAndNotify()
    await __testing.checkAndNotify()

    expect(replaceMock).not.toHaveBeenCalled()
    document.body.removeChild(dialog)
  })

  it('사용자가 토스트를 dismiss 하면 같은 버전은 자동 새로고침 대상에서 제외된다', async () => {
    mockVersion('101', 'newhash')

    await __testing.checkAndNotify() // 1회 → 토스트
    // 사용자가 토스트를 닫음
    const opts = toastMock.mock.calls[0][1]
    opts.onDismiss()

    await __testing.checkAndNotify() // 2회
    await __testing.checkAndNotify() // 3회 — 평소라면 자동 refresh 이지만 dismiss 됨

    expect(replaceMock).not.toHaveBeenCalled()
  })

  it('또 다른 신버전이 등장하면 감지 카운트가 초기화된다', async () => {
    // 1회차: v101
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: { build_number: '101', commit_sha: 'hash1' } }),
    })
    // 2~4회차: v102 (다른 신버전)
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ data: { build_number: '102', commit_sha: 'hash2' } }),
    })

    await __testing.checkAndNotify() // v101, count=1
    await __testing.checkAndNotify() // v102 (다른 버전) → count=1
    await __testing.checkAndNotify() // v102 → count=2

    expect(replaceMock).not.toHaveBeenCalled()

    await __testing.checkAndNotify() // v102 → count=3 → auto refresh
    expect(replaceMock).toHaveBeenCalledTimes(1)
  })

  it('서버 버전이 임베드 버전과 같아지면 카운터가 리셋된다 (롤백 대비)', async () => {
    mockVersion('101', 'newhash')
    await __testing.checkAndNotify() // count=1
    await __testing.checkAndNotify() // count=2

    // 서버 롤백 시뮬레이션: 서버 버전이 우리 임베드 버전과 같아짐
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ data: { build_number: '100', commit_sha: 'embedded' } }),
    })
    await __testing.checkAndNotify() // 리셋되어야 함

    // 다시 새 버전 올라옴 (같은 v101 이지만 카운터는 0부터 다시 셈)
    mockVersion('101', 'newhash')
    await __testing.checkAndNotify() // count=1
    await __testing.checkAndNotify() // count=2

    expect(replaceMock).not.toHaveBeenCalled()
  })
})

describe('useVersionCheck — isSafeToReload', () => {
  it('기본 상태에서는 true', () => {
    expect(__testing.isSafeToReload()).toBe(true)
  })

  it('textarea 에 포커스가 있으면 false', () => {
    const ta = document.createElement('textarea')
    document.body.appendChild(ta)
    ta.focus()
    expect(__testing.isSafeToReload()).toBe(false)
    document.body.removeChild(ta)
  })

  it('열린 dialog 가 있으면 false', () => {
    const dialog = document.createElement('div')
    dialog.setAttribute('role', 'dialog')
    dialog.setAttribute('data-state', 'open')
    document.body.appendChild(dialog)
    expect(__testing.isSafeToReload()).toBe(false)
    document.body.removeChild(dialog)
  })
})
