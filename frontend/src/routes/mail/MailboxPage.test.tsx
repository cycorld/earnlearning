import { describe, it, expect, vi, beforeEach, beforeAll } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from '@/test/test-utils'
import MailboxPage from './MailboxPage'

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

// jsdom 폴리필: Radix Tabs 등이 쓰는 API
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

// 승인된 개인 메일함 (address_id 필수 계약)
const personalApproved = {
  address_id: 1,
  kind: 'user',
  company_id: null,
  name: '홍길동',
  local_part: 'me123',
  email: 'me123@earnlearning.com',
  status: 'approved',
}

const inboxEmail = {
  id: 1,
  direction: 'inbox',
  from_addr: 'alice@earnlearning.com',
  to_addr: 'me123@earnlearning.com',
  subject: '안녕하세요',
  snippet: '첫 메일이에요',
  read: false,
  has_attachments: true,
  created_at: now,
}

const inboxDetail = {
  ...inboxEmail,
  read: true,
  body_text: '메일 본문 내용입니다',
  body_html: '',
  in_reply_to: null,
  attachments: [{ id: 10, filename: 'report.pdf', mime: 'application/pdf', size: 2048 }],
}

// 승인 개인 메일함 + 기본 목록 응답
function setupClaimed(sentEmails: unknown[] = []) {
  mockApiGet.mockImplementation((path: string) => {
    if (path === '/mail/mailboxes')
      return Promise.resolve({ mailboxes: [personalApproved] })
    if (path.startsWith('/mail?box=inbox'))
      return Promise.resolve({ emails: [inboxEmail], total: 1 })
    if (path.startsWith('/mail?box=sent'))
      return Promise.resolve({ emails: sentEmails, total: sentEmails.length })
    if (/^\/mail\/\d+$/.test(path)) return Promise.resolve(inboxDetail)
    return Promise.resolve({ emails: [], total: 0 })
  })
  mockApiPost.mockResolvedValue({ id: 99 })
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('MailboxPage', () => {
  it('메일함이 없으면 주소 신청 화면을 보여준다', async () => {
    mockApiGet.mockResolvedValue({ mailboxes: [] })
    renderWithProviders(<MailboxPage />)

    await waitFor(() => {
      expect(screen.getByText('내 이메일 주소 신청')).toBeInTheDocument()
    })
    expect(
      screen.getByText(/승인 후에는 변경할 수 없습니다/),
    ).toBeInTheDocument()
  })

  it('주소를 신청하면 POST 후 승인 대기 화면으로 전환된다', async () => {
    // 최초엔 빈 메일함, 신청 후 재조회에서 pending 개인 메일함이 나오도록
    let mailboxesResp: { mailboxes: unknown[] } = { mailboxes: [] }
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/mail/mailboxes') return Promise.resolve(mailboxesResp)
      return Promise.resolve({ emails: [], total: 0 })
    })
    mockApiPost.mockImplementation(() => {
      mailboxesResp = {
        mailboxes: [
          {
            address_id: 5,
            kind: 'user',
            company_id: null,
            name: '홍길동',
            local_part: 'jane99',
            email: 'jane99@earnlearning.com',
            status: 'pending',
          },
        ],
      }
      return Promise.resolve({
        local_part: 'jane99',
        email: 'jane99@earnlearning.com',
        status: 'pending',
      })
    })

    const user = userEvent.setup()
    renderWithProviders(<MailboxPage />)

    const input = await screen.findByPlaceholderText('jane99')
    await user.type(input, 'jane99')
    await user.click(screen.getByRole('button', { name: /이 주소로 신청하기/ }))

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith('/mail/address', {
        local_part: 'jane99',
      })
    })
    // 메일함이 아니라 승인 대기 화면으로 전환
    await waitFor(() => {
      expect(screen.getByText(/관리자 승인 대기 중/)).toBeInTheDocument()
    })
    expect(screen.queryByText('받은편지함')).not.toBeInTheDocument()
  })

  it('pending 상태면 승인 대기 화면을 보여준다 (메일함 아님)', async () => {
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/mail/mailboxes')
        return Promise.resolve({
          mailboxes: [
            {
              address_id: 3,
              kind: 'user',
              company_id: null,
              name: '홍길동',
              local_part: 'me123',
              email: 'me123@earnlearning.com',
              status: 'pending',
            },
          ],
        })
      return Promise.resolve({ emails: [], total: 0 })
    })
    renderWithProviders(<MailboxPage />)

    await waitFor(() => {
      expect(screen.getByText(/관리자 승인 대기 중/)).toBeInTheDocument()
    })
    expect(screen.getByText('me123@earnlearning.com')).toBeInTheDocument()
    expect(screen.queryByText('받은편지함')).not.toBeInTheDocument()
  })

  it('rejected 상태면 재신청 폼을 보여주고 재신청하면 POST 한다', async () => {
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/mail/mailboxes')
        return Promise.resolve({
          mailboxes: [
            {
              address_id: 4,
              kind: 'user',
              company_id: null,
              name: '홍길동',
              local_part: 'me123',
              email: 'me123@earnlearning.com',
              status: 'rejected',
            },
          ],
        })
      return Promise.resolve({ emails: [], total: 0 })
    })
    mockApiPost.mockResolvedValue({
      local_part: 'me123',
      email: 'me123@earnlearning.com',
      status: 'pending',
    })

    const user = userEvent.setup()
    renderWithProviders(<MailboxPage />)

    await waitFor(() => {
      expect(screen.getByText(/반려되었습니다/)).toBeInTheDocument()
    })
    // 이전 local_part 로 프리필됨
    const input = (await screen.findByPlaceholderText('jane99')) as HTMLInputElement
    expect(input.value).toBe('me123')

    await user.click(screen.getByRole('button', { name: /이 주소로 신청하기/ }))

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith('/mail/address', {
        local_part: 'me123',
      })
    })
  })

  it('승인된 메일함이면 메일함을 렌더한다', async () => {
    setupClaimed()
    renderWithProviders(<MailboxPage />)

    await waitFor(() => {
      expect(screen.getByText('받은편지함')).toBeInTheDocument()
    })
    // 목록 조회 시 address_id 를 함께 보낸다
    expect(mockApiGet).toHaveBeenCalledWith(
      expect.stringContaining('address_id=1'),
    )
  })

  it('목록의 읽지 않은 메일은 굵게 + 점 표시된다', async () => {
    setupClaimed()
    renderWithProviders(<MailboxPage />)

    await waitFor(() => {
      expect(screen.getByText('alice@earnlearning.com')).toBeInTheDocument()
    })
    // 보낸사람이 굵게
    expect(screen.getByText('alice@earnlearning.com').className).toContain('font-bold')
    // 안읽음 점
    expect(screen.getByTestId('unread-dot')).toBeInTheDocument()
  })

  it('행을 클릭하면 상세를 조회한다', async () => {
    setupClaimed()
    const user = userEvent.setup()
    renderWithProviders(<MailboxPage />)

    await waitFor(() => {
      expect(screen.getByText('안녕하세요')).toBeInTheDocument()
    })
    await user.click(screen.getByText('안녕하세요'))

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/mail/1')
    })
    expect(await screen.findByText('메일 본문 내용입니다')).toBeInTheDocument()
    expect(screen.getByText('report.pdf')).toBeInTheDocument()
  })

  it('답장은 Re: 제목이 채워지고 in_reply_to_id 로 전송한다', async () => {
    setupClaimed([])
    const user = userEvent.setup()
    renderWithProviders(<MailboxPage />)

    await waitFor(() => {
      expect(screen.getByText('안녕하세요')).toBeInTheDocument()
    })
    await user.click(screen.getByText('안녕하세요'))

    const replyBtn = await screen.findByRole('button', { name: /답장/ })
    await user.click(replyBtn)

    // 제목 Re: 프리필, 받는 사람 = 보낸사람
    const subject = (await screen.findByLabelText('제목')) as HTMLInputElement
    expect(subject.value).toBe('Re: 안녕하세요')
    const to = screen.getByLabelText('받는 사람') as HTMLInputElement
    expect(to.value).toBe('alice@earnlearning.com')

    await user.click(screen.getByRole('button', { name: /보내기/ }))

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith('/mail/send', {
        address_id: 1,
        to: 'alice@earnlearning.com',
        subject: 'Re: 안녕하세요',
        body_text: '',
        in_reply_to_id: 1,
      })
    })
  })

  it('새 메일을 보내면 보낸편지함에 나타난다', async () => {
    const sentItem = {
      id: 2,
      direction: 'sent',
      from_addr: 'me123@earnlearning.com',
      to_addr: 'bob@earnlearning.com',
      subject: '보낸 제목',
      snippet: '보낸 내용',
      read: true,
      has_attachments: false,
      created_at: now,
    }
    setupClaimed([sentItem])
    const user = userEvent.setup()
    renderWithProviders(<MailboxPage />)

    await waitFor(() => {
      expect(screen.getByText('안녕하세요')).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: /새 메일/ }))

    await user.type(screen.getByLabelText('받는 사람'), 'bob@earnlearning.com')
    await user.type(screen.getByLabelText('제목'), '보낸 제목')
    await user.click(screen.getByRole('button', { name: /보내기/ }))

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith('/mail/send', {
        address_id: 1,
        to: 'bob@earnlearning.com',
        subject: '보낸 제목',
        body_text: '',
        in_reply_to_id: null,
      })
    })
    // 보낸편지함으로 전환되어 받는사람이 노출
    await waitFor(() => {
      expect(screen.getByText('bob@earnlearning.com')).toBeInTheDocument()
    })
  })
})

