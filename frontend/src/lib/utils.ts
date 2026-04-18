import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatMoney(amount: number): string {
  return new Intl.NumberFormat('ko-KR').format(amount) + '원'
}

// YYYY-MM-DD 문자열이나 ISO 8601 타임스탬프를 "YYYY. M. D." 형태로 변환한다.
// 타임존 변환을 피하기 위해 앞 10자(날짜 부분)만 파싱한다 — 서버가 DATE를 넘기든
// DATETIME을 넘기든 동일한 결과를 돌려주기 위함.
export function formatDate(s: string | null | undefined): string {
  if (!s) return ''
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(s)
  if (match) {
    return `${match[1]}. ${Number(match[2])}. ${Number(match[3])}.`
  }
  return s
}

/**
 * 사용자 이름을 "이름(소속/학번2자리)" 형태로 표시합니다.
 * - department + student_id: "홍길동(컴공/33)"
 * - department만: "홍길동(컴공)"
 * - student_id만: "홍길동(33)"
 * - 둘 다 없으면: "홍길동"
 */
export function displayName(
  user: { name?: string; department?: string; student_id?: string } | null | undefined,
): string {
  if (!user?.name) return '?'
  const dept = user.department ? shortenDept(user.department) : ''
  const sid = user.student_id ? user.student_id.slice(0, 2) : ''
  if (!dept && !sid) return user.name
  const suffix = [dept, sid].filter(Boolean).join('/')
  return `${user.name}(${suffix})`
}

function shortenDept(dept: string): string {
  // 긴 학과명을 축약 (2~3글자)
  const map: Record<string, string> = {
    '컴퓨터공학': '컴공',
    '컴퓨터공학과': '컴공',
    '경영학': '경영',
    '경영학과': '경영',
    '경제학': '경제',
    '경제학과': '경제',
    '디자인': '디자인',
    '국제학': '국제',
    '국제학과': '국제',
    '미디어학': '미디어',
    '미디어학과': '미디어',
    '소프트웨어학': 'SW',
    '소프트웨어학과': 'SW',
    '인공지능': 'AI',
    '인공지능학과': 'AI',
  }
  return map[dept] || (dept.length > 3 ? dept.slice(0, 3) : dept)
}
