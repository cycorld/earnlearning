import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  renderWithProviders,
  setMockUser,
  mockAdmin,
  mockStudent,
} from '@/test/test-utils'
import {
  allMockPosts,
  mockChannels,
  mockClassrooms,
  mockCommentsForPost1,
  paginatePosts,
  paginateComments,
  postWithNoAuthor,
  postWithLongContent,
  pinnedPost,
} from '@/test/mock-data'
import FeedPage from './FeedPage'

// ─── API Mock ─────────────────────────────────────────────────

const mockApiGet = vi.fn()
const mockApiPost = vi.fn()
const mockApiPut = vi.fn()
const mockApiDel = vi.fn()

vi.mock('@/lib/api', () => ({
  api: {
    get: (...args: unknown[]) => mockApiGet(...args),
    post: (...args: unknown[]) => mockApiPost(...args),
    put: (...args: unknown[]) => mockApiPut(...args),
    del: (...args: unknown[]) => mockApiDel(...args),
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

// ─── Default API Responses ────────────────────────────────────

function setupDefaultApiMocks() {
  const page1 = paginatePosts(allMockPosts, 1, 20)

  mockApiGet.mockImplementation((path: string) => {
    if (path === '/classrooms') return Promise.resolve(mockClassrooms)
    if (path.includes('/channels'))
      return Promise.resolve(mockChannels)
    if (path.includes('/posts?'))
      return Promise.resolve(page1)
    if (path.match(/\/posts\/\d+\/comments/))
      return Promise.resolve(paginateComments(mockCommentsForPost1, 1, 50))
    return Promise.resolve([])
  })

  mockApiPost.mockImplementation(() => Promise.resolve({ liked: true, reward: 0 }))
  mockApiPut.mockImplementation(() => Promise.resolve({}))
  mockApiDel.mockImplementation(() => Promise.resolve({}))
}

beforeEach(() => {
  vi.clearAllMocks()
  setupDefaultApiMocks()
})

// ─── Tests ────────────────────────────────────────────────────

describe('FeedPage 타임라인', () => {
  it('게시글 목록이 렌더링된다', async () => {
    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText(/테스트 게시글 #1번/)).toBeInTheDocument()
    })
  })

  it('각 게시글에 작성자 이름이 표시된다', async () => {
    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      // 25개 중 20개 (1페이지), 다양한 작성자 이름이 표시되어야 함
      expect(screen.getAllByText(/김학생/).length).toBeGreaterThan(0)
      expect(screen.getAllByText(/이개발/).length).toBeGreaterThan(0)
    })
  })

  it('게시글 내용이 표시된다', async () => {
    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText(/테스트 게시글 #2번 내용입니다/)).toBeInTheDocument()
      expect(screen.getByText(/테스트 게시글 #3번 내용입니다/)).toBeInTheDocument()
    })
  })

  it('좋아요 수와 댓글 수가 표시된다', async () => {
    const postsData = paginatePosts(
      [{ ...allMockPosts[0], like_count: 7, comment_count: 3 }],
      1,
      20,
    )
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/classrooms') return Promise.resolve(mockClassrooms)
      if (path.includes('/channels')) return Promise.resolve(mockChannels)
      if (path.includes('/posts?')) return Promise.resolve(postsData)
      return Promise.resolve([])
    })

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText('7')).toBeInTheDocument()
      expect(screen.getByText('3')).toBeInTheDocument()
    })
  })

  it('채널 탭이 표시된다', async () => {
    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      const tabs = screen.getAllByRole('tab')
      const tabTexts = tabs.map((t) => t.textContent)
      expect(tabTexts).toContain('전체')
      expect(tabTexts).toContain('공지')
      expect(tabTexts).toContain('자유')
      expect(tabTexts).toContain('과제')
    })
  })

  it('채널 탭 클릭 시 해당 채널의 게시글을 요청한다', async () => {
    const user = userEvent.setup()
    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getAllByRole('tab').length).toBeGreaterThan(0)
    })

    // role=tab 중 "자유" 텍스트를 가진 탭 클릭
    const freeTab = screen.getAllByRole('tab').find((t) => t.textContent === '자유')!
    await user.click(freeTab)

    await waitFor(() => {
      const calls = mockApiGet.mock.calls.map((c) => c[0])
      expect(calls.some((c: string) => c.includes('channel_id=2'))).toBe(true)
    })
  })

  it('태그가 있는 게시글에 태그 뱃지가 표시된다', async () => {
    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      // id % 3 === 0 인 포스트에 '프로젝트', '테스트' 태그가 있음
      expect(screen.getAllByText('#프로젝트').length).toBeGreaterThan(0)
      expect(screen.getAllByText('#테스트').length).toBeGreaterThan(0)
    })
  })

  it('게시글이 없으면 빈 상태 메시지가 표시된다', async () => {
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/classrooms') return Promise.resolve(mockClassrooms)
      if (path.includes('/channels')) return Promise.resolve(mockChannels)
      if (path.includes('/posts?'))
        return Promise.resolve({ data: [], pagination: { page: 1, limit: 20, total: 0, total_pages: 0 } })
      return Promise.resolve([])
    })

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText('아직 게시물이 없습니다.')).toBeInTheDocument()
    })
  })

  it('클래스룸이 없으면 참여 폼이 표시된다', async () => {
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/classrooms') return Promise.resolve([])
      return Promise.resolve([])
    })

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText('클래스룸 참여')).toBeInTheDocument()
      expect(screen.getByPlaceholderText('초대 코드')).toBeInTheDocument()
    })
  })

  it('author가 없는 게시글도 크래시 없이 렌더링된다', async () => {
    const postsData = paginatePosts([postWithNoAuthor], 1, 20)
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/classrooms') return Promise.resolve(mockClassrooms)
      if (path.includes('/channels')) return Promise.resolve(mockChannels)
      if (path.includes('/posts?')) return Promise.resolve(postsData)
      return Promise.resolve([])
    })

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      // author가 없어도 게시글이 크래시 없이 렌더링되어야 함
      expect(screen.getByText(/테스트 게시글 #101번/)).toBeInTheDocument()
      // displayName(undefined) → '?'가 작성자 이름과 아바타 fallback에 표시됨
      expect(screen.getAllByText('?').length).toBeGreaterThan(0)
    })
  })

  it('고정 게시글에 핀 아이콘이 표시된다', async () => {
    const postsData = paginatePosts([pinnedPost, ...allMockPosts.slice(0, 5)], 1, 20)
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/classrooms') return Promise.resolve(mockClassrooms)
      if (path.includes('/channels')) return Promise.resolve(mockChannels)
      if (path.includes('/posts?')) return Promise.resolve(postsData)
      return Promise.resolve([])
    })

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText(/중요 공지사항입니다/)).toBeInTheDocument()
    })
  })
})

