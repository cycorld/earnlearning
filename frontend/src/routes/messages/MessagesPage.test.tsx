import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { mockStudent, renderWithProviders, setMockUser } from '@/test/test-utils'
import MessagesPage from './MessagesPage'

const mockApiGet = vi.fn()
const mockNavigate = vi.fn()

vi.mock('@/lib/api', () => ({ api: { get: (...args: unknown[]) => mockApiGet(...args) } }))
vi.mock('@/hooks/use-ws', () => ({ useWebSocket: vi.fn() }))

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return { ...actual, useNavigate: () => mockNavigate }
})

beforeEach(() => {
  vi.clearAllMocks()
  setMockUser({ ...mockStudent, active_classroom_id: 2 })
  mockApiGet.mockImplementation((path: string) => {
    if (path === '/dm/conversations') return Promise.resolve([])
    if (path === '/classrooms/2/students') return Promise.resolve([
      { user_id: 7, name: '김학생', department: '컴퓨터공학', student_id: '20260007', avatar_url: '' },
      { user_id: 8, name: '이창업', department: '경영학', student_id: '20260008', avatar_url: '' },
    ])
    return Promise.resolve([])
  })
})

describe('MessagesPage compose', () => {
  it('활성 강의실 학생을 이름·학과·학번으로 검색하고 대화로 이동한다', async () => {
    const user = userEvent.setup()
    renderWithProviders(<MessagesPage />)
    await user.click(await screen.findByRole('button', { name: '메시지 쓰기' }))
    await waitFor(() => expect(mockApiGet).toHaveBeenCalledWith('/classrooms/2/students'))

    const search = screen.getByLabelText('학생 검색')
    await user.type(search, '컴퓨터')
    expect(screen.getByText('김학생')).toBeInTheDocument()
    expect(screen.queryByText('이창업')).not.toBeInTheDocument()

    await user.clear(search)
    await user.type(search, '20260008')
    await user.click(screen.getByText('이창업'))
    expect(mockNavigate).toHaveBeenCalledWith('/messages/8')
  })
})
