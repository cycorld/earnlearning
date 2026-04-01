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
                  {children}
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
