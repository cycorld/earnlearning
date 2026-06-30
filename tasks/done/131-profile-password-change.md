---
id: 131
title: 프로필 페이지 비밀번호 변경 기능
priority: medium
type: feat
branch: feat/131-profile-password-change
created: 2026-06-12
---

## 배경
로그인 상태에서 비밀번호를 바꿀 방법이 없음 (#128 비밀번호 찾기는 비로그인용).

## 작업 내용
- [x] Backend: `PUT /api/auth/password` — 현재 비밀번호 검증(bcrypt) + 새 비밀번호(8자+) 재해싱 저장
- [x] Frontend: 프로필 페이지에 "비밀번호 변경" 다이얼로그 (현재/새/확인 3필드)
- [x] 통합 테스트 (TDD): 성공 플로우(구 비번 거부·새 비번 로그인), 현재 비번 오류, 약한 비번, 미인증
