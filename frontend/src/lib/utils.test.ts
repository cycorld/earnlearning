import { describe, it, expect } from 'vitest'
import { formatDate } from './utils'

describe('formatDate', () => {
  // #025 회귀: ISO 타임스탬프(`2026-04-12T00:00:00Z`)를 날짜로 그대로 렌더하던 버그 방어.
  it('ISO 타임스탬프를 한국어 날짜 포맷으로 변환한다', () => {
    expect(formatDate('2026-04-12T00:00:00Z')).toBe('2026. 4. 12.')
  })

  it('YYYY-MM-DD 문자열을 한국어 날짜 포맷으로 변환한다', () => {
    expect(formatDate('2026-04-12')).toBe('2026. 4. 12.')
  })

  it('월/일에 앞자리 0이 없어야 한다 (한국 로케일 규칙)', () => {
    expect(formatDate('2026-01-05')).toBe('2026. 1. 5.')
  })

  it('빈 문자열은 빈 문자열로 반환한다', () => {
    expect(formatDate('')).toBe('')
  })

  it('null / undefined 는 빈 문자열로 반환한다', () => {
    expect(formatDate(null)).toBe('')
    expect(formatDate(undefined)).toBe('')
  })

  it('알 수 없는 포맷은 원본 그대로 반환한다 (방어적)', () => {
    expect(formatDate('??')).toBe('??')
  })
})
