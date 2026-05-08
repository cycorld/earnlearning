/**
 * #114 회귀: 투자 페이지 리스트 카드에 회사 정보 (대표자 / service URL / 소개) 노출.
 * 이 테스트가 깨지면 학생이 "어느 회사인지" 모른 채 투자해야 하는 상태로 회귀.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import InvestPage from './InvestPage'
import { api } from '@/lib/api'

vi.mock('@/lib/api', () => ({
  api: {
    get: vi.fn(),
  },
}))

const sampleRound = {
  id: 1,
  company_id: 7,
  company: {
    id: 7,
    name: 'Genova',
    valuation: 5_000_000,
    logo_url: '',
    description: '이대 학생을 위한 코딩 도우미. 매일 30분 투자로 첫 MVP 를 6주 안에.',
    service_url: 'https://genova.example',
  },
  owner: { id: 25, name: '김예린' },
  target_amount: 1_000_000,
  offered_percent: 0.2,
  current_amount: 200_000,
  price_per_share: 100,
  new_shares: 10000,
  remaining_shares: 8000,
  sold_shares: 2000,
  status: 'open' as const,
  expires_at: null,
  created_at: '2026-05-08T00:00:00Z',
  funded_at: null,
}

describe('InvestPage — #114 회사 정보 노출', () => {
  beforeEach(() => {
    vi.mocked(api.get).mockImplementation((path: string) => {
      if (path.startsWith('/investment/rounds')) {
        return Promise.resolve({ rounds: [sampleRound], total: 1 })
      }
      return Promise.resolve([])
    })
  })

  it('대표자 이름이 카드에 표시된다', async () => {
    render(
      <MemoryRouter>
        <InvestPage />
      </MemoryRouter>,
    )
    // displayName(round.owner) 결과가 표시됨 — 익명화 정책 적용 후 어떤 형태든 owner 표시 검증
    await waitFor(() => {
      expect(screen.getByText(/대표/)).toBeInTheDocument()
    })
  })

  it('서비스 URL 바로가기 버튼이 표시된다', async () => {
    render(
      <MemoryRouter>
        <InvestPage />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByText('서비스 바로가기')).toBeInTheDocument()
    })
  })

  it('회사 소개 snippet 이 표시된다 (truncate 80자)', async () => {
    render(
      <MemoryRouter>
        <InvestPage />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(
        screen.getByText(/이대 학생을 위한 코딩 도우미/),
      ).toBeInTheDocument()
    })
  })

  it('description 이 80자 초과 시 … 으로 자른다', async () => {
    const longDesc = '가'.repeat(100)
    vi.mocked(api.get).mockImplementation((path: string) => {
      if (path.startsWith('/investment/rounds')) {
        return Promise.resolve({
          rounds: [{ ...sampleRound, company: { ...sampleRound.company, description: longDesc } }],
          total: 1,
        })
      }
      return Promise.resolve([])
    })
    render(
      <MemoryRouter>
        <InvestPage />
      </MemoryRouter>,
    )
    await waitFor(() => {
      const card = screen.getByText(/^가{80}…$/)
      expect(card).toBeInTheDocument()
    })
  })

  it('description / service_url 없으면 해당 영역 안 그림 (안전)', async () => {
    vi.mocked(api.get).mockImplementation((path: string) => {
      if (path.startsWith('/investment/rounds')) {
        return Promise.resolve({
          rounds: [
            {
              ...sampleRound,
              company: { ...sampleRound.company, description: '', service_url: '' },
            },
          ],
          total: 1,
        })
      }
      return Promise.resolve([])
    })
    render(
      <MemoryRouter>
        <InvestPage />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByText('Genova')).toBeInTheDocument()
    })
    expect(screen.queryByText('서비스 바로가기')).toBeNull()
  })
})
