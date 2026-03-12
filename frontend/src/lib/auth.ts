const TOKEN_KEY = 'el_token'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function removeToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

interface JwtPayload {
  user_id: number
  email: string
  role: string
  status: string
  exp: number
}

export function parseToken(token: string): JwtPayload | null {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) return null
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const payload = JSON.parse(atob(base64))
    return {
      user_id: payload.sub ?? payload.user_id,
      email: payload.email,
      role: payload.role,
      status: payload.status,
      exp: payload.exp,
    }
  } catch {
    return null
  }
}

export function isTokenExpired(token: string): boolean {
  const payload = parseToken(token)
  if (!payload) return true
  return Date.now() >= payload.exp * 1000
}
