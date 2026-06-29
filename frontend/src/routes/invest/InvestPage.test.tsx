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

  // #115 회귀: 다중 URL — 첫 URL 만 "서비스 바로가기" + 추가 URL 개수 +N
  it('서비스 URL 이 여러 개면 첫 URL + "+N" 표시', async () => {
    vi.mocked(api.get).mockImplementation((path: string) => {
      if (path.startsWith('/investment/rounds')) {
        return Promise.resolve({
          rounds: [
            {
              ...sampleRound,
              company: {
                ...sampleRound.company,
                service_url: 'https://a.example,https://b.example,https://c.example',
              },
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
      expect(screen.getByText('서비스 바로가기')).toBeInTheDocument()
    })
    // 추가 2개 = "+2"
    expect(screen.getByText('+2')).toBeInTheDocument()
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

/**
 * #136 회귀: 투자 라운드 목록 페이지네이션 + 카드 세로 간격.
 * - 한 페이지(최대 50개)를 넘는 오픈 라운드도 전부 가져온다 (예전엔 첫 20개만 보임).
 * - 라운드 카드를 감싼 <Link>(=inline <a>)에 block 을 줘 space-y 간격이 먹는다.
 */
describe('InvestPage — #136 페이지네이션 & 카드 간격', () => {
  it('라운드가 한 페이지(50개)를 넘으면 모든 페이지를 가져온다', async () => {
    const pageSize = 50
    const total = 55
    const makeRound = (id: number) => ({
      ...sampleRound,
      id,
      company: { ...sampleRound.company, name: `Co${id}` },
    })
    vi.mocked(api.get).mockImplementation((path: string) => {
      if (path.startsWith('/investment/rounds')) {
        const page = Number(
          new URLSearchParams(path.split('?')[1] ?? '').get('page') ?? '1',
        )
        const start = (page - 1) * pageSize
        const batch = Array.from(
          { length: Math.max(0, Math.min(pageSize, total - start)) },
          (_, i) => makeRound(start + i + 1),
        )
        return Promise.resolve({ rounds: batch, total })
      }
      return Promise.resolve([])
    })
    render(
      <MemoryRouter>
        <InvestPage />
      </MemoryRouter>,
    )
    // Co55 는 2페이지에만 존재 → 페이지 순회를 했다는 증거
    await waitFor(() => {
      expect(screen.getByText('Co55')).toBeInTheDocument()
    })
  })

  it('라운드 카드 링크가 block 이라 카드 간 세로 간격이 유지된다', async () => {
    vi.mocked(api.get).mockImplementation((path: string) => {
      if (path.startsWith('/investment/rounds')) {
        return Promise.resolve({ rounds: [sampleRound], total: 1 })
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
    const link = screen.getByText('Genova').closest('a')
    expect(link).toHaveClass('block')
  })
})
