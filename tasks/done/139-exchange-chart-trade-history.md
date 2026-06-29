---
id: 139
title: 거래소 종목 상세 — 체결 차트 + 거래이력 + 주문 UX 개선
priority: medium
type: feat
branch: feat/139-exchange-chart-history
created: 2026-06-30
---

거래소 상세 페이지(호가창)를 "종목 상세"로 재설계. 체결 내역 리스트 + 가격 차트 추가, 매수/매도 주문을 편하게 쓸 수 있게 개선.

## 화면 기획
1. 헤더: 회사 로고+이름, 현재가(大) + 등락률(▲▼ 색상)
2. 가격 차트: 체결가 추이 (자체 SVG area 차트, 차트 라이브러리 없음)
3. 주문 패널: [매수]/[매도] 탭 토글(드롭다운 대체), 수량·가격, 총액, 호가 클릭 시 가격 자동입력
4. 호가창 + 체결 내역(거래이력) 나란히 표시
5. 내 주문 (이 종목)

## 백엔드
- 신규: `GET /exchange/trades/:companyId?limit=` — 체결 내역(차트 데이터원)
- 버그 수정: `GetOrderbook` 핸들러가 `intParam(c,"id")` 사용 → 라우트는 `:companyId` → 항상 400. 오더북이 한 번도 렌더된 적 없음. `companyId`로 수정.
- TDD: `GetCompanyTrades` repo 테스트 먼저 작성

## 검증
- 백엔드 스모크 + repo 테스트 통과
- 프론트 타입체크/빌드 통과
