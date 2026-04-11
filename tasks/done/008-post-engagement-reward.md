---
id: 008
title: 게시글 좋아요/댓글 보상 시스템
priority: high
type: feat
branch: feat/engagement-reward
created: 2026-04-03
---

## 설명
게시글에 좋아요/댓글이 달리면 글쓴이에게 가상화폐 보상을 지급한다.

## 규칙
- 좋아요 1개당: 글쓴이에게 10원 보상
- 댓글 1개당: 글쓴이에게 100원 보상
- 자기 글에 자기가 좋아요/댓글 → 보상 없음
- 좋아요 취소 시: 10원 회수 (글쓴이 지갑에서 차감)
- 댓글 삭제 시: 100원 회수

## 작업 내용
- Backend: LikePost에서 좋아요 시 글쓴이 지갑에 credit, 취소 시 debit
- Backend: CreateComment에서 댓글 작성 시 글쓴이 지갑에 credit
- Backend: 댓글 삭제 API 추가 + 삭제 시 보상 회수
- transaction 기록: tx_type "like_reward" / "comment_reward"
- 자기 자신 글에 좋아요/댓글은 보상 제외
- Frontend: 좋아요/댓글 시 토스트로 보상 알림 (선택)
