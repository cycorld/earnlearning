import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders, setMockUser, mockAdmin } from '@/test/test-utils'
import GrantNewPage from './GrantNewPage'

const mockApiPost = vi.fn()
const mockNavigate = vi.fn()

vi.mock('@/lib/api', () => ({
  api: {
    get: vi.fn(),
    post: (...args: unknown[]) => mockApiPost(...args),
    put: vi.fn(),
    del: vi.fn(),
  },
}))

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

beforeEach(() => {
  vi.clearAllMocks()
  setMockUser(mockAdmin)
})

describe('GrantNewPage', () => {
  it('과제 등록 폼이 렌더링된다', () => {
    renderWithProviders(<GrantNewPage />)

    expect(screen.getByText('정부과제 등록')).toBeInTheDocument()
    expect(screen.getByLabelText('제목')).toBeInTheDocument()
    expect(screen.getByLabelText(/보상 금액/)).toBeInTheDocument()
    expect(screen.getByLabelText(/최대 지원자/)).toBeInTheDocument()
    expect(screen.getByText('등록하기')).toBeInTheDocument()
  })

  it('필수 필드가 비어있으면 제출되지 않는다', async () => {
    const user = userEvent.setup()
    renderWithProviders(<GrantNewPage />)

    await user.click(screen.getByText('등록하기'))

    expect(mockApiPost).not.toHaveBeenCalled()
  })

  it('폼 제출 시 API를 호출하고 상세 페이지로 이동한다', async () => {
    mockApiPost.mockResolvedValue({ id: 42 })
    const user = userEvent.setup()
    renderWithProviders(<GrantNewPage />)

    await user.type(screen.getByLabelText('제목'), '새 과제')
    await user.type(screen.getByLabelText(/보상 금액/), '5000')

    await user.click(screen.getByText('등록하기'))

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith('/admin/grants', expect.objectContaining({
        title: '새 과제',
        reward: 5000,
      }))
      expect(mockNavigate).toHaveBeenCalledWith('/grant/42')
    })
  })

  it('API 에러 시 에러 메시지가 표시된다', async () => {
    mockApiPost.mockRejectedValue(new Error('등록에 실패했습니다.'))
    const user = userEvent.setup()
    renderWithProviders(<GrantNewPage />)

    await user.type(screen.getByLabelText('제목'), '에러 테스트')
    await user.type(screen.getByLabelText(/보상 금액/), '1000')

    await user.click(screen.getByText('등록하기'))

    await waitFor(() => {
      expect(screen.getByText('등록에 실패했습니다.')).toBeInTheDocument()
    })
  })

  it('뒤로가기 링크가 과제 목록으로 연결된다', () => {
    renderWithProviders(<GrantNewPage />)

    const backLink = screen.getByText('과제 목록으로')
    expect(backLink.closest('a')).toHaveAttribute('href', '/grant')
  })
})
