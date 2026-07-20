import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from '@/test/test-utils'
import { CompanyMailSection } from './CompanyMailSection'

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

beforeEach(() => {
  vi.clearAllMocks()
})

describe('CompanyMailSection', () => {
  it('회사 메일이 없으면 등록 폼을 보여주고 제출 시 POST 한다', async () => {
    mockApiGet.mockResolvedValue({ mailboxes: [] })
    mockApiPost.mockResolvedValue({
      local_part: 'acompany',
      email: 'acompany@earnlearning.com',
      status: 'pending',
    })

    const user = userEvent.setup()
    renderWithProviders(<CompanyMailSection companyId={7} />)

    const input = await screen.findByPlaceholderText('acompany')
    await user.type(input, 'acompany')
    await user.click(screen.getByRole('button', { name: /이 주소로 신청하기/ }))

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith('/companies/7/mail-address', {
        local_part: 'acompany',
      })
    })
  })

  it('승인된 회사 메일은 읽기전용 주소로 보여준다', async () => {
    mockApiGet.mockResolvedValue({
      mailboxes: [
        {
          address_id: 2,
          kind: 'company',
          company_id: 7,
          name: '에이컴퍼니',
          local_part: 'acompany',
          email: 'acompany@earnlearning.com',
          status: 'approved',
        },
      ],
    })

    renderWithProviders(<CompanyMailSection companyId={7} />)

    await waitFor(() => {
      expect(screen.getByText('acompany@earnlearning.com')).toBeInTheDocument()
    })
    // 등록 폼(신청 버튼)은 없어야 한다
    expect(
      screen.queryByRole('button', { name: /이 주소로 신청하기/ }),
    ).not.toBeInTheDocument()
  })

  it('반려 상태면 반려 안내와 재신청 폼을 보여준다', async () => {
    mockApiGet.mockResolvedValue({
      mailboxes: [
        {
          address_id: 2,
          kind: 'company',
          company_id: 7,
          name: '에이컴퍼니',
          local_part: 'acompany',
          email: 'acompany@earnlearning.com',
          status: 'rejected',
        },
      ],
    })

    renderWithProviders(<CompanyMailSection companyId={7} />)

    await waitFor(() => {
      expect(screen.getByText(/반려되었습니다/)).toBeInTheDocument()
    })
    expect(
      screen.getByRole('button', { name: /이 주소로 신청하기/ }),
    ).toBeInTheDocument()
  })
})
