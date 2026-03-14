import { render, type RenderOptions } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { type ReactNode, createElement } from 'react'
import type { User } from '@/types'

// Mock user
export const mockAdmin: User = {
  id: 1,
  email: 'admin@test.com',
  name: '최용철',
  role: 'admin',
  status: 'approved',
  department: '관리자',
  student_id: '0000000000',
  bio: '',
  avatar_url: '',
}

export const mockStudent: User = {
  id: 2,
  email: 'student@test.com',
  name: '김학생',
  role: 'student',
  status: 'approved',
  department: '컴퓨터공학과',
  student_id: '2026000001',
  bio: '',
  avatar_url: 'https://example.com/avatar.png',
}

// Mock auth context value
let mockUser: User | null = mockStudent

export function setMockUser(user: User | null) {
  mockUser = user
}

// Mock useAuth
vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({
    user: mockUser,
    isLoading: false,
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
    refreshUser: vi.fn(),
  }),
}))

// Mock sonner toast
vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
}))

// Custom render with providers
function AllProviders({ children }: { children: ReactNode }) {
  return createElement(MemoryRouter, null, children)
}

export function renderWithProviders(
  ui: React.ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>,
) {
  return render(ui, { wrapper: AllProviders, ...options })
}

export { render }
