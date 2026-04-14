---
id: 025
title: 공시 카드에 기간이 ISO 타임스탬프로 표시됨
priority: low
type: fix
branch: fix/disclosure-period-display
created: 2026-04-13
---

## 문제

회사 상세 페이지 `DisclosureSection`에서 공시 카드의 기간(`period_from ~ period_to`)이 포맷되지 않은 ISO 8601 타임스탬프 그대로 렌더된다.

## 재현

1. 회사 상세 페이지 이동
2. 공시 섹션에서 기존 공시 카드 확인
3. 기간 표시 영역에 다음과 같이 나타남:
   ```
   2026-04-12T00:00:00Z ~ 2026-04-12T00:00:00Z
   ```

기대: `2026. 4. 12. ~ 2026. 4. 12.` (한국어 로케일 날짜 포맷)

## 발견 경위

#023 청산 기능 검증 중 자동 생성된 청산 공시 카드에서 확인. 백엔드 `ExecuteLiquidation`이 `time.Now().Format("2006-01-02")` 로 `period_from/to`를 보내지만, DB에서 `DATE` 타입이 아니고 `DATETIME`으로 내려오면서 SQLite가 타임스탬프로 저장 → 프론트가 그대로 렌더.

실제로 일반 사용자가 직접 작성한 공시는 `<input type="date">` 값(YYYY-MM-DD 문자열)을 쓰기 때문에 문제없이 표시되지만, 청산 자동 공시는 `Format("2006-01-02")` 결과를 repo가 SQLite DATE로 암시적 변환하지 못하는 듯.

## 원인

두 가지 중 하나:
1. **백엔드**: `company_disclosures.period_from` 컬럼 타입이 `DATE`지만 Go가 `time.Time`으로 bind → SQLite가 ISO 타임스탬프로 저장
2. **프론트엔드**: 원본 값이 어떻든 `new Date(...).toLocaleDateString('ko-KR')` 으로 포맷해주지 않아 raw 문자열이 그대로 나옴

## 해결 방안

### 옵션 A (권장, 프론트 수정)
`frontend/src/routes/company/DisclosureSection.tsx` 에서 period 표시 로직을 포맷 헬퍼로 감싼다:

```tsx
function formatDate(s: string): string {
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  return d.toLocaleDateString('ko-KR')
}
// ...
<span>{formatDate(d.period_from)} ~ {formatDate(d.period_to)}</span>
```

이 방식은 DB에 어떻게 저장되어 있든(date string, ISO datetime) 모두 올바르게 표시한다.

### 옵션 B (백엔드 수정)
`ExecuteLiquidation`에서 `period_from/to`를 순수 date string으로 넘기도록 보장 + repo가 문자열 타입으로 받도록 함. 이미 `time.Now().Format("2006-01-02")`를 쓰고 있으나 SQLite bind 시 Go의 time.Time 자동 변환이 개입할 수 있으므로, 엔티티 필드를 명확히 string으로 유지할 것.

## 영향도

- **기능**: 없음 — 날짜 데이터는 내부적으로 올바름
- **사용자 체감**: 낮음 — 공시 카드의 기간만 못생기게 보임
- **작업 범위**: 아주 작음 — `formatDate` 헬퍼 한 함수 + 1~2곳 사용

## 참고

- 관련 티켓: #024 (disclosure status 하드코딩)
- 발견 PR: #64 스테이지 검증
