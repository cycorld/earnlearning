---
id: 134
title: 게시글 수정 시 신규 멘션 알림 발송
priority: high
type: fix
branch: fix/134-update-post-mention-notify
created: 2026-06-16
---

## 배경
#132 멘션 기능은 CreatePost/댓글 생성 경로에만 알림 발송 로직(`notifyMentions`)이 있음.
`UpdatePost`는 멘션 알림을 보내지 않아, 게시글을 수정하며 새로 추가한 멘션은 대상에게 알림이 가지 않음.
(게시글 213을 `@최용철`→`@[최용철](user:1)`로 수정했으나 알림 미발송으로 발견)

## 작업 내용
- `UpdatePost`에서 **신규 멘션만** 알림 발송 (구본문에 이미 있던 멘션은 재알림 금지 = 수정 스팸 방지)
- 구현: `extractMentions(구본문)` 을 exclude set 으로 `notifyMentions` 에 전달
- `notifyMentions` 에 `exclude map[int]bool` 파라미터 추가, CreatePost/댓글 호출부는 `nil` 전달

## 규칙 (TDD)
- 수정으로 추가된 멘션 → 알림 1건
- 구본문에 이미 있던 멘션 → 재수정해도 재알림 없음
- 동일 본문 재저장 → 신규 알림 없음

## 별도 (1회성)
- 게시글 213 멘션 대상(user:1 최용철) prod 알림 1회성 발송 — 과거 수정 건 소급