describe('FeedPage 좋아요', () => {
  it('좋아요 버튼 클릭 시 API를 호출하고 카운트가 변경된다', async () => {
    const user = userEvent.setup()
    const post = { ...allMockPosts[0], like_count: 77, comment_count: 0, is_liked: false }
    const postsData = paginatePosts([post], 1, 20)

    mockApiGet.mockImplementation((path: string) => {
      if (path === '/classrooms') return Promise.resolve(mockClassrooms)
      if (path.includes('/channels')) return Promise.resolve(mockChannels)
      if (path.includes('/posts?')) return Promise.resolve(postsData)
      return Promise.resolve([])
    })
    mockApiPost.mockImplementation((path: string) => {
      if (path === `/posts/${post.id}/like`) return Promise.resolve({ liked: true, reward: 10 })
      return Promise.resolve({})
    })

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText('77')).toBeInTheDocument()
    })

    // 좋아요 버튼 클릭
    const likeButton = screen.getByText('77').closest('button')!
    await user.click(likeButton)

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith(`/posts/${post.id}/like`)
      expect(screen.getByText('78')).toBeInTheDocument()
    })
  })
})

describe('FeedPage 댓글', () => {
  it('댓글 버튼 클릭 시 댓글 목록을 로드한다', async () => {
    const user = userEvent.setup()
    const post = { ...allMockPosts[0], comment_count: 30 }
    const postsData = paginatePosts([post], 1, 20)

    mockApiGet.mockImplementation((path: string) => {
      if (path === '/classrooms') return Promise.resolve(mockClassrooms)
      if (path.includes('/channels')) return Promise.resolve(mockChannels)
      if (path.includes('/posts?')) return Promise.resolve(postsData)
      if (path.match(/\/posts\/\d+\/comments/))
        return Promise.resolve(paginateComments(mockCommentsForPost1, 1, 50))
      return Promise.resolve([])
    })

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText('30')).toBeInTheDocument()
    })

    // 댓글 버튼 클릭
    const commentButton = screen.getByText('30').closest('button')!
    await user.click(commentButton)

    await waitFor(() => {
      // 댓글이 로드되어 댓글 작성자 이름이 표시됨
      expect(screen.getByText(/댓글 #1번입니다/)).toBeInTheDocument()
    })
  })

  it('댓글이 없으면 빈 상태 메시지가 표시된다', async () => {
    const user = userEvent.setup()
    const post = { ...allMockPosts[0], comment_count: 0 }
    const postsData = paginatePosts([post], 1, 20)

    mockApiGet.mockImplementation((path: string) => {
      if (path === '/classrooms') return Promise.resolve(mockClassrooms)
      if (path.includes('/channels')) return Promise.resolve(mockChannels)
      if (path.includes('/posts?')) return Promise.resolve(postsData)
      if (path.match(/\/posts\/\d+\/comments/))
        return Promise.resolve({ data: [], pagination: { page: 1, limit: 50, total: 0, total_pages: 0 } })
      return Promise.resolve([])
    })

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText('0')).toBeInTheDocument()
    })

    const commentButton = screen.getByText('0').closest('button')!
    await user.click(commentButton)

    await waitFor(() => {
      expect(screen.getByText('아직 댓글이 없습니다.')).toBeInTheDocument()
    })
  })

  it('댓글 입력 후 전송하면 API가 호출된다', async () => {
    const user = userEvent.setup()
    const post = { ...allMockPosts[0], comment_count: 0 }
    const postsData = paginatePosts([post], 1, 20)

    const newComment = {
      id: 999,
      post_id: post.id,
      author: { id: 2, name: '김학생', avatar_url: '' },
      content: '새 댓글입니다',
      media: [],
      created_at: new Date().toISOString(),
    }

    mockApiGet.mockImplementation((path: string) => {
      if (path === '/classrooms') return Promise.resolve(mockClassrooms)
      if (path.includes('/channels')) return Promise.resolve(mockChannels)
      if (path.includes('/posts?')) return Promise.resolve(postsData)
      if (path.match(/\/posts\/\d+\/comments/))
        return Promise.resolve({ data: [], pagination: { page: 1, limit: 50, total: 0, total_pages: 0 } })
      return Promise.resolve([])
    })
    mockApiPost.mockResolvedValue(newComment)

    renderWithProviders(<FeedPage />)

    // 댓글 섹션 열기
    await waitFor(() => {
      expect(screen.getByText('0')).toBeInTheDocument()
    })
    const commentButton = screen.getByText('0').closest('button')!
    await user.click(commentButton)

    await waitFor(() => {
      expect(screen.getByPlaceholderText(/댓글을 입력하세요/)).toBeInTheDocument()
    })

    // 댓글 입력
    const textarea = screen.getByPlaceholderText(/댓글을 입력하세요/)
    await user.type(textarea, '새 댓글입니다')

    // 전송 버튼 클릭
    const sendButton = screen.getByRole('button', { name: '댓글' })
    await user.click(sendButton)

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith(
        `/posts/${post.id}/comments`,
        expect.objectContaining({ content: '새 댓글입니다' }),
      )
    })
  })
})

