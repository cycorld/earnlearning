import { useState, useEffect, useCallback } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import type { Post, Channel, Comment, PaginatedData } from '@/types'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  Heart,
  MessageCircle,
  Pin,
  Plus,
  Send,
  Loader2,
  ChevronDown,
  ChevronUp,
  LogIn,
  X,
  Hash,
  Pencil,
  Trash2,
  User,
  MessageSquare,
} from 'lucide-react'
import { toast } from 'sonner'
import { MarkdownEditor } from '@/components/MarkdownEditor'
import { MarkdownContent } from '@/components/MarkdownContent'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { displayName } from '@/lib/utils'

interface Classroom {
  id: number
  name: string
  invite_code: string
}

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return '방금'
  if (mins < 60) return `${mins}분 전`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}시간 전`
  const days = Math.floor(hours / 24)
  return `${days}일 전`
}

export default function FeedPage() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [classrooms, setClassrooms] = useState<Classroom[]>([])
  const [selectedClassroom, setSelectedClassroom] = useState<number | null>(null)
  const [channels, setChannels] = useState<Channel[]>([])
  const [activeChannel, setActiveChannel] = useState('all')
  const [posts, setPosts] = useState<Post[]>([])
  const [loading, setLoading] = useState(true)
  const [classroomLoading, setClassroomLoading] = useState(true)
  const [currentPage, setCurrentPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)

  // Join classroom
  const [inviteCode, setInviteCode] = useState('')
  const [joining, setJoining] = useState(false)

  // Tag filter
  const [activeTag, setActiveTag] = useState('')

  // Create post
  const [newPostOpen, setNewPostOpen] = useState(false)
  const [newPostContent, setNewPostContent] = useState('')
  const [newPostTags, setNewPostTags] = useState('')
  const [postChannelId, setPostChannelId] = useState<number | null>(null)
  const [creating, setCreating] = useState(false)

  // Edit post
  const [editPostId, setEditPostId] = useState<number | null>(null)
  const [editPostContent, setEditPostContent] = useState('')
  const [editPostTags, setEditPostTags] = useState('')
  const [editPostOpen, setEditPostOpen] = useState(false)
  const [editing, setEditing] = useState(false)

  // Delete post
  const [deletePostId, setDeletePostId] = useState<number | null>(null)
  const [deleteConfirmText, setDeleteConfirmText] = useState('')
  const [deleting, setDeleting] = useState(false)

  // Comments
  const [expandedPost, setExpandedPost] = useState<number | null>(null)
  const [comments, setComments] = useState<Record<number, Comment[]>>({})
  const [commentInput, setCommentInput] = useState<Record<number, string>>({})
  const [commentLoading, setCommentLoading] = useState<Record<number, boolean>>({})

  // Load classrooms
  const fetchClassrooms = useCallback(async (showLoading = true) => {
    if (showLoading) setClassroomLoading(true)
    try {
      const data = await api.get<Classroom[]>('/classrooms')
      const list = data ?? []
      setClassrooms(list)
      if (list.length > 0) {
        setSelectedClassroom((prev) => prev ?? list[0].id)
      }
    } catch {
      setClassrooms([])
    } finally {
      if (showLoading) setClassroomLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchClassrooms()
  }, [fetchClassrooms, location.key])

  // Refetch classrooms when the page becomes visible (e.g. after external join)
  useEffect(() => {
    const handleVisibility = () => {
      if (document.visibilityState === 'visible') {
        fetchClassrooms(false)
      }
    }
    const handleFocus = () => fetchClassrooms(false)
    document.addEventListener('visibilitychange', handleVisibility)
    window.addEventListener('focus', handleFocus)
    return () => {
      document.removeEventListener('visibilitychange', handleVisibility)
      window.removeEventListener('focus', handleFocus)
    }
  }, [fetchClassrooms])

  // Load channels when classroom changes
  useEffect(() => {
    if (!selectedClassroom) {
      setChannels([])
      return
    }
    api
      .get<Channel[]>(`/classrooms/${selectedClassroom}/channels`)
      .then((data) => {
        setChannels(data)
        setActiveChannel('all')
      })
      .catch(() => setChannels([]))
  }, [selectedClassroom])

  // Load posts
  const fetchPosts = useCallback(async (page = 1) => {
    if (!selectedClassroom) {
      setPosts([])
      setLoading(false)
      return
    }
    setLoading(true)
    try {
      const channelParam =
        activeChannel !== 'all' ? `&channel_id=${activeChannel}` : ''
      const tagParam = activeTag ? `&tag=${encodeURIComponent(activeTag)}` : ''
      const data = await api.get<PaginatedData<Post>>(
        `/posts?classroom_id=${selectedClassroom}${channelParam}${tagParam}&page=${page}&limit=20`,
      )
      setPosts(data.data)
      setCurrentPage(data.pagination.page)
      setTotalPages(data.pagination.total_pages)
    } catch {
      setPosts([])
    } finally {
      setLoading(false)
    }
  }, [selectedClassroom, activeChannel, activeTag])

  useEffect(() => {
    fetchPosts()
  }, [fetchPosts])

  // Like toggle
  const handleLike = async (postId: number, _isLiked: boolean) => {
    try {
      const result = await api.post<{ liked: boolean; reward: number }>(
        `/posts/${postId}/like`,
      )
      setPosts((prev) =>
        prev.map((p) =>
          p.id === postId
            ? {
                ...p,
                is_liked: result.liked,
                like_count: p.like_count + (result.liked ? 1 : -1),
              }
            : p,
        ),
      )
      if (result.reward > 0) {
        toast.success(`좋아요 보상 +${result.reward}원이 글쓴이에게 지급되었습니다`)
      }
    } catch {
      // ignore
    }
  }

  // Join classroom
  const handleJoinClassroom = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!inviteCode.trim()) return
    setJoining(true)
    try {
      await api.post('/classrooms/join', { code: inviteCode.trim() })
      toast.success('클래스룸에 참여했습니다!')
      setInviteCode('')
      await fetchClassrooms(false)
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : '참여에 실패했습니다.'
      toast.error(message)
    } finally {
      setJoining(false)
    }
  }

  // Create post
  const handleCreatePost = async () => {
    if (!newPostContent.trim() || !postChannelId) return
    setCreating(true)
    try {
      const tags = newPostTags
        .split(',')
        .map((t) => t.trim().replace(/^#/, ''))
        .filter(Boolean)
      await api.post(`/channels/${postChannelId}/posts`, {
        content: newPostContent.trim(),
        post_type: 'normal',
        tags: JSON.stringify(tags),
      })
      toast.success('게시물이 작성되었습니다.')
      setNewPostContent('')
      setNewPostTags('')
      setPostChannelId(null)
      setNewPostOpen(false)
      fetchPosts()
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : '게시물 작성에 실패했습니다.'
      toast.error(message)
    } finally {
      setCreating(false)
    }
  }

  // Open edit dialog
  const openEditPost = (post: Post) => {
    setEditPostId(post.id)
    setEditPostContent(post.content)
    let tags: string[] = []
    if (Array.isArray(post.tags)) {
      tags = post.tags
    } else if (typeof post.tags === 'string') {
      try { tags = JSON.parse(post.tags) } catch { tags = [] }
    }
    setEditPostTags(tags.join(', '))
    setEditPostOpen(true)
  }

  // Update post
  const handleUpdatePost = async () => {
    if (!editPostContent.trim() || !editPostId) return
    setEditing(true)
    try {
      const tags = editPostTags
        .split(',')
        .map((t) => t.trim().replace(/^#/, ''))
        .filter(Boolean)
      const updated = await api.put<Post>(`/posts/${editPostId}`, {
        content: editPostContent.trim(),
        tags: JSON.stringify(tags),
      })
      setPosts((prev) =>
        prev.map((p) => (p.id === editPostId ? { ...p, content: updated.content, tags: updated.tags } : p)),
      )
      toast.success('게시물이 수정되었습니다.')
      setEditPostOpen(false)
      setEditPostId(null)
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : '게시물 수정에 실패했습니다.'
      toast.error(message)
    } finally {
      setEditing(false)
    }
  }

  // Delete post
  const handleDeletePost = async () => {
    if (!deletePostId || deleteConfirmText !== '삭제') return
    setDeleting(true)
    try {
      await api.del(`/posts/${deletePostId}`)
      setPosts((prev) => prev.filter((p) => p.id !== deletePostId))
      toast.success('게시물이 삭제되었습니다.')
      setDeletePostId(null)
      setDeleteConfirmText('')
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : '삭제에 실패했습니다.')
    } finally {
      setDeleting(false)
    }
  }

  // Load comments
  const toggleComments = async (postId: number) => {
    if (expandedPost === postId) {
      setExpandedPost(null)
      return
    }
    setExpandedPost(postId)
    if (!comments[postId]) {
      setCommentLoading((prev) => ({ ...prev, [postId]: true }))
      try {
        const data = await api.get<PaginatedData<Comment>>(
          `/posts/${postId}/comments?page=1&limit=50`,
        )
        setComments((prev) => ({ ...prev, [postId]: data.data }))
      } catch {
        setComments((prev) => ({ ...prev, [postId]: [] }))
      } finally {
        setCommentLoading((prev) => ({ ...prev, [postId]: false }))
      }
    }
  }

  // Create comment
  const handleCreateComment = async (postId: number) => {
    const content = commentInput[postId]?.trim()
    if (!content) return
    try {
      const newComment = await api.post<Comment>(`/posts/${postId}/comments`, {
        content,
      })
      setComments((prev) => ({
        ...prev,
        [postId]: [...(prev[postId] || []), newComment],
      }))
      setCommentInput((prev) => ({ ...prev, [postId]: '' }))
      // Check if comment is on someone else's post → reward toast
      const post = posts.find((p) => p.id === postId)
      if (post && post.author?.id !== user?.id) {
        toast.success('댓글 보상 +100원이 글쓴이에게 지급되었습니다')
      }
      setPosts((prev) =>
        prev.map((p) =>
          p.id === postId ? { ...p, comment_count: p.comment_count + 1 } : p,
        ),
      )
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : '댓글 작성에 실패했습니다.'
      toast.error(message)
    }
  }

  // Delete comment
  const handleDeleteComment = async (postId: number, commentId: number) => {
    try {
      await api.del(`/posts/${postId}/comments/${commentId}`)
      setComments((prev) => ({
        ...prev,
        [postId]: (prev[postId] || []).filter((c) => c.id !== commentId),
      }))
      setPosts((prev) =>
        prev.map((p) =>
          p.id === postId
            ? { ...p, comment_count: Math.max(0, p.comment_count - 1) }
            : p,
        ),
      )
      toast.success('댓글이 삭제되었습니다')
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : '댓글 삭제에 실패했습니다.'
      toast.error(message)
    }
  }

  if (classroomLoading) {
    return (
      <div className="flex justify-center py-8">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  // No classrooms: show join form
  if (classrooms.length === 0) {
    return (
      <div className="mx-auto max-w-lg p-4">
        <Card>
          <CardContent className="p-6 text-center">
            <LogIn className="mx-auto mb-4 h-12 w-12 text-muted-foreground" />
            <h2 className="text-lg font-semibold">클래스룸 참여</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              초대 코드를 입력하여 클래스룸에 참여하세요.
            </p>
            <form
              onSubmit={handleJoinClassroom}
              className="mt-4 flex gap-2"
            >
              <Input
                placeholder="초대 코드"
                value={inviteCode}
                onChange={(e) => setInviteCode(e.target.value)}
                className="flex-1"
              />
              <Button type="submit" disabled={joining || !inviteCode.trim()}>
                {joining && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                참여
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
      {/* Classroom selector (if multiple) */}
      {classrooms.length > 1 && (
        <select
          value={selectedClassroom ?? ''}
          onChange={(e) => setSelectedClassroom(Number(e.target.value))}
          className="w-full rounded-md border bg-background px-3 py-2 text-sm"
        >
          {classrooms.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
      )}

      {/* Channel tabs */}
      {channels.length > 0 && (
        <Tabs value={activeChannel} onValueChange={setActiveChannel}>
          <TabsList className="w-full justify-start overflow-x-auto">
            <TabsTrigger value="all">전체</TabsTrigger>
            {channels.map((ch) => (
              <TabsTrigger key={ch.id} value={String(ch.id)}>
                {ch.name}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      )}

      {/* Create post button */}
      <Dialog open={newPostOpen} onOpenChange={setNewPostOpen}>
        <DialogTrigger asChild>
          <Button className="w-full gap-2">
            <Plus className="h-4 w-4" />
            새 게시물 작성
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>새 게시물</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>채널</Label>
              <select
                value={postChannelId ?? ''}
                onChange={(e) => setPostChannelId(Number(e.target.value) || null)}
                className="w-full rounded-md border bg-background px-3 py-2 text-sm"
              >
                <option value="">채널을 선택하세요</option>
                {channels
                  .filter(
                    (ch) =>
                      ch.write_role === 'all' ||
                      (ch.write_role === 'admin' && user?.role === 'admin'),
                  )
                  .map((ch) => (
                    <option key={ch.id} value={ch.id}>
                      {ch.name}
                    </option>
                  ))}
              </select>
            </div>
            <div className="space-y-2">
              <Label>내용</Label>
              <MarkdownEditor
                value={newPostContent}
                onChange={setNewPostContent}
                placeholder="마크다운으로 작성하세요... (이미지 붙여넣기 가능)"
                rows={6}
              />
            </div>
            <div className="space-y-2">
              <Label className="flex items-center gap-1">
                <Hash className="h-3.5 w-3.5" />
                태그 (쉼표로 구분)
              </Label>
              <Input
                placeholder="예: 공지, 중요, 프로젝트"
                value={newPostTags}
                onChange={(e) => setNewPostTags(e.target.value)}
              />
              {newPostTags.trim() && (
                <div className="flex flex-wrap gap-1">
                  {newPostTags
                    .split(',')
                    .map((t) => t.trim().replace(/^#/, ''))
                    .filter(Boolean)
                    .map((tag) => (
                      <Badge key={tag} variant="outline" className="text-xs">
                        #{tag}
                      </Badge>
                    ))}
                </div>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button
              onClick={handleCreatePost}
              disabled={creating || !newPostContent.trim() || !postChannelId}
            >
              {creating && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              게시
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit post dialog */}
      <Dialog open={editPostOpen} onOpenChange={setEditPostOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>게시물 수정</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>내용</Label>
              <MarkdownEditor
                value={editPostContent}
                onChange={setEditPostContent}
                placeholder="마크다운으로 작성하세요... (이미지 붙여넣기 가능)"
                rows={6}
              />
            </div>
            <div className="space-y-2">
              <Label className="flex items-center gap-1">
                <Hash className="h-3.5 w-3.5" />
                태그 (쉼표로 구분)
              </Label>
              <Input
                placeholder="예: 공지, 중요, 프로젝트"
                value={editPostTags}
                onChange={(e) => setEditPostTags(e.target.value)}
              />
              {editPostTags.trim() && (
                <div className="flex flex-wrap gap-1">
                  {editPostTags
                    .split(',')
                    .map((t) => t.trim().replace(/^#/, ''))
                    .filter(Boolean)
                    .map((tag) => (
                      <Badge key={tag} variant="outline" className="text-xs">
                        #{tag}
                      </Badge>
                    ))}
                </div>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setEditPostOpen(false)}
            >
              취소
            </Button>
            <Button
              onClick={handleUpdatePost}
              disabled={editing || !editPostContent.trim()}
            >
              {editing && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              수정
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Active tag filter */}
      {activeTag && (
        <div className="flex items-center gap-2 rounded-lg border bg-primary/5 px-3 py-2">
          <Hash className="h-4 w-4 text-primary" />
          <span className="text-sm">
            <span className="font-medium text-primary">#{activeTag}</span> 태그로 필터링 중
          </span>
          <Button
            variant="ghost"
            size="icon"
            className="ml-auto h-6 w-6"
            onClick={() => setActiveTag('')}
          >
            <X className="h-3.5 w-3.5" />
          </Button>
        </div>
      )}

      {/* Posts list */}
      {loading ? (
        <div className="flex justify-center py-8">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        </div>
      ) : posts.length === 0 ? (
        <div className="py-12 text-center text-muted-foreground">
          아직 게시물이 없습니다.
        </div>
      ) : (
        <div className="space-y-3">
          {posts.map((post) => (
            <Card key={post.id} className="transition-colors hover:bg-accent/30">
              <CardContent className="p-4">
                <div className="flex items-start gap-3">
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild disabled={!post.author?.id}>
                      <button className="shrink-0">
                        <Avatar className="h-9 w-9">
                          <AvatarImage src={post.author?.avatar_url} />
                          <AvatarFallback>
                            {post.author?.name?.charAt(0) || '?'}
                          </AvatarFallback>
                        </Avatar>
                      </button>
                    </DropdownMenuTrigger>
                    {post.author?.id && (
                      <DropdownMenuContent align="start">
                        <DropdownMenuItem onClick={() => navigate(`/profile/${post.author!.id}`)}>
                          <User className="mr-2 h-4 w-4" />
                          프로필 보기
                        </DropdownMenuItem>
                        {post.author.id !== user?.id && (
                          <DropdownMenuItem onClick={() => navigate(`/messages/${post.author!.id}`)}>
                            <MessageSquare className="mr-2 h-4 w-4" />
                            메시지 보내기
                          </DropdownMenuItem>
                        )}
                      </DropdownMenuContent>
                    )}
                  </DropdownMenu>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">
                        {displayName(post.author)}
                      </span>
                      {activeChannel === 'all' && post.channel && (
                        <Badge variant="secondary" className="text-xs">
                          {post.channel.name}
                        </Badge>
                      )}
                      {post.pinned && <Pin className="h-3 w-3 text-primary" />}
                      <span className="ml-auto flex items-center gap-1 text-xs text-muted-foreground">
                        <Link
                          to={`/post/${post.id}`}
                          className="hover:underline"
                          title="이 글의 고유 링크"
                        >
                          {timeAgo(post.created_at)}
                        </Link>
                        {(post.author?.id === user?.id || user?.role === 'admin') && (
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <button className="ml-1 rounded p-0.5 hover:bg-muted">
                                <Pencil className="h-3 w-3" />
                              </button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem onClick={() => openEditPost(post)}>
                                <Pencil className="mr-2 h-3.5 w-3.5" />
                                수정
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                onClick={() => { setDeletePostId(post.id); setDeleteConfirmText('') }}
                                className="text-destructive focus:text-destructive"
                              >
                                <Trash2 className="mr-2 h-3.5 w-3.5" />
                                삭제
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        )}
                      </span>
                    </div>
                    <MarkdownContent
                      content={post.content}
                      maxLines={6}
                      className="mt-1 text-sm"
                    />
                    {(() => {
                      let tags: string[] = [];
                      if (Array.isArray(post.tags)) {
                        tags = post.tags;
                      } else if (typeof post.tags === 'string') {
                        try { tags = JSON.parse(post.tags); } catch { tags = (post.tags as string).split(',').filter(Boolean); }
                      }
                      return tags.length > 0 ? (
                        <div className="mt-2 flex flex-wrap gap-1">
                          {tags.map((tag) => (
                            <Badge
                              key={tag}
                              variant={activeTag === tag ? 'default' : 'outline'}
                              className="cursor-pointer text-xs transition-colors hover:bg-primary/10"
                              onClick={() => setActiveTag(activeTag === tag ? '' : tag)}
                            >
                              #{tag}
                            </Badge>
                          ))}
                        </div>
                      ) : null;
                    })()}
                    <div className="mt-2 flex items-center gap-4 text-xs text-muted-foreground">
                      <button
                        onClick={() => handleLike(post.id, post.is_liked)}
                        className={`flex items-center gap-1 ${post.is_liked ? 'text-red-500' : ''}`}
                      >
                        <Heart
                          className={`h-4 w-4 ${post.is_liked ? 'fill-current' : ''}`}
                        />
                        {post.like_count}
                      </button>
                      <button
                        onClick={() => toggleComments(post.id)}
                        className="flex items-center gap-1"
                      >
                        <MessageCircle className="h-4 w-4" />
                        {post.comment_count}
                        {expandedPost === post.id ? (
                          <ChevronUp className="h-3 w-3" />
                        ) : (
                          <ChevronDown className="h-3 w-3" />
                        )}
                      </button>
                    </div>

                    {/* Comments section */}
                    {expandedPost === post.id && (
                      <div className="mt-3 space-y-2 border-t pt-3">
                        {commentLoading[post.id] ? (
                          <div className="flex justify-center py-2">
                            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                          </div>
                        ) : (
                          <>
                            {(comments[post.id] || []).map((c) => (
                              <div key={c.id} className="flex items-start gap-2">
                                <DropdownMenu>
                                  <DropdownMenuTrigger asChild disabled={!c.author?.id}>
                                    <button className="shrink-0">
                                      <Avatar className="h-6 w-6">
                                        <AvatarImage src={c.author?.avatar_url} />
                                        <AvatarFallback className="text-xs">
                                          {c.author?.name?.charAt(0) || '?'}
                                        </AvatarFallback>
                                      </Avatar>
                                    </button>
                                  </DropdownMenuTrigger>
                                  {c.author?.id && (
                                    <DropdownMenuContent align="start">
                                      <DropdownMenuItem onClick={() => navigate(`/profile/${c.author!.id}`)}>
                                        <User className="mr-2 h-4 w-4" />
                                        프로필 보기
                                      </DropdownMenuItem>
                                      {c.author.id !== user?.id && (
                                        <DropdownMenuItem onClick={() => navigate(`/messages/${c.author!.id}`)}>
                                          <MessageSquare className="mr-2 h-4 w-4" />
                                          메시지 보내기
                                        </DropdownMenuItem>
                                      )}
                                    </DropdownMenuContent>
                                  )}
                                </DropdownMenu>
                                <div className="min-w-0 flex-1">
                                  <div className="flex items-center gap-1">
                                    <span className="text-xs font-medium">
                                      {displayName(c.author)}
                                    </span>
                                    <span className="text-xs text-muted-foreground">
                                      {timeAgo(c.created_at)}
                                    </span>
                                    {(c.author?.id === user?.id || user?.role === 'admin') && (
                                      <button
                                        onClick={() => handleDeleteComment(post.id, c.id)}
                                        className="ml-auto rounded p-0.5 text-muted-foreground hover:bg-muted hover:text-destructive"
                                        title="댓글 삭제"
                                      >
                                        <Trash2 className="h-3 w-3" />
                                      </button>
                                    )}
                                  </div>
                                  <MarkdownContent
                                    content={c.content}
                                    maxLines={4}
                                    className="text-xs"
                                  />
                                </div>
                              </div>
                            ))}
                            {(comments[post.id] || []).length === 0 && (
                              <p className="text-center text-xs text-muted-foreground">
                                아직 댓글이 없습니다.
                              </p>
                            )}
                          </>
                        )}
                        <div className="space-y-1">
                          <MarkdownEditor
                            value={commentInput[post.id] || ''}
                            onChange={(v) =>
                              setCommentInput((prev) => ({
                                ...prev,
                                [post.id]: v,
                              }))
                            }
                            placeholder="댓글을 입력하세요 (마크다운 지원)"
                            rows={2}
                            compact
                          />
                          <div className="flex justify-end">
                            <Button
                              size="sm"
                              className="h-7 gap-1 px-3 text-xs"
                              onClick={() => handleCreateComment(post.id)}
                              disabled={!commentInput[post.id]?.trim()}
                            >
                              <Send className="h-3 w-3" />
                              댓글
                            </Button>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 py-4">
              <Button
                variant="outline"
                size="sm"
                disabled={currentPage <= 1}
                onClick={() => fetchPosts(currentPage - 1)}
              >
                이전
              </Button>
              <span className="text-sm text-muted-foreground">
                {currentPage} / {totalPages}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={currentPage >= totalPages}
                onClick={() => fetchPosts(currentPage + 1)}
              >
                다음
              </Button>
            </div>
          )}
        </div>
      )}

      {/* Delete confirmation dialog */}
      <Dialog open={deletePostId !== null} onOpenChange={(open) => { if (!open) { setDeletePostId(null); setDeleteConfirmText('') } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>게시글 삭제</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              이 게시글을 정말 삭제하시겠습니까? 댓글과 좋아요도 함께 삭제됩니다.
            </p>
            <p className="text-sm font-medium">
              확인을 위해 아래에 <span className="text-destructive">"삭제"</span>를 입력하세요.
            </p>
            <Input
              value={deleteConfirmText}
              onChange={(e) => setDeleteConfirmText(e.target.value)}
              placeholder="삭제"
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setDeletePostId(null); setDeleteConfirmText('') }}>
              취소
            </Button>
            <Button
              variant="destructive"
              disabled={deleteConfirmText !== '삭제' || deleting}
              onClick={handleDeletePost}
            >
              {deleting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              삭제
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
