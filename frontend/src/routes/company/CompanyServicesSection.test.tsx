import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from '@/test/test-utils'
// test-utils 가 sonner 를 전역 mock 하므로 그 뒤에 import 해야 mock 인스턴스를 받는다.
import { toast } from 'sonner'
import { ApiError } from '@/lib/api'
import type { CompanyService } from '@/types'
import { CompanyServicesSection } from './CompanyServicesSection'

// ─── API Mock ─────────────────────────────────────────────────
const mockApiGet = vi.fn()
const mockApiPost = vi.fn()
const mockApiPut = vi.fn()
const mockApiDel = vi.fn()

vi.mock('@/lib/api', () => ({
  api: {
    get: (...args: unknown[]) => mockApiGet(...args),
    post: (...args: unknown[]) => mockApiPost(...args),
    put: (...args: unknown[]) => mockApiPut(...args),
    patch: vi.fn(),
    del: (...args: unknown[]) => mockApiDel(...args),
  },
  ApiError: class extends Error {
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

const toastError = vi.mocked(toast.error)
const toastSuccess = vi.mocked(toast.success)

function makeService(overrides: Partial<CompanyService> = {}): CompanyService {
  return {
    id: 10,
    company_id: 1,
    name: '에이앱',
    url: 'https://a-app.example.com',
    validation_status: 'unvalidated',
    validation_checked_at: null,
    validation_detail: '',
    rybbit_status: 'not_connected',
    rybbit_site_id: '',
    rybbit_connected_at: null,
    connect_ready: false,
    ...overrides,
  }
}

/**
 * GET 응답 시퀀스를 세팅한다.
 * 첫 번째 = 초기 목록, 이후 = 각 refetch 응답, 마지막 응답은 계속 반복(terminal fallback)되어
 * 추가 refetch 가 일어나도 테스트가 깨지지 않는다.
 */
function setupServices(initial: CompanyService[], ...updates: CompanyService[][]) {
  mockApiGet.mockReset()
  mockApiGet.mockResolvedValueOnce(initial)
  for (const u of updates) mockApiGet.mockResolvedValueOnce(u)
  mockApiGet.mockResolvedValue(updates.length ? updates[updates.length - 1] : initial)
}

beforeEach(() => {
  vi.clearAllMocks()
  mockApiGet.mockReset()
  mockApiPost.mockReset()
  mockApiPut.mockReset()
  mockApiDel.mockReset()
})

describe('CompanyServicesSection 목록 렌더', () => {
  it('여러 서비스를 이름·URL·상태 뱃지와 함께 보여준다', async () => {
    setupServices([
      makeService({
        id: 10,
        name: '에이앱',
        url: 'https://a-app.example.com',
        validation_status: 'valid',
        rybbit_status: 'connected',
        rybbit_site_id: '42',
      }),
      makeService({
        id: 11,
        name: '비앱',
        url: 'https://b-app.example.com',
      }),
    ])

    renderWithProviders(<CompanyServicesSection companyId={1} isOwner={false} />)

    expect(await screen.findByText('에이앱')).toBeInTheDocument()
    expect(screen.getByText('비앱')).toBeInTheDocument()
    expect(mockApiGet).toHaveBeenCalledWith('/companies/1/services')

    const linkA = screen.getByRole('link', { name: /a-app\.example\.com/ })
    expect(linkA).toHaveAttribute('href', 'https://a-app.example.com')
    expect(linkA).toHaveAttribute('target', '_blank')
    expect(screen.getByRole('link', { name: /b-app\.example\.com/ })).toHaveAttribute(
      'href',
      'https://b-app.example.com',
    )

    // 서비스별 뱃지 (A: 검증됨/연동됨, B: 미검증/미연동)
    expect(screen.getByText('검증됨')).toBeInTheDocument()
    expect(screen.getByText('연동됨')).toBeInTheDocument()
    expect(screen.getByText('미검증')).toBeInTheDocument()
    expect(screen.getByText('미연동')).toBeInTheDocument()
  })

  it('소유자가 아니면 목록만 보이고 추가 폼·검증/연동 버튼은 없다', async () => {
    setupServices([
      makeService({ id: 10, name: '에이앱', validation_status: 'valid', connect_ready: true }),
    ])

    renderWithProviders(<CompanyServicesSection companyId={1} isOwner={false} />)

    expect(await screen.findByText('에이앱')).toBeInTheDocument()
    expect(screen.getByText('검증됨')).toBeInTheDocument()

    expect(screen.queryByRole('button', { name: /추가/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /검증하기/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /연동하기/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /수정/ })).not.toBeInTheDocument()
  })
})

describe('CompanyServicesSection 연동 버튼 활성화', () => {
  it('connect_ready 가 서비스별로 버튼 활성화를 결정한다', async () => {
    setupServices([
      makeService({
        id: 10,
        name: '에이앱',
        validation_status: 'valid',
        connect_ready: true,
      }),
      makeService({
        id: 11,
        name: '비앱',
        validation_status: 'unvalidated',
        connect_ready: false,
      }),
    ])

    renderWithProviders(<CompanyServicesSection companyId={1} isOwner />)

    expect(await screen.findByText('에이앱')).toBeInTheDocument()

    expect(screen.getByRole('button', { name: '에이앱 연동하기' })).toBeEnabled()
    expect(screen.getByRole('button', { name: '비앱 연동하기' })).toBeDisabled()
  })

  it('재연동 필요 상태는 뱃지를 보여주고 connect_ready 로만 버튼이 열린다', async () => {
    setupServices([
      makeService({
        id: 10,
        name: '에이앱',
        validation_status: 'unvalidated',
        rybbit_status: 'needs_reconnect',
        rybbit_site_id: '42',
        connect_ready: false,
      }),
      makeService({
        id: 11,
        name: '비앱',
        validation_status: 'valid',
        rybbit_status: 'needs_reconnect',
        rybbit_site_id: '43',
        connect_ready: true,
      }),
    ])

    renderWithProviders(<CompanyServicesSection companyId={1} isOwner />)

    expect(await screen.findByText('에이앱')).toBeInTheDocument()
    expect(screen.getAllByText('재연동 필요')).toHaveLength(2)

    expect(screen.getByRole('button', { name: '에이앱 재연동' })).toBeDisabled()
    expect(screen.getByRole('button', { name: '비앱 재연동' })).toBeEnabled()
  })
})

describe('CompanyServicesSection 검증', () => {
  it('검증하기를 누르면 validate 를 POST 하고 목록을 다시 불러온다', async () => {
    const before = makeService({ id: 11, name: '비앱', validation_status: 'unvalidated' })
    const after = makeService({
      id: 11,
      name: '비앱',
      validation_status: 'valid',
      validation_checked_at: '2026-07-27T00:00:00Z',
      connect_ready: true,
    })
    setupServices([before], [after])
    mockApiPost.mockResolvedValue(after)

    const user = userEvent.setup()
    renderWithProviders(<CompanyServicesSection companyId={1} isOwner />)

    await screen.findByText('비앱')
    expect(screen.getByText('미검증')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '비앱 검증하기' }))

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith('/companies/1/services/11/validate')
    })
    expect(await screen.findByText('검증됨')).toBeInTheDocument()
    expect(toastSuccess).toHaveBeenCalled()
    expect(mockApiGet).toHaveBeenCalledTimes(2)
  })

  it('검증 실패(200 응답)면 한국어 사유로 toast.error 한다', async () => {
    const before = makeService({ id: 11, name: '비앱' })
    const after = makeService({
      id: 11,
      name: '비앱',
      validation_status: 'invalid',
      validation_detail: 'private_ip',
      validation_checked_at: '2026-07-27T00:00:00Z',
      connect_ready: false,
    })
    setupServices([before], [after])
    mockApiPost.mockResolvedValue(after)

    const user = userEvent.setup()
    renderWithProviders(<CompanyServicesSection companyId={1} isOwner />)

    await screen.findByText('비앱')
    await user.click(screen.getByRole('button', { name: '비앱 검증하기' }))

    await waitFor(() => {
      expect(toastError).toHaveBeenCalledWith('내부/사설 주소는 사용할 수 없어요')
    })
    expect(toastSuccess).not.toHaveBeenCalled()
    expect(await screen.findByText('검증 실패')).toBeInTheDocument()
    expect(screen.getByText('내부/사설 주소는 사용할 수 없어요')).toBeInTheDocument()
  })
})