describe('FeedPage 새 게시물 작성', () => {
  it('새 게시물 작성 버튼이 표시된다', async () => {
    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText('새 게시물 작성')).toBeInTheDocument()
    })
  })
})

describe('FeedPage 긴 내용 접기/펼치기', () => {
  it('긴 게시글에 "더보기" 버튼이 표시된다', async () => {
    const postsData = paginatePosts([postWithLongContent], 1, 20)
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/classrooms') return Promise.resolve(mockClassrooms)
      if (path.includes('/channels')) return Promise.resolve(mockChannels)
      if (path.includes('/posts?')) return Promise.resolve(postsData)
      return Promise.resolve([])
    })

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText('더보기')).toBeInTheDocument()
    })
  })

  it('"더보기" 클릭 시 "접기"로 변경된다', async () => {
    const user = userEvent.setup()
    const postsData = paginatePosts([postWithLongContent], 1, 20)
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/classrooms') return Promise.resolve(mockClassrooms)
      if (path.includes('/channels')) return Promise.resolve(mockChannels)
      if (path.includes('/posts?')) return Promise.resolve(postsData)
      return Promise.resolve([])
    })

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText('더보기')).toBeInTheDocument()
    })

    await user.click(screen.getByText('더보기'))

    await waitFor(() => {
      expect(screen.getByText('접기')).toBeInTheDocument()
      expect(screen.queryByText('더보기')).not.toBeInTheDocument()
    })
  })

  it('짧은 게시글에는 "더보기" 버튼이 없다', async () => {
    const shortPost = { ...allMockPosts[0], content: '짧은 글입니다.' }
    const postsData = paginatePosts([shortPost], 1, 20)
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/classrooms') return Promise.resolve(mockClassrooms)
      if (path.includes('/channels')) return Promise.resolve(mockChannels)
      if (path.includes('/posts?')) return Promise.resolve(postsData)
      return Promise.resolve([])
    })

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText('짧은 글입니다.')).toBeInTheDocument()
    })

    expect(screen.queryByText('더보기')).not.toBeInTheDocument()
  })
})

