/**
 * #117 회귀: 게시글 상세 페이지에서 댓글이 안 보이던 버그.
 *
 * 원인: 백엔드 GET /api/posts/:id/comments 가 {data: [...], pagination} 으로 wrap 응답.
 * PostDetailPage 가 Comment[] 로 받아 Array.isArray() 체크 → 항상 false → setComments([]).
 *
 * Fix: PaginatedData<Comment> 로 받고 .data 로 unwrap.
 * 이 테스트가 깨지면 학생들이 댓글 못 보는 회귀로 복귀.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/test-utils'
import {
  allMockPosts,
  mockCommentsForPost1,
  paginateComments,
} from '@/test/mock-data'
import PostDetailPage from './PostDetailPage'

const mockApiGet = vi.fn()
const mockApiPost = vi.fn()

vi.mock('@/lib/api', () => ({
  api: {
    get: (...args: unknown[]) => mockApiGet(...args),
    post: (...args: unknown[]) => mockApiPost(...args),
    put: vi.fn(),
    del: vi.fn(),
  },
  ApiError: class extends Error {
    code: string
    status: number
    constructor(code: string, message: string, status: number) {
      super(message)
      this.code = code
      this.status = status
    }
  },
}))

vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router-dom')>()
  return {
    ...actual,
    useParams: () => ({ id: '1' }),
    useNavigate: () => vi.fn(),
  }
})

beforeEach(() => {
  vi.clearAllMocks()
  // Backend shape: GET /posts/1 → Post directly,
  //                GET /posts/1/comments?... → PaginatedData<Comment>
  mockApiGet.mockImplementation((path: string) => {
    if (path.match(/^\/posts\/\d+\/comments/)) {
      return Promise.resolve(paginateComments(mockCommentsForPost1, 1, 200))
    }
    if (path.match(/^\/posts\/\d+$/)) {
      return Promise.resolve(allMockPosts.find((p) => p.id === 1))
    }
    return Promise.resolve(null)
  })
})

describe('PostDetailPage — #117 댓글 unwrap 회귀', () => {
  it('백엔드 {data:[],pagination} 응답을 unwrap 해서 댓글 개수가 정확히 표시된다', async () => {
    renderWithProviders(<PostDetailPage />)

    // mockCommentsForPost1 길이는 30 — pagination wrap 응답에서 .data 로 풀어야 30 개로 셈
    await waitFor(() => {
      expect(screen.getByText(/댓글 30개/)).toBeInTheDocument()
    })
  })

  it('백엔드 응답이 빈 배열이면 "댓글이 없습니다" 표시', async () => {
    mockApiGet.mockImplementation((path: string) => {
      if (path.match(/^\/posts\/\d+\/comments/)) {
        return Promise.resolve({
          data: [],
          pagination: { page: 1, limit: 0, total: 0, total_pages: 0 },
        })
      }
      if (path.match(/^\/posts\/\d+$/)) {
        return Promise.resolve(allMockPosts.find((p) => p.id === 1))
      }
      return Promise.resolve(null)
    })
    renderWithProviders(<PostDetailPage />)
    await waitFor(() => {
      expect(screen.getByText(/댓글 0개/)).toBeInTheDocument()
    })
  })

  it('API 호출 실패 시 catch fallback 으로 댓글 0개 표시 (앱 crash 방지)', async () => {
    mockApiGet.mockImplementation((path: string) => {
      if (path.match(/^\/posts\/\d+\/comments/)) {
        return Promise.reject(new Error('network'))
      }
      if (path.match(/^\/posts\/\d+$/)) {
        return Promise.resolve(allMockPosts.find((p) => p.id === 1))
      }
      return Promise.resolve(null)
    })
    renderWithProviders(<PostDetailPage />)
    await waitFor(() => {
      expect(screen.getByText(/댓글 0개/)).toBeInTheDocument()
    })
  })
})
