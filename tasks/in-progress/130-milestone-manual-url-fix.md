---
id: 130
title: 평가지표 MVP 링크 직접 수정이 자동 동기화에 덮어써지는 버그 수정
priority: high
type: fix
branch: fix/130-milestone-manual-url-fix
created: 2026-06-12
---

## 배경
학생이 평가지표 페이지에서 잘못 자동 집계된 MVP1/2 링크를 "다시 제출"로 직접 수정해도,
페이지 재조회 시 `SyncAuto`가 회사/정부과제에서 추출한 URL로 도로 덮어씀 (수정이 안 되는 것처럼 보임).
인접 버그: rejected 자동 항목도 같은 URL로 매 조회마다 Upsert → status가 pending으로 리셋되고 교수 코멘트(admin_note) 삭제됨.

(참고: admin 승인 페이지는 /admin/milestones "평가지표 매트릭스"로 이미 존재 — 추가 작업 불필요)

## 작업 내용
- [x] SyncAuto: `source_type=manual` 항목은 자동 동기화로 덮어쓰지 않음 (학생 직접 제출 우선)
- [x] SyncAuto: 기존 항목과 URL·source가 동일하면 Upsert 스킵 (rejected 상태·admin_note 보존)
- [x] 회귀 테스트: 수동 수정 후 재조회 시 URL 유지 / reject 후 재조회 시 상태·코멘트 유지 / 자동 URL 변경 시 갱신은 계속 동작
