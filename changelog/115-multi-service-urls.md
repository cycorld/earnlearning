# 115. 회사 service_url 을 쉼표 구분 다중 URL 지원 (#114 후속)

**날짜**: 2026-05-08
**태그**: 투자, UX, URL, 회사정보

## 배경
#114 에서 회사 service_url 을 노출시켰는데, 한 회사가 여러 채널을 가지는 경우가 흔함 (웹 + Instagram + GitHub …). DB 컬럼 `companies.service_url TEXT` 그대로 두고 **쉼표 구분 문자열**로 다중 URL 저장 (schema 변경 X).

## 추가

### Frontend `lib/urls.ts` — 단일 진실
- `parseServiceUrls(raw)` — 쉼표 split + trim + 빈 piece 제거 → 배열
- `formatServiceUrls(arr)` — 역방향
- `isValidHttpUrl(s)` — http/https 만 허용 (ftp/mailto/javascript: 거부)
- `isValidServiceUrls(raw)` — 다중 URL 한 piece 라도 invalid 면 false
- `shortenUrl(url)` — 표시용 protocol 제거

### Frontend display
- **CompanyDetailPage**: 헤더에 다중 URL 각각 별도 링크
- **InvestPage 리스트 카드**: 첫 URL "서비스 바로가기" 버튼 + 추가 N개 있으면 `+N` 배지
- **InvestDetailPage 헤더**: 모든 URL 별도 링크

### Frontend edit
- **CompanyDetailPage 편집 폼**: input type="text" (단일 URL 강제 풀음) + placeholder `"https://my-app.com, https://instagram.com/myapp"` + hint "쉼표(,) 로 구분"
- 클라이언트 검증: 각 piece 가 valid http/https URL 인지 (저장 전 toast.error)

### Backend validation
`backend/internal/application/company_usecase.go`:
- `validateServiceURLs(raw)` — 쉼표 split → 각 piece url.Parse + scheme check (http/https) + host check
- 빈 문자열 OK (URL 0개 허용)
- `UpdateCompany` usecase 에서 호출 → invalid 시 error 반환 + 정규화된 (trim 된) 문자열 저장

## 회귀 테스트

### Frontend (vitest)
- `lib/urls.test.ts` — 23 tests (parse / format / isValidHttpUrl / isValidServiceUrls / shortenUrl)
- `routes/invest/InvestPage.test.tsx` — `+2` 배지 회귀 테스트 추가

### Backend (Go integration)
- `TestCompanyUpdate_ServiceURL_Multi_Success` — 3개 URL 정규화 round-trip
- `TestCompanyUpdate_ServiceURL_Empty_OK` — 빈 문자열 허용
- `TestCompanyUpdate_ServiceURL_Invalid_Reject` — 5개 invalid 케이스 (protocol 없음 / ftp / javascript / 한 piece invalid / 호스트 없음)

전체: frontend 179 pass · backend 33 pass.

## 미포함 (의도)
- 스키마 변경 X — TEXT 컬럼 그대로 사용
- 다른 회사 필드 변경 X
- 외부 링크 미리보기 (OG 메타 fetch) X
- URL 정렬·드래그 reorder X — 사용자가 입력 순서 그대로 유지
