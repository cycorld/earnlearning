import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, act } from '@testing-library/react'
import { MemoryRouter, Routes, Route, useNavigate } from 'react-router-dom'
import { GAPageViewTracker } from './use-ga-pageview'
import * as analytics from '@/lib/analytics'

function Nav({ to }: { to: string }) {
  const navigate = useNavigate()
  return (
    <button data-testid="nav" onClick={() => navigate(to)}>
      go
    </button>
  )
}

describe('GAPageViewTracker', () => {
  beforeEach(() => {
    vi.spyOn(analytics, 'trackPageView').mockImplementation(() => undefined)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('mount 시 현재 경로로 page_view 1회', () => {
    render(
      <MemoryRouter initialEntries={['/feed']}>
        <GAPageViewTracker />
        <Routes>
          <Route path="/feed" element={<div />} />
        </Routes>
      </MemoryRouter>,
    )
    expect(analytics.trackPageView).toHaveBeenCalledTimes(1)
    expect(analytics.trackPageView).toHaveBeenCalledWith('/feed')
  })

  it('라우트 이동 시 새 경로로 page_view 추가 발사', () => {
    const { getByTestId } = render(
      <MemoryRouter initialEntries={['/feed']}>
        <GAPageViewTracker />
        <Routes>
          <Route path="/feed" element={<Nav to="/wallet" />} />
          <Route path="/wallet" element={<div />} />
        </Routes>
      </MemoryRouter>,
    )
    expect(analytics.trackPageView).toHaveBeenCalledTimes(1)
    act(() => {
      getByTestId('nav').click()
    })
    expect(analytics.trackPageView).toHaveBeenCalledTimes(2)
    expect(analytics.trackPageView).toHaveBeenLastCalledWith('/wallet')
  })

  it('search query string 도 함께 기록한다', () => {
    render(
      <MemoryRouter initialEntries={['/post/42?ref=feed']}>
        <GAPageViewTracker />
        <Routes>
          <Route path="/post/:id" element={<div />} />
        </Routes>
      </MemoryRouter>,
    )
    expect(analytics.trackPageView).toHaveBeenCalledWith('/post/42?ref=feed')
  })
})