describe('CompanyServicesSection Rybbit 연동', () => {
  it('연동하기 성공 시 connect 를 POST 하고 연동됨으로 바뀐다', async () => {
    const before = makeService({
      id: 10,
      name: '에이앱',
      validation_status: 'valid',
      connect_ready: true,
    })
    const after = makeService({
      id: 10,
      name: '에이앱',
      validation_status: 'valid',
      rybbit_status: 'connected',
      rybbit_site_id: '42',
      rybbit_connected_at: '2026-07-27T00:00:00Z',
      connect_ready: false,
    })
    setupServices([before], [after])
    mockApiPost.mockResolvedValue(after)

    const user = userEvent.setup()
    renderWithProviders(<CompanyServicesSection companyId={1} isOwner />)

    await screen.findByText('에이앱')
    expect(screen.getByText('미연동')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '에이앱 연동하기' }))

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith('/companies/1/services/10/rybbit/connect')
    })
    expect(toastSuccess).toHaveBeenCalledWith('연동 완료')
    expect(await screen.findByText('연동됨')).toBeInTheDocument()
  })

  it('연동 실패 시 서버 메시지로 toast.error 하고 상태는 그대로다', async () => {
    const before = makeService({
      id: 10,
      name: '에이앱',
      validation_status: 'valid',
      connect_ready: true,
    })
    setupServices([before])
    mockApiPost.mockRejectedValue(
      new ApiError('RYBBIT_CONNECT_FAILED', 'Rybbit 연동에 실패했어요.', 502),
    )

    const user = userEvent.setup()
    renderWithProviders(<CompanyServicesSection companyId={1} isOwner />)

    await screen.findByText('에이앱')
    await user.click(screen.getByRole('button', { name: '에이앱 연동하기' }))

    await waitFor(() => {
      expect(toastError).toHaveBeenCalledWith('Rybbit 연동에 실패했어요.')
    })
    expect(screen.getByText('미연동')).toBeInTheDocument()
    expect(toastSuccess).not.toHaveBeenCalled()
  })
})