describe('FeedPage 댓글 작성자 표시', () => {
  it('댓글에 작성자 이름이 표시된다', async () => {
    const user = userEvent.setup()
    const post = { ...allMockPosts[0], comment_count: 30 }
    const postsData = paginatePosts([post], 1, 20)

    mockApiGet.mockImplementation((path: string) => {
      if (path === '/classrooms') return Promise.resolve(mockClassrooms)
      if (path.includes('/channels')) return Promise.resolve(mockChannels)
      if (path.includes('/posts?')) return Promise.resolve(postsData)
      if (path.match(/\/posts\/\d+\/comments/))
        return Promise.resolve(paginateComments(mockCommentsForPost1, 1, 50))
      return Promise.resolve([])
    })

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText('30')).toBeInTheDocument()
    })

    const commentButton = screen.getByText('30').closest('button')!
    await user.click(commentButton)

    await waitFor(() => {
      // 댓글 작성자들의 이름이 표시됨
      expect(screen.getAllByText(/김학생/).length).toBeGreaterThan(0)
      expect(screen.getAllByText(/이개발/).length).toBeGreaterThan(0)
    })
  })

  it('30개 댓글이 모두 렌더링된다', async () => {
    const user = userEvent.setup()
    const post = { ...allMockPosts[0], comment_count: 30 }
    const postsData = paginatePosts([post], 1, 20)

    mockApiGet.mockImplementation((path: string) => {
      if (path === '/classrooms') return Promise.resolve(mockClassrooms)
      if (path.includes('/channels')) return Promise.resolve(mockChannels)
      if (path.includes('/posts?')) return Promise.resolve(postsData)
      if (path.match(/\/posts\/\d+\/comments/))
        return Promise.resolve(paginateComments(mockCommentsForPost1, 1, 50))
      return Promise.resolve([])
    })

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText('30')).toBeInTheDocument()
    })

    const commentButton = screen.getByText('30').closest('button')!
    await user.click(commentButton)

    await waitFor(() => {
      // 30개 댓글 중 마지막 댓글까지 표시됨
      expect(screen.getByText(/댓글 #30번입니다/)).toBeInTheDocument()
      expect(screen.getByText(/댓글 #1번입니다/)).toBeInTheDocument()
      expect(screen.getByText(/댓글 #15번입니다/)).toBeInTheDocument()
    })
  })
})

describe('FeedPage 좋아요 취소', () => {
  it('이미 좋아요한 게시글 클릭 시 좋아요가 취소된다', async () => {
    const user = userEvent.setup()
    const post = { ...allMockPosts[0], like_count: 10, is_liked: true }
    const postsData = paginatePosts([post], 1, 20)

    mockApiGet.mockImplementation((path: string) => {
      if (path === '/classrooms') return Promise.resolve(mockClassrooms)
      if (path.includes('/channels')) return Promise.resolve(mockChannels)
      if (path.includes('/posts?')) return Promise.resolve(postsData)
      return Promise.resolve([])
    })
    mockApiPost.mockImplementation((path: string) => {
      if (path === `/posts/${post.id}/like`) return Promise.resolve({ liked: false, reward: 0 })
      return Promise.resolve({})
    })

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText('10')).toBeInTheDocument()
    })

    const likeButton = screen.getByText('10').closest('button')!
    await user.click(likeButton)

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith(`/posts/${post.id}/like`)
      expect(screen.getByText('9')).toBeInTheDocument()
    })
  })
})

