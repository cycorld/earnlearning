import { describe, it, expect, vi, beforeEach, beforeAll } from 'vitest'
import { screen, waitFor, fireEvent, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders, setMockUser, mockStudent } from '@/test/test-utils'
// sonner 모킹은 test-utils 가 등록한다 → 반드시 그 뒤에 import 해야 목이 잡힌다.
import { toast } from 'sonner'
import ConversationPage from './ConversationPage'
// 목 팩토리에서 만든 ApiError 를 그대로 써야 instanceof 가 맞는다.
import { ApiError } from '@/lib/api'
const MockApiError = ApiError as unknown as new (code: string, message: string, status: number) => Error

// #184 DM 첨부 — 전송(FormData)·검증(개수/크기/확장자)·수신 렌더(이미지 inline / 그 외 다운로드)

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
  // apiFetch 계약 = fetch + 인증 헤더 → 테스트에선 전역 fetch 로 그대로 흘린다.
  apiFetch: (...args: unknown[]) => (fetch as never as (...a: unknown[]) => unknown)(...args),
  ApiError: class ApiError extends Error {
    code: string
    status: number
    constructor(code: string, message: string, status: number) {
      super(message)
      this.code = code
      this.status = status
      this.name = 'ApiError'
    }
  },
}))

// 실시간 소켓은 테스트에서 무력화
vi.mock('@/hooks/use-ws', () => ({ useWebSocket: vi.fn() }))

// 라우트 파라미터 (/messages/:userId)
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return { ...actual, useParams: () => ({ userId: '2' }) }
})

const now = new Date().toISOString()

function msg(over: Record<string, unknown> = {}) {
  return {
    id: 1,
    sender_id: 2,
    receiver_id: 3,
    content: '안녕',
    is_read: true,
    created_at: now,
    attachments: [],
    ...over,
  }
}

let messages: unknown[] = []
let openSpy: ReturnType<typeof vi.fn>
let clickSpy: ReturnType<typeof vi.fn>

beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn()
})

beforeEach(() => {
  vi.clearAllMocks()
  setMockUser({ ...mockStudent, id: 3 })
  messages = []
  mockApiGet.mockImplementation((path: string) => {
    if (path.startsWith('/users/')) return Promise.resolve({ name: '상대', avatar_url: '' })
    if (path.startsWith('/dm/messages/')) return Promise.resolve(messages)
    return Promise.resolve(null)
  })
  mockApiPost.mockResolvedValue(msg())
  mockApiPut.mockResolvedValue({})

  openSpy = vi.fn()
  vi.stubGlobal('open', openSpy)
  URL.createObjectURL = vi.fn(() => 'blob:mock')
  URL.revokeObjectURL = vi.fn()
  clickSpy = vi.fn()
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(clickSpy)
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue({
      ok: true,
      blob: async () => new Blob(['bytes'], { type: 'application/octet-stream' }),
      headers: { get: () => 'application/octet-stream' },
    }),
  )
})

function fileInput() {
  return screen.getByTestId('dm-file-input') as HTMLInputElement
}

function select(files: File[]) {
  fireEvent.change(fileInput(), { target: { files } })
}

function makeFile(name: string, type: string, size = 10) {
  const f = new File(['x'], name, { type })
  Object.defineProperty(f, 'size', { value: size })
  return f
}

async function renderPage() {
  renderWithProviders(<ConversationPage />)
  await screen.findByPlaceholderText('메시지를 입력하세요')
}

describe('#184 DM 첨부 렌더', () => {
  it('이미지 첨부는 인라인 img 로 렌더한다', async () => {
    messages = [
      msg({
        attachments: [
          {
            id: 10,
            message_id: 1,
            filename: 'photo.png',
            stored_name: 'a.png',
            mime: 'image/png',
            size: 1234,
            created_at: now,
          },
        ],
      }),
    ]
    await renderPage()

    const img = await screen.findByAltText('photo.png')
    expect(img.tagName).toBe('IMG')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith('/api/dm/attachments/10')
  })

  it('PDF 첨부는 img 가 아니라 다운로드 컨트롤로 렌더되고 클릭 시 파일을 받아온다', async () => {
    messages = [
      msg({
        attachments: [
          {
            id: 11,
            message_id: 1,
            filename: 'report.pdf',
            stored_name: 'b.pdf',
            mime: 'application/pdf',
            size: 2048,
            created_at: now,
          },
        ],
      }),
    ]
    const user = userEvent.setup()
    await renderPage()

    expect(screen.queryByAltText('report.pdf')).not.toBeInTheDocument()
    const btn = await screen.findByRole('button', { name: /report\.pdf/ })
    await user.click(btn)

    await waitFor(() => {
      expect(vi.mocked(fetch)).toHaveBeenCalledWith('/api/dm/attachments/11')
    })
  })
})

