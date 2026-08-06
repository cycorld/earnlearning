import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
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
  it('SPA 내부 라우트 (/grant/14) 는 클릭 가능한 <a> 로 렌더', () => {
    const { container } = renderMd('[지원하러 가기](/grant/14)')
    const link = container.querySelector('a')
    expect(link).not.toBeNull()
    expect(link?.getAttribute('href')).toBe('/grant/14')
  })

  it('알려지지 않은 비-절대 path (/wiki/없는문서) 는 비활성화 (span)', () => {
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

describe('MarkdownContent — preview and line breaks', () => {
  it('uses a 12-line preview before showing more', () => {
    const content = Array.from({ length: 13 }, (_, i) => `line ${i + 1}`).join('\n')
    const { container, getByText } = renderMd(content)
    expect(getByText('더보기')).toBeInTheDocument()
    expect(container.querySelector('[style*="18em"]')).not.toBeNull()
  })

  it('renders a single newline as a line break', () => {
    const { container } = renderMd('첫 줄\n둘째 줄')
    expect(container.querySelector('.markdown-body br')).not.toBeNull()
  })
})
