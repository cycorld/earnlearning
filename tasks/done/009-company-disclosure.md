---
id: 009
title: 회사 공시 + 정부 수익금 입금 시스템
priority: high
type: feat
branch: feat/company-disclosure
created: 2026-04-03
---

## 설명
학생들이 1주일 동안의 회사 성과를 공시(disclosure)로 정리해서 올리면, 관리자(교수)가 팩트 체크 후 정부에서 수익금을 회사 법인 계좌(company_wallets)에 입금한다.

## 핵심 흐름
1. **학생(대표)**: 회사 상세 페이지에서 "공시 작성" → 1주간 성과 내용 작성 (마크다운)
2. **관리자 리뷰**: 공시 목록에서 내용 확인 → 팩트 체크 → 수익금 금액 결정 → 승인
3. **승인 시**: 정부(관리자 지갑)에서 회사 법인 계좌(company_wallets)로 수익금 입금
4. **학생 알림**: 공시 승인 + 수익금 입금 알림

## 계좌 구조 (기존)
- **개인 지갑**: `wallets` 테이블 (user별)
- **법인 계좌**: `company_wallets` 테이블 (company별, 별도 balance)
- **법인 거래내역**: `company_transactions` 테이블

## DB 스키마 (신규)
```sql
CREATE TABLE IF NOT EXISTS company_disclosures (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id  INTEGER NOT NULL REFERENCES companies(id),
    author_id   INTEGER NOT NULL REFERENCES users(id),
    content     TEXT NOT NULL,
    period_from DATE NOT NULL,        -- 공시 기간 시작
    period_to   DATE NOT NULL,        -- 공시 기간 종료
    status      TEXT NOT NULL DEFAULT 'pending',  -- pending, approved, rejected
    reward      INTEGER DEFAULT 0,     -- 승인 시 지급된 수익금
    admin_note  TEXT DEFAULT '',       -- 관리자 코멘트
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## API
- `POST /companies/:id/disclosures` — 공시 작성 (대표만)
- `GET /companies/:id/disclosures` — 공시 목록 조회
- `GET /companies/:id/disclosures/:did` — 공시 상세
- `POST /admin/disclosures/:did/approve` — 승인 + 수익금 입금 (관리자)
- `POST /admin/disclosures/:did/reject` — 거절 (관리자)

## 프론트엔드
- 회사 상세 페이지에 "공시" 탭/섹션 추가
- 공시 작성 폼 (기간 선택 + 마크다운 에디터)
- 공시 목록 (상태 뱃지 + 수익금 표시)
- 관리자 페이지에 공시 리뷰 화면 (승인/거절 + 수익금 입력)
