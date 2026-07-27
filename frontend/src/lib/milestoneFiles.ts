import { toast } from 'sonner'

import { getToken } from './auth'
import type { MilestoneFile } from './milestone'

// #125 사업계획서 비공개 첨부 — 공통 헬퍼.

export function formatFileSize(n: number): string {
  if (n < 1024) return `${n}B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)}KB`
  return `${(n / (1024 * 1024)).toFixed(1)}MB`
}

// #176 새 탭(inline)으로 열어도 안전한 타입만 화이트리스트.
// blob: URL 은 이를 만든 페이지와 same-origin 이므로, HTML blob 을 window.open 하면
// 서버가 붙인 Content-Disposition/nosniff 와 무관하게 우리 출처에서 스크립트가 실행된다.
// 따라서 화이트리스트 밖(특히 HTML)은 반드시 <a download> 로 저장만 시킨다.
function isInlineSafe(mime: string, filename: string): boolean {
  const ext = filename.toLowerCase().match(/\.[^.]+$/)?.[0] ?? ''
  if (ext === '.html' || ext === '.htm') return false
  const type = mime.split(';')[0]!.trim().toLowerCase()
  return type === 'application/pdf' || type.startsWith('image/')
}

function triggerDownload(url: string, filename: string) {
  const a = document.createElement('a')
  a.href = url
  a.download = filename || 'download'
  a.rel = 'noopener'
  document.body.appendChild(a)
  a.click()
  a.remove()
}

// 다운로드/열람 — 인증 헤더가 필요하므로 fetch 로 blob 을 받는다.
// 서버가 owner/admin 권한을 검증하므로 권한 없으면 403 → 에러 토스트.
export async function openMilestoneFile(file: Pick<MilestoneFile, 'id'> & Partial<MilestoneFile>) {
  try {
    const res = await fetch(`/api/milestones/files/${file.id}`, {
      headers: { Authorization: `Bearer ${getToken()}` },
    })
    if (!res.ok) throw new Error(String(res.status))
    const raw = await res.blob()
    const filename = file.filename ?? ''
    const serverType = res.headers.get('Content-Type') ?? raw.type ?? ''
    const inline = isInlineSafe(serverType, filename)
    // 서버 응답이 잘못된 타입이더라도 렌더링되지 않도록 blob 타입을 강제한다.
    const blob = inline ? raw : new Blob([raw], { type: 'application/octet-stream' })
    const url = URL.createObjectURL(blob)
    if (inline) {
      window.open(url, '_blank', 'noopener')
    } else {
      triggerDownload(url, filename)
    }
    setTimeout(() => URL.revokeObjectURL(url), 60_000)
  } catch {
    toast.error('파일을 열 수 없습니다.')
  }
}
