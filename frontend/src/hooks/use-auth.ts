import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from 'react'
import { useNavigate } from 'react-router-dom'
import { createElement } from 'react'
import type { User } from '@/types'
import { api } from '@/lib/api'
import { getToken, setToken, removeToken, isTokenExpired } from '@/lib/auth'
import { wsClient } from '@/lib/ws'

interface RegisterData {
  email: string
  password: string
  name: string
  department: string
  student_id: string
}

interface AuthContextValue {
  user: User | null
  isLoading: boolean
  login: (email: string, password: string) => Promise<void>
  register: (data: RegisterData) => Promise<void>
  logout: () => void
  refreshUser: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const navigate = useNavigate()

  const fetchUser = useCallback(async () => {
    const token = getToken()
    if (!token || isTokenExpired(token)) {
      removeToken()
      setUser(null)
      setIsLoading(false)
      return
    }

    try {
      const userData = await api.get<User>('/auth/me')
      setUser(userData)
      wsClient.connect(token)
    } catch {
      removeToken()
      setUser(null)
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchUser()
    return () => {
      wsClient.disconnect()
    }
  }, [fetchUser])

  const login = useCallback(
    async (email: string, password: string) => {
      const result = await api.post<{ token: string; user: User }>(
        '/auth/login',
        { email, password },
      )
      setToken(result.token)
      setUser(result.user)
      wsClient.connect(result.token)
    },
    [],
  )

  const register = useCallback(async (data: RegisterData) => {
    await api.post('/auth/register', data)
  }, [])

  const logout = useCallback(() => {
    wsClient.disconnect()
    removeToken()
    setUser(null)
    navigate('/login')
  }, [navigate])

  const refreshUser = useCallback(async () => {
    const token = getToken()
    if (!token) return
    try {
      const userData = await api.get<User>('/auth/me')
      setUser(userData)
    } catch {
      // ignore refresh errors
    }
  }, [])

  return createElement(
    AuthContext.Provider,
    {
      value: { user, isLoading, login, register, logout, refreshUser },
    },
    children,
  )
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth는 AuthProvider 내부에서만 사용할 수 있습니다.')
  }
  return context
}
