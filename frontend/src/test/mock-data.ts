import type { Post, Comment, Channel, PaginatedData, User } from '@/types'

// ─── Channels ────────────────────────────────────────────────
export const mockChannels: Channel[] = [
  { id: 1, name: '공지', slug: 'notice', channel_type: 'notice', write_role: 'admin' },
  { id: 2, name: '자유', slug: 'free', channel_type: 'free', write_role: 'all' },
  { id: 3, name: '과제', slug: 'assignment', channel_type: 'assignment', write_role: 'admin' },
  { id: 4, name: '쇼케이스', slug: 'showcase', channel_type: 'showcase', write_role: 'all' },
]

// ─── Posts (25개 — 페이지네이션 테스트용) ──────────────────────

function makePost(id: number, overrides?: Partial<Post>): Post {
  const authors = [
    { id: 1, name: '최용철', avatar_url: '', student_id: '0000000000' },
    { id: 2, name: '김학생', avatar_url: 'https://example.com/avatar2.png', student_id: '2026000001' },
    { id: 3, name: '이개발', avatar_url: '', student_id: '2026000002' },
    { id: 4, name: '박창업', avatar_url: 'https://example.com/avatar4.png', student_id: '2026000003' },
    { id: 5, name: '정디자인', avatar_url: '', student_id: '2026000004' },
  ]
  const author = authors[id % authors.length]
  const channel = mockChannels[id % mockChannels.length]
  const minutesAgo = id * 3
  const createdAt = new Date(Date.now() - minutesAgo * 60 * 1000).toISOString()

  return {
    id,
    channel,
    author,
    content: `테스트 게시글 #${id}번 내용입니다. 이것은 자동 생성된 더미 데이터입니다.`,
    post_type: 'normal',
    media: [],
    tags: id % 3 === 0 ? ['프로젝트', '테스트'] : id % 2 === 0 ? ['공지'] : [],
    like_count: Math.floor(Math.random() * 20),
    comment_count: Math.floor(Math.random() * 10),
    is_liked: id % 4 === 0,
    pinned: id === 1,
    created_at: createdAt,
    ...overrides,
  }
}

// 25개 포스트 생성
export const allMockPosts: Post[] = Array.from({ length: 25 }, (_, i) => makePost(i + 1))

// 페이지네이션 헬퍼
export function paginatePosts(
  posts: Post[],
  page: number,
  limit: number,
): PaginatedData<Post> {
  const start = (page - 1) * limit
  const data = posts.slice(start, start + limit)
  return {
    data,
    pagination: {
      page,
      limit,
      total: posts.length,
      total_pages: Math.ceil(posts.length / limit),
    },
  }
}

// ─── Comments (30개 — 댓글 페이지네이션 테스트용) ──────────────

function makeComment(id: number, postId: number): Comment {
  const authors = [
    { id: 2, name: '김학생', avatar_url: '' },
    { id: 3, name: '이개발', avatar_url: '' },
    { id: 4, name: '박창업', avatar_url: '' },
    { id: 5, name: '정디자인', avatar_url: '' },
    { id: 1, name: '최용철', avatar_url: '' },
  ]
  const minutesAgo = id * 2
  return {
    id,
    post_id: postId,
    author: authors[id % authors.length],
    content: `댓글 #${id}번입니다. 게시글 ${postId}에 대한 의견입니다.`,
    media: [],
    created_at: new Date(Date.now() - minutesAgo * 60 * 1000).toISOString(),
  }
}

// 포스트 1번에 30개 댓글
export const mockCommentsForPost1: Comment[] = Array.from({ length: 30 }, (_, i) =>
  makeComment(i + 1, 1),
)

// 포스트 2번에 5개 댓글
export const mockCommentsForPost2: Comment[] = Array.from({ length: 5 }, (_, i) =>
  makeComment(i + 100, 2),
)

export function paginateComments(
  comments: Comment[],
  page: number,
  limit: number,
): PaginatedData<Comment> {
  const start = (page - 1) * limit
  const data = comments.slice(start, start + limit)
  return {
    data,
    pagination: {
      page,
      limit,
      total: comments.length,
      total_pages: Math.ceil(comments.length / limit),
    },
  }
}

// ─── Classrooms ──────────────────────────────────────────────

export const mockClassrooms = [
  { id: 1, name: '스타트업을 위한 코딩입문 A반', invite_code: 'ABC123' },
]

// ─── Users (20명 — 승인 테스트용) ─────────────────────────────

const koreanNames = [
  '김민수', '이서연', '박지훈', '최수아', '정우진',
  '강예린', '조현우', '윤서윤', '임도현', '한지은',
  '송민재', '오하영', '배준서', '홍서진', '류시우',
  '권나연', '남도윤', '문하은', '장서준', '신유진',
]

const departments = [
  '컴퓨터공학과', '경영학과', '디자인학과', '산업공학과', '전자공학과',
  '미디어학과', '국제학부', '소프트웨어학과', '통계학과', '경제학과',
]

function makeUser(index: number, statusOverride?: User['status']): User {
  const id = index + 10 // admin=1 이므로 10부터 시작
  const statuses: User['status'][] = ['pending', 'pending', 'approved', 'pending', 'rejected']
  return {
    id,
    email: `student${id}@ewha.ac.kr`,
    name: koreanNames[index % koreanNames.length],
    role: 'student',
    status: statusOverride ?? statuses[index % statuses.length],
    department: departments[index % departments.length],
    student_id: `202600${String(id).padStart(4, '0')}`,
    bio: '',
    avatar_url: index % 3 === 0 ? `https://example.com/avatar${id}.png` : '',
  }
}

// 20명 유저 (다양한 상태: pending, approved, rejected)
export const allMockUsers: User[] = Array.from({ length: 20 }, (_, i) => makeUser(i))

// 상태별 필터
export const pendingUsers = allMockUsers.filter((u) => u.status === 'pending')
export const approvedUsers = allMockUsers.filter((u) => u.status === 'approved')
export const rejectedUsers = allMockUsers.filter((u) => u.status === 'rejected')

// 유저 페이지네이션 헬퍼
export function paginateUsers(
  users: User[],
  page: number,
  limit: number,
): PaginatedData<User> {
  const start = (page - 1) * limit
  const data = users.slice(start, start + limit)
  return {
    data,
    pagination: {
      page,
      limit,
      total: users.length,
      total_pages: Math.ceil(users.length / limit),
    },
  }
}

// ─── 특수 케이스 포스트 ───────────────────────────────────────

export const postWithLongContent: Post = makePost(100, {
  content: `# 긴 게시글 제목

이것은 매우 긴 게시글입니다. 마크다운 렌더링이 제대로 되는지 테스트합니다.

## 섹션 1
- 항목 1
- 항목 2
- 항목 3

## 섹션 2
\`\`\`javascript
const hello = 'world';
console.log(hello);
\`\`\`

## 섹션 3
> 인용문도 잘 표시되어야 합니다.

마지막 줄입니다.`,
})

export const postWithNoAuthor: Post = makePost(101, {
  author: undefined,
})

export const pinnedPost: Post = makePost(102, {
  pinned: true,
  content: '📌 중요 공지사항입니다.',
  author: { id: 1, name: '최용철', avatar_url: '', student_id: '0000000000' },
})
