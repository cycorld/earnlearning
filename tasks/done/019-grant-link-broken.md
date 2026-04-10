---
id: 019
title: 공지 게시글의 정부과제 바로가기 링크 동작 안 함
priority: high
type: fix
branch: fix/grant-link-broken
created: 2026-04-10
---

## 현상

프로덕션 공지 게시글에 정부과제 바로가기 링크가 있는데 클릭해도 페이지 이동이 안 됨.
PWA 이슈일 수 있음.

## 조사 계획

1. 프로덕션에서 browse 로 공지 게시글 확인 → 링크 형태 파악
2. 링크 클릭 시 동작 확인 (React Router vs 외부 링크 vs 앵커)
3. 프론트 라우터에 /grants 경로 등록 여부 확인
4. 수정 + 테스트 + 스테이지 배포 + prod promote
