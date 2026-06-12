import { toast } from 'sonner'

import { getToken } from './auth'
import type { MilestoneFile } from './milestone'

// #125 사업계획서 비공개 첨부 — 공통 헬퍼.

export function formatFileSize(n: number): string {
  if (n < 1024) return `${n}B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)}KB`
  return `${(n / (1024 * 1024)).toFixed(1)}MB`
}

// 다운로드/열람 — 인증 헤더가 필요하므로 fetch 로 blob 을 받아 새 탭에 연다.
// 서버가 owner/admin 권한을 검증하므로 권한 없으면 403 → 에러 토스트.
export async function openMilestoneFile(file: Pick<MilestoneFile, 'id'>) {
  try {
    const res = await fetch(`/api/milestones/files/${file.id}`, {
      headers: { Authorization: `Bearer ${getToken()}` },
    })
    if (!res.ok) throw new Error(String(res.status))
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    window.open(url, '_blank')
    setTimeout(() => URL.revokeObjectURL(url), 60_000)
  } catch {
    toast.error('파일을 열 수 없습니다.')
  }
}
