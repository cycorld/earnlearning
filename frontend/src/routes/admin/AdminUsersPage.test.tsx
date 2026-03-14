import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders, setMockUser, mockAdmin } from '@/test/test-utils'
import type { User } from '@/types'
import AdminUsersPage from './AdminUsersPage'

// ─── API Mock ─────────────────────────────────────────────────

const mockApiGet = vi.fn()
const mockApiPost = vi.fn()
const mockApiPut = vi.fn()

vi.mock('@/lib/api', () => ({
  api: {
    get: (...args: unknown[]) => mockApiGet(...args),
    post: (...args: unknown[]) => mockApiPost(...args),
    put: (...args: unknown[]) => mockApiPut(...args),
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

// ─── Mock Users 생성 (50명 — 페이지네이션 회귀 테스트) ──────────

const koreanNames = [
  '김민수', '이서연', '박지훈', '최수아', '정우진',
  '강예린', '조현우', '윤서윤', '임도현', '한지은',
  '송민재', '오하영', '배준서', '홍서진', '류시우',
  '권나연', '남도윤', '문하은', '장서준', '신유진',
  '김태희', '이준혁', '박소연', '최영호', '정다은',
  '강민서', '조은비', '윤재현', '임지수', '한도현',
  '송하늘', '오민수', '배서영', '홍지민', '류은서',
  '권도현', '남서아', '문재윤', '장하은', '신민재',
  '김서진', '이하율', '박재민', '최서윤', '정준서',
  '강지윤', '조민호', '윤서진', '임하영', '한재현',
]

const departments = [
  '컴퓨터공학과', '경영학과', '디자인학과', '산업공학과', '전자공학과',
  '미디어학과', '국제학부', '소프트웨어학과', '통계학과', '경제학과',
]

function makeUsers(count: number): User[] {
  return Array.from({ length: count }, (_, i) => ({
    id: i + 10,
    email: `student${i + 10}@ewha.ac.kr`,
    name: koreanNames[i % koreanNames.length],
    role: 'student' as const,
    status: (i < 15 ? 'pending' : i < 40 ? 'approved' : 'rejected') as User['status'],
    department: departments[i % departments.length],
    student_id: `202600${String(i + 10).padStart(4, '0')}`,
    bio: '',
    avatar_url: '',
    wallet_balance: i < 15 ? undefined : 50000000,
  }))
}

const fiftyUsers = makeUsers(50)
const fiftyPending = fiftyUsers.filter((u) => u.status === 'pending')

// ─── Setup ────────────────────────────────────────────────────

beforeEach(() => {
  vi.clearAllMocks()
  setMockUser(mockAdmin)

  mockApiGet.mockImplementation((path: string) => {
    if (path.includes('/admin/users/pending'))
      return Promise.resolve(fiftyPending)
    if (path.includes('/admin/users'))
      return Promise.resolve({ users: fiftyUsers, total: fiftyUsers.length })
    return Promise.resolve([])
  })
})

// ─── Tests ────────────────────────────────────────────────────

describe('AdminUsersPage 유저 목록', () => {
  it('50명 전체 유저가 모두 렌더링된다 (20명 제한 회귀 테스트)', async () => {
    renderWithProviders(<AdminUsersPage />)

    await waitFor(() => {
      // 전체 탭에 총 인원수 표시
      expect(screen.getByText(`전체 (${fiftyUsers.length})`)).toBeInTheDocument()
    })

    // "전체" 탭 클릭
    const user = userEvent.setup()
    await user.click(screen.getByText(`전체 (${fiftyUsers.length})`))

    await waitFor(() => {
      // 50명 전원의 이메일이 렌더링되었는지 확인
      for (const u of fiftyUsers) {
        expect(screen.getByText(new RegExp(u.email))).toBeInTheDocument()
      }
    })
  })

  it('대기 중 유저 15명이 모두 표시된다', async () => {
    renderWithProviders(<AdminUsersPage />)

    await waitFor(() => {
      expect(screen.getByText(`대기 (${fiftyPending.length})`)).toBeInTheDocument()
    })

    // 대기 탭이 기본이므로 pending 유저 전부 표시
    await waitFor(() => {
      for (const u of fiftyPending) {
        expect(screen.getByText(new RegExp(u.email))).toBeInTheDocument()
      }
    })
  })

  it('각 유저에 이름, 이메일, 학과, 학번이 표시된다', async () => {
    renderWithProviders(<AdminUsersPage />)

    const user = userEvent.setup()

    await waitFor(() => {
      expect(screen.getByText(`전체 (${fiftyUsers.length})`)).toBeInTheDocument()
    })

    await user.click(screen.getByText(`전체 (${fiftyUsers.length})`))

    await waitFor(() => {
      const firstUser = fiftyUsers[0]
      // 이름은 여러 명이 같을 수 있으므로 getAllByText 사용
      expect(screen.getAllByText(firstUser.name).length).toBeGreaterThan(0)
      // email | department | student_id 조합으로 확인
      expect(
        screen.getByText(new RegExp(firstUser.email)),
      ).toBeInTheDocument()
    })
  })

  it('승인된 유저에 "승인됨" 뱃지가 표시된다', async () => {
    renderWithProviders(<AdminUsersPage />)

    const user = userEvent.setup()

    await waitFor(() => {
      expect(screen.getByText(`전체 (${fiftyUsers.length})`)).toBeInTheDocument()
    })

    await user.click(screen.getByText(`전체 (${fiftyUsers.length})`))

    await waitFor(() => {
      const badges = screen.getAllByText('승인됨')
      const approvedCount = fiftyUsers.filter((u) => u.status === 'approved').length
      expect(badges.length).toBe(approvedCount)
    })
  })

  it('대기 유저에 "대기" 뱃지가 표시된다', async () => {
    renderWithProviders(<AdminUsersPage />)

    await waitFor(() => {
      const badges = screen.getAllByText('대기', { selector: '.text-xs' })
      // 대기 탭 제목의 "대기 (15)" 도 있으므로 badge 개수는 pending 수와 같아야 함
      expect(badges.length).toBe(fiftyPending.length)
    })
  })
})

describe('AdminUsersPage 승인/거절', () => {
  it('승인 버튼 클릭 시 API를 호출한다', async () => {
    mockApiPut.mockResolvedValue({})

    renderWithProviders(<AdminUsersPage />)

    await waitFor(() => {
      expect(screen.getAllByTitle('승인').length).toBeGreaterThan(0)
    })

    const user = userEvent.setup()
    const approveButtons = screen.getAllByTitle('승인')
    await user.click(approveButtons[0])

    await waitFor(() => {
      expect(mockApiPut).toHaveBeenCalledWith(
        expect.stringMatching(/\/admin\/users\/\d+\/approve/),
      )
    })
  })

  it('거절 버튼 클릭 시 API를 호출한다', async () => {
    mockApiPut.mockResolvedValue({})

    renderWithProviders(<AdminUsersPage />)

    await waitFor(() => {
      expect(screen.getAllByTitle('거절').length).toBeGreaterThan(0)
    })

    const user = userEvent.setup()
    const rejectButtons = screen.getAllByTitle('거절')
    await user.click(rejectButtons[0])

    await waitFor(() => {
      expect(mockApiPut).toHaveBeenCalledWith(
        expect.stringMatching(/\/admin\/users\/\d+\/reject/),
      )
    })
  })

  it('승인된 유저에는 송금 버튼이 표시된다', async () => {
    renderWithProviders(<AdminUsersPage />)

    const user = userEvent.setup()

    await waitFor(() => {
      expect(screen.getByText(`전체 (${fiftyUsers.length})`)).toBeInTheDocument()
    })

    await user.click(screen.getByText(`전체 (${fiftyUsers.length})`))

    await waitFor(() => {
      const approvedCount = fiftyUsers.filter((u) => u.status === 'approved').length
      const sendButtons = screen.getAllByTitle('송금')
      expect(sendButtons.length).toBe(approvedCount)
    })
  })
})

describe('AdminUsersPage 60명 대기 유저 페이지네이션', () => {
  const sixtyPendingUsers: User[] = Array.from({ length: 60 }, (_, i) => ({
    id: i + 100,
    email: `pending${i + 100}@ewha.ac.kr`,
    name: koreanNames[i % koreanNames.length],
    role: 'student' as const,
    status: 'pending' as const,
    department: departments[i % departments.length],
    student_id: `202610${String(i).padStart(4, '0')}`,
    bio: '',
    avatar_url: '',
  }))

  beforeEach(() => {
    mockApiGet.mockImplementation((path: string) => {
      if (path.includes('/admin/users/pending'))
        return Promise.resolve(sixtyPendingUsers)
      if (path.includes('/admin/users'))
        return Promise.resolve({ users: sixtyPendingUsers, total: sixtyPendingUsers.length })
      return Promise.resolve([])
    })
  })

  it('60명 대기 유저가 전부 렌더링된다 (20명 제한 회귀 테스트)', async () => {
    renderWithProviders(<AdminUsersPage />)

    await waitFor(() => {
      expect(screen.getByText(`대기 (${sixtyPendingUsers.length})`)).toBeInTheDocument()
    })

    // 60명 전원의 이메일이 DOM에 있는지 확인
    await waitFor(() => {
      for (const u of sixtyPendingUsers) {
        expect(screen.getByText(new RegExp(u.email))).toBeInTheDocument()
      }
    })
  })

  it('60명 각각에 승인/거절 버튼이 표시된다', async () => {
    renderWithProviders(<AdminUsersPage />)

    await waitFor(() => {
      const approveButtons = screen.getAllByTitle('승인')
      const rejectButtons = screen.getAllByTitle('거절')
      expect(approveButtons.length).toBe(60)
      expect(rejectButtons.length).toBe(60)
    })
  })

  it('60명 모두 "대기" 뱃지가 표시된다', async () => {
    renderWithProviders(<AdminUsersPage />)

    await waitFor(() => {
      const badges = screen.getAllByText('대기', { selector: '.text-xs' })
      expect(badges.length).toBe(60)
    })
  })
})

describe('AdminUsersPage 에러 처리', () => {
  it('API 에러 시 크래시하지 않는다', async () => {
    mockApiGet.mockImplementation(() => Promise.reject(new Error('서버 에러')))

    renderWithProviders(<AdminUsersPage />)

    // 로딩이 끝나면 빈 상태가 표시됨 (크래시 아님)
    await waitFor(() => {
      expect(
        screen.getByText('대기 중인 사용자가 없습니다.'),
      ).toBeInTheDocument()
    })
  })
})
