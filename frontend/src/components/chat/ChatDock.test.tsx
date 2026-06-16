/**
 * #135 회귀: 챗봇 FAB가 페이지 전송버튼을 가리는 문제.
 *  - A: DM 대화 라우트(/messages/:id)에선 FAB 숨김
 *  - A: 외부 텍스트 입력 포커스 시 FAB fade(pointer-events-none)
 *
 * 이 테스트가 깨지면 FAB가 다시 composer 위에 겹칠 위험.
 */
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({ user: { role: 'student', status: 'approved' } }),
}))

import ChatDock from './ChatDock'

const renderAt = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <ChatDock />
    </MemoryRouter>,
  )

const fab = () => screen.queryByRole('button', { name: /챗봇 조교 열기/ })

describe('#135 FAB 컨텍스트 숨김', () => {
  it('일반 페이지(/feed)에선 FAB 표시', () => {
    renderAt('/feed')
    expect(fab()).not.toBeNull()
  })

  it('DM 대화(/messages/123)에선 FAB 숨김', () => {
    renderAt('/messages/123')
    expect(fab()).toBeNull()
  })

  it('DM 목록(/messages)에선 FAB 표시 (composer 없음)', () => {
    renderAt('/messages')
    expect(fab()).not.toBeNull()
  })

  it('외부 텍스트 입력 포커스 시 FAB는 pointer-events-none 로 숨김', () => {
    render(
      <MemoryRouter initialEntries={['/feed']}>
        <input data-testid="external" />
        <ChatDock />
      </MemoryRouter>,
    )
    const before = fab()
    expect(before?.className).not.toContain('pointer-events-none')
    fireEvent.focusIn(screen.getByTestId('external'))
    expect(fab()?.className).toContain('pointer-events-none')
  })
})
