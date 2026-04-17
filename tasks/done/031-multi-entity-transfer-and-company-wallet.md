---
id: 031
title: 송금 주체 확장(개인↔법인) + 법인 Wallet 페이지
priority: high
type: feat
branch: feat/multi-entity-transfer
created: 2026-04-17
---

# 송금 주체 확장(개인↔법인) + 법인 Wallet 페이지

## 배경
현재 송금은 **개인 → 개인** 만 가능. 백엔드 `wallet_usecase.go:239-243`에서 `target_type == "company"` 를 명시적으로 거부함.
회사 상세 페이지는 잔액만 읽기 전용으로 표시하고, 법인 지갑 거래내역/송금 UI가 없어 학생들이 "개인 재정"과 "법인 재정"의 분리를 체감하기 어려움.

## 목표 (3가지 갭 + 법인 Wallet 페이지)

### Gap 1 — 백엔드 Transfer가 법인을 수신자로 허용
- `target_type: "company"` 지원 → `CreditCompanyWallet` 호출
- `target_user_id` 의미를 target_type에 따라 분기 (user일 때는 user_id, company일 때는 company_id로 해석)
- `SearchRecipients` 가 법인도 반환 (`type: "company"` 포함)

### Gap 2 — 법인이 송금 주체가 될 수 있게
- 새 엔드포인트: `POST /api/companies/:id/transfer`
  - 권한: 대표(founder) 또는 공동창업자만
  - 대상: user 또는 company
  - 실패 시 롤백 (트랜잭션)
- TxType 추가: `TxCompanyTransfer` (company_transactions에 기록)

### Gap 3 — 법인 Wallet 페이지 신설
- 라우트: `/company/:id/wallet`
- 회사 상세 페이지에서 "지갑 관리" 버튼으로 진입
- 표시:
  - 법인 잔액 (크게)
  - 거래 내역 (company_transactions)
  - "송금하기" 버튼 (대표/공동창업자만 노출)
- 교육 목적상 **개인 지갑과 시각적으로 다른 톤** 으로 표시 (예: 아이콘/배지, 컬러 차별)

### 백엔드 신규 엔드포인트
- `GET /api/companies/:id/wallet` — 법인 지갑 + 최근 거래
- `GET /api/companies/:id/transactions` — 법인 거래 내역 (페이지네이션)
- `POST /api/companies/:id/transfer` — 법인에서 출금 송금

### 프론트엔드
- `frontend/src/routes/company/CompanyWalletPage.tsx` (신규)
- `CompanyDetailPage.tsx`에 "지갑 관리" 진입 버튼
- `WalletPage.tsx` 수신자 선택 UI에서 법인 배지 표시

### 후속 (이 티켓 범위 밖)
- 기존 지갑 페이지에 드롭다운으로 "개인/○○법인 계좌" 전환 UI → 다음 티켓

## 테스트 (TDD)
- Backend:
  - 개인→법인 송금 성공 (wallets -1000, company_wallets +1000, 트랜잭션 2건)
  - 법인→개인 송금 성공 (company_wallets -500, wallets +500)
  - 법인→법인 송금 성공
  - 법인 송금 권한 없는 유저 거부 (403)
  - 잔액 부족 시 거부 + 롤백
- Frontend:
  - 수신자 검색에서 법인이 나오고 type='company' 로 표시
  - CompanyWalletPage 잔액/거래내역 렌더
  - 대표 계정에서만 "송금하기" 노출

## 알림 연동 체크리스트
- `CreateNotification` 호출 시:
  - `reference_type: "company"` (법인이 수신/발신 주체일 때) — 이미 `/company/:id` 매핑 존재
  - `reference_type: "transaction"` (기존 개인 송금 수신자 알림) — 이미 `/wallet` 매핑 존재
- 법인 지갑 관련 신규 reference_type 필요 시 `NotificationsPage.tsx` 매핑 추가
