import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { createElement } from 'react'
import PendingPage from './PendingPage'

// ─── Mocks ─────────────────────────────────────────────────
const mockNavigate = vi.fn()
const mockRefreshUser = vi.fn().mockResolvedValue(undefined)
const mockLogout = vi.fn()
const mockApiPost = vi.fn()

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return { ...actual, useNavigate: () => mockNavigate }
})

// pending 유저 고정 — refresh 응답으로만 승인 전이를 검증한다.
vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({
    user: {
      id: 2,
      email: 'pending@test.com',
      name: '대기학생',
      role: 'student',
      status: 'pending',
      department: '컴퓨터공학과',
      student_id: '2026000001',
      bio: '',
      avatar_url: '',
    },
    isLoading: false,
    login: vi.fn(),
    register: vi.fn(),
    logout: mockLogout,
    refreshUser: mockRefreshUser,
  }),
}))

vi.mock('@/hooks/use-ws', () => ({
  useWebSocket: vi.fn(),
}))

vi.mock('@/lib/api', () => ({
  api: {
    get: vi.fn(),
    post: (...args: unknown[]) => mockApiPost(...args),
    put: vi.fn(),
    del: vi.fn(),
  },
}))

function renderPending() {
  return render(createElement(MemoryRouter, null, createElement(PendingPage)))
}

// ─── Tests ─────────────────────────────────────────────────
describe('PendingPage', () => {
  let originalLocation: Location
  let replaceMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    mockRefreshUser.mockResolvedValue(undefined)
    localStorage.clear()
    vi.useFakeTimers()

    // 승인 시 호출되는 window.location.replace 를 스파이로 (jsdom 은 실제 이동 미구현).
    originalLocation = window.location
    replaceMock = vi.fn()
    Object.defineProperty(window, 'location', {
      writable: true,
      configurable: true,
      value: { href: 'http://localhost/', replace: replaceMock },
    })
  })

  afterEach(() => {
    vi.useRealTimers()
    Object.defineProperty(window, 'location', {
      writable: true,
      configurable: true,
      value: originalLocation,
    })
  })

  it('승인 대기 화면을 렌더하고 5초마다 refresh 를 폴링한다', async () => {
    mockApiPost.mockResolvedValue({ token: 'still-pending', user: { status: 'pending' } })
    renderPending()

    expect(screen.getByText('관리자 승인을 기다리고 있습니다.')).toBeInTheDocument()
    // 마운트 직후에는 아직 폴링 전
    expect(mockApiPost).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(5000)

    expect(mockApiPost).toHaveBeenCalledWith('/auth/refresh')
  })

  it('승인되면 새 토큰을 저장하고 /feed 로 전체 새로고침한다', async () => {
    mockApiPost.mockResolvedValue({ token: 'approved-token', user: { status: 'approved' } })
    renderPending()

    await vi.advanceTimersByTimeAsync(5000)

    expect(mockApiPost).toHaveBeenCalledWith('/auth/refresh')
    expect(localStorage.getItem('el_token')).toBe('approved-token')
    // SPA 이동이 아니라 전체 새로고침으로 앱을 재부팅 (WS 재연결 목적)
    expect(replaceMock).toHaveBeenCalledWith('/feed')
    expect(mockNavigate).not.toHaveBeenCalled()
  })

  it('탭 복귀(visibilitychange) 시 5초를 기다리지 않고 즉시 refresh 를 확인한다', async () => {
    mockApiPost.mockResolvedValue({ token: 'still-pending', user: { status: 'pending' } })
    renderPending()

    expect(mockApiPost).not.toHaveBeenCalled()

    Object.defineProperty(document, 'visibilityState', {
      value: 'visible',
      configurable: true,
    })
    document.dispatchEvent(new Event('visibilitychange'))

    // 인터벌(5초) 진행 없이도 즉시 호출되어야 한다
    expect(mockApiPost).toHaveBeenCalledWith('/auth/refresh')
    // 대기 중인 마이크로태스크 정리
    await vi.advanceTimersByTimeAsync(0)
  })

  it('refresh 가 계속 pending 을 반환하는 동안은 이동하지 않는다', async () => {
    mockApiPost.mockResolvedValue({ token: 'still-pending', user: { status: 'pending' } })
    renderPending()

    await vi.advanceTimersByTimeAsync(15000) // 3회 폴링

    expect(mockApiPost.mock.calls.length).toBeGreaterThanOrEqual(3)
    expect(mockNavigate).not.toHaveBeenCalled()
  })
})
