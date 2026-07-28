import { beforeEach, describe, expect, it, vi } from 'vitest'

import { apiFetch } from './api'
import { getToken, setToken } from './auth'

// #184 첨부 다운로드도 공용 토큰 갱신 경로를 타야 한다.

describe('apiFetch', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
    localStorage.clear()
    setToken('old-token')
  })

  it('/api 로 시작하지 않는 경로에는 BASE_URL 을 붙이고 인증 헤더를 넣는다', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 200 })
    vi.stubGlobal('fetch', fetchMock)

    await apiFetch('/dm/attachments/7')

    expect(fetchMock.mock.calls[0]![0]).toBe('/api/dm/attachments/7')
    expect(fetchMock.mock.calls[0]![1].headers).toMatchObject({
      Authorization: 'Bearer old-token',
    })
  })

  it('/api 로 시작하는 전체 경로는 그대로 쓴다', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 200 })
    vi.stubGlobal('fetch', fetchMock)

    await apiFetch('/api/dm/attachments/7')

    expect(fetchMock.mock.calls[0]![0]).toBe('/api/dm/attachments/7')
  })

  it('401 이면 토큰을 갱신하고 한 번만 재시도한다', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: false, status: 401 })
      .mockResolvedValueOnce({ ok: true, status: 200, json: async () => ({ data: { token: 'new-token' } }) })
      .mockResolvedValueOnce({ ok: true, status: 200 })
    vi.stubGlobal('fetch', fetchMock)

    const res = await apiFetch('/api/dm/attachments/7')

    expect(res.ok).toBe(true)
    expect(getToken()).toBe('new-token')
    expect(fetchMock).toHaveBeenCalledTimes(3)
    expect(fetchMock.mock.calls[1]![0]).toBe('/api/auth/refresh')
    expect(fetchMock.mock.calls[2]![1].headers).toMatchObject({
      Authorization: 'Bearer new-token',
    })
  })

  it('401 이 아닌 실패 응답은 그대로 돌려준다(throw 하지 않음)', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: false, status: 403 })
    vi.stubGlobal('fetch', fetchMock)

    const res = await apiFetch('/api/dm/attachments/7')
    expect(res.status).toBe(403)
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })
})
