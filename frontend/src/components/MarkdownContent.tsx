import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { ChevronDown, ChevronUp } from 'lucide-react'

interface MarkdownContentProps {
  content: string
  maxLines?: number
  className?: string
}

// SPA 내부 라우트 prefix (App.tsx 의 <Route path="..."/> 첫 segment 와 동기화).
// 이 목록에 있는 경로만 SPA 네비게이션 처리. 나머지 비-절대 path 는 #086 처럼 비활성화.
const SPA_ROUTES = new Set([
  'admin', 'bank', 'changelog', 'company', 'developer', 'exchange',
  'feed', 'grant', 'grants', 'invest', 'llm', 'login', 'market',
  'messages', 'notifications', 'oauth', 'pending', 'post', 'profile',
  'register', 'wallet',
])

function isInternalSPARoute(href: string): boolean {
  if (!href.startsWith('/') || href.startsWith('/uploads/')) return false
  const seg = href.slice(1).split(/[/?#]/, 1)[0]
  return SPA_ROUTES.has(seg)
}

export function MarkdownContent({
  content,
  maxLines = 8,
  className = '',
}: MarkdownContentProps) {
  const navigate = useNavigate()
  const [expanded, setExpanded] = useState(false)

  const lineCount = content.split('\n').length
  const charCount = content.length
  const isLong = lineCount > maxLines || charCount > 400

  return (
    <div className={className}>
      <div
        className={`markdown-body break-words ${
          !expanded && isLong ? 'line-clamp-[8] overflow-hidden' : ''
        }`}
        style={
          !expanded && isLong
            ? { maxHeight: `${maxLines * 1.5}em`, overflow: 'hidden' }
            : undefined
        }
      >
        <ReactMarkdown
          remarkPlugins={[[remarkGfm, { singleTilde: false }]]}
          components={{
            img: ({ src, alt }) => (
              <img
                src={src}
                alt={alt || ''}
                className="max-h-64 max-w-full rounded-md object-contain"
                loading="lazy"
              />
            ),
            a: ({ href, children }) => {
              const isUpload = href?.startsWith('/uploads/')
              const isAbsolute = !!href && /^(https?:|mailto:|tel:|\/uploads\/)/i.test(href)
              const isInternal = !!href && isInternalSPARoute(href)
              // 안전장치 (#086): 챗봇이 가끔 상대경로 링크를 만들 때, 그대로 두면
              // SPA 라우터 fallback 이 메인 페이지를 띄움. 비-내부/비-절대 링크는 비활성화.
              if (!isAbsolute && !isInternal && href) {
                return (
                  <span
                    className="underline decoration-dotted text-muted-foreground"
                    title={`유효하지 않은 링크: ${href}`}
                  >
                    {typeof children === 'string' ? decodeURIComponent(children) : children}
                  </span>
                )
              }
              if (isInternal) {
                // SPA 내부: react-router 로 클라이언트 네비게이션 (페이지 reload 안 함).
                // <a href> 도 유지 → 가운데/우클릭으로 새 탭 가능.
                return (
                  <a
                    href={href}
                    onClick={(e) => {
                      if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return
                      e.preventDefault()
                      navigate(href!)
                    }}
                  >
                    {typeof children === 'string' ? decodeURIComponent(children) : children}
                  </a>
                )
              }
              return (
                <a
                  href={href}
                  target="_blank"
                  rel="noopener noreferrer"
                  {...(isUpload ? { download: '' } : {})}
                  onClick={isUpload ? (e) => {
                    // PWA standalone 모드에서 파일 다운로드 보장
                    e.preventDefault()
                    window.open(href!, '_blank')
                  } : undefined}
                >
                  {typeof children === 'string' ? decodeURIComponent(children) : children}
                </a>
              )
            },
          }}
        >
          {content}
        </ReactMarkdown>
      </div>
      {isLong && (
        <button
          onClick={() => setExpanded(!expanded)}
          className="mt-1 flex items-center gap-0.5 text-xs text-primary hover:underline"
        >
          {expanded ? (
            <>
              접기 <ChevronUp className="h-3 w-3" />
            </>
          ) : (
            <>
              더보기 <ChevronDown className="h-3 w-3" />
            </>
          )}
        </button>
      )}
    </div>
  )
}
