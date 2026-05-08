/**
 * URL helpers — #115 회사 service_url 다중 URL 지원.
 *
 * DB 는 단일 TEXT 컬럼에 쉼표 구분으로 저장 ("https://a.com,https://b.com").
 * 이 helper 가 parse/format/validate 의 단일 진실.
 */

/**
 * 쉼표 구분 문자열 → URL 배열.
 * - 각 piece trim
 * - 빈 piece 제거
 * - 결과 0개 이상 가능
 */
export function parseServiceUrls(raw: string | undefined | null): string[] {
  if (!raw) return []
  return raw
    .split(',')
    .map((s) => s.trim())
    .filter((s) => s.length > 0)
}

/**
 * URL 배열 → 쉼표 구분 문자열 (저장용).
 * 빈 배열 → 빈 문자열.
 */
export function formatServiceUrls(urls: string[]): string {
  return urls
    .map((s) => s.trim())
    .filter((s) => s.length > 0)
    .join(',')
}

/** http/https URL 인지 검사. */
export function isValidHttpUrl(s: string): boolean {
  try {
    const u = new URL(s)
    return u.protocol === 'http:' || u.protocol === 'https:'
  } catch {
    return false
  }
}

/**
 * 쉼표 구분 문자열의 모든 piece 가 valid http/https URL 인지 검증.
 * 빈 문자열은 valid (URL 0개 OK).
 * 모든 piece valid 이면 true, 하나라도 invalid 이면 false.
 */
export function isValidServiceUrls(raw: string | undefined | null): boolean {
  const urls = parseServiceUrls(raw)
  return urls.every(isValidHttpUrl)
}

/**
 * 표시용 — `https://` `http://` 를 벗긴 짧은 호스트+path.
 * 빈 문자열은 그대로 빈 문자열.
 */
export function shortenUrl(url: string): string {
  return url.replace(/^https?:\/\//, '')
}
