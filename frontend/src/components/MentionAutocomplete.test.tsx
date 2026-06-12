import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useState } from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { MarkdownEditor } from './MarkdownEditor'

// ─── API Mock ─────────────────────────────────────────────────

const mockApiGet = vi.fn()

vi.mock('@/lib/api', () => ({
  api: {
    get: (...args: unknown[]) => mockApiGet(...args),
    post: vi.fn(),
    put: vi.fn(),
    del: vi.fn(),
  },
}))

const mentionUser = {
  id: 7,
  name: '김멘션',
  department: '컴퓨터공학과',
  student_id: '20학번',
  avatar_url: '',
}

// MarkdownEditor는 제어 컴포넌트 — 상태를 들고 있는 래퍼로 렌더
function renderEditor() {
  let current = ''
  const Wrapper = () => {
    const [value, setValue] = useState('')
    current = value
    return (
      <MemoryRouter>
        <MarkdownEditor value={value} onChange={setValue} placeholder="내용 입력" />
      </MemoryRouter>
    )
  }
  const utils = render(<Wrapper />)
  return { ...utils, getValue: () => current }
}

describe('#132 멘션 자동완성 — 이메일 타이핑 간섭 없음', () => {
  beforeEach(() => {
    mockApiGet.mockReset()
    mockApiGet.mockResolvedValue([])
  })

  it('단어 중간 @(이메일)는 드롭다운도 검색 호출도 없다', async () => {
    const user = userEvent.setup()
    const { getValue } = renderEditor()

    const textarea = screen.getByPlaceholderText('내용 입력')
    await user.type(textarea, 'abc@gmail.com')

    // debounce(200ms)보다 길게 대기 후 검색 미호출 확인
    await new Promise((r) => setTimeout(r, 350))
    expect(mockApiGet).not.toHaveBeenCalled()
    expect(screen.queryByText('김멘션')).toBeNull()
    expect(getValue()).toBe('abc@gmail.com')
  })

  it('줄 시작 @라도 검색 결과 없으면 드롭다운 없이 그대로 타이핑된다', async () => {
    const user = userEvent.setup()
    const { getValue } = renderEditor()

    const textarea = screen.getByPlaceholderText('내용 입력')
    await user.type(textarea, '@gmail.com 으로 보내주세요')

    await new Promise((r) => setTimeout(r, 350))
    expect(screen.queryByText('김멘션')).toBeNull()
    expect(getValue()).toBe('@gmail.com 으로 보내주세요')
  })

  it('드롭다운이 떠도 선택 없이 계속 타이핑하면 텍스트 그대로 (마크업 삽입 안 됨)', async () => {
    // 검색어 '김'만 매치 → 드롭다운 표시, 검색어가 길어지면('김x…') 결과 없음 → 닫힘
    mockApiGet.mockImplementation((path: unknown) => {
      const q = decodeURIComponent(String(path))
      return Promise.resolve(q.endsWith('q=김') ? [mentionUser] : [])
    })
    const user = userEvent.setup()
    const { getValue } = renderEditor()

    const textarea = screen.getByPlaceholderText('내용 입력')
    await user.type(textarea, '@김')

    // 드롭다운 표시 확인
    await waitFor(() => {
      expect(screen.getByText('김멘션')).toBeInTheDocument()
    })

    // 선택하지 않고 계속 타이핑 → 결과 없음 → 드롭다운 닫힘
    await user.type(textarea, 'x입니다')
    await waitFor(() => {
      expect(screen.queryByText('김멘션')).toBeNull()
    })

    // Enter는 멘션 선택이 아니라 줄바꿈
    await user.keyboard('{Enter}')
    expect(getValue()).toBe('@김x입니다\n')
    expect(getValue()).not.toContain('](user:')
  })
})
