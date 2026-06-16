import { describe, it, expect, beforeEach } from 'vitest'

import {
  FAB_ANCHORS,
  loadAnchor,
  saveAnchor,
  nearestAnchor,
  type FabAnchor,
} from './fab-position'

const W = 400
const H = 800

describe('nearestAnchor', () => {
  it('각 앵커 근처 좌표는 그 앵커로 스냅된다', () => {
    const cases: Array<[FabAnchor, { x: number; y: number }]> = [
      ['top-left', { x: 10, y: 10 }],
      ['top-right', { x: 390, y: 10 }],
      ['mid-left', { x: 10, y: 400 }],
      ['mid-right', { x: 390, y: 400 }],
      ['bottom-left', { x: 10, y: 790 }],
      ['bottom-right', { x: 390, y: 790 }],
    ]
    for (const [expected, point] of cases) {
      expect(nearestAnchor(point, W, H)).toBe(expected)
    }
  })

  it('좌/우는 x 화면 중앙 기준으로 갈린다', () => {
    expect(nearestAnchor({ x: 199, y: 400 }, W, H).endsWith('left')).toBe(true)
    expect(nearestAnchor({ x: 201, y: 400 }, W, H).endsWith('right')).toBe(true)
  })

  it('항상 6개 앵커 중 하나를 반환', () => {
    const a = nearestAnchor({ x: 123, y: 456 }, W, H)
    expect(FAB_ANCHORS).toContain(a)
  })
})

describe('load/save anchor', () => {
  beforeEach(() => localStorage.clear())

  it('저장한 앵커를 그대로 읽는다', () => {
    saveAnchor('mid-left')
    expect(loadAnchor()).toBe('mid-left')
  })

  it('저장값 없으면 기본 bottom-right', () => {
    expect(loadAnchor()).toBe('bottom-right')
  })

  it('손상된 값은 기본값으로 폴백', () => {
    localStorage.setItem('chatdock:fab-anchor', 'garbage')
    expect(loadAnchor()).toBe('bottom-right')
  })
})
