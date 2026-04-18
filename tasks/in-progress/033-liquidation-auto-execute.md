---
id: 033
title: 청산 안건 — 세금 20% UX 안내 + 가결 시 자동 집행
priority: high
type: feat
branch: feat/liquidation-auto-execute
created: 2026-04-18
---

## 배경
- 청산 집행(#023)은 구현되어 있으나, 가결 후 owner가 `POST /companies/:id/proposals/:pid/execute-liquidation` 을 **수동 호출**해야 분배가 이루어진다.
- 주주 입장에서 "투표 가결"과 "자산 분배"의 연결이 불명확하며, owner가 집행을 잊으면 주주 자산이 회사 지갑에 잠긴다.
- 또 청산 안건을 상정할 때 **세금 20% 공제 후 분배** 된다는 사실이 UI에 미노출이라, 주주가 의사결정에 필요한 정보를 충분히 받지 못함.

## 작업
1. **백엔드 — 자동 집행**: `closeProposal` 에서 청산 안건이 passed 되면 즉시 `ExecuteLiquidation` 을 자동 호출. (owner 대신 proposer의 user_id 사용). 실패해도 proposal 은 passed 상태 유지 (idempotent). 실행 후 결과를 notification body에 첨부.
2. **백엔드 — 세금 안내 자동 prefix**: 청산 안건 생성 시 `description` 앞에 세금 20% 공제 안내 문구를 자동 삽입. (운영자가 수동으로 추가하지 않아도 항상 노출)
3. **프론트엔드 — 청산 제안 폼 안내 배너**: 청산 안건 타입 선택 시 세금 20% 공제 안내 + 주주별 분배 계산 예시 표시.
4. **회귀 테스트**:
   - 청산 안건 가결 → 자동 집행 → 회사 dissolved + 분배금 지급 확인
   - 세금 안내 prefix 포함 확인
5. **changelog** 058 추가

## 주의사항
- `#022` 공지 포스트 / `#023` 청산 기능은 기존대로 유지 (세금 20% 정책 불변)
- 이미 청산된 CND(회사 26)는 수동 집행 이력 있음 — 이번 개선과 무관
