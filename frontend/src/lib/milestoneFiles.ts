import { toast } from 'sonner'

import { apiFetch } from './api'
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

// #184 fetch 를 기다린 뒤 window.open 하면 사용자 제스처가 끊겨 팝업이 차단된다.
// → 확장자로 미리 짐작해서 클릭 즉시 빈 탭을 열어두고, blob 이 준비되면 그 탭을 이동시킨다.
// 여기는 어디까지나 "미리 열지" 결정일 뿐, 실제 inline 여부는 서버 Content-Type 으로 다시 판단한다.
// .html/.htm 은 목록에 없으므로 구조적으로 절대 미리 열리지 않는다.
const PREOPEN_EXT = ['.pdf', '.jpg', '.jpeg', '.png', '.gif', '.webp']

function guessInline(filename: string): boolean {
  const ext = filename.toLowerCase().match(/\.[^.]+$/)?.[0] ?? ''
  return PREOPEN_EXT.includes(ext)
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
// 서버가 권한을 검증하므로 권한 없으면 403 → 에러 토스트.
export async function openPrivateFile(url: string, filename: string) {
  // noopener 를 주면 window.open 이 항상 null 을 반환해 이 방식이 무너진다.
  // 우리가 만든 same-origin blob 을 우리 탭에서 여는 것이라 문제 없다.
  const pre = guessInline(filename) ? window.open('', '_blank') : null
  try {
    const res = await apiFetch(url)
    if (!res.ok) throw new Error(String(res.status))
    const raw = await res.blob()
    const serverType = res.headers.get('Content-Type') ?? raw.type ?? ''
    const inline = isInlineSafe(serverType, filename)
    // 서버 응답이 잘못된 타입이더라도 렌더링되지 않도록 blob 타입을 강제한다.
    const blob = inline ? raw : new Blob([raw], { type: 'application/octet-stream' })
    const blobURL = URL.createObjectURL(blob)
    if (inline) {
      if (pre) {
        pre.location.replace(blobURL)
      } else if (!window.open(blobURL, '_blank')) {
        // 팝업 차단 → 다운로드로 대체.
        triggerDownload(blobURL, filename)
      }
    } else {
      pre?.close()
      triggerDownload(blobURL, filename)
    }
    setTimeout(() => URL.revokeObjectURL(blobURL), 60_000)
  } catch {
    pre?.close()
    toast.error('파일을 열 수 없습니다.')
  }
}

export function openMilestoneFile(file: Pick<MilestoneFile, 'id'> & Partial<MilestoneFile>) {
  return openPrivateFile(`/api/milestones/files/${file.id}`, file.filename ?? '')
}
