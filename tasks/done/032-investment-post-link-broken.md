---
id: 032
title: 투자유치 공지 자동 포스트 링크 경로 오류(/investment → /invest/:id)
priority: high
type: fix
branch: fix/investment-post-link
created: 2026-04-17
---

## 문제
프로덕션에서 투자유치가 시작되면 피드에 자동 공지 포스트가 생성되는데, 본문/링크가 `/investment` 로 되어 있어 클릭 시 404가 뜬다. 프론트엔드 라우트는 `/invest/:id`이다.

## 작업
1. 백엔드에서 투자유치 공지 포스트를 생성하는 지점 찾아 경로를 `/invest/:id`로 수정
2. 회귀 테스트 추가 (포스트 본문/링크가 올바른 경로인지 검증)
3. 프로덕션 DB의 기존 포스트 데이터도 `/investment` → `/invest/:id` 로 UPDATE
4. changelog 추가

## 확인 루트
- CLAUDE.md 알림 체크리스트: `investment` reference_type → `/invest/:id` 매핑이 이미 되어 있음 → 포스트 쪽에서만 어긋났을 가능성
