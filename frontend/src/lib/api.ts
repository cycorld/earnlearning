import type { ApiResponse } from '@/types'
import { getToken, setToken, removeToken } from './auth'

const BASE_URL = '/api'

class ApiError extends Error {
  code: string
  status: number

  constructor(code: string, message: string, status: number) {
    super(message)
    this.code = code
    this.status = status
    this.name = 'ApiError'
  }
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {}

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  if (body && !(body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
  }

  const res = await fetch(`${BASE_URL}${path}`, {
    method,
    headers,
    body: body
      ? body instanceof FormData
        ? body
        : JSON.stringify(body)
      : undefined,
  })

  if (!res.ok) {
    if (res.status === 401 && !path.startsWith('/auth/')) {
      // Try to refresh the token before giving up
      const refreshed = await tryRefreshToken()
      if (refreshed) {
        // Retry the original request with the new token
        return request<T>(method, path, body)
      }
      removeToken()
      window.location.href = '/login'
      throw new ApiError('UNAUTHORIZED', '세션이 만료되었습니다. 다시 로그인해주세요.', 401)
    }
    let errorData: ApiResponse<null> | null = null
    try {
      errorData = await res.json()
    } catch {
      // ignore parse errors
    }
    throw new ApiError(
      errorData?.error?.code || 'UNKNOWN',
      errorData?.error?.message || res.statusText,
      res.status,
    )
  }

  const data: ApiResponse<T> = await res.json()
  return data.data
}

let refreshPromise: Promise<boolean> | null = null

async function tryRefreshToken(): Promise<boolean> {
  // Deduplicate concurrent refresh attempts
  if (refreshPromise) return refreshPromise

  refreshPromise = (async () => {
    try {
      const token = getToken()
      if (!token) return false

      const res = await fetch(`${BASE_URL}/auth/refresh`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` },
      })

      if (!res.ok) return false

      const data: ApiResponse<{ token: string }> = await res.json()
      setToken(data.data.token)
      return true
    } catch {
      return false
    } finally {
      refreshPromise = null
    }
  })()

  return refreshPromise
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
  del: <T>(path: string) => request<T>('DELETE', path),
}

export { ApiError }
