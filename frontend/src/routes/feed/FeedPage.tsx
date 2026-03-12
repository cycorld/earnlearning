import { useState, useEffect, useCallback } from 'react'
import { api } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import type { Post, Channel, Comment, PaginatedData } from '@/types'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
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
} from 'lucide-react'
import { toast } from 'sonner'

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
  const [classrooms, setClassrooms] = useState<Classroom[]>([])
  const [selectedClassroom, setSelectedClassroom] = useState<number | null>(null)
  const [channels, setChannels] = useState<Channel[]>([])
  const [activeChannel, setActiveChannel] = useState('all')
  const [posts, setPosts] = useState<Post[]>([])
  const [loading, setLoading] = useState(true)
  const [classroomLoading, setClassroomLoading] = useState(true)

  // Join classroom
  const [inviteCode, setInviteCode] = useState('')
  const [joining, setJoining] = useState(false)

  // Create post
  const [newPostOpen, setNewPostOpen] = useState(false)
  const [newPostContent, setNewPostContent] = useState('')
  const [postChannelId, setPostChannelId] = useState<number | null>(null)
  const [creating, setCreating] = useState(false)

  // Comments
  const [expandedPost, setExpandedPost] = useState<number | null>(null)
  const [comments, setComments] = useState<Record<number, Comment[]>>({})
  const [commentInput, setCommentInput] = useState<Record<number, string>>({})
  const [commentLoading, setCommentLoading] = useState<Record<number, boolean>>({})

  // Load classrooms
  useEffect(() => {
    setClassroomLoading(true)
    api
      .get<Classroom[]>('/classrooms')
      .then((data) => {
        setClassrooms(data)
        if (data.length > 0) {
          setSelectedClassroom(data[0].id)
        }
      })
      .catch(() => {
        setClassrooms([])
      })
      .finally(() => setClassroomLoading(false))
  }, [])

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
  const fetchPosts = useCallback(async () => {
    if (!selectedClassroom) {
      setPosts([])
      setLoading(false)
      return
    }
    setLoading(true)
    try {
      const channelParam =
        activeChannel !== 'all' ? `&channel_id=${activeChannel}` : ''
      const data = await api.get<PaginatedData<Post>>(
        `/posts?classroom_id=${selectedClassroom}${channelParam}&page=1&limit=20`,
      )
      setPosts(data.data)
    } catch {
      setPosts([])
    } finally {
      setLoading(false)
    }
  }, [selectedClassroom, activeChannel])

  useEffect(() => {
    fetchPosts()
  }, [fetchPosts])

  // Like toggle
  const handleLike = async (postId: number, isLiked: boolean) => {
    try {
      if (isLiked) {
        await api.del(`/posts/${postId}/like`)
      } else {
        await api.post(`/posts/${postId}/like`)
      }
      setPosts((prev) =>
        prev.map((p) =>
          p.id === postId
            ? {
                ...p,
                is_liked: !isLiked,
                like_count: p.like_count + (isLiked ? -1 : 1),
              }
            : p,
        ),
      )
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
      const data = await api.get<Classroom[]>('/classrooms')
      setClassrooms(data)
      if (data.length > 0 && !selectedClassroom) {
        setSelectedClassroom(data[0].id)
      }
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
      await api.post(`/channels/${postChannelId}/posts`, {
        content: newPostContent.trim(),
        post_type: 'normal',
      })
      toast.success('게시물이 작성되었습니다.')
      setNewPostContent('')
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
    <div className="mx-auto max-w-lg space-y-4 p-4">
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
          <Button className="w-full gap-2" variant="outline">
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
              <Textarea
                placeholder="내용을 입력하세요"
                value={newPostContent}
                onChange={(e) => setNewPostContent(e.target.value)}
                rows={5}
              />
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
                  <Avatar className="h-9 w-9 shrink-0">
                    <AvatarImage src={post.author?.avatar_url} />
                    <AvatarFallback>
                      {post.author?.name?.charAt(0) || '?'}
                    </AvatarFallback>
                  </Avatar>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">
                        {post.author?.name}
                      </span>
                      {post.channel && (
                        <Badge variant="secondary" className="text-xs">
                          {post.channel.name}
                        </Badge>
                      )}
                      {post.pinned && <Pin className="h-3 w-3 text-primary" />}
                      <span className="ml-auto text-xs text-muted-foreground">
                        {timeAgo(post.created_at)}
                      </span>
                    </div>
                    <p className="mt-1 whitespace-pre-wrap text-sm">
                      {post.content}
                    </p>
                    {post.tags.length > 0 && (
                      <div className="mt-2 flex flex-wrap gap-1">
                        {post.tags.map((tag) => (
                          <Badge key={tag} variant="outline" className="text-xs">
                            #{tag}
                          </Badge>
                        ))}
                      </div>
                    )}
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
                              <div key={c.id} className="flex gap-2">
                                <Avatar className="h-6 w-6 shrink-0">
                                  <AvatarImage src={c.author?.avatar_url} />
                                  <AvatarFallback className="text-xs">
                                    {c.author?.name?.charAt(0) || '?'}
                                  </AvatarFallback>
                                </Avatar>
                                <div className="min-w-0">
                                  <div className="flex items-center gap-1">
                                    <span className="text-xs font-medium">
                                      {c.author?.name}
                                    </span>
                                    <span className="text-xs text-muted-foreground">
                                      {timeAgo(c.created_at)}
                                    </span>
                                  </div>
                                  <p className="text-xs">{c.content}</p>
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
                        <div className="flex gap-2">
                          <Input
                            placeholder="댓글을 입력하세요"
                            value={commentInput[post.id] || ''}
                            onChange={(e) =>
                              setCommentInput((prev) => ({
                                ...prev,
                                [post.id]: e.target.value,
                              }))
                            }
                            onKeyDown={(e) => {
                              if (e.key === 'Enter' && !e.shiftKey) {
                                e.preventDefault()
                                handleCreateComment(post.id)
                              }
                            }}
                            className="h-8 text-xs"
                          />
                          <Button
                            size="sm"
                            variant="ghost"
                            className="h-8 w-8 p-0"
                            onClick={() => handleCreateComment(post.id)}
                            disabled={!commentInput[post.id]?.trim()}
                          >
                            <Send className="h-4 w-4" />
                          </Button>
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
