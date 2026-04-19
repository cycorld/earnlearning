import { useState } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { ChevronDown, ChevronUp } from 'lucide-react'

interface MarkdownContentProps {
  content: string
  maxLines?: number
  className?: string
}

export function MarkdownContent({
  content,
  maxLines = 8,
  className = '',
}: MarkdownContentProps) {
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
              // 안전장치 (#086): 챗봇이 가끔 상대경로 링크를 만들 때, 그대로 두면
              // SPA 라우터 fallback 이 메인 페이지를 띄움. 링크 비활성화하고 텍스트만.
              if (!isAbsolute && href) {
                return (
                  <span
                    className="underline decoration-dotted text-muted-foreground"
                    title={`유효하지 않은 링크: ${href}`}
                  >
                    {typeof children === 'string' ? decodeURIComponent(children) : children}
                  </span>
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
