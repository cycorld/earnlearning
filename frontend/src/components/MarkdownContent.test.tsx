import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { MarkdownContent } from './MarkdownContent'

function renderMd(content: string) {
  return render(
    <MemoryRouter>
      <MarkdownContent content={content} />
    </MemoryRouter>,
  )
}

describe('MarkdownContent — link rendering', () => {
  // Regression for #099: SPA 내부 라우트 (예: /grant/14) 가 #086 fix 로 비활성화됐던 회귀.
  it('SPA 내부 라우트 (/grant/14) 는 클릭 가능한 <a> 로 렌더', () => {
    const { container } = renderMd('[지원하러 가기](/grant/14)')
    const link = container.querySelector('a')
    expect(link).not.toBeNull()
    expect(link?.getAttribute('href')).toBe('/grant/14')
  })

  it('알려지지 않은 비-절대 path (/wiki/없는문서) 는 #086 처럼 비활성화 (span)', () => {
    const { container } = renderMd('[잘못된 출처](/wiki/없는문서)')
    expect(container.querySelector('a')).toBeNull()
    expect(container.querySelector('span[title*="유효하지 않은"]')).not.toBeNull()
  })

  it('http 절대 URL 은 외부 링크 (target=_blank)', () => {
    const { container } = renderMd('[Anthropic](https://anthropic.com)')
    const link = container.querySelector('a')
    expect(link?.getAttribute('href')).toBe('https://anthropic.com')
    expect(link?.getAttribute('target')).toBe('_blank')
  })

  it('/uploads/ 는 외부 링크 + download', () => {
    const { container } = renderMd('![img](/uploads/foo.png)')
    // img 태그로 렌더되므로 a 가 아닐 수 있음 — 텍스트 링크로 테스트
    const linkContainer = renderMd('[파일](/uploads/foo.pdf)').container
    const link = linkContainer.querySelector('a')
    expect(link?.getAttribute('href')).toBe('/uploads/foo.pdf')
    expect(link?.getAttribute('target')).toBe('_blank')
    expect(container).toBeTruthy()
  })

  it.each([
    '/feed', '/wallet', '/llm', '/profile/3', '/grant', '/notifications', '/admin/chat',
  ])('알려진 SPA prefix %s 는 활성 링크', (path) => {
    const { container } = renderMd(`[가기](${path})`)
    const link = container.querySelector('a')
    expect(link, `${path} should render <a>`).not.toBeNull()
    expect(link?.getAttribute('href')).toBe(path)
  })
})
