import type { CSSProperties } from 'react'

// 챗봇 FAB가 스냅되는 6개 모서리 앵커 (#135)
export const FAB_ANCHORS = [
  'top-left',
  'top-right',
  'mid-left',
  'mid-right',
  'bottom-left',
  'bottom-right',
] as const

export type FabAnchor = (typeof FAB_ANCHORS)[number]

const DEFAULT_ANCHOR: FabAnchor = 'bottom-right'
const STORAGE_KEY = 'chatdock:fab-anchor'

export function loadAnchor(): FabAnchor {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v && (FAB_ANCHORS as readonly string[]).includes(v)) return v as FabAnchor
  } catch {
    /* localStorage 접근 불가 환경 — 기본값 */
  }
  return DEFAULT_ANCHOR
}

export function saveAnchor(anchor: FabAnchor): void {
  try {
    localStorage.setItem(STORAGE_KEY, anchor)
  } catch {
    /* 무시 */
  }
}

// 각 앵커의 대략적 화면 좌표(중심점) — 스냅 거리 계산용.
// 상단바(~56px)·바텀네브(~64px)를 피하도록 여백 확보.
function anchorPoint(anchor: FabAnchor, w: number, h: number): { x: number; y: number } {
  const [v, hor] = anchor.split('-') as ['top' | 'mid' | 'bottom', 'left' | 'right']
  const x = hor === 'left' ? 40 : w - 40
  const y = v === 'top' ? 96 : v === 'bottom' ? h - 96 : h / 2
  return { x, y }
}

// 릴리즈 좌표에서 가장 가까운 앵커로 스냅.
export function nearestAnchor(point: { x: number; y: number }, w: number, h: number): FabAnchor {
  let best: FabAnchor = DEFAULT_ANCHOR
  let bestDist = Infinity
  for (const a of FAB_ANCHORS) {
    const p = anchorPoint(a, w, h)
    const d = (p.x - point.x) ** 2 + (p.y - point.y) ** 2
    if (d < bestDist) {
      bestDist = d
      best = a
    }
  }
  return best
}

// 앵커 → fixed 포지셔닝 인라인 스타일 (safe-area 반영).
export function anchorStyle(anchor: FabAnchor): CSSProperties {
  const [v, hor] = anchor.split('-') as ['top' | 'mid' | 'bottom', 'left' | 'right']
  const s: CSSProperties = {}
  if (hor === 'left') s.left = '1rem'
  else s.right = '1rem'
  if (v === 'top') s.top = 'calc(env(safe-area-inset-top) + 4.5rem)'
  else if (v === 'bottom') s.bottom = 'calc(5rem + env(safe-area-inset-bottom))'
  else {
    // 세로 중앙 — transform 대신 marginTop(-FAB높이/2)로 정렬해 hover/active transform과 충돌 회피.
    s.top = '50%'
    s.marginTop = '-1.5rem'
  }
  return s
}
