import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { CoinChip } from './CoinChip'

describe('CoinChip', () => {
  it('금액을 천 단위 구분자와 함께 표시한다', () => {
    const { container } = render(<CoinChip amount={1500} />)
    expect(container.textContent).toContain('1,500')
  })

  it('💰 아이콘이 렌더링된다', () => {
    const { container } = render(<CoinChip amount={100} />)
    expect(container.textContent).toContain('💰')
  })

  it('showSign=true + 양수는 + 접두사를 붙인다', () => {
    const { container } = render(<CoinChip amount={200} showSign />)
    expect(container.textContent).toContain('+200')
  })

  it('showSign=true + 음수는 coral 변형으로 렌더링되고 - 접두사를 붙인다', () => {
    const { container } = render(<CoinChip amount={-500} showSign />)
    expect(container.textContent).toContain('-500')
    const root = container.firstChild as HTMLElement
    expect(root.getAttribute('data-tone')).toBe('loss')
  })

  it('showSign=true + 0 은 중립 톤', () => {
    const { container } = render(<CoinChip amount={0} showSign />)
    const root = container.firstChild as HTMLElement
    expect(root.getAttribute('data-tone')).toBe('neutral')
  })

  it('기본 톤은 gain', () => {
    const { container } = render(<CoinChip amount={500} />)
    const root = container.firstChild as HTMLElement
    expect(root.getAttribute('data-tone')).toBe('gain')
  })

  it('size=sm / md / lg prop 을 받는다', () => {
    const sizes = ['sm', 'md', 'lg'] as const
    sizes.forEach((size) => {
      const { container } = render(<CoinChip amount={1} size={size} />)
      const root = container.firstChild as HTMLElement
      expect(root.getAttribute('data-size')).toBe(size)
    })
  })
})
