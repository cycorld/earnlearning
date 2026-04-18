---
id: 070
title: /grant 및 기타 리스트 페이지 카드 간격 전수 조사 + 수정
priority: medium
type: fix
branch: fix/list-card-spacing-round2
created: 2026-04-18
---

## 배경
`https://earnlearning.com/grant` 에서 카드들이 서로 여백 없이 붙어 있음.
#066 (리스트 카드 간격 1차 정리) 에서 3 페이지만 잡고 `/grant` 는 놓쳤음.

## 스코프
1. `GrantListPage` 컨테이너 스페이싱 확인 + 수정
2. **비슷한 화면 전수 검사** — routes 하위 모든 `.map` 호출 부모 컨테이너 훑기
3. 기준: 세로 리스트 형태 카드는 `space-y-4` (16px) 또는 `gap-4` 로 통일
4. 발견된 모든 케이스 한 PR 로 수정

## 확인 리스트
- [ ] `/grant` (GrantListPage)
- [ ] 전수 검사 스크립트 재실행 (`space-y-0/1/2` 또는 space-y 없음)
- [ ] 각 후보를 직접 읽고 사용자가 볼 카드 리스트인지 판단 (카드 내부 요소 그룹은 예외)
- [ ] 스크린샷으로 before/after 확인
