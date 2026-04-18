import { describe, it, expect, vi } from 'vitest'
import { render, fireEvent } from '@testing-library/react'
import { SkillTree, type SkillNode } from './SkillTree'

const NODES: SkillNode[] = [
  { id: 'w1', label: '1주차', status: 'completed' },
  { id: 'w2', label: '2주차', status: 'available' },
  { id: 'w3', label: '3주차', status: 'locked' },
  { id: 'w4', label: '4주차', status: 'locked' },
]

describe('SkillTree', () => {
  it('모든 노드 라벨을 렌더링한다', () => {
    const { container } = render(<SkillTree nodes={NODES} />)
    expect(container.textContent).toContain('1주차')
    expect(container.textContent).toContain('2주차')
    expect(container.textContent).toContain('3주차')
    expect(container.textContent).toContain('4주차')
  })

  it('노드에 data-status 속성을 부여한다', () => {
    const { container } = render(<SkillTree nodes={NODES} />)
    const nodeEls = container.querySelectorAll('[data-slot="skill-node"]')
    expect(nodeEls.length).toBe(4)
    expect(nodeEls[0].getAttribute('data-status')).toBe('completed')
    expect(nodeEls[1].getAttribute('data-status')).toBe('available')
    expect(nodeEls[2].getAttribute('data-status')).toBe('locked')
  })

  it('locked 노드는 자물쇠 아이콘을 표시한다', () => {
    const { container } = render(<SkillTree nodes={NODES} />)
    const nodeEls = container.querySelectorAll('[data-slot="skill-node"]')
    expect(nodeEls[2].textContent).toContain('🔒')
  })

  it('완료 노드는 체크 아이콘을 표시한다', () => {
    const { container } = render(<SkillTree nodes={NODES} />)
    const nodeEls = container.querySelectorAll('[data-slot="skill-node"]')
    expect(nodeEls[0].textContent).toContain('✓')
  })

  it('available 노드 클릭 시 onSelect 콜백이 호출된다', () => {
    const onSelect = vi.fn()
    const { container } = render(<SkillTree nodes={NODES} onSelect={onSelect} />)
    const nodeEls = container.querySelectorAll<HTMLElement>('[data-slot="skill-node"]')
    fireEvent.click(nodeEls[1])
    expect(onSelect).toHaveBeenCalledWith('w2')
  })

  it('locked 노드 클릭 시 onSelect 콜백이 호출되지 않는다', () => {
    const onSelect = vi.fn()
    const { container } = render(<SkillTree nodes={NODES} onSelect={onSelect} />)
    const nodeEls = container.querySelectorAll<HTMLElement>('[data-slot="skill-node"]')
    fireEvent.click(nodeEls[2])
    expect(onSelect).not.toHaveBeenCalled()
  })

  it('노드 사이에 연결선(connector)을 렌더링한다', () => {
    const { container } = render(<SkillTree nodes={NODES} />)
    const connectors = container.querySelectorAll('[data-slot="skill-connector"]')
    expect(connectors.length).toBe(NODES.length - 1)
  })

  it('진행도가 완료된 노드 쪽 connector 는 active 상태', () => {
    const { container } = render(<SkillTree nodes={NODES} />)
    const connectors = container.querySelectorAll('[data-slot="skill-connector"]')
    // 1주차(completed) → 2주차(available): 완료 이후의 connector 는 active
    expect(connectors[0].getAttribute('data-active')).toBe('true')
    // 2주차(available) → 3주차(locked): 아직 잠금이므로 inactive
    expect(connectors[1].getAttribute('data-active')).toBe('false')
  })
})