describe('MailboxPage 메일함 선택기', () => {
  const companyApproved = {
    address_id: 2,
    kind: 'company',
    company_id: 7,
    name: '에이컴퍼니',
    local_part: 'acompany',
    email: 'acompany@earnlearning.com',
    status: 'approved',
  }

  it('여러 메일함을 보여주고 전환 시 address_id 를 바꿔 조회한다', async () => {
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/mail/mailboxes')
        return Promise.resolve({ mailboxes: [personalApproved, companyApproved] })
      return Promise.resolve({ emails: [], total: 0 })
    })

    const user = userEvent.setup()
    renderWithProviders(<MailboxPage />)

    // 기본 선택 = 첫 승인 메일함(개인, address_id=1)
    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith(
        expect.stringContaining('address_id=1'),
      )
    })
    // 두 메일함 탭이 모두 노출
    expect(screen.getByRole('tab', { name: /홍길동/ })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /에이컴퍼니/ })).toBeInTheDocument()

    // 회사 메일함으로 전환
    await user.click(screen.getByRole('tab', { name: /에이컴퍼니/ }))

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith(
        expect.stringContaining('address_id=2'),
      )
    })
  })

  const sharedApproved = {
    address_id: 9,
    kind: 'shared',
    company_id: null,
    name: '고객지원',
    local_part: 'support',
    email: 'support@earnlearning.com',
    status: 'approved',
  }

  it('메일함이 하나면 이름과 구분 뱃지를 비대화형 헤더로 보여준다', async () => {
    const sharedOnly = {
      address_id: 12,
      kind: 'shared',
      company_id: null,
      name: '언러닝 안내',
      local_part: 'hello',
      email: 'hello@earnlearning.com',
      status: 'approved',
    }
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/mail/mailboxes')
        return Promise.resolve({ mailboxes: [sharedOnly] })
      return Promise.resolve({ emails: [], total: 0 })
    })

    renderWithProviders(<MailboxPage />)

    await waitFor(() => {
      expect(screen.getByText('언러닝 안내')).toBeInTheDocument()
    })
    // 공용 구분 뱃지 노출
    expect(screen.getByText('공용')).toBeInTheDocument()
    // 메일함이 하나뿐이면 선택 탭은 렌더하지 않는다
    expect(
      screen.queryByRole('tab', { name: /언러닝 안내/ }),
    ).not.toBeInTheDocument()
    // 메일함은 사용 가능
    expect(screen.getByText('받은편지함')).toBeInTheDocument()
  })

  it('공용 메일함을 공용 뱃지와 함께 보여주고 전환 시 해당 address_id 로 조회한다', async () => {
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/mail/mailboxes')
        return Promise.resolve({ mailboxes: [personalApproved, sharedApproved] })
      return Promise.resolve({ emails: [], total: 0 })
    })

    const user = userEvent.setup()
    renderWithProviders(<MailboxPage />)

    // 공용 메일함 탭 + 공용 뱃지 노출
    await waitFor(() => {
      expect(screen.getByRole('tab', { name: /고객지원/ })).toBeInTheDocument()
    })
    expect(screen.getByText('공용')).toBeInTheDocument()

    // 공용 메일함으로 전환 → address_id=9 로 조회
    await user.click(screen.getByRole('tab', { name: /고객지원/ }))

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith(
        expect.stringContaining('address_id=9'),
      )
    })
  })

  it('개인 pending + 회사 approved 이면 회사 메일함을 사용할 수 있다', async () => {
    const personalPending = {
      ...personalApproved,
      status: 'pending',
    }
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/mail/mailboxes')
        return Promise.resolve({ mailboxes: [personalPending, companyApproved] })
      return Promise.resolve({ emails: [], total: 0 })
    })

    renderWithProviders(<MailboxPage />)

    // 회사 메일함이 기본 선택되어 목록 조회
    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith(
        expect.stringContaining('address_id=2'),
      )
    })
    // 메일함 UI 노출
    expect(screen.getByText('받은편지함')).toBeInTheDocument()
    // 개인 탭은 대기중이라 비활성
    expect(screen.getByRole('tab', { name: /홍길동/ })).toBeDisabled()
  })
})

