import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { createElement } from 'react'
import LoginPage from './LoginPage'

// ─── Mocks ─────────────────────────────────────────────────

const mockLogin = vi.fn()
const mockNavigate = vi.fn()

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({
    login: mockLogin,
    user: null,
    isLoading: false,
    register: vi.fn(),
    logout: vi.fn(),
    refreshUser: vi.fn(),
  }),
}))

function renderLogin() {
  return render(
    createElement(MemoryRouter, null, createElement(LoginPage)),
  )
}

// ─── Tests ─────────────────────────────────────────────────

describe('LoginPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  describe('아이디 저장', () => {
    it('기본적으로 아이디 저장 체크박스가 체크되어 있어야 한다', () => {
      renderLogin()
      const checkbox = screen.getByLabelText('아이디 저장')
      expect(checkbox).toBeChecked()
    })

    it('로그인 시 아이디 저장이 체크되면 이메일이 localStorage에 저장된다', async () => {
      mockLogin.mockResolvedValueOnce(undefined)
      const user = userEvent.setup()
      renderLogin()

      await user.type(screen.getByLabelText('이메일'), 'student@test.com')
      await user.type(screen.getByLabelText('비밀번호'), 'password123')
      await user.click(screen.getByRole('button', { name: '로그인' }))

      await waitFor(() => {
        expect(localStorage.getItem('el_saved_email')).toBe('student@test.com')
        expect(localStorage.getItem('el_remember_email')).toBe('true')
      })
    })

    it('아이디 저장 해제 후 로그인하면 저장된 이메일이 삭제된다', async () => {
      localStorage.setItem('el_saved_email', 'old@test.com')
      mockLogin.mockResolvedValueOnce(undefined)
      const user = userEvent.setup()
      renderLogin()

      // Uncheck remember email
      await user.click(screen.getByLabelText('아이디 저장'))
      await user.clear(screen.getByLabelText('이메일'))
      await user.type(screen.getByLabelText('이메일'), 'new@test.com')
      await user.type(screen.getByLabelText('비밀번호'), 'password123')
      await user.click(screen.getByRole('button', { name: '로그인' }))

      await waitFor(() => {
        expect(localStorage.getItem('el_saved_email')).toBeNull()
        expect(localStorage.getItem('el_remember_email')).toBe('false')
      })
    })

    it('저장된 이메일이 있으면 페이지 로드 시 자동으로 채워진다', () => {
      localStorage.setItem('el_saved_email', 'saved@test.com')
      localStorage.setItem('el_remember_email', 'true')
      renderLogin()

      const emailInput = screen.getByLabelText('이메일') as HTMLInputElement
      expect(emailInput.value).toBe('saved@test.com')
    })

    it('아이디 저장이 false로 저장되어 있으면 이메일을 복원하지 않는다', () => {
      localStorage.setItem('el_saved_email', 'saved@test.com')
      localStorage.setItem('el_remember_email', 'false')
      renderLogin()

      const emailInput = screen.getByLabelText('이메일') as HTMLInputElement
      expect(emailInput.value).toBe('')
    })
  })

  describe('로그인 유지', () => {
    it('기본적으로 로그인 유지 체크박스가 체크되어 있지 않아야 한다', () => {
      renderLogin()
      const checkbox = screen.getByLabelText('로그인 유지')
      expect(checkbox).not.toBeChecked()
    })

    it('로그인 유지 체크 시 remember_me=true로 login 호출된다', async () => {
      mockLogin.mockResolvedValueOnce(undefined)
      const user = userEvent.setup()
      renderLogin()

      await user.type(screen.getByLabelText('이메일'), 'test@test.com')
      await user.type(screen.getByLabelText('비밀번호'), 'password123')
      await user.click(screen.getByLabelText('로그인 유지'))
      await user.click(screen.getByRole('button', { name: '로그인' }))

      await waitFor(() => {
        expect(mockLogin).toHaveBeenCalledWith('test@test.com', 'password123', true)
      })
    })

    it('로그인 유지 미체크 시 remember_me=false로 login 호출된다', async () => {
      mockLogin.mockResolvedValueOnce(undefined)
      const user = userEvent.setup()
      renderLogin()

      await user.type(screen.getByLabelText('이메일'), 'test@test.com')
      await user.type(screen.getByLabelText('비밀번호'), 'password123')
      await user.click(screen.getByRole('button', { name: '로그인' }))

      await waitFor(() => {
        expect(mockLogin).toHaveBeenCalledWith('test@test.com', 'password123', false)
      })
    })

    it('로그인 유지 체크 상태가 localStorage에 저장된다', async () => {
      mockLogin.mockResolvedValueOnce(undefined)
      const user = userEvent.setup()
      renderLogin()

      await user.type(screen.getByLabelText('이메일'), 'test@test.com')
      await user.type(screen.getByLabelText('비밀번호'), 'password123')
      await user.click(screen.getByLabelText('로그인 유지'))
      await user.click(screen.getByRole('button', { name: '로그인' }))

      await waitFor(() => {
        expect(localStorage.getItem('el_remember_me')).toBe('true')
      })
    })

    it('이전에 로그인 유지를 체크했으면 다음 방문 시 체크되어 있어야 한다', () => {
      localStorage.setItem('el_remember_me', 'true')
      renderLogin()

      const checkbox = screen.getByLabelText('로그인 유지')
      expect(checkbox).toBeChecked()
    })
  })

  describe('로그인 동작', () => {
    it('로그인 성공 시 /feed로 네비게이트된다', async () => {
      mockLogin.mockResolvedValueOnce(undefined)
      const user = userEvent.setup()
      renderLogin()

      await user.type(screen.getByLabelText('이메일'), 'test@test.com')
      await user.type(screen.getByLabelText('비밀번호'), 'password123')
      await user.click(screen.getByRole('button', { name: '로그인' }))

      await waitFor(() => {
        expect(mockNavigate).toHaveBeenCalledWith('/feed', { replace: true })
      })
    })

    it('로그인 실패 시 에러 메시지가 표시된다', async () => {
      mockLogin.mockRejectedValueOnce(new Error('이메일 또는 비밀번호가 올바르지 않습니다.'))
      const user = userEvent.setup()
      renderLogin()

      await user.type(screen.getByLabelText('이메일'), 'test@test.com')
      await user.type(screen.getByLabelText('비밀번호'), 'wrongpass')
      await user.click(screen.getByRole('button', { name: '로그인' }))

      await waitFor(() => {
        expect(screen.getByText('이메일 또는 비밀번호가 올바르지 않습니다.')).toBeInTheDocument()
      })
    })
  })
})
