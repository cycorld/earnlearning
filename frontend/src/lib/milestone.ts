/**
 * #119 학생 4대 평가지표 — pure utilities.
 *
 * 백엔드(`backend/internal/domain/milestone/url_filter.go`)와 deny-list/규칙을 동기화.
 * 한 곳에서 바꾸면 다른 곳도 같이 바꿀 것.
 */

export const MILESTONE_TYPES = ['mvp1', 'mvp2', 'business_plan', 'retrospective'] as const
export type MilestoneType = (typeof MILESTONE_TYPES)[number]

export const MILESTONE_LABELS: Record<MilestoneType, string> = {
  mvp1: '1차 MVP 배포',
  mvp2: '2차 MVP 배포',
  business_plan: '사업계획서 제출',
  retrospective: '한 학기 회고 발표',
}

export const MILESTONE_DEADLINES: Record<MilestoneType, string> = {
  mvp1: '7주차까지',
  mvp2: '12주차까지',
  business_plan: '14주차',
  retrospective: '보강 1주차',
}

export type MilestoneStatus = 'pending' | 'approved' | 'rejected'
export type MilestoneSourceType = 'manual' | 'company' | 'grant'

export interface Milestone {
  id: number
  student_id: number
  type: MilestoneType
  source_type: MilestoneSourceType
  source_ref_id?: number | null
  url: string
  content: string
  status: MilestoneStatus
  admin_note: string
  approved_by?: number | null
  approved_at?: string | null
  created_at: string
  updated_at: string
}

export interface StudentRef {
  id: number
  name: string
  student_id: string
  department: string
}

export interface StudentProgress {
  student: StudentRef
  milestones: (Milestone | null)[] // ordered by MILESTONE_TYPES; nulls allowed
  approved_count: number
  group: '' | 'A' | 'B' | 'C' | 'D'
}

/**
 * 연습용 도메인 — 자동 detect 에서 제외.
 * 호스트(서브도메인 포함) suffix 매칭. 백엔드와 동일.
 */
export const DENY_HOST_SUFFIXES = [
  'aistudio.google.com',
  'ai.studio',
  'claude.ai',
  'chatgpt.com',
  'chat.openai.com',
  'gemini.google.com',
  'bard.google.com',
  'localhost',
  '127.0.0.1',
]

export function isValidMilestoneURL(raw: string): boolean {
  const s = (raw ?? '').trim()
  if (!s) return false
  let u: URL
  try {
    u = new URL(s)
  } catch {
    return false
  }
  if (u.protocol !== 'http:' && u.protocol !== 'https:') return false
  const host = u.hostname.toLowerCase()
  if (!host) return false
  for (const deny of DENY_HOST_SUFFIXES) {
    if (host === deny || host.endsWith('.' + deny)) return false
  }
  return true
}

/**
 * approved 개수로 그룹 분류.
 * 4 → A / 3 → B / 2 → C / 1 → D / 0 → '' (그룹 없음)
 */
export function classifyGroup(approvedCount: number): '' | 'A' | 'B' | 'C' | 'D' {
  if (approvedCount === 4) return 'A'
  if (approvedCount === 3) return 'B'
  if (approvedCount === 2) return 'C'
  if (approvedCount === 1) return 'D'
  return ''
}

export const GROUP_DESCRIPTIONS: Record<string, string> = {
  A: '4개 평가지표 모두 완료',
  B: '3개 완료',
  C: '2개 완료',
  D: '1개 완료',
  '': '아직 승인된 평가지표 없음',
}