describe('CompanyServicesSection 서비스 추가', () => {
  it('https 가 아니면 API 를 호출하지 않고, https 면 POST 한다', async () => {
    setupServices([])
    mockApiPost.mockResolvedValue(makeService({ id: 12, name: '씨앱', url: 'https://foo.com' }))

    const user = userEvent.setup()
    renderWithProviders(<CompanyServicesSection companyId={1} isOwner />)

    const nameInput = await screen.findByLabelText('서비스 이름')
    const urlInput = screen.getByLabelText('서비스 주소')

    await user.type(nameInput, '씨앱')
    await user.type(urlInput, 'http://foo.com')
    await user.click(screen.getByRole('button', { name: '추가' }))

    expect(toastError).toHaveBeenCalledWith('https:// URL만 등록할 수 있어요')
    expect(mockApiPost).not.toHaveBeenCalled()

    await user.clear(urlInput)
    await user.type(urlInput, 'https://foo.com')
    await user.click(screen.getByRole('button', { name: '추가' }))

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith('/companies/1/services', {
        name: '씨앱',
        url: 'https://foo.com',
      })
    })
  })

  it('중복 URL 이면 서버 메시지를 toast.error 한다', async () => {
    setupServices([])
    mockApiPost.mockRejectedValue(
      new ApiError('DUPLICATE_SERVICE_URL', '이미 등록된 URL이에요', 409),
    )

    const user = userEvent.setup()
    renderWithProviders(<CompanyServicesSection companyId={1} isOwner />)

    const nameInput = await screen.findByLabelText('서비스 이름')
    await user.type(nameInput, '씨앱')
    await user.type(screen.getByLabelText('서비스 주소'), 'https://foo.com')
    await user.click(screen.getByRole('button', { name: '추가' }))

    await waitFor(() => {
      expect(toastError).toHaveBeenCalledWith('이미 등록된 URL이에요')
    })
  })
})

