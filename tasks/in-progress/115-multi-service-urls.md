---
id: 115
title: 회사 service_url 을 쉼표 구분 다중 URL 지원 (#114 후속)
priority: medium
type: feat
branch: feat/multi-service-urls
created: 2026-05-08
---

## 배경
#114 에서 회사 service_url 을 투자 페이지에 노출시켰는데, 한 회사가 **여러 서비스 URL** 을 가지는 경우가 흔함:
- 웹사이트 + Instagram + GitHub
- 메인 도메인 + 베타
- 한국어 + 영문 사이트

DB 칼럼 (`companies.service_url TEXT DEFAULT ''`) 은 그대로 두고 **쉼표 구분 문자열**로 저장 (schema 변경 X, 마이그레이션 X).

## 작업

### 새 helper
- `frontend/src/lib/urls.ts`:
  - `parseServiceUrls(raw)` — 쉼표 split + trim + empty filter
  - `isValidHttpUrl(s)` — http/https URL 검증
- `frontend/src/lib/urls.test.ts` — 단위 테스트

### Frontend display
- `CompanyDetailPage` 헤더: 다중 URL 을 각각 렌더 (현재 1개만)
- `InvestPage` 리스트 카드: 첫 URL "서비스 바로가기" + 추가 URL 개수 (e.g., "+2 더보기")
- `InvestDetailPage` 헤더: 다중 URL 모두 표시

### Frontend edit
- `CompanyDetailPage` 편집 폼: input type="text" (URL 단일 검증 풀고) + placeholder "https://..., https://..." + 보조 hint "쉼표(,) 로 구분"
- 클라이언트 사이드 validation: 각 piece 가 valid http/https URL 인지

### Backend validation
- `UpdateCompany` usecase: 쉼표 split → 각 piece 가 valid URL 인지 체크. 빈 문자열은 OK.

### 회귀 테스트
- helper unit tests
- CompanyDetailPage 편집·디스플레이 회귀
- InvestPage 다중 URL 표시

## 미포함
- Schema 변경 X (TEXT 그대로)
- 다른 회사 필드 변경 X
- 외부 링크 미리보기 X
