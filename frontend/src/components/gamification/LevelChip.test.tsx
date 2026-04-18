import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { LevelChip, LEVELS, type Level } from './LevelChip'

describe('LevelChip', () => {
  it('레벨 이름을 표시한다', () => {
    const { container } = render(<LevelChip level="Gold" />)
    expect(container.textContent).toContain('Gold')
  })

  it('data-level 속성에 현재 레벨을 노출한다', () => {
    const { container } = render(<LevelChip level="Silver" />)
    const root = container.firstChild as HTMLElement
    expect(root.getAttribute('data-level')).toBe('Silver')
  })

  it('모든 레벨(Seed, Bronze, Silver, Gold, Diamond)을 렌더링할 수 있다', () => {
    LEVELS.forEach((level: Level) => {
      const { container } = render(<LevelChip level={level} />)
      const root = container.firstChild as HTMLElement
      expect(root.getAttribute('data-level')).toBe(level)
    })
  })

  it('showIcon=false 일 때 아이콘을 숨긴다', () => {
    const { container } = render(<LevelChip level="Diamond" showIcon={false} />)
    expect(container.querySelectorAll('[data-slot="level-icon"]').length).toBe(0)
  })

  it('기본 상태에선 레벨 아이콘을 표시한다', () => {
    const { container } = render(<LevelChip level="Bronze" />)
    expect(container.querySelectorAll('[data-slot="level-icon"]').length).toBe(1)
  })
})
