---
id: 006
title: 서브페이지 네비게이션 헤더 고정 (sticky)
priority: medium
type: fix
branch: fix/sticky-nav-header
created: 2026-04-03
---

## 설명
정부과제 상세, 프로필 등 서브페이지에서 스크롤이 길어지면 "← 과제 목록으로" 같은 네비게이션 링크가 화면 밖으로 사라져 다시 돌아가기 힘듦.

## 작업 내용
- 서브페이지의 상단 네비게이션(뒤로가기 + 페이지 제목)을 `sticky top-0`으로 고정
- 적용 대상: GrantDetailPage, UserProfilePage, ConversationPage, BusinessCardPage 등
- 기존 Header와 겹치지 않도록 `top` 값 조정