describe('#184 DM 첨부 선택/전송', () => {
  it('파일을 고르면 미리보기 칩이 뜨고 X 로 제거된다', async () => {
    const user = userEvent.setup()
    await renderPage()

    select([makeFile('doc.pdf', 'application/pdf')])
    expect(await screen.findByText('doc.pdf')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /doc\.pdf 제거/ }))
    await waitFor(() => {
      expect(screen.queryByText('doc.pdf')).not.toBeInTheDocument()
    })
  })

  it('파일이 있으면 FormData 로 전송한다', async () => {
    const user = userEvent.setup()
    await renderPage()

    await user.type(screen.getByPlaceholderText('메시지를 입력하세요'), '자료요')
    select([makeFile('doc.pdf', 'application/pdf')])
    await screen.findByText('doc.pdf')

    await user.click(screen.getByRole('button', { name: '보내기' }))

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith('/dm/messages', expect.any(FormData))
    })
    const fd = mockApiPost.mock.calls[0][1] as FormData
    expect(fd.get('receiver_id')).toBe('2')
    expect(fd.get('content')).toBe('자료요')
    expect(fd.getAll('files')).toHaveLength(1)
    expect((fd.getAll('files')[0] as File).name).toBe('doc.pdf')
  })

  it('파일이 없으면 기존 JSON 본문으로 전송한다', async () => {
    const user = userEvent.setup()
    await renderPage()

    await user.type(screen.getByPlaceholderText('메시지를 입력하세요'), '안녕')
    await user.click(screen.getByRole('button', { name: '보내기' }))

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith('/dm/messages', {
        receiver_id: 2,
        content: '안녕',
      })
    })
  })

  it('첨부만 있어도(본문 비어있어도) 전송할 수 있다', async () => {
    const user = userEvent.setup()
    await renderPage()

    const sendBtn = screen.getByRole('button', { name: '보내기' })
    expect(sendBtn).toBeDisabled()

    select([makeFile('doc.pdf', 'application/pdf')])
    await screen.findByText('doc.pdf')
    expect(sendBtn).not.toBeDisabled()

    await user.click(sendBtn)
    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith('/dm/messages', expect.any(FormData))
    })
    const fd = mockApiPost.mock.calls[0][1] as FormData
    expect(fd.get('content')).toBe('')
  })
})

describe('#184 DM 전송 실패 처리', () => {
  it('413 이면 raw statusText 대신 한글 용량 안내를 띄우고 입력을 유지한다', async () => {
    const user = userEvent.setup()
    mockApiPost.mockRejectedValue(
      new MockApiError('UNKNOWN', 'Request Entity Too Large', 413),
    )
    await renderPage()

    await user.type(screen.getByPlaceholderText('메시지를 입력하세요'), '자료요')
    select([makeFile('doc.pdf', 'application/pdf')])
    await screen.findByText('doc.pdf')
    await user.click(screen.getByRole('button', { name: '보내기' }))

    await waitFor(() => expect(vi.mocked(toast.error)).toHaveBeenCalled())
    const shown = vi.mocked(toast.error).mock.calls.at(-1)![0] as string
    expect(shown).not.toContain('Request Entity Too Large')
    expect(shown).toMatch(/용량|크기/)
    // 재시도할 수 있도록 본문과 첨부는 그대로 남는다.
    expect(screen.getByPlaceholderText('메시지를 입력하세요')).toHaveValue('자료요')
    expect(screen.getByText('doc.pdf')).toBeInTheDocument()
  })

  it('서버 메시지가 있으면 그대로 보여준다', async () => {
    const user = userEvent.setup()
    mockApiPost.mockRejectedValue(
      new MockApiError('DM_BLOCKED', '차단된 사용자입니다.', 403),
    )
    await renderPage()

    await user.type(screen.getByPlaceholderText('메시지를 입력하세요'), '안녕')
    await user.click(screen.getByRole('button', { name: '보내기' }))

    await waitFor(() =>
      expect(vi.mocked(toast.error)).toHaveBeenCalledWith('차단된 사용자입니다.'),
    )
  })

  it('envelope 가 아닌 에러(code=UNKNOWN)는 statusText 대신 일반 안내를 띄운다', async () => {
    const user = userEvent.setup()
    mockApiPost.mockRejectedValue(new MockApiError('UNKNOWN', 'Bad Gateway', 502))
    await renderPage()

    await user.type(screen.getByPlaceholderText('메시지를 입력하세요'), '안녕')
    await user.click(screen.getByRole('button', { name: '보내기' }))

    await waitFor(() => expect(vi.mocked(toast.error)).toHaveBeenCalled())
    expect(vi.mocked(toast.error).mock.calls.at(-1)![0]).not.toContain('Bad Gateway')
  })
})

