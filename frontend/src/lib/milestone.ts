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

// #125 사업계획서 비공개 첨부 파일
export interface MilestoneFile {
  id: number
  student_id: number
  filename: string
  mime_type: string
  size: number
  created_at: string
}

export interface Milestone {
  id: number
  student_id: number
  type: MilestoneType
  source_type: MilestoneSourceType
  source_ref_id?: number | null
  url: string
  content: string
  files?: MilestoneFile[] // #125 business_plan 첨부
  status: MilestoneStatus
  admin_note: string
  approved_by?: number | null
  approved_at?: string | null
  // #120 retrospective 만 채워짐
  ai_score?: number | null
  ai_reasoning?: string
  ai_signals?: string
  ai_evaluated_at?: string | null
  created_at: string
  updated_at: string
}

// #120 essay 평가 응답
export interface EssayScoreSignal {
  key: string
  label: string
  value: number
  weight: number
  hint: string
}

export interface EssayScoreResult {
  heuristic_score: number
  llm_score: number // -1 if LLM 없음
  combined_score: number
  llm_reasoning: string
  signals: EssayScoreSignal[]
}

/** AI 작성 확률 점수 → 컬러 클래스 + 라벨 */
export function aiScoreMeta(score: number): { label: string; chip: string; tone: 'good' | 'warn' | 'danger' } {
  if (score < 30) return { label: '사람이 쓴 글 같음', chip: 'bg-emerald-100 text-emerald-700', tone: 'good' }
  if (score < 60) return { label: '약간 AI 같음', chip: 'bg-amber-100 text-amber-700', tone: 'warn' }
  return { label: 'AI 작성 가능성 높음', chip: 'bg-red-100 text-red-700', tone: 'danger' }
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
  // #125 성적/자산 — 본인 대시보드에서만 채워짐
  asset_total?: number
  group_size?: number
  asset_rank?: number
  asset_percentile?: number // 같은 그룹 내 상위 N% (1~100), 0 = 미산정
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
