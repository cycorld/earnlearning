import { describe, it, expect, vi, beforeEach, beforeAll } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import {
  renderWithProviders,
  setMockUser,
  mockStudent,
} from '@/test/test-utils'
import Header from './Header'

// ─── API Mock ─────────────────────────────────────────────────
const mockApiGet = vi.fn()
const mockApiPost = vi.fn()

vi.mock('@/lib/api', () => ({
  api: {
    get: (...args: unknown[]) => mockApiGet(...args),
    post: (...args: unknown[]) => mockApiPost(...args),
    put: vi.fn(),
    del: vi.fn(),
  },
  ApiError: class extends Error {
    code: string
    status: number
    constructor(code: string, message: string, status: number) {
      super(message)
      this.code = code
      this.status = status
    }
  },
}))

// WebSocket 훅은 헤더가 실시간 배지 갱신에 쓰지만 테스트에선 무력화한다.
vi.mock('@/hooks/use-ws', () => ({ useWebSocket: vi.fn() }))

// ─── jsdom 폴리필: 헤더 안의 ClassroomSwitcher(Radix 드롭다운)가 쓰는 API ──
beforeAll(() => {
  Element.prototype.hasPointerCapture = vi.fn()
  Element.prototype.setPointerCapture = vi.fn()
  Element.prototype.releasePointerCapture = vi.fn()
  Element.prototype.scrollIntoView = vi.fn()
  // @ts-expect-error jsdom 에 ResizeObserver 없음
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
})

beforeEach(() => {
  vi.clearAllMocks()
  setMockUser({ ...mockStudent, active_classroom_id: 1 })
  mockApiGet.mockImplementation((path: string) => {
    if (path === '/classrooms')
      return Promise.resolve([
        { id: 1, name: 'A반' },
        { id: 2, name: 'B반' },
      ])
    if (path.startsWith('/notifications'))
      return Promise.resolve({ unread_count: 3 })
    if (path === '/dm/unread-count')
      return Promise.resolve({ unread_count: 0 })
    return Promise.resolve({})
  })
  mockApiPost.mockResolvedValue({})
})

describe('#178 반응형 헤더', () => {
  it('a. 헤더 내부 컨테이너가 모바일 2줄(flex-wrap)·데스크톱 1줄(sm:flex-nowrap)로 감싸진다', () => {
    const { container } = renderWithProviders(<Header />)
    const inner = container.querySelector('header > div')!
    expect(inner).not.toBeNull()
    expect(inner.classList.contains('flex-wrap')).toBe(true)
    expect(inner.classList.contains('sm:flex-nowrap')).toBe(true)
  })

  it('b. ClassroomSwitcher 래퍼가 모바일에선 전체 너비로 줄바꿈되고 현재 강의실명이 보인다', async () => {
    const { container } = renderWithProviders(<Header />)
    // 현재 강의실 이름이 스위처 트리거에 렌더된다
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /A반/ })).toBeInTheDocument()
    })
    const wrapper = container.querySelector('.basis-full')!
    expect(wrapper).not.toBeNull()
    expect(wrapper.classList.contains('basis-full')).toBe(true)
    expect(wrapper.classList.contains('order-last')).toBe(true)
    expect(wrapper.classList.contains('sm:basis-auto')).toBe(true)
    expect(wrapper.classList.contains('sm:order-none')).toBe(true)
  })

  it('c. 메시지·알림 아이콘 링크가 44px 이상 탭 타깃(h-11 w-11)을 갖는다', () => {
    renderWithProviders(<Header />)
    const messages = screen.getByRole('link', { name: '메시지' })
    const notifications = screen.getByRole('link', { name: '알림' })
    expect(messages.className).toContain('h-11')
    expect(messages.className).toContain('w-11')
    expect(notifications.className).toContain('h-11')
    expect(notifications.className).toContain('w-11')
  })

  it('d. 커밋 sha 부분은 모바일에서 숨기고(hidden sm:inline) 빌드 번호는 항상 보인다', () => {
    renderWithProviders(<Header />)
    // 테스트 환경에선 빌드 번호가 'dev', 커밋 sha 가 'local' 로 렌더된다.
    const buildSpan = screen.getByText('dev')
    expect(buildSpan.className).not.toContain('hidden')
    // 커밋 sha 는 별도 span 에 담겨 모바일에서 숨겨진다
    const shaSpan = screen.getByText(/local/)
    expect(shaSpan.className).toContain('hidden')
    expect(shaSpan.className).toContain('sm:inline')
  })

  it('e. 알림 링크에 안읽음 배지 3 이 표시된다(구조 개편 회귀)', async () => {
    renderWithProviders(<Header />)
    await waitFor(() => {
      expect(screen.getByText('3')).toBeInTheDocument()
    })
  })
})
