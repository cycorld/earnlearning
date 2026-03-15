import { useState } from 'react'
import { api } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { ArrowLeft, Send, Megaphone } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useToast } from '@/hooks/use-toast'

export default function AdminAnnouncePage() {
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [sending, setSending] = useState(false)
  const { toast } = useToast()

  const handleSend = async () => {
    if (!title.trim() || !body.trim()) {
      toast({ title: '제목과 내용을 모두 입력해주세요', variant: 'destructive' })
      return
    }

    if (!confirm('전체 유저에게 공지 알림을 보내시겠습니까?')) return

    setSending(true)
    try {
      const result = await api.post<{ message: string; sent: number }>(
        '/admin/notifications/announce',
        { title: title.trim(), body: body.trim() },
      )
      toast({ title: `${result.sent}명에게 공지 알림을 보냈습니다` })
      setTitle('')
      setBody('')
    } catch (e: any) {
      toast({ title: e.message || '전송 실패', variant: 'destructive' })
    } finally {
      setSending(false)
    }
  }

  return (
    <div className="mx-auto max-w-lg space-y-4 p-4">
      <div className="flex items-center gap-2">
        <Link to="/admin" className="rounded-full p-1 hover:bg-accent">
          <ArrowLeft className="h-5 w-5" />
        </Link>
        <Megaphone className="h-5 w-5 text-primary" />
        <h1 className="text-xl font-bold">공지 알림</h1>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">전체 공지 보내기</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">제목</label>
            <Input
              placeholder="공지 제목을 입력하세요"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">내용</label>
            <Textarea
              placeholder="공지 내용을 입력하세요"
              value={body}
              onChange={(e) => setBody(e.target.value)}
              rows={4}
            />
          </div>
          <Button
            onClick={handleSend}
            disabled={sending || !title.trim() || !body.trim()}
            className="w-full"
          >
            <Send className="mr-2 h-4 w-4" />
            {sending ? '전송 중...' : '전체 유저에게 공지 보내기'}
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
