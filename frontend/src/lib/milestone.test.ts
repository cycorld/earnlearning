import { describe, expect, it } from 'vitest'
import {
  MILESTONE_TYPES,
  MILESTONE_LABELS,
  classifyGroup,
  isValidMilestoneURL,
} from './milestone'

describe('isValidMilestoneURL', () => {
  it('accepts vercel.app and custom domains', () => {
    expect(isValidMilestoneURL('https://my-mvp.vercel.app')).toBe(true)
    expect(isValidMilestoneURL('https://my-mvp.vercel.app/path')).toBe(true)
    expect(isValidMilestoneURL('https://example.com')).toBe(true)
    expect(isValidMilestoneURL('https://my.netlify.app')).toBe(true)
    expect(isValidMilestoneURL('https://student.github.io/project')).toBe(true)
  })

  it('rejects practice domains (deny list)', () => {
    expect(isValidMilestoneURL('https://ai.studio/apps/1')).toBe(false)
    expect(isValidMilestoneURL('https://aistudio.google.com/prompts/new')).toBe(false)
    expect(isValidMilestoneURL('https://claude.ai/chat/abc')).toBe(false)
    expect(isValidMilestoneURL('https://www.claude.ai/foo')).toBe(false)
    expect(isValidMilestoneURL('https://chatgpt.com/c/1')).toBe(false)
    expect(isValidMilestoneURL('https://gemini.google.com/app')).toBe(false)
    expect(isValidMilestoneURL('http://localhost:3000')).toBe(false)
    expect(isValidMilestoneURL('http://127.0.0.1:5173')).toBe(false)
  })

  it('rejects invalid input', () => {
    expect(isValidMilestoneURL('')).toBe(false)
    expect(isValidMilestoneURL('   ')).toBe(false)
    expect(isValidMilestoneURL('example.com')).toBe(false) // no scheme
    expect(isValidMilestoneURL('ftp://example.com')).toBe(false)
    expect(isValidMilestoneURL('javascript:alert(1)')).toBe(false)
  })
})

describe('classifyGroup', () => {
  it('maps approved count → group', () => {
    expect(classifyGroup(0)).toBe('')
    expect(classifyGroup(1)).toBe('D')
    expect(classifyGroup(2)).toBe('C')
    expect(classifyGroup(3)).toBe('B')
    expect(classifyGroup(4)).toBe('A')
  })

  it('handles out-of-range values gracefully', () => {
    expect(classifyGroup(5)).toBe('')
    expect(classifyGroup(-1)).toBe('')
  })
})

describe('MILESTONE constants', () => {
  it('has 4 types in the order matching syllabus-actual.md', () => {
    expect(MILESTONE_TYPES).toEqual(['mvp1', 'mvp2', 'business_plan', 'retrospective'])
  })

  it('labels each type', () => {
    for (const t of MILESTONE_TYPES) {
      expect(MILESTONE_LABELS[t]).toBeTruthy()
    }
  })
})
