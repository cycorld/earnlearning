import { useState } from 'react'
import { api } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { ArrowLeft, Send, Megaphone } from 'lucide-react'
import { Link } from 'react-router-dom'

export default function AdminAnnouncePage() {
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [sendNotify, setSendNotify] = useState(true)
  const [sending, setSending] = useState(false)
  const [result, setResult] = useState<string | null>(null)

  const handleSend = async () => {
    if (!title.trim() || !body.trim()) {
      alert('제목과 내용을 모두 입력해주세요')
      return
    }

    const notifyText = sendNotify
      ? '푸시 + 이메일 알림과 함께'
      : '앱 내 알림만'
    if (!confirm(`전체 유저에게 공지를 보내시겠습니까?\n(${notifyText} 전송)`)) return

    setSending(true)
    setResult(null)
    try {
      const res = await api.post<{ message: string; sent: number }>(
        '/admin/notifications/announce',
        { title: title.trim(), body: body.trim(), send_notify: sendNotify },
      )
      setResult(`${res.sent}명에게 공지 알림을 보냈습니다`)
      setTitle('')
      setBody('')
    } catch (e: any) {
      alert(e.message || '전송 실패')
    } finally {
      setSending(false)
    }
  }

  return (
    <div className="mx-auto max-w-lg space-y-5 p-4">
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
          <label className="flex items-center gap-2 rounded-md border px-3 py-2.5 cursor-pointer hover:bg-accent/50 transition-colors">
            <input
              type="checkbox"
              checked={sendNotify}
              onChange={(e) => setSendNotify(e.target.checked)}
              className="h-4 w-4 rounded border-gray-300 accent-primary"
            />
            <div className="flex-1">
              <span className="text-sm font-medium">푸시 + 이메일 알림 보내기</span>
              <p className="text-xs text-muted-foreground">
                {sendNotify
                  ? '앱 내 알림, 푸시 알림, 이메일이 모두 전송됩니다'
                  : '앱 내 알림만 생성됩니다 (푸시/이메일 미전송)'}
              </p>
            </div>
          </label>
          <Button
            onClick={handleSend}
            disabled={sending || !title.trim() || !body.trim()}
            className="w-full"
          >
            <Send className="mr-2 h-4 w-4" />
            {sending ? '전송 중...' : '전체 유저에게 공지 보내기'}
          </Button>
          {result && (
            <p className="text-center text-sm text-success font-medium">{result}</p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
