import { useEffect, useState, useCallback } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { api, ApiError } from '@/lib/api'
import type { Post, Comment } from '@/types'
import { Card, CardContent } from '@/components/ui/card'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import { toast } from 'sonner'
import { ArrowLeft, Heart, MessageCircle, Loader2, Link as LinkIcon } from 'lucide-react'
import { MarkdownContent } from '@/components/MarkdownContent'

function parseTags(tags: unknown): string[] {
  if (Array.isArray(tags)) return tags as string[]
  if (typeof tags === 'string') {
    try { return JSON.parse(tags) } catch { return [] }
  }
  return []
}

function timeAgo(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString('ko-KR', { dateStyle: 'short', timeStyle: 'short' })
}

export default function PostDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [post, setPost] = useState<Post | null>(null)
  const [comments, setComments] = useState<Comment[]>([])
  const [loading, setLoading] = useState(true)
  const [notFound, setNotFound] = useState(false)

  const [commentInput, setCommentInput] = useState('')
  const [submittingComment, setSubmittingComment] = useState(false)
  const [likeBusy, setLikeBusy] = useState(false)

  const fetchPost = useCallback(async () => {
    if (!id) return
    setLoading(true)
    try {
      const p = await api.get<Post>(`/posts/${id}`)
      setPost(p)
      const cs = await api.get<Comment[]>(`/posts/${id}/comments?page=1&limit=200`).catch(() => [])
      setComments(Array.isArray(cs) ? cs : [])
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setNotFound(true)
      } else {
        toast.error(err instanceof ApiError ? err.message : '게시물을 불러오지 못했습니다.')
      }
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    fetchPost()
  }, [fetchPost])

  const handleLike = async () => {
    if (!post || likeBusy) return
    setLikeBusy(true)
    try {
      await api.post(`/posts/${post.id}/like`, {})
      setPost({
        ...post,
        is_liked: !post.is_liked,
        like_count: post.like_count + (post.is_liked ? -1 : 1),
      })
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '좋아요 실패')
    } finally {
      setLikeBusy(false)
    }
  }

  const handleAddComment = async () => {
    if (!post || !commentInput.trim() || submittingComment) return
    setSubmittingComment(true)
    try {
      const created = await api.post<Comment>(`/posts/${post.id}/comments`, {
        content: commentInput.trim(),
      })
      setComments((prev) => [...prev, created])
      setCommentInput('')
      setPost({ ...post, comment_count: post.comment_count + 1 })
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '댓글 작성 실패')
    } finally {
      setSubmittingComment(false)
    }
  }

  const handleCopyLink = async () => {
    try {
      await navigator.clipboard.writeText(window.location.href)
      toast.success('링크를 복사했어요.')
    } catch {
      toast.error('복사 실패 — 주소창 URL을 수동으로 복사해주세요.')
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (notFound || !post) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-8">
        <Button variant="ghost" size="sm" onClick={() => navigate('/feed')} className="mb-4">
          <ArrowLeft className="mr-1 h-4 w-4" /> 피드로 돌아가기
        </Button>
        <Card>
          <CardContent className="py-10 text-center text-sm text-muted-foreground">
            <p className="mb-2">찾으시는 게시물이 없어요.</p>
            <p className="text-xs">삭제되었거나 권한이 없을 수 있습니다.</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  const tags = parseTags(post.tags)
  const author = post.author

  return (
    <div className="mx-auto max-w-2xl px-4 py-4">
      <div className="mb-3 flex items-center justify-between">
        <Button variant="ghost" size="sm" onClick={() => navigate('/feed')}>
          <ArrowLeft className="mr-1 h-4 w-4" /> 피드
        </Button>
        <Button variant="ghost" size="sm" onClick={handleCopyLink}>
          <LinkIcon className="mr-1 h-3 w-3" /> 링크 복사
        </Button>
      </div>

      <Card>
        <CardContent className="p-5">
          <div className="mb-3 flex items-center gap-2">
            {author && (
              <Link to={`/profile/${author.id}`} className="flex items-center gap-2">
                <Avatar className="h-8 w-8">
                  {author.avatar_url ? <AvatarImage src={author.avatar_url} /> : null}
                  <AvatarFallback>{author.name?.[0] ?? '?'}</AvatarFallback>
                </Avatar>
                <div className="text-sm">
                  <div className="font-medium">{author.name}</div>
                  <div className="text-xs text-muted-foreground">
                    {post.channel?.name ? `#${post.channel.name} · ` : ''}
                    {timeAgo(post.created_at)}
                  </div>
                </div>
              </Link>
            )}
          </div>

          <MarkdownContent content={post.content} />

          {tags.length > 0 && (
            <div className="mt-3 flex flex-wrap gap-1">
              {tags.map((t) => (
                <Badge key={t} variant="secondary" className="text-xs">#{t}</Badge>
              ))}
            </div>
          )}

          <div className="mt-4 flex items-center gap-3 border-t pt-3 text-sm">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleLike}
              disabled={likeBusy}
              className={post.is_liked ? 'text-coral' : ''}
            >
              <Heart className={`mr-1 h-4 w-4 ${post.is_liked ? 'fill-current' : ''}`} />
              {post.like_count}
            </Button>
            <span className="flex items-center text-muted-foreground">
              <MessageCircle className="mr-1 h-4 w-4" />
              {post.comment_count}
            </span>
          </div>
        </CardContent>
      </Card>

      <div className="mt-4 space-y-2">
        <h3 className="px-1 text-sm font-medium text-muted-foreground">댓글 {comments.length}개</h3>
        {comments.length === 0 ? (
          <p className="px-2 py-6 text-center text-xs text-muted-foreground">아직 댓글이 없어요. 첫 댓글을 남겨보세요.</p>
        ) : (
          comments.map((c) => (
            <div key={c.id} className="rounded-md border bg-card p-3 text-sm">
              <div className="mb-1 flex items-center gap-2">
                {c.author && (
                  <>
                    <Avatar className="h-6 w-6">
                      {c.author.avatar_url ? <AvatarImage src={c.author.avatar_url} /> : null}
                      <AvatarFallback>{c.author.name?.[0] ?? '?'}</AvatarFallback>
                    </Avatar>
                    <span className="text-xs font-medium">{c.author.name}</span>
                  </>
                )}
                <span className="ml-auto text-[10px] text-muted-foreground">{timeAgo(c.created_at)}</span>
              </div>
              <p className="whitespace-pre-wrap text-sm">{c.content}</p>
            </div>
          ))
        )}

        <div className="mt-3 rounded-md border p-3">
          <Textarea
            value={commentInput}
            onChange={(e) => setCommentInput(e.target.value)}
            placeholder="댓글 작성..."
            rows={2}
            className="resize-none border-0 p-0 shadow-none focus-visible:ring-0"
          />
          <div className="mt-2 flex justify-end">
            <Button size="sm" disabled={submittingComment || !commentInput.trim()} onClick={handleAddComment}>
              {submittingComment && <Loader2 className="mr-1 h-3 w-3 animate-spin" />}
              등록
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
