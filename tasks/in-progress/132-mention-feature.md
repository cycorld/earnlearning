---
id: 132
title: 게시글/댓글 @멘션 기능 (자동완성 + 멘션 알림 + 알림 탭)
priority: high
type: feat
branch: feat/132-mentions
created: 2026-06-12
---

게시글/댓글 본문에서 `@`로 유저를 멘션하면 멘션된 유저에게 알림이 가고,
알림 페이지에서 "멘션" 탭으로 모아볼 수 있다.

## 요구사항
- `@` 입력 시 자동완성 드롭다운 → 선택 시 user_id 확정 저장 (동명이인 대응, 자유 텍스트 아님)
- 멘션 대상: approved 전체 유저 (~59명)
- 멘션된 유저에게 알림 발송 (새 NotifType `mention`)
- 알림 페이지 탭 분리: 전체 / 멘션
- 멘션 알림 클릭 → 글 상세 이동. 댓글 멘션이면 부모 게시글 상세 + `#comment-<id>` 앵커 스크롤
- 본문 렌더 시 멘션 하이라이트

## 구현 순서 (TDD)
1. 유저 검색 엔드포인트 `GET /api/users/search?q=` (이름/학번 부분일치, approved만)
2. 본문 멘션 마크업 `@[이름](user:ID)` 파싱
3. CreatePost/CreateComment에서 멘션 파싱 → 멘션 유저에 알림 (NotifType `mention` 신규)
4. 알림 목록 `?type=mention` 필터
5. 프론트: 알림 탭 + getReferencePath/getNotifIcon 매핑, @ 자동완성 입력, 멘션 하이라이트, 댓글 앵커 스크롤
