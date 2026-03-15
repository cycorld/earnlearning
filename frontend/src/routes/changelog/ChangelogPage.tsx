import { useEffect, useState, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import 'highlight.js/styles/github-dark.min.css'
import { ArrowLeft, Calendar, Tag, BookOpen, ChevronRight } from 'lucide-react'
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
    const currentIndex = entries.findIndex((e) => e.slug === selectedSlug)
    const prevEntry = currentIndex < entries.length - 1 ? entries[currentIndex + 1] : null
    const nextEntry = currentIndex > 0 ? entries[currentIndex - 1] : null

    return (
      <div className="mx-auto max-w-3xl px-4 py-6">
        <Button variant="ghost" size="sm" onClick={goBack} className="mb-4 gap-1">
          <ArrowLeft className="h-4 w-4" />
          목록으로
        </Button>
        {entry && (
          <div className="mb-6">
            <div className="mb-2 flex items-center gap-2 text-sm text-muted-foreground">
              <Calendar className="h-4 w-4" />
              <span>{entry.date}</span>
              <span className="text-muted-foreground/50">|</span>
              <span className="font-mono text-xs">#{entry.slug.split('-')[0]}</span>
            </div>
            <div className="flex flex-wrap gap-1.5">
              {entry.tags.map((tag) => (
                <Badge key={tag} variant="secondary" className="text-xs">
                  {tag}
                </Badge>
              ))}
            </div>
          </div>
        )}
        <article className="markdown-body">
          <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeHighlight]}>
            {content}
          </ReactMarkdown>
        </article>

        {/* Prev / Next navigation */}
        <div className="mt-10 flex items-stretch gap-3 border-t pt-6">
          {prevEntry ? (
            <button
              onClick={() => selectEntry(prevEntry.slug)}
              className="flex flex-1 flex-col items-start rounded-lg border p-3 text-left transition-colors hover:bg-accent"
            >
              <span className="text-xs text-muted-foreground">이전</span>
              <span className="mt-1 text-sm font-medium line-clamp-1">{prevEntry.title}</span>
            </button>
          ) : (
            <div className="flex-1" />
          )}
          {nextEntry ? (
            <button
              onClick={() => selectEntry(nextEntry.slug)}
              className="flex flex-1 flex-col items-end rounded-lg border p-3 text-right transition-colors hover:bg-accent"
            >
              <span className="text-xs text-muted-foreground">다음</span>
              <span className="mt-1 text-sm font-medium line-clamp-1">{nextEntry.title}</span>
            </button>
          ) : (
            <div className="flex-1" />
          )}
        </div>
      </div>
    )
  }

  // List view
  return (
    <div className="mx-auto max-w-3xl px-4 py-6">
      <div className="mb-2 flex items-center gap-2">
        <BookOpen className="h-5 w-5 text-primary" />
        <h1 className="text-xl font-bold">개발일지</h1>
        <Badge variant="outline" className="ml-1 text-xs font-normal">
          {entries.length}편
        </Badge>
      </div>
      <p className="mb-6 text-sm text-muted-foreground">
        EarnLearning이 어떻게 만들어지고 있는지, 그 과정을 기록합니다.
        기획부터 배포까지, AI와 함께하는 실전 개발 과정을 살펴보세요.
      </p>
      <div className="space-y-2">
        {entries.map((entry, i) => {
          const num = entry.slug.split('-')[0]
          return (
            <button
              key={entry.slug}
              onClick={() => selectEntry(entry.slug)}
              className="group flex w-full items-center gap-3 rounded-lg border p-4 text-left transition-colors hover:bg-accent"
            >
              <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-bold text-primary">
                {num}
              </div>
              <div className="min-w-0 flex-1">
                <h2 className="text-sm font-semibold leading-snug">{entry.title}</h2>
                <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
                  <span className="flex items-center gap-1 text-[11px] text-muted-foreground">
                    <Calendar className="h-3 w-3" />
                    {entry.date}
                  </span>
                  {entry.tags.slice(0, 3).map((tag) => (
                    <Badge key={tag} variant="outline" className="px-1.5 py-0 text-[10px]">
                      {tag}
                    </Badge>
                  ))}
                  {entry.tags.length > 3 && (
                    <span className="text-[10px] text-muted-foreground">
                      +{entry.tags.length - 3}
                    </span>
                  )}
                </div>
              </div>
              <ChevronRight className="h-4 w-4 flex-shrink-0 text-muted-foreground/50 transition-transform group-hover:translate-x-0.5" />
            </button>
          )
        })}
      </div>
      {entries.length === 0 && (
        <p className="text-center text-muted-foreground">아직 작성된 개발일지가 없습니다.</p>
      )}
    </div>
  )
}
