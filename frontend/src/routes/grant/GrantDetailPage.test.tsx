import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders, setMockUser, mockAdmin, mockStudent } from '@/test/test-utils'
import GrantDetailPage from './GrantDetailPage'

const mockApiGet = vi.fn()
const mockApiPost = vi.fn()

vi.mock('@/lib/api', () => ({
  api: {
    get: (...args: unknown[]) => mockApiGet(...args),
    post: (...args: unknown[]) => mockApiPost(...args),
    put: vi.fn(),
    del: vi.fn(),
  },
}))

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useParams: () => ({ id: '1' }),
  }
})

const mockGrant = {
  id: 1,
  admin_id: 1,
  title: '리액트 스터디',
  description: '리액트 기초 학습 과제입니다',
  reward: 5000,
  max_applicants: 10,
  status: 'open',
  admin: { id: 1, name: '최용철' },
  applications: [],
  created_at: '2026-01-01T00:00:00Z',
}

const mockGrantWithApps = {
  ...mockGrant,
  applications: [
    { id: 1, grant_id: 1, user_id: 2, proposal: '지원합니다', status: 'pending', user: { id: 2, name: '김학생' }, created_at: '2026-01-02T00:00:00Z' },
    { id: 2, grant_id: 1, user_id: 3, proposal: '저도 지원합니다', status: 'approved', user: { id: 3, name: '이개발' }, created_at: '2026-01-03T00:00:00Z' },
  ],
}

function setupMocks(grant = mockGrant) {
  mockApiGet.mockImplementation(() => Promise.resolve(grant))
  mockApiPost.mockImplementation(() => Promise.resolve({ id: 99, grant_id: 1, proposal: '지원', status: 'pending' }))
}

beforeEach(() => {
  vi.clearAllMocks()
  setMockUser(mockStudent)
})

describe('GrantDetailPage', () => {
  it('과제 상세 정보가 렌더링된다', async () => {
    setupMocks()
    renderWithProviders(<GrantDetailPage />)

    await waitFor(() => {
      expect(screen.getByText('리액트 스터디')).toBeInTheDocument()
      expect(screen.getByText(/5,000원/)).toBeInTheDocument()
      expect(screen.getByText('10명')).toBeInTheDocument()
    })
  })

  it('과제 설명이 표시된다', async () => {
    setupMocks()
    renderWithProviders(<GrantDetailPage />)

    await waitFor(() => {
      expect(screen.getByText(/리액트 기초 학습 과제입니다/)).toBeInTheDocument()
    })
  })

  it('모집 중 상태 뱃지가 표시된다', async () => {
    setupMocks()
    renderWithProviders(<GrantDetailPage />)

    await waitFor(() => {
      expect(screen.getByText('모집 중')).toBeInTheDocument()
    })
  })

  it('학생에게 지원 버튼이 표시된다', async () => {
    setupMocks()
    renderWithProviders(<GrantDetailPage />)

    await waitFor(() => {
      expect(screen.getByText('이 과제에 지원하기')).toBeInTheDocument()
    })
  })

  it('지원 버튼 클릭 시 지원서 폼이 표시된다', async () => {
    setupMocks()
    const user = userEvent.setup()
    renderWithProviders(<GrantDetailPage />)

    await waitFor(() => {
      expect(screen.getByText('이 과제에 지원하기')).toBeInTheDocument()
    })

    await user.click(screen.getByText('이 과제에 지원하기'))

    await waitFor(() => {
      expect(screen.getByText('지원서')).toBeInTheDocument()
      expect(screen.getByText('지원하기')).toBeInTheDocument()
    })
  })

  it('이미 지원한 학생에게 안내 메시지가 표시된다', async () => {
    const grantWithMyApp = {
      ...mockGrant,
      applications: [
        { id: 1, grant_id: 1, user_id: 2, proposal: '지원', status: 'pending', user: { id: 2, name: '김학생' }, created_at: '2026-01-02T00:00:00Z' },
      ],
    }
    setupMocks(grantWithMyApp)
    renderWithProviders(<GrantDetailPage />)

    await waitFor(() => {
      expect(screen.getByText('이미 지원한 과제입니다. 승인을 기다려주세요.')).toBeInTheDocument()
    })
  })

  it('관리자에게 과제 종료 버튼이 표시된다', async () => {
    setMockUser(mockAdmin)
    setupMocks()
    renderWithProviders(<GrantDetailPage />)

    await waitFor(() => {
      expect(screen.getByText('과제 종료')).toBeInTheDocument()
    })
  })

  it('관리자에게 지원 버튼이 표시되지 않는다', async () => {
    setMockUser(mockAdmin)
    setupMocks()
    renderWithProviders(<GrantDetailPage />)

    await waitFor(() => {
      expect(screen.getByText('리액트 스터디')).toBeInTheDocument()
    })
    expect(screen.queryByText('이 과제에 지원하기')).not.toBeInTheDocument()
  })

  it('지원자 목록이 렌더링된다', async () => {
    setupMocks(mockGrantWithApps)
    renderWithProviders(<GrantDetailPage />)

    await waitFor(() => {
      expect(screen.getByText('지원자 (2명)')).toBeInTheDocument()
      expect(screen.getByText('김학생')).toBeInTheDocument()
      expect(screen.getByText('이개발')).toBeInTheDocument()
    })
  })

  it('관리자에게 승인 버튼이 표시된다 (pending 상태)', async () => {
    setMockUser(mockAdmin)
    setupMocks(mockGrantWithApps)
    renderWithProviders(<GrantDetailPage />)

    await waitFor(() => {
      expect(screen.getByText('승인 (보상 지급)')).toBeInTheDocument()
    })
  })

  it('종료된 과제에 지원 버튼이 없다', async () => {
    const closedGrant = { ...mockGrant, status: 'closed' }
    setupMocks(closedGrant)
    renderWithProviders(<GrantDetailPage />)

    await waitFor(() => {
      expect(screen.getByText('종료')).toBeInTheDocument()
    })
    expect(screen.queryByText('이 과제에 지원하기')).not.toBeInTheDocument()
  })

  it('과제를 찾을 수 없을 때 에러 메시지가 표시된다', async () => {
    mockApiGet.mockRejectedValue(new Error('Not Found'))
    renderWithProviders(<GrantDetailPage />)

    await waitFor(() => {
      expect(screen.getByText('과제를 찾을 수 없습니다.')).toBeInTheDocument()
    })
  })
})
