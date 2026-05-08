import { describe, it, expect } from 'vitest'
import {
  parseServiceUrls,
  formatServiceUrls,
  isValidHttpUrl,
  isValidServiceUrls,
  shortenUrl,
} from './urls'

describe('parseServiceUrls', () => {
  it('빈 입력 (undefined / null / "") → 빈 배열', () => {
    expect(parseServiceUrls(undefined)).toEqual([])
    expect(parseServiceUrls(null)).toEqual([])
    expect(parseServiceUrls('')).toEqual([])
  })

  it('단일 URL → 1-element 배열', () => {
    expect(parseServiceUrls('https://a.com')).toEqual(['https://a.com'])
  })

  it('쉼표 구분 다중 URL → N-element 배열', () => {
    expect(parseServiceUrls('https://a.com,https://b.com')).toEqual([
      'https://a.com',
      'https://b.com',
    ])
  })

  it('주변 공백 / 쉼표 주변 공백 trim', () => {
    expect(parseServiceUrls('  https://a.com , https://b.com  ')).toEqual([
      'https://a.com',
      'https://b.com',
    ])
  })

  it('빈 piece (연속 쉼표, 끝 쉼표) 는 제거', () => {
    expect(parseServiceUrls('https://a.com,,https://b.com,')).toEqual([
      'https://a.com',
      'https://b.com',
    ])
  })

  it('whitespace-only piece 도 제거', () => {
    expect(parseServiceUrls('https://a.com,   ,https://b.com')).toEqual([
      'https://a.com',
      'https://b.com',
    ])
  })
})

describe('formatServiceUrls', () => {
  it('빈 배열 → 빈 문자열', () => {
    expect(formatServiceUrls([])).toBe('')
  })

  it('1개 URL', () => {
    expect(formatServiceUrls(['https://a.com'])).toBe('https://a.com')
  })

  it('2개 URL — 쉼표 구분 (공백 X)', () => {
    expect(formatServiceUrls(['https://a.com', 'https://b.com'])).toBe(
      'https://a.com,https://b.com',
    )
  })

  it('빈 piece 는 저장 시 제거', () => {
    expect(formatServiceUrls(['https://a.com', '', '  ', 'https://b.com'])).toBe(
      'https://a.com,https://b.com',
    )
  })

  it('parse → format roundtrip 안정', () => {
    const raw = 'https://a.com, https://b.com'
    expect(formatServiceUrls(parseServiceUrls(raw))).toBe(
      'https://a.com,https://b.com',
    )
  })
})

describe('isValidHttpUrl', () => {
  it('http / https 만 valid', () => {
    expect(isValidHttpUrl('https://a.com')).toBe(true)
    expect(isValidHttpUrl('http://a.com')).toBe(true)
    expect(isValidHttpUrl('https://a.com/path?q=1')).toBe(true)
  })

  it('ftp / mailto / javascript 등은 invalid (보안)', () => {
    expect(isValidHttpUrl('ftp://a.com')).toBe(false)
    expect(isValidHttpUrl('mailto:a@b.com')).toBe(false)
    expect(isValidHttpUrl('javascript:alert(1)')).toBe(false)
  })

  it('protocol 없는 문자열은 invalid', () => {
    expect(isValidHttpUrl('a.com')).toBe(false)
    expect(isValidHttpUrl('www.a.com')).toBe(false)
  })

  it('빈 문자열 invalid', () => {
    expect(isValidHttpUrl('')).toBe(false)
  })
})

describe('isValidServiceUrls', () => {
  it('빈 입력 → valid (URL 0개 OK)', () => {
    expect(isValidServiceUrls('')).toBe(true)
    expect(isValidServiceUrls(undefined)).toBe(true)
  })

  it('모든 piece 가 valid http/https → true', () => {
    expect(isValidServiceUrls('https://a.com,https://b.com,http://c.com')).toBe(
      true,
    )
  })

  it('한 piece 라도 invalid → false', () => {
    expect(isValidServiceUrls('https://a.com,not-a-url')).toBe(false)
    expect(isValidServiceUrls('https://a.com,javascript:alert(1)')).toBe(false)
  })

  it('빈 piece 는 무시 (parseServiceUrls 가 걸러냄)', () => {
    expect(isValidServiceUrls('https://a.com,,https://b.com')).toBe(true)
  })
})

describe('shortenUrl', () => {
  it('https:// 제거', () => {
    expect(shortenUrl('https://a.com/path')).toBe('a.com/path')
  })

  it('http:// 제거', () => {
    expect(shortenUrl('http://a.com')).toBe('a.com')
  })

  it('protocol 없으면 그대로', () => {
    expect(shortenUrl('a.com')).toBe('a.com')
  })

  it('빈 문자열은 빈 문자열', () => {
    expect(shortenUrl('')).toBe('')
  })
})
