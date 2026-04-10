---
id: 017
title: 기업 정보 수정 (이름/이미지) + 다른 학생 기업 둘러보기
priority: high
type: feat
branch: feat/company-edit-and-browse-all
created: 2026-04-10
---

## 배경

스테이지 사용 중 발견된 두 가지 요구:

1. **기업 정보 수정 시 이름 변경 안 됨**: 프론트는 `name` 을 보내는데
   백엔드 핸들러는 `description` / `logo_url` 만 받음 → name 은 무시되거나
   에러 없이 저장 안 됨. 또한 로고 이미지 수정 UI 가 detail page 편집 폼에 없음.
2. **학생이 자기 회사만 볼 수 있음**: 회사 메뉴에서 본인 소유 회사만 표시.
   다른 학생 회사를 둘러볼 수 없음 → 투자, 비즈니스 카드 교환 등의 학습 활동 제약.

## 목표

### 1. 기업 정보 수정
- [ ] 백엔드 `UpdateCompany` usecase / handler 가 `name` 도 받도록 확장
- [ ] 백엔드 `UpdateCompany` usecase / handler / repo 가 `logo_url` 도 받도록 확장
  (현재 repo.Update 는 description/logo_url/business_card 만 SET 하지만 usecase 가
   logo_url 안 넘겨주고 있음)
- [ ] 이름 변경 시 UNIQUE 제약 충돌 → 친절한 에러 매핑 (`ErrDuplicateName` → 409)
- [ ] 프론트 detail page 편집 다이얼로그에 로고 업로드 (또는 URL 입력) 추가
- [ ] 변경 후 detail page 가 새 값 즉시 반영

### 2. 학생용 전체 기업 목록
- [ ] 백엔드 `GET /companies` (approved 학생 가능, OAuth read:company)
  - 기존 admin 전용 `GET /admin/companies` 와 별개
  - 페이지네이션 (선택), 정렬은 created_at DESC
  - 응답에 `is_owner` 또는 owner 정보 포함 (탭 분리 UI 위해)
- [ ] 프론트 `CompanyListPage`:
  - "내 회사" / "전체 기업" 탭 또는 섹션 분리
  - 모든 회사 카드 표시 (이름, 로고, 설명 일부, 소유자 이름)
  - 각 카드 클릭 시 detail page 진입

## TDD

### 회귀 테스트 (백엔드)
- [ ] `TestUpdateCompany_NameChange_Success`
- [ ] `TestUpdateCompany_NameChange_Duplicate` → 409 ErrDuplicateName
- [ ] `TestUpdateCompany_LogoURLChange_Success`
- [ ] `TestUpdateCompany_NotOwner_Forbidden`
- [ ] `TestListAllCompanies_Approved` → 모든 회사 반환, 본인 + 타인 포함
- [ ] `TestListAllCompanies_NoAuth` → 401
- [ ] smoke test 에 `GET /api/companies` 추가

### 프론트엔드
- [ ] CompanyListPage 의 "전체 기업" 탭 렌더 + 카드 클릭 시 detail 이동
- [ ] CompanyDetailPage 편집 폼에 로고 업로드 input

## 검증

- [x] go test 통과
- [x] frontend tsc + build + vitest 통과
- [ ] 스테이지 배포
- [ ] 브라우저 검증:
  - 기업 이름 변경 → 페이지에 즉시 반영
  - 로고 이미지 변경 → 즉시 반영
  - 다른 학생 회사 카드 클릭 → detail 진입
  - 이름 중복 시 친절한 에러 표시

## 비-목표

- 회사 검색/필터 (이번 작업에 안 함)
- 회사 삭제 (해체) 기능
- 다른 학생 회사 수정 (본인 회사만 수정 가능)
- 회사 카테고리/태그