describe('FeedPage API 에러 처리', () => {
  it('게시글 로드 실패 시 크래시하지 않는다', async () => {
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/classrooms') return Promise.resolve(mockClassrooms)
      if (path.includes('/channels')) return Promise.resolve(mockChannels)
      if (path.includes('/posts?')) return Promise.reject(new Error('서버 에러'))
      return Promise.resolve([])
    })

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText('아직 게시물이 없습니다.')).toBeInTheDocument()
    })
  })

  it('클래스룸 로드 실패 시 참여 폼이 표시된다', async () => {
    mockApiGet.mockImplementation(() => Promise.reject(new Error('네트워크 에러')))

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText('클래스룸 참여')).toBeInTheDocument()
    })
  })
})

describe('FeedPage 게시물 수정 - 관리자 카테고리 변경 (#175)', () => {
  // 학생(id=2)이 '자유'(id=2) 채널에 쓴 글. 관리자(id=1)와 작성자가 다름.
  const studentPost = {
    ...allMockPosts[0],
    id: 501,
    author: { id: 2, name: '김학생', avatar_url: '', student_id: '2026000001' },
    channel: mockChannels[1], // { id: 2, name: '자유' }
    content: '학생이 쓴 글입니다',
    tags: [] as string[],
  }

  function setupSinglePost(post: typeof studentPost) {
    const data = paginatePosts([post], 1, 20)
    mockApiGet.mockImplementation((path: string) => {
      if (path === '/classrooms') return Promise.resolve(mockClassrooms)
      if (path.includes('/channels')) return Promise.resolve(mockChannels)
      if (path.includes('/posts?')) return Promise.resolve(data)
      if (path.match(/\/posts\/\d+\/comments/))
        return Promise.resolve(paginateComments(mockCommentsForPost1, 1, 50))
      return Promise.resolve([])
    })
  }

  // 수정 연필 트리거는 접근 가능한 이름이 없으므로 고유 className 으로 찾는다.
  function findEditPencil() {
    return screen
      .getAllByRole('button')
      .find((b) => b.className.includes('hover:bg-muted'))
  }

  afterEach(() => {
    setMockUser(mockStudent)
  })

  it('관리자가 학생 글의 수정 다이얼로그를 열면 카테고리 선택이 표시된다', async () => {
    setMockUser(mockAdmin)
    const user = userEvent.setup({ pointerEventsCheck: 0 })
    setupSinglePost(studentPost)

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText(/학생이 쓴 글입니다/)).toBeInTheDocument()
    })

    await user.click(findEditPencil()!)
    await user.click(await screen.findByText('수정'))

    const select = (await screen.findByLabelText('카테고리')) as HTMLSelectElement
    expect(select).toBeInTheDocument()
    // 모든 채널이 옵션으로 노출된다
    expect(within(select).getByRole('option', { name: '공지' })).toBeInTheDocument()
    expect(within(select).getByRole('option', { name: '자유' })).toBeInTheDocument()
    expect(within(select).getByRole('option', { name: '과제' })).toBeInTheDocument()
  })

  it('학생이 자기 글의 수정 다이얼로그를 열면 카테고리 선택이 없다', async () => {
    setMockUser(mockStudent) // 기본값이지만 명시
    const user = userEvent.setup({ pointerEventsCheck: 0 })
    const ownPost = { ...studentPost, id: 502 } // author id=2 = mockStudent
    setupSinglePost(ownPost)

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText(/학생이 쓴 글입니다/)).toBeInTheDocument()
    })

    await user.click(findEditPencil()!)
    await user.click(await screen.findByText('수정'))

    // 수정 다이얼로그는 열리지만 카테고리 선택은 없어야 한다
    await screen.findByText('게시물 수정')
    expect(screen.queryByLabelText('카테고리')).not.toBeInTheDocument()
  })

  it('관리자가 카테고리를 변경해 저장하면 channel_id 를 전송하고 뱃지가 갱신된다', async () => {
    setMockUser(mockAdmin)
    const user = userEvent.setup({ pointerEventsCheck: 0 })
    setupSinglePost(studentPost)
    // 백엔드 PUT 응답의 channel 은 null (채널명 join 안 함) — 로컬 channels 로 해석되어야 함
    mockApiPut.mockResolvedValue({
      id: 501,
      channel: null,
      content: '학생이 쓴 글입니다',
      tags: [],
    })

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText(/학생이 쓴 글입니다/)).toBeInTheDocument()
    })

    // 초기 뱃지 = 자유 (카드 내부로 스코프해서 채널 탭과 구분)
    const card = screen
      .getByText(/학생이 쓴 글입니다/)
      .closest('[data-slot="card"]') as HTMLElement
    expect(within(card).getByText('자유')).toBeInTheDocument()

    await user.click(findEditPencil()!)
    await user.click(await screen.findByText('수정'))

    const select = (await screen.findByLabelText('카테고리')) as HTMLSelectElement
    await user.selectOptions(select, '1') // 공지 (id=1)

    await user.click(screen.getByRole('button', { name: '수정' }))

    await waitFor(() => {
      expect(mockApiPut).toHaveBeenCalledWith(
        '/posts/501',
        expect.objectContaining({ channel_id: 1 }),
      )
    })

    // 응답 channel 이 null 이어도 로컬 channels 목록에서 '공지'로 해석되어 뱃지가 갱신된다
    await waitFor(() => {
      const updatedCard = screen
        .getByText(/학생이 쓴 글입니다/)
        .closest('[data-slot="card"]') as HTMLElement
      expect(within(updatedCard).getByText('공지')).toBeInTheDocument()
      expect(within(updatedCard).queryByText('자유')).not.toBeInTheDocument()
    })
  })

  it('학생 저장 시 payload 에 channel_id 키가 없다', async () => {
    setMockUser(mockStudent)
    const user = userEvent.setup({ pointerEventsCheck: 0 })
    const ownPost = { ...studentPost, id: 503 }
    setupSinglePost(ownPost)
    mockApiPut.mockResolvedValue({
      id: 503,
      channel: null,
      content: '학생이 쓴 글입니다',
      tags: [],
    })

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText(/학생이 쓴 글입니다/)).toBeInTheDocument()
    })

    await user.click(findEditPencil()!)
    await user.click(await screen.findByText('수정'))
    await screen.findByText('게시물 수정')

    await user.click(screen.getByRole('button', { name: '수정' }))

    await waitFor(() => {
      expect(mockApiPut).toHaveBeenCalled()
    })
    const body = mockApiPut.mock.calls[0][1] as Record<string, unknown>
    expect(Object.keys(body)).not.toContain('channel_id')
  })

  it('관리자는 다른 사람이 쓴 글에도 수정 연필 버튼이 보인다 (회귀)', async () => {
    setMockUser(mockAdmin)
    setupSinglePost(studentPost) // 작성자 id=2, 관리자 id=1

    renderWithProviders(<FeedPage />)

    await waitFor(() => {
      expect(screen.getByText(/학생이 쓴 글입니다/)).toBeInTheDocument()
    })

    expect(findEditPencil()).toBeTruthy()
  })
})