// ─── #173 알림 딥링크 (?open=<메일id>) ───────────────────────
import { MemoryRouter } from 'react-router-dom'
import { render } from '@testing-library/react'

const companyApproved2 = {
  address_id: 2,
  kind: 'company',
  company_id: 7,
  name: '길동컴퍼니',
  local_part: 'gil-co',
  email: 'gil-co@earnlearning.com',
  status: 'approved',
}

describe('MailboxPage 딥링크 (#173)', () => {
  beforeEach(() => {
    mockApiGet.mockReset()
    mockApiPost.mockReset()
  })

  it('?open=<id> 로 진입하면 소속 메일함을 선택하고 해당 메일 상세를 연다', async () => {
    const deepDetail = {
      ...inboxDetail,
      id: 9,
      address_id: 2,
      subject: '회사로 온 메일',
      body_text: '딥링크 본문',
      to_addr: 'gil-co@earnlearning.com',
    }
    mockApiGet.mockImplementation((url: string) => {
      if (url === '/mail/mailboxes')
        return Promise.resolve({ mailboxes: [personalApproved, companyApproved2] })
      if (url === '/mail/address')
        return Promise.resolve({ local_part: 'me123', email: 'me123@earnlearning.com', status: 'approved' })
      if (url === '/mail/9') return Promise.resolve(deepDetail)
      if (url.startsWith('/mail?')) return Promise.resolve({ emails: [], total: 0 })
      return Promise.resolve(null)
    })

    render(
      <MemoryRouter initialEntries={['/mail?open=9']}>
        <MailboxPage />
      </MemoryRouter>,
    )

    // 상세가 열리고
    await waitFor(() => {
      expect(screen.getByText('딥링크 본문')).toBeInTheDocument()
    })
    // 소속(회사) 메일함 스코프로 목록을 조회했는지
    await waitFor(() => {
      const listCalls = mockApiGet.mock.calls
        .map((c) => String(c[0]))
        .filter((u) => u.startsWith('/mail?'))
      expect(listCalls.some((u) => u.includes('address_id=2'))).toBe(true)
    })
  })
})
