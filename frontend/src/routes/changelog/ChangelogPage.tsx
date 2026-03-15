import { useEffect, useState, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { ArrowLeft, Calendar, Tag, BookOpen } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

interface ChangelogEntry {
  slug: string
  title: string
  date: string
  tags: string[]
}

export default function ChangelogPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [entries, setEntries] = useState<ChangelogEntry[]>([])
  const [content, setContent] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const selectedSlug = searchParams.get('entry')

  useEffect(() => {
    fetch('/changelog/index.json')
      .then((r) => r.json())
      .then((data: ChangelogEntry[]) => {
        setEntries(data.sort((a, b) => b.slug.localeCompare(a.slug)))
        setLoading(false)
      })
      .catch(() => setLoading(false))
  }, [])

  useEffect(() => {
    if (!selectedSlug) {
      setContent(null)
      return
    }
    setLoading(true)
    fetch(`/changelog/${selectedSlug}.md`)
      .then((r) => {
        if (!r.ok) throw new Error('Not found')
        return r.text()
      })
      .then((text) => {
        // Remove frontmatter
        const stripped = text.replace(/^---[\s\S]*?---\n*/, '')
        setContent(stripped)
        setLoading(false)
      })
      .catch(() => {
        setContent('# 페이지를 찾을 수 없습니다')
        setLoading(false)
      })
  }, [selectedSlug])

  const selectEntry = useCallback(
    (slug: string) => {
      setSearchParams({ entry: slug })
    },
    [setSearchParams],
  )

  const goBack = useCallback(() => {
    setSearchParams({})
  }, [setSearchParams])

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="text-muted-foreground">불러오는 중...</div>
      </div>
    )
  }

  // Detail view
  if (selectedSlug && content) {
    const entry = entries.find((e) => e.slug === selectedSlug)
    return (
      <div className="mx-auto max-w-3xl px-4 py-6">
        <Button variant="ghost" size="sm" onClick={goBack} className="mb-4 gap-1">
          <ArrowLeft className="h-4 w-4" />
          목록으로
        </Button>
        {entry && (
          <div className="mb-6 flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
            <Calendar className="h-4 w-4" />
            <span>{entry.date}</span>
            {entry.tags.map((tag) => (
              <Badge key={tag} variant="secondary" className="text-xs">
                {tag}
              </Badge>
            ))}
          </div>
        )}
        <article className="prose prose-sm dark:prose-invert max-w-none prose-headings:scroll-mt-20 prose-h2:text-lg prose-h2:font-bold prose-h2:border-b prose-h2:pb-2 prose-h2:mt-8 prose-h3:text-base prose-pre:bg-muted prose-pre:text-foreground prose-code:text-sm">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
        </article>
      </div>
    )
  }

  // List view
  return (
    <div className="mx-auto max-w-3xl px-4 py-6">
      <div className="mb-6 flex items-center gap-2">
        <BookOpen className="h-5 w-5 text-primary" />
        <h1 className="text-xl font-bold">개발일지</h1>
      </div>
      <p className="mb-6 text-sm text-muted-foreground">
        EarnLearning이 어떻게 만들어지고 있는지, 그 과정을 기록합니다.
        기획부터 배포까지, AI와 함께하는 실전 개발 과정을 살펴보세요.
      </p>
      <div className="space-y-3">
        {entries.map((entry) => (
          <button
            key={entry.slug}
            onClick={() => selectEntry(entry.slug)}
            className="w-full rounded-lg border p-4 text-left transition-colors hover:bg-accent"
          >
            <div className="flex items-start justify-between gap-2">
              <div className="min-w-0 flex-1">
                <h2 className="font-semibold">{entry.title}</h2>
                <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                  <span className="flex items-center gap-1">
                    <Calendar className="h-3 w-3" />
                    {entry.date}
                  </span>
                  {entry.tags.map((tag) => (
                    <Badge key={tag} variant="outline" className="text-[10px]">
                      <Tag className="mr-0.5 h-2.5 w-2.5" />
                      {tag}
                    </Badge>
                  ))}
                </div>
              </div>
            </div>
          </button>
        ))}
      </div>
      {entries.length === 0 && (
        <p className="text-center text-muted-foreground">아직 작성된 개발일지가 없습니다.</p>
      )}
    </div>
  )
}
