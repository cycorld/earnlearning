---
slug: 062-disclosure-period-display
title: 공시 카드 기간이 ISO 타임스탬프로 표시되던 문제 수정
date: 2026-04-18
tags: [fix, 공시, 날짜포맷, 프론트엔드, 유틸, 단위테스트]
---

# 공시 카드 기간이 ISO 타임스탬프로 표시되던 문제 수정

## 왜 필요했는가

#033 청산 기능에서 회사가 청산되면 청산 내역을 자동 공시로 남기는데, 그 공시 카드의 **기간** 칸에 이런 게 떴습니다:

```
2026-04-12T00:00:00Z ~ 2026-04-12T00:00:00Z
```

학생들이 쓰는 UI에 내부 타임스탬프가 노출되는 못생긴 상태. 원래는 `2026. 4. 12. ~ 2026. 4. 12.` 같은 한국어 날짜로 보여야 해요.

**왜 이렇게 나왔나?**
- 사용자가 직접 공시를 작성할 때는 `<input type="date">` 가 `YYYY-MM-DD` 문자열을 보내므로 DB에도 그대로 저장되고 그대로 렌더되어 문제 없음.
- 반면 청산 자동 공시는 백엔드가 `time.Now().Format("2006-01-02")` 로 날짜를 넘기지만, SQLite 가 DATE 컬럼에 문자열로 bind 하는 과정에서 ISO 타임스탬프로 저장되고, 프론트가 그대로 렌더.

저장 포맷이 경로마다 달라지는 건 이미 잠재적인 버그이고, **프론트가 포맷을 책임지는 것**이 가장 견고합니다.

## 무엇을 했는가

### 1. `formatDate` 유틸 추가

`frontend/src/lib/utils.ts` 에 공용 날짜 포매터 추가:

```ts
export function formatDate(s: string | null | undefined): string {
  if (!s) return ''
  // YYYY-MM-DD 또는 ISO 8601 문자열의 앞 10자만 파싱 — 타임존 변환 회피
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(s)
  if (match) {
    return `${match[1]}. ${Number(match[2])}. ${Number(match[3])}.`
  }
  return s
}
```

**설계 포인트: `new Date()` 를 피함.**
`new Date("2026-04-12")` 는 UTC 자정으로 해석되는데, `toLocaleDateString('ko-KR')` 로 포맷하면 로컬 타임존이 UTC 서쪽(예: 미국)일 때 하루 당겨집니다. 테스트도 CI 머신의 TZ 에 따라 빨간색이 나오죠. 그래서 문자열 앞 10자(`YYYY-MM-DD`)를 정규식으로 뽑아서 그 정수만 그대로 찍습니다. **결정적이고 타임존 독립적.**

### 2. TDD: 단위 테스트 먼저 (Red)

`frontend/src/lib/utils.test.ts` 를 생성, 6가지 케이스:
- ISO 타임스탬프 → 한국어 날짜
- YYYY-MM-DD → 한국어 날짜
- 앞자리 0 제거 (`2026-01-05` → `2026. 1. 5.`)
- 빈 문자열 / null / undefined → 빈 문자열
- 알 수 없는 포맷 → 원본 반환 (방어적)

구현 전 실행하면 `TypeError: formatDate is not a function` 으로 6/6 FAIL. 구현 후 6/6 PASS.

### 3. 사용처 2개 파일에 적용

- `frontend/src/routes/company/DisclosureSection.tsx` — 회사 상세 페이지의 공시 카드 + 상세 다이얼로그
- `frontend/src/routes/admin/AdminDisclosuresPage.tsx` — 관리자 공시 리스트 + 리뷰 다이얼로그

총 4곳의 `{d.period_from} ~ {d.period_to}` 를 `{formatDate(d.period_from)} ~ {formatDate(d.period_to)}` 로 교체.

## 사용한 프롬프트

```
백로그 중에 현재 반영 필요한거 하나씩 적용해줘.
```

AI가 #025 를 두 번째로 골라서 처리. 작업 중 DisclosureSection 만 고치려다가 `grep` 로 `period_from` / `period_to` 사용처를 훑어보니 AdminDisclosuresPage 에도 동일한 패턴이 있어서 함께 고쳤습니다. (관리자 페이지만 수정 안 된 채 머지됐다면 같은 버그가 다른 URL 로 남았을 것.)

## 배운 점

- **날짜/시간은 "어디서 포맷하느냐" 가 시스템 설계의 결정.** DB/백엔드가 일관된 포맷을 주려고 애써도 경로가 여러 개면 결국 한 곳은 놓친다. 프론트에서 포맷을 책임지는 편이 방어적.
- **`new Date()` 파싱은 타임존 함정.** 날짜 전용 데이터(YYYY-MM-DD)를 다룰 때는 문자열 조작으로 포매팅하는 게 더 안전하고 테스트도 결정적.
- **같은 버그의 복수 인스턴스를 찾는 습관:** `grep` 로 관련 필드를 추적해야 한 파일만 고쳐 놓고 "고쳤다" 선언하는 실수를 막을 수 있음.

## 관련 티켓

- #025 (backlog → done) — 이 PR
- #024 — 같은 PR 시리즈의 첫 픽스 (SQL status 하드코딩)
- #023 / #033 — 원본 청산 기능
