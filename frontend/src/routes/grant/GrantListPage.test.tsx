import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders, setMockUser, mockAdmin, mockStudent } from '@/test/test-utils'
import GrantListPage from './GrantListPage'

const mockApiGet = vi.fn()

vi.mock('@/lib/api', () => ({
  api: {
    get: (...args: unknown[]) => mockApiGet(...args),
    post: vi.fn(),
    put: vi.fn(),
    del: vi.fn(),
  },
}))

const mockGrants = [
  { id: 1, title: '리액트 스터디', description: '리액트 기초', reward: 5000, max_applicants: 10, status: 'open', created_at: '2026-01-01T00:00:00Z' },
  { id: 2, title: '파이썬 과제', description: '파이썬 실습', reward: 3000, max_applicants: 0, status: 'open', created_at: '2026-01-02T00:00:00Z' },
  { id: 3, title: '종료된 과제', description: '이미 종료', reward: 1000, max_applicants: 5, status: 'closed', created_at: '2026-01-03T00:00:00Z' },
]

function setupMocks(grants = mockGrants) {
  mockApiGet.mockImplementation((path: string) => {
    if (path.includes('/grants')) {
      const url = new URLSearchParams(path.split('?')[1])
      const status = url.get('status')
      const filtered = status ? grants.filter(g => g.status === status) : grants
      return Promise.resolve({
        data: filtered,
        pagination: { page: 1, limit: 20, total: filtered.length, total_pages: 1 },
      })
    }
    return Promise.resolve([])
  })
}

beforeEach(() => {
  vi.clearAllMocks()
  setMockUser(mockStudent)
})

describe('GrantListPage', () => {
  it('과제 목록이 렌더링된다', async () => {
    setupMocks()
    renderWithProviders(<GrantListPage />)

    await waitFor(() => {
      expect(screen.getByText('리액트 스터디')).toBeInTheDocument()
      expect(screen.getByText('파이썬 과제')).toBeInTheDocument()
      expect(screen.getByText('종료된 과제')).toBeInTheDocument()
    })
  })

  it('보상 금액이 표시된다', async () => {
    setupMocks()
    renderWithProviders(<GrantListPage />)

    await waitFor(() => {
      expect(screen.getByText(/5,000원/)).toBeInTheDocument()
      expect(screen.getByText(/3,000원/)).toBeInTheDocument()
    })
  })

  it('정원이 표시된다 (max_applicants > 0)', async () => {
    setupMocks()
    renderWithProviders(<GrantListPage />)

    await waitFor(() => {
      expect(screen.getByText('정원 10명')).toBeInTheDocument()
    })
  })

  it('과제가 없으면 빈 상태 메시지가 표시된다', async () => {
    setupMocks([])
    renderWithProviders(<GrantListPage />)

    await waitFor(() => {
      expect(screen.getByText('등록된 과제가 없습니다.')).toBeInTheDocument()
    })
  })

  it('학생에게는 과제 등록 버튼이 보이지 않는다', async () => {
    setMockUser(mockStudent)
    setupMocks()
    renderWithProviders(<GrantListPage />)

    await waitFor(() => {
      expect(screen.getByText('리액트 스터디')).toBeInTheDocument()
    })
    expect(screen.queryByText('과제 등록')).not.toBeInTheDocument()
  })

  it('관리자에게는 과제 등록 버튼이 보인다', async () => {
    setMockUser(mockAdmin)
    setupMocks()
    renderWithProviders(<GrantListPage />)

    await waitFor(() => {
      expect(screen.getByText('과제 등록')).toBeInTheDocument()
    })
  })

  it('상태 필터 셀렉트 박스가 렌더링된다', async () => {
    setupMocks()
    renderWithProviders(<GrantListPage />)

    await waitFor(() => {
      expect(screen.getByText('리액트 스터디')).toBeInTheDocument()
    })

    expect(screen.getByRole('combobox')).toBeInTheDocument()
  })
})