describe('#184 DM 이미지 첨부 접근성/실패 처리', () => {
  const imgMsg = () =>
    msg({
      attachments: [
        {
          id: 10,
          message_id: 1,
          filename: 'photo.png',
          stored_name: 'a.png',
          mime: 'image/png',
          size: 1234,
          created_at: now,
        },
      ],
    })

  it('이미지는 버튼으로 감싸 키보드로 열 수 있다', async () => {
    messages = [imgMsg()]
    await renderPage()

    const btn = await screen.findByRole('button', { name: /photo\.png/ })
    expect(btn).toHaveAttribute('type', 'button')
    expect(btn.querySelector('img')).not.toBeNull()
  })

  it('이미지 로드 실패 시 파일명·재시도·다운로드를 보여주고 재시도하면 다시 요청한다', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500 }))
    messages = [imgMsg()]
    const user = userEvent.setup()
    await renderPage()

    expect(await screen.findByText('photo.png')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '다운로드' })).toBeInTheDocument()

    const calls = vi.mocked(fetch).mock.calls.length
    await user.click(screen.getByRole('button', { name: '재시도' }))
    await waitFor(() => {
      expect(vi.mocked(fetch).mock.calls.length).toBeGreaterThan(calls)
    })
  })
})

describe('#184 중복 제출 방어', () => {
  it('같은 렌더에서 제출이 두 번 발생해도 한 번만 전송한다', async () => {
    mockApiPost.mockImplementation(() => new Promise(() => {})) // 계속 진행 중
    await renderPage()

    await screen.findByPlaceholderText('메시지를 입력하세요')
    fireEvent.change(screen.getByPlaceholderText('메시지를 입력하세요'), {
      target: { value: '안녕' },
    })

    // sending state 가 리렌더로 반영되기 전에 두 번 제출 (Enter 연타 + 클릭).
    // 하나의 act 안에서 연달아 dispatch 해야 state 갱신이 사이에 끼어들지 않는다.
    const form = document.querySelector('form')!
    await act(async () => {
      form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
      form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    })

    await waitFor(() => expect(mockApiPost).toHaveBeenCalledTimes(1))
  })
})

describe('#184 전송 중 파일 컨트롤 잠금', () => {
  it('전송 중에는 첨부 버튼·파일 입력·칩 제거가 비활성화된다', async () => {
    const user = userEvent.setup()
    let resolvePost: (v: unknown) => void = () => {}
    mockApiPost.mockImplementation(() => new Promise((r) => { resolvePost = r }))
    await renderPage()

    select([makeFile('doc.pdf', 'application/pdf')])
    await screen.findByText('doc.pdf')
    await user.click(screen.getByRole('button', { name: '보내기' }))

    await waitFor(() => {
      expect(screen.getByRole('button', { name: '파일 첨부' })).toBeDisabled()
    })
    expect(fileInput()).toBeDisabled()
    expect(screen.getByRole('button', { name: /doc\.pdf 제거/ })).toBeDisabled()

    resolvePost(msg())
    await waitFor(() => {
      expect(screen.getByRole('button', { name: '파일 첨부' })).not.toBeDisabled()
    })
  })
})

describe('#184 DM 첨부 클라이언트 검증', () => {
  it('10MB 초과 파일은 거부한다', async () => {
    await renderPage()
    select([makeFile('big.pdf', 'application/pdf', 11 * 1024 * 1024)])

    await waitFor(() => expect(vi.mocked(toast.error)).toHaveBeenCalled())
    expect(screen.queryByText('big.pdf')).not.toBeInTheDocument()
  })

  it('허용되지 않은 확장자는 거부한다', async () => {
    await renderPage()
    select([makeFile('evil.html', 'text/html')])

    await waitFor(() => expect(vi.mocked(toast.error)).toHaveBeenCalled())
    expect(screen.queryByText('evil.html')).not.toBeInTheDocument()
  })

  it('4개를 넘으면 거부한다', async () => {
    await renderPage()
    select([
      makeFile('a.pdf', 'application/pdf'),
      makeFile('b.pdf', 'application/pdf'),
      makeFile('c.pdf', 'application/pdf'),
      makeFile('d.pdf', 'application/pdf'),
      makeFile('e.pdf', 'application/pdf'),
    ])

    await waitFor(() => expect(vi.mocked(toast.error)).toHaveBeenCalled())
    expect(screen.queryByText('e.pdf')).not.toBeInTheDocument()
  })

  it('합계 20MB 를 넘으면 거부한다', async () => {
    await renderPage()
    select([
      makeFile('a.pdf', 'application/pdf', 9 * 1024 * 1024),
      makeFile('b.pdf', 'application/pdf', 9 * 1024 * 1024),
      makeFile('c.pdf', 'application/pdf', 9 * 1024 * 1024),
    ])

    await waitFor(() => expect(vi.mocked(toast.error)).toHaveBeenCalled())
    expect(screen.queryByText('c.pdf')).not.toBeInTheDocument()
  })
})
