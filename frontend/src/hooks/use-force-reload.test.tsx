import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, act } from '@testing-library/react'
import { createElement } from 'react'

// sonner mock (toast)
vi.mock('sonner', () => {
  const toastFn = vi.fn()
  return {
    toast: Object.assign(toastFn, {
      success: vi.fn(),
      error: vi.fn(),
      info: vi.fn(),
      dismiss: vi.fn(),
    }),
  }
})

// wsClient mock — 훅이 등록하는 핸들러를 잡아 두었다가 테스트에서 직접 호출.
let capturedHandler: ((data: unknown) => void) | null = null
const unsubscribeMock = vi.fn()
vi.mock('@/lib/ws', () => ({
  wsClient: {
    on: vi.fn((_event: string, cb: (data: unknown) => void) => {
      capturedHandler = cb
      return unsubscribeMock
    }),
  },
}))

import { toast as mockedToast } from 'sonner'
const toastMock = mockedToast as unknown as ReturnType<typeof vi.fn> & {
  dismiss: ReturnType<typeof vi.fn>
}

import { useForceReload } from './use-force-reload'

function TestComponent() {
  useForceReload()
  return null
}

function renderHook() {
  return render(createElement(TestComponent))
}

describe('useForceReload', () => {
  let replaceMock: ReturnType<typeof vi.fn>
  let originalLocation: Location

  beforeEach(() => {
    toastMock.mockClear()
    toastMock.dismiss.mockClear()
    capturedHandler = null
    unsubscribeMock.mockClear()

    originalLocation = window.location
    replaceMock = vi.fn()
    Object.defineProperty(window, 'location', {
      writable: true,
      configurable: true,
      value: { href: 'http://localhost/', replace: replaceMock },
    })

    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
    Object.defineProperty(window, 'location', {
      writable: true,
      configurable: true,
      value: originalLocation,
    })
  })

  it('force_reload 이벤트를 wsClient 에 구독한다', () => {
    renderHook()
    expect(capturedHandler).toBeTruthy()
  })

  it('이벤트 수신 시 토스트를 띄운다', () => {
    renderHook()
    act(() => {
      capturedHandler!({ reason: '청산 롤아웃' })
    })

    expect(toastMock).toHaveBeenCalled()
    const [title, opts] = toastMock.mock.calls[0]
    expect(title).toContain('관리자 강제 새로고침')
    expect(opts.description).toContain('5초')
    expect(opts.description).toContain('청산 롤아웃')
    expect(opts.action.label).toBe('취소')
  })

  it('5초 카운트다운 후 자동 새로고침 (location.replace 호출)', async () => {
    renderHook()
    act(() => {
      capturedHandler!({ reason: 'rollout' })
    })

    // 5초 경과
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000)
    })

    expect(replaceMock).toHaveBeenCalledTimes(1)
  })

  it('사용자가 취소를 누르면 새로고침 안 한다', async () => {
    renderHook()
    act(() => {
      capturedHandler!({ reason: 'rollout' })
    })

    // 2초 후 취소
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000)
    })

    const opts = toastMock.mock.calls[toastMock.mock.calls.length - 1][1]
    act(() => {
      opts.action.onClick()
    })

    // 남은 시간 경과
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000)
    })

    expect(replaceMock).not.toHaveBeenCalled()
    expect(toastMock.dismiss).toHaveBeenCalledWith('force-reload')
  })

  it('중복 브로드캐스트가 오면 무시한다 (카운트다운 진행 중)', () => {
    renderHook()
    act(() => {
      capturedHandler!({ reason: 'first' })
    })
    const firstCallCount = toastMock.mock.calls.length

    act(() => {
      capturedHandler!({ reason: 'duplicate' })
    })

    // 두 번째 호출은 아무 동작 없음 — toast 호출 수 증가 없음
    expect(toastMock.mock.calls.length).toBe(firstCallCount)
  })

  it('카운트다운 중 description 이 매초 감소한다', async () => {
    renderHook()
    act(() => {
      capturedHandler!({ reason: '' })
    })

    // 초기: 5초
    expect(toastMock.mock.calls[0][1].description).toContain('5초')

    // 1초 경과: 4초
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })
    const latest = toastMock.mock.calls[toastMock.mock.calls.length - 1][1]
    expect(latest.description).toContain('4초')
  })

  it('언마운트 시 ws 구독이 해제된다', () => {
    const { unmount } = renderHook()
    unmount()
    expect(unsubscribeMock).toHaveBeenCalled()
  })
})
