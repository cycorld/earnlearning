import { describe, it, expect, vi, beforeEach, beforeAll } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  renderWithProviders,
  setMockUser,
  mockStudent,
} from '@/test/test-utils'
import ClassroomSwitcher from './ClassroomSwitcher'

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

// ─── jsdom 폴리필: Radix 드롭다운/다이얼로그는 포인터·스크롤 API 를 쓴다 ──
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

const reloadMock = vi.fn()

beforeEach(() => {
  vi.clearAllMocks()
  Object.defineProperty(window, 'location', {
    configurable: true,
    value: { ...window.location, reload: reloadMock },
  })
  // 활성 강의실 id=1 을 가진 학생
  setMockUser({ ...mockStudent, active_classroom_id: 1 })
  mockApiGet.mockResolvedValue([{ id: 1, name: 'A반', code: 'ABC123', initial_capital: 0 }])
  mockApiPost.mockResolvedValue({})
})

describe('ClassroomSwitcher', () => {
  // #159 회귀: 강의실이 1개여도 드롭다운(전환/입장 진입점)으로 렌더돼야 한다.
  it('강의실이 1개여도 드롭다운 버튼으로 렌더된다', async () => {
    renderWithProviders(<ClassroomSwitcher />)
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /A반/ })).toBeInTheDocument()
    })
  })

  // #178 반응형: 트리거가 모바일에선 전체 너비, 데스크톱에선 auto 너비여야 한다.
  it('트리거 버튼이 모바일 전체 너비(w-full)·데스크톱 auto(sm:w-auto) 클래스를 갖는다', async () => {
    renderWithProviders(<ClassroomSwitcher />)
    const trigger = await screen.findByRole('button', { name: /A반/ })
    expect(trigger.className).toContain('w-full')
    expect(trigger.className).toContain('sm:w-auto')
  })

  // #159 회귀: 드롭다운에 "초대 코드로 입장" 항목이 있고, 코드 제출 시 조인 API 를 호출한다.
  it('초대 코드로 입장 항목으로 새 강의실에 참여한다', async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 })
    renderWithProviders(<ClassroomSwitcher />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /A반/ })).toBeInTheDocument()
    })

    // 드롭다운 열기 → 입장 항목 노출
    await user.click(screen.getByRole('button', { name: /A반/ }))
    const joinItem = await screen.findByText('초대 코드로 입장')
    expect(joinItem).toBeInTheDocument()

    // 항목 클릭 → 다이얼로그 오픈
    await user.click(joinItem)
    const input = await screen.findByPlaceholderText(/초대 코드/)

    // 코드 입력 후 제출
    await user.type(input, 'xyz789')
    await user.click(screen.getByRole('button', { name: '강의실 입장' }))

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith('/classrooms/join', { code: 'XYZ789' })
    })
  })
})
