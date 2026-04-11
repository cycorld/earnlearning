# 048. 회사 공시 + 서비스 URL + 정부 수익금 입금 시스템

> **날짜**: 2026-04-12
> **태그**: `feat`, `회사`, `공시`, `백엔드`, `프론트엔드`

## 무엇을 했나요?

회사 정보에 **서비스 URL** 필드를 추가하고, 학생(대표)이 주간 성과를 **공시**로 작성하면 교수(관리자)가 팩트체크 후 **수익금을 법인 계좌에 입금**하는 시스템을 만들었습니다.

## 서비스 URL

회사에 자신이 만든 서비스 URL을 등록할 수 있습니다. 등록하면 회사 상세 페이지 헤더에 링크가 표시됩니다.

- Backend: Company 엔티티에 `service_url` 필드 추가
- DB: `ALTER TABLE companies ADD COLUMN service_url` 마이그레이션
- Frontend: 수정 다이얼로그에 URL 입력, 헤더에 외부 링크 아이콘과 함께 표시

## 공시 시스템

### 흐름
1. **학생(대표)**: 회사 상세 → "공시" 섹션 → "작성" 버튼 → 기간 + 성과 내용 작성
2. **관리자**: 관리자 페이지 → "공시 관리" → 내용 확인 → 수익금 결정 → 승인/거절
3. **승인 시**: 정부(시스템)에서 회사 법인 계좌로 수익금 자동 입금

### API
- `POST /companies/:id/disclosures` — 공시 작성 (대표만)
- `GET /companies/:id/disclosures` — 공시 목록
- `GET /admin/disclosures` — 관리자 전체 조회
- `POST /admin/disclosures/:did/approve` — 승인 + 수익금 입금
- `POST /admin/disclosures/:did/reject` — 거절

### DB
`company_disclosures` 테이블: id, company_id, author_id, content, period_from, period_to, status(pending/approved/rejected), reward, admin_note

## 어떻게 만들었나요?

TDD로 진행했습니다.

1. 실패하는 테스트 작성 (service_url 업데이트, 공시 생성→승인 흐름, 거절, 권한 검증, 관리자 목록)
2. 백엔드 구현: entity → repository → usecase → handler → router 순서
3. 프론트엔드: DisclosureSection 컴포넌트 (작성/목록/상세 다이얼로그), AdminDisclosuresPage

## 배운 점

- `ALTER TABLE ... ADD COLUMN`으로 프로덕션 DB 안전하게 마이그레이션
- 공시 승인 시 법인 계좌 입금은 `CreditCompanyWallet`로 트랜잭션 기록과 함께 처리
- 소유자만 공시를 작성할 수 있도록 `ErrNotOwner` 검증 필수
