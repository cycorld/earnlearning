import { describe, it, expect, vi, beforeEach, beforeAll } from 'vitest'
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders, setMockUser, mockAdmin } from '@/test/test-utils'
import AdminMailAddressesPage from './AdminMailAddressesPage'

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

// jsdom 폴리필: Radix Tabs
beforeAll(() => {
  Element.prototype.hasPointerCapture = vi.fn()
  Element.prototype.scrollIntoView = vi.fn()
  // @ts-expect-error jsdom 에 ResizeObserver 없음
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
})

const now = new Date().toISOString()

const pendingRows = [
  {
    id: 11,
    user_id: 100,
    user_name: '홍길동',
    user_email: 'gildong@ewha.ac.kr',
    local_part: 'gil.dong',
    status: 'pending',
    created_at: now,
    owner_type: 'user' as const,
    owner_name: '홍길동',
  },
  {
    id: 12,
    user_id: 101,
    user_name: '김대표',
    user_email: 'ceo@ewha.ac.kr',
    local_part: 'acompany',
    status: 'pending',
    created_at: now,
    owner_type: 'company' as const,
    owner_name: '에이컴퍼니',
  },
]

// 전체 계정(registry) — 혼합 상태
const allRows = [
  ...pendingRows,
  {
    id: 20,
    user_id: 102,
    user_name: '박대표',
    user_email: 'park@ewha.ac.kr',
    local_part: 'beta',
    status: 'approved',
    created_at: now,
    owner_type: 'company' as const,
    owner_name: '베타컴퍼니',
  },
  {
    id: 21,
    user_id: 103,
    user_name: '김학생',
    user_email: 'kim@ewha.ac.kr',
    local_part: 'badname',
    status: 'rejected',
    created_at: now,
    owner_type: 'user' as const,
    owner_name: '김학생',
  },
  {
    id: 22,
    user_id: 0,
    user_name: '',
    user_email: '',
    local_part: 'support',
    status: 'approved',
    created_at: now,
    owner_type: 'shared' as const,
    owner_name: '고객지원',
  },
]

const sharedRows = [
  {
    address_id: 50,
    local_part: 'support',
    display_name: '고객지원',
    email: 'support@earnlearning.com',
    grants: [{ user_id: 200, user_name: '이지원', revoked: false }],
  },
]

function setupDefault() {
  mockApiGet.mockImplementation((path: string) => {
    if (path.includes('/admin/mail/addresses')) {
      if (path.includes('status=all')) return Promise.resolve(allRows)
      return Promise.resolve(pendingRows)
    }
    if (path.includes('/admin/mail/shared')) return Promise.resolve(sharedRows)
    if (path.includes('/users/search')) return Promise.resolve([])
    return Promise.resolve([])
  })
  mockApiPost.mockResolvedValue({})
}

beforeEach(() => {
  vi.clearAllMocks()
  setMockUser(mockAdmin)
})

describe('AdminMailAddressesPage 승인 대기', () => {
  it('대기 중인 신청 행을 개인/회사 구분과 함께 렌더한다', async () => {
    setupDefault()
    renderWithProviders(<AdminMailAddressesPage />)

    await waitFor(() => {
      expect(screen.getByText('홍길동')).toBeInTheDocument()
    })
    expect(screen.getByText('에이컴퍼니')).toBeInTheDocument()
    // 구분 뱃지
    expect(screen.getByText('개인')).toBeInTheDocument()
    expect(screen.getByText('회사')).toBeInTheDocument()
    // 신청 주소
    expect(screen.getByText('gil.dong@earnlearning.com')).toBeInTheDocument()
    expect(screen.getByText('acompany@earnlearning.com')).toBeInTheDocument()
  })

  it('승인 버튼 클릭 시 approve API를 호출한다', async () => {
    setupDefault()
    renderWithProviders(<AdminMailAddressesPage />)

    await waitFor(() => {
      expect(screen.getAllByTitle('승인').length).toBeGreaterThan(0)
    })

    const user = userEvent.setup()
    await user.click(screen.getAllByTitle('승인')[0])

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith(
        expect.stringMatching(/\/admin\/mail\/addresses\/\d+\/approve/),
      )
    })
  })

  it('반려 버튼 클릭 시 reject API를 호출한다', async () => {
    setupDefault()
    renderWithProviders(<AdminMailAddressesPage />)

    await waitFor(() => {
      expect(screen.getAllByTitle('반려').length).toBeGreaterThan(0)
    })

    const user = userEvent.setup()
    await user.click(screen.getAllByTitle('반려')[0])

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith(
        expect.stringMatching(/\/admin\/mail\/addresses\/\d+\/reject/),
      )
    })
  })

  it('대기 신청이 없으면 빈 상태를 보여준다', async () => {
    mockApiGet.mockImplementation((path: string) => {
      if (path.includes('/admin/mail/shared')) return Promise.resolve([])
      return Promise.resolve([])
    })
    renderWithProviders(<AdminMailAddressesPage />)

    await waitFor(() => {
      expect(screen.getByText('대기 중인 신청이 없습니다.')).toBeInTheDocument()
    })
  })
})

