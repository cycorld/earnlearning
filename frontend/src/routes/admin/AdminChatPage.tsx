import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowLeft, RefreshCw, Sparkles } from 'lucide-react'
import { toast } from 'sonner'

import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'

interface Skill {
  id: number
  slug: string
  name: string
  description: string
  default_model: string
  default_reasoning_effort?: string
  tools_allowed?: string[]
  wiki_scope?: string[]
  enabled: boolean
  admin_only: boolean
  updated_at: string
}

interface WikiMeta {
  slug: string
  path: string
  title: string
  notion_page_id?: string
  synced_at?: string
  updated_at: string
}

export default function AdminChatPage() {
  const [skills, setSkills] = useState<Skill[]>([])
  const [wikiDocs, setWikiDocs] = useState<WikiMeta[]>([])
  const [loading, setLoading] = useState(true)
  const [reindexing, setReindexing] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [s, w] = await Promise.all([
        api.get<Skill[]>('/chat/skills'),
        api.get<WikiMeta[]>('/admin/chat/wiki'),
      ])
      setSkills(s ?? [])
      setWikiDocs(w ?? [])
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '불러오기 실패')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const reindex = async () => {
    setReindexing(true)
    try {
      const out = await api.post<{ indexed: number; status: string }>('/admin/chat/wiki/reindex')
      toast.success(`${out.indexed}개 문서 재인덱싱 완료`)
      await load()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '재인덱싱 실패')
    } finally {
      setReindexing(false)
    }
  }

  const toggleEnabled = async (sk: Skill) => {
    try {
      await api.put(`/admin/chat/skills/${sk.id}`, { ...sk, enabled: !sk.enabled })
      toast.success(sk.enabled ? `${sk.name} 비활성화` : `${sk.name} 활성화`)
      await load()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '수정 실패')
    }
  }

  if (loading) {
    return (
      <div className="flex min-h-[50vh] items-center justify-center">
        <Spinner />
      </div>
    )
  }

  return (
    <div className="container mx-auto max-w-4xl space-y-6 px-4 py-6">
      <div className="flex items-center gap-2">
        <Link to="/admin">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="h-4 w-4" />
            관리자
          </Button>
        </Link>
        <Sparkles className="h-5 w-5 text-highlight" />
        <h1>챗봇 관리</h1>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>스킬 ({skills.length})</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="mb-3 text-sm text-muted-foreground">
            스킬 = 시스템 프롬프트 + 사용 가능 도구 + 위키 범위. 관리자 전용 스킬(<code>skill_designer</code>)
            로 대화하며 새 스킬을 만들 수 있습니다 (챗봇 위젯에서 "스킬 설계자 (관리자)" 선택).
          </p>
          <div className="space-y-2">
            {skills.map((sk) => (
              <div key={sk.id} className="flex items-start justify-between gap-3 rounded-lg border p-3">
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <code className="text-xs">{sk.slug}</code>
                    <strong className="text-sm">{sk.name}</strong>
                    {sk.admin_only && (
                      <span className="rounded bg-primary/15 px-1.5 py-0.5 text-[10px] text-primary">관리자 전용</span>
                    )}
                    {!sk.enabled && (
                      <span className="rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">비활성</span>
                    )}
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">{sk.description}</p>
                  <p className="mt-1 font-mono text-[10px] text-muted-foreground">
                    model: {sk.default_model}
                    {sk.default_reasoning_effort ? ` · effort: ${sk.default_reasoning_effort}` : ''}
                    {sk.tools_allowed && sk.tools_allowed.length > 0
                      ? ` · tools: ${sk.tools_allowed.join(', ')}`
                      : ''}
                  </p>
                </div>
                <Button
                  size="sm"
                  variant={sk.enabled ? 'outline' : 'default'}
                  onClick={() => void toggleEnabled(sk)}
                >
                  {sk.enabled ? '비활성화' : '활성화'}
                </Button>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>위키 문서 ({wikiDocs.length})</span>
            <Button size="sm" variant="outline" onClick={() => void reindex()} disabled={reindexing}>
              <RefreshCw className={`h-4 w-4 ${reindexing ? 'animate-spin' : ''}`} />
              재인덱싱
            </Button>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="mb-3 text-sm text-muted-foreground">
            <code>docs/llm-wiki/**/*.md</code> 파일을 FTS5 인덱스로 로드합니다. 파일이 변경됐을 때
            재인덱싱 버튼을 눌러주세요. (정식 반영은 재배포)
          </p>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-left text-xs uppercase text-muted-foreground">
                  <th className="py-2 pr-3">Slug</th>
                  <th className="py-2 pr-3">제목</th>
                  <th className="py-2 pr-3">Notion</th>
                  <th className="py-2 text-right">최종 갱신</th>
                </tr>
              </thead>
              <tbody>
                {wikiDocs.map((d) => (
                  <tr key={d.slug} className="border-b last:border-none">
                    <td className="py-2 pr-3 font-mono text-[11px]">{d.slug}</td>
                    <td className="py-2 pr-3">{d.title}</td>
                    <td className="py-2 pr-3 text-[11px] text-muted-foreground">
                      {d.notion_page_id ? d.notion_page_id.slice(0, 8) + '…' : '—'}
                    </td>
                    <td className="py-2 text-right text-[11px] text-muted-foreground">
                      {new Date(d.updated_at).toLocaleDateString('ko-KR')}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
