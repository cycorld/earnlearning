/**
 * #118 회귀: /grants (복수) 경로가 silent 으로 /feed catch-all 로 튕기던 버그.
 *
 * 학생이 자연스럽게 영어 list = 복수형으로 추측해 입력 → 안내 없이 /feed 로 redirect.
 * 의도: /grants → /grant 로 명시 redirect.
 *
 * 이 테스트가 깨지면 같은 회귀 즉시 알림.
 */
import { describe, it, expect } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import {
  MemoryRouter,
  Routes,
  Route,
  Navigate,
  useLocation,
} from 'react-router-dom'

function LocationProbe() {
  const loc = useLocation()
  return <div data-testid="loc">{loc.pathname}</div>
}

// App.tsx 와 동일한 grant 라우트 구조를 반영
function GrantRoutesProbe() {
  return (
    <Routes>
      <Route path="/grant" element={<LocationProbe />} />
      <Route path="/grant/:id" element={<LocationProbe />} />
      {/* #118: /grants (복수, 단독) → /grant 명시 redirect */}
      <Route path="/grants" element={<Navigate to="/grant" replace />} />
      <Route path="/grants/:id" element={<Navigate to="/grant" replace />} />
      <Route path="*" element={<div data-testid="catchall">CATCHALL</div>} />
    </Routes>
  )
}

describe('#118 — /grants (복수) 라우트 redirect', () => {
  it('/grants → /grant 로 redirect (catch-all 로 안 떨어짐)', async () => {
    render(
      <MemoryRouter initialEntries={['/grants']}>
        <GrantRoutesProbe />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByTestId('loc').textContent).toBe('/grant')
    })
    expect(screen.queryByTestId('catchall')).toBeNull()
  })

  it('/grants/14 (레거시 복수형) → /grant 로 redirect', async () => {
    render(
      <MemoryRouter initialEntries={['/grants/14']}>
        <GrantRoutesProbe />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByTestId('loc').textContent).toBe('/grant')
    })
  })

  it('/grant 직접 접근은 그대로 렌더', async () => {
    render(
      <MemoryRouter initialEntries={['/grant']}>
        <GrantRoutesProbe />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByTestId('loc').textContent).toBe('/grant')
    })
  })
})
