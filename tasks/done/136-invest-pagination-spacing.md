---
id: 136
title: 투자처 목록 페이지네이션 + 카드 세로 간격 수정
priority: medium
type: fix
branch: fix/136-invest-pagination-spacing
created: 2026-06-29
---

## 배경
- 투자 라운드(투자처) 목록이 20개까지만 노출. 백엔드는 페이지네이션(limit/page, max 50)을 지원하지만 프론트가 `?status=open`만 보내 기본 limit=20 의 첫 페이지만 가져옴 → 21번째부터 안 보임.
- 라운드 탭 카드들이 세로 간격 없이 붙어 보임. 카드를 감싼 `<Link>`(=inline `<a>`)에 `space-y-3`(margin-top)가 먹지 않음. portfolio/dividends 탭은 `<Card>` 직접 렌더라 정상.

## 작업 내용
- [x] Frontend: `fetchAllOpenRounds()` 추가 — `limit=50` 으로 페이지 순회하며 `total` 전부 수집 (50페이지 안전 상한).
- [x] Frontend: 라운드 카드 `<Link>` 에 `className="block"` → inline anchor 를 block 으로, `space-y-3` 간격 복구.
- [x] 회귀 테스트 (TDD): 한 페이지(50개) 초과 시 전 페이지 수집 / 라운드 링크 `block` 클래스 유지. (InvestPage.test.tsx, 8/8 통과)
