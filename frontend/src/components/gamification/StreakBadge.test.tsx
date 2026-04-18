import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { StreakBadge } from './StreakBadge'

describe('StreakBadge', () => {
  it('연속 일수를 텍스트로 표시한다', () => {
    const { container } = render(<StreakBadge days={7} />)
    expect(container.textContent).toContain('7')
  })

  it('🔥 아이콘이 렌더링된다', () => {
    const { container } = render(<StreakBadge days={3} />)
    expect(container.textContent).toContain('🔥')
  })

  it('0일일 때는 cold 변형으로 렌더링된다', () => {
    const { container } = render(<StreakBadge days={0} />)
    const root = container.firstChild as HTMLElement
    expect(root.getAttribute('data-variant')).toBe('cold')
  })

  it('1~6일은 warm 변형', () => {
    const { container } = render(<StreakBadge days={5} />)
    const root = container.firstChild as HTMLElement
    expect(root.getAttribute('data-variant')).toBe('warm')
  })

  it('7일 이상은 hot 변형', () => {
    const { container } = render(<StreakBadge days={14} />)
    const root = container.firstChild as HTMLElement
    expect(root.getAttribute('data-variant')).toBe('hot')
  })

  it('label prop 이 있으면 단위 텍스트를 커스터마이즈한다', () => {
    const { container } = render(<StreakBadge days={3} label="연속 제출" />)
    expect(container.textContent).toContain('연속 제출')
  })

  it('음수는 0으로 clamp 된다', () => {
    const { container } = render(<StreakBadge days={-1} />)
    const root = container.firstChild as HTMLElement
    expect(root.getAttribute('data-variant')).toBe('cold')
    expect(container.textContent).toContain('0')
  })
})