describe('AdminMailAddressesPage 전체 계정', () => {
  it('전체 탭에서 혼합 상태 계정을 상태 뱃지와 함께 렌더한다', async () => {
    setupDefault()
    const user = userEvent.setup()
    renderWithProviders(<AdminMailAddressesPage />)

    await waitFor(() => {
      expect(screen.getByText('전체 계정 (5)')).toBeInTheDocument()
    })

    await user.click(screen.getByRole('tab', { name: /전체 계정/ }))

    // 혼합 상태 계정 노출
    await waitFor(() => {
      expect(screen.getByText('베타컴퍼니')).toBeInTheDocument()
    })
    expect(screen.getByText('badname@earnlearning.com')).toBeInTheDocument()
    // 상태 뱃지: 승인됨 2건, 반려됨 1건, 대기중 존재
    expect(screen.getAllByText('승인됨').length).toBe(2)
    expect(screen.getByText('반려됨')).toBeInTheDocument()
    expect(screen.getAllByText('대기중').length).toBeGreaterThan(0)
    // 공용 소유 뱃지 노출
    expect(screen.getByText('공용')).toBeInTheDocument()
  })

  it('반려된 계정의 승인 버튼이 approve API를 POST 한다 (관리자 오버라이드)', async () => {
    setupDefault()
    const user = userEvent.setup()
    renderWithProviders(<AdminMailAddressesPage />)

    await waitFor(() => {
      expect(screen.getByText('전체 계정 (5)')).toBeInTheDocument()
    })
    await user.click(screen.getByRole('tab', { name: /전체 계정/ }))

    // 반려 행(badname)을 찾아 그 안의 승인 버튼 클릭
    const rejectedRow = (
      await screen.findByText('badname@earnlearning.com')
    ).closest('.rounded-lg') as HTMLElement
    await user.click(within(rejectedRow).getByTitle('승인'))

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith(
        '/admin/mail/addresses/21/approve',
      )
    })
  })
})

describe('AdminMailAddressesPage 공용 메일함', () => {
  it('공용 메일함 탭에서 생성 폼이 POST 한다', async () => {
    setupDefault()
    const user = userEvent.setup()
    renderWithProviders(<AdminMailAddressesPage />)

    await waitFor(() => {
      expect(screen.getByText('승인 대기 (2)')).toBeInTheDocument()
    })

    // 공용 메일함 탭으로 이동
    await user.click(screen.getByRole('tab', { name: /공용 메일함/ }))

    const localInput = await screen.findByPlaceholderText('hello')
    const nameInput = screen.getByPlaceholderText('고객지원')
    await user.type(localInput, 'newshared')
    await user.type(nameInput, '신규팀')
    await user.click(screen.getByRole('button', { name: /만들기/ }))

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith('/admin/mail/shared', {
        local_part: 'newshared',
        display_name: '신규팀',
      })
    })
  })

  it('사용자를 검색해 권한을 부여하면 grants API를 POST 한다', async () => {
    mockApiGet.mockImplementation((path: string) => {
      if (path.includes('/admin/mail/addresses')) return Promise.resolve([])
      if (path.includes('/admin/mail/shared')) return Promise.resolve(sharedRows)
      if (path.includes('/users/search'))
        return Promise.resolve([
          {
            id: 300,
            name: '박신입',
            department: '경영학과',
            student_id: '2026001234',
            avatar_url: '',
          },
        ])
      return Promise.resolve([])
    })
    const user = userEvent.setup()
    renderWithProviders(<AdminMailAddressesPage />)

    await waitFor(() => {
      expect(screen.getByText('승인 대기 (0)')).toBeInTheDocument()
    })
    await user.click(screen.getByRole('tab', { name: /공용 메일함/ }))

    const searchInput = await screen.findByPlaceholderText(
      '사용자 이름·학번으로 검색',
    )
    await user.type(searchInput, '박신입')

    // 검색 결과가 뜨면 클릭
    const result = await screen.findByText('박신입')
    await user.click(result)

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith('/admin/mail/shared/50/grants', {
        user_id: 300,
      })
    })
  })

  it('권한 부여 후 refetch 되어도 공용 메일함 탭이 유지된다', async () => {
    mockApiGet.mockImplementation((path: string) => {
      if (path.includes('/admin/mail/addresses')) {
        if (path.includes('status=all')) return Promise.resolve(allRows)
        return Promise.resolve(pendingRows)
      }
      if (path.includes('/admin/mail/shared')) return Promise.resolve(sharedRows)
      if (path.includes('/users/search'))
        return Promise.resolve([
          {
            id: 300,
            name: '박신입',
            department: '경영학과',
            student_id: '2026001234',
            avatar_url: '',
          },
        ])
      return Promise.resolve([])
    })
    mockApiPost.mockResolvedValue({})

    const user = userEvent.setup()
    renderWithProviders(<AdminMailAddressesPage />)

    await waitFor(() => {
      expect(screen.getByText('승인 대기 (2)')).toBeInTheDocument()
    })
    await user.click(screen.getByRole('tab', { name: /공용 메일함/ }))

    const searchInput = await screen.findByPlaceholderText(
      '사용자 이름·학번으로 검색',
    )
    await user.type(searchInput, '박신입')
    await user.click(await screen.findByText('박신입'))

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith('/admin/mail/shared/50/grants', {
        user_id: 300,
      })
    })

    // refetch(로딩 스왑) 후에도 공용 메일함 탭이 활성 상태로 유지
    await waitFor(() => {
      expect(screen.getByText('새 공용 메일함')).toBeInTheDocument()
    })
    expect(screen.getByRole('tab', { name: /공용 메일함/ })).toHaveAttribute(
      'data-state',
      'active',
    )
  })

  it('권한 회수 버튼이 revoke API를 POST 한다', async () => {
    setupDefault()
    const user = userEvent.setup()
    renderWithProviders(<AdminMailAddressesPage />)

    await waitFor(() => {
      expect(screen.getByText('승인 대기 (2)')).toBeInTheDocument()
    })
    await user.click(screen.getByRole('tab', { name: /공용 메일함/ }))

    const revokeBtn = await screen.findByTitle('권한 회수')
    await user.click(revokeBtn)

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith(
        '/admin/mail/shared/50/grants/200/revoke',
      )
    })
  })
})
