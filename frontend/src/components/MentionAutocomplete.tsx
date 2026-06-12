import { useState, useEffect, useCallback } from 'react'
import { api } from '@/lib/api'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'

// #132 @멘션 자동완성 — 텍스트 입력에서 '@검색어'를 감지해 드롭다운을 띄우고,
// 선택 시 @[이름](user:ID) 마크업을 삽입한다 (백엔드 mentionRegex와 동기화).

export interface MentionUser {
  id: number
  name: string
  department: string
  student_id: string
  avatar_url: string
}

interface MentionState {
  /** 텍스트 내 '@' 인덱스 */
  start: number
  query: string
}

export function useMentionAutocomplete(
  value: string,
  onChange: (v: string) => void,
  textareaRef: React.RefObject<HTMLTextAreaElement | null>,
) {
  const [mention, setMention] = useState<MentionState | null>(null)
  const [users, setUsers] = useState<MentionUser[]>([])
  const [highlight, setHighlight] = useState(0)

  // 입력 변화 시 호출 — 커서 앞의 '@검색어' 감지 (공백/줄시작 뒤 @만 인정)
  const detect = useCallback((text: string, cursor: number) => {
    const before = text.slice(0, cursor)
    const m = /(?:^|\s)@([^\s@]*)$/.exec(before)
    if (m) {
      setMention({ start: cursor - m[1].length - 1, query: m[1] })
    } else {
      setMention(null)
    }
  }, [])

  useEffect(() => {
    if (!mention || mention.query.length === 0) {
      setUsers([])
      return
    }
    const t = setTimeout(async () => {
      try {
        const res = await api.get<MentionUser[]>(
          `/users/search?q=${encodeURIComponent(mention.query)}`,
        )
        setUsers(res ?? [])
        setHighlight(0)
      } catch {
        setUsers([])
      }
    }, 200)
    return () => clearTimeout(t)
  }, [mention])

  const select = useCallback(
    (u: MentionUser) => {
      if (!mention) return
      const el = textareaRef.current
      // 커서가 아니라 detect 시점의 '@검색어' 끝 위치 기준 — 클릭 등으로 커서가
      // 이동해도(텍스트 변화 없음) mention state가 가리키는 구간만 치환한다
      const cursor = mention.start + 1 + mention.query.length
      const markup = `@[${u.name}](user:${u.id}) `
      const next = value.slice(0, mention.start) + markup + value.slice(cursor)
      onChange(next)
      setMention(null)
      setUsers([])
      requestAnimationFrame(() => {
        if (!el) return
        el.focus()
        const pos = mention.start + markup.length
        el.setSelectionRange(pos, pos)
      })
    },
    [mention, value, onChange, textareaRef],
  )

  const open = mention !== null && users.length > 0

  const onKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (!open) return
      if (e.nativeEvent.isComposing) return // 한글 IME 조합 중 Enter 무시
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setHighlight((h) => (h + 1) % users.length)
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setHighlight((h) => (h - 1 + users.length) % users.length)
      } else if (e.key === 'Enter' || e.key === 'Tab') {
        e.preventDefault()
        select(users[highlight])
      } else if (e.key === 'Escape') {
        setMention(null)
        setUsers([])
      }
    },
    [open, users, highlight, select],
  )

  // 부모는 textarea를 relative 컨테이너로 감싸고 이 엘리먼트를 함께 렌더한다.
  const dropdown = open ? (
    <div className="absolute left-0 right-0 top-full z-50 mt-1 max-h-56 overflow-auto rounded-md border bg-popover shadow-md">
      {users.map((u, i) => (
        <button
          key={u.id}
          type="button"
          className={`flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-accent ${
            i === highlight ? 'bg-accent' : ''
          }`}
          // mousedown: textarea blur 전에 선택 처리
          onMouseDown={(e) => {
            e.preventDefault()
            select(u)
          }}
        >
          <Avatar className="h-6 w-6">
            {u.avatar_url ? <AvatarImage src={u.avatar_url} /> : null}
            <AvatarFallback>{u.name?.[0] ?? '?'}</AvatarFallback>
          </Avatar>
          <span className="font-medium">{u.name}</span>
          <span className="text-xs text-muted-foreground">
            {[u.department, u.student_id].filter(Boolean).join(' · ')}
          </span>
        </button>
      ))}
    </div>
  ) : null

  return { detect, onKeyDown, dropdown }
}