describe('CompanyServicesSection 서비스 수정', () => {
  it('URL 을 바꾸면 PUT 하고 재연동 필요 상태를 보여준다', async () => {
    const before = makeService({
      id: 10,
      name: '에이앱',
      url: 'https://a-app.example.com',
      validation_status: 'valid',
      rybbit_status: 'connected',
      rybbit_site_id: '42',
      connect_ready: false,
    })
    const after = makeService({
      id: 10,
      name: '에이앱',
      url: 'https://new.example.com',
      validation_status: 'unvalidated',
      rybbit_status: 'needs_reconnect',
      rybbit_site_id: '42',
      connect_ready: false,
    })
    setupServices([before], [after])
    mockApiPut.mockResolvedValue(after)

    const user = userEvent.setup()
    renderWithProviders(<CompanyServicesSection companyId={1} isOwner />)

    await screen.findByText('에이앱')
    await user.click(screen.getByRole('button', { name: '에이앱 수정' }))

    const dialog = await screen.findByRole('dialog')
    const urlInput = within(dialog).getByLabelText('서비스 주소')
    await user.clear(urlInput)
    await user.type(urlInput, 'https://new.example.com')
    await user.click(within(dialog).getByRole('button', { name: '저장' }))

    await waitFor(() => {
      expect(mockApiPut).toHaveBeenCalledWith('/companies/1/services/10', {
        name: '에이앱',
        url: 'https://new.example.com',
      })
    })
    expect(await screen.findByText('재연동 필요')).toBeInTheDocument()
  })
})

describe('CompanyServicesSection 서비스 삭제', () => {
  it('삭제를 확인하면 DELETE 하고 목록에서 사라진다', async () => {
    const a = makeService({ id: 10, name: '에이앱', url: 'https://a-app.example.com' })
    const b = makeService({ id: 11, name: '비앱', url: 'https://b-app.example.com' })
    // 삭제 후 refetch 응답에는 에이앱이 없다.
    setupServices([a, b], [b])
    mockApiDel.mockResolvedValue({ message: 'deleted' })
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)

    const user = userEvent.setup()
    renderWithProviders(<CompanyServicesSection companyId={1} isOwner />)

    await screen.findByText('에이앱')
    // 서비스마다 삭제 버튼이 있다.
    expect(screen.getByRole('button', { name: '에이앱 삭제' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '비앱 삭제' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '에이앱 삭제' }))

    expect(confirmSpy).toHaveBeenCalled()
    await waitFor(() => {
      expect(mockApiDel).toHaveBeenCalledWith('/companies/1/services/10')
    })
    // refetch 결과가 화면에 반영된다.
    await waitFor(() => {
      expect(screen.queryByText('에이앱')).not.toBeInTheDocument()
    })
    expect(screen.getByText('비앱')).toBeInTheDocument()
    expect(toastSuccess).toHaveBeenCalledWith('서비스를 삭제했어요')
  })

  it('확인창에서 취소하면 DELETE 를 호출하지 않는다', async () => {
    setupServices([makeService({ id: 10, name: '에이앱' })])
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false)

    const user = userEvent.setup()
    renderWithProviders(<CompanyServicesSection companyId={1} isOwner />)

    await screen.findByText('에이앱')
    await user.click(screen.getByRole('button', { name: '에이앱 삭제' }))

    expect(confirmSpy).toHaveBeenCalled()
    expect(mockApiDel).not.toHaveBeenCalled()
    expect(screen.getByText('에이앱')).toBeInTheDocument()
    expect(toastSuccess).not.toHaveBeenCalled()
  })

  it('소유자가 아니면 삭제 버튼이 없다', async () => {
    setupServices([makeService({ id: 10, name: '에이앱' })])

    renderWithProviders(<CompanyServicesSection companyId={1} isOwner={false} />)

    expect(await screen.findByText('에이앱')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /삭제/ })).not.toBeInTheDocument()
  })

  it('삭제 실패 시 toast.error 하고 목록은 그대로다', async () => {
    setupServices([makeService({ id: 10, name: '에이앱' })])
    mockApiDel.mockRejectedValue(
      new ApiError('NOT_FOUND', '서비스를 찾을 수 없어요.', 404),
    )
    vi.spyOn(window, 'confirm').mockReturnValue(true)

    const user = userEvent.setup()
    renderWithProviders(<CompanyServicesSection companyId={1} isOwner />)

    await screen.findByText('에이앱')
    await user.click(screen.getByRole('button', { name: '에이앱 삭제' }))

    await waitFor(() => {
      expect(toastError).toHaveBeenCalledWith('서비스를 찾을 수 없어요.')
    })
    expect(screen.getByText('에이앱')).toBeInTheDocument()
    expect(toastSuccess).not.toHaveBeenCalled()
  })
})
