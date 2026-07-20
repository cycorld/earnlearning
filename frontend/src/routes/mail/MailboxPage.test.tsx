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

const inboxEmail = {
  id: 1,
  direction: 'inbox',
  from_addr: 'alice@earnlearning.com',
  to_addr: 'me@earnlearning.com',
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

// 주소가 있는 상태 + 기본 목록 응답
function setupClaimed(sentEmails: unknown[] = []) {
  mockApiGet.mockImplementation((path: string) => {
    if (path === '/mail/address')
      return Promise.resolve({ local_part: 'me', email: 'me@earnlearning.com' })
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
  it('주소가 없으면 주소 만들기 화면을 보여준다', async () => {
    mockApiGet.mockResolvedValue({ local_part: null, email: null })
    renderWithProviders(<MailboxPage />)

    await waitFor(() => {
      expect(screen.getByText('내 이메일 주소 만들기')).toBeInTheDocument()
    })
    expect(screen.getByText(/한 번 정하면 바꿀 수 없습니다/)).toBeInTheDocument()
  })

  it('주소를 만들면 POST 후 메일함으로 전환된다', async () => {
    // 최초엔 null, 생성 후 메일함이 뜨도록 이후 목록 응답도 준비
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/mail/address')
        return Promise.resolve({ local_part: null, email: null })
      if (path.startsWith('/mail?box=inbox'))
        return Promise.resolve({ emails: [], total: 0 })
      return Promise.resolve({ emails: [], total: 0 })
    })
    mockApiPost.mockResolvedValue({
      local_part: 'jane99',
      email: 'jane99@earnlearning.com',
    })

    const user = userEvent.setup()
    renderWithProviders(<MailboxPage />)

    const input = await screen.findByPlaceholderText('jane99')
    await user.type(input, 'jane99')
    await user.click(screen.getByRole('button', { name: /이 주소로 만들기/ }))

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith('/mail/address', {
        local_part: 'jane99',
      })
    })
    // 메일함(받은편지함 탭) 으로 전환
    await waitFor(() => {
      expect(screen.getByText('받은편지함')).toBeInTheDocument()
    })
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
      from_addr: 'me@earnlearning.com',
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
