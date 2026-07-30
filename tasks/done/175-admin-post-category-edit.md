---
id: 175
title: 관리자 게시물 수정 시 카테고리(채널) 변경 기능
priority: medium
type: feat
branch: feat/175-admin-post-category-edit
created: 2026-07-23
---

## 배경
관리자는 학생 게시물의 본문/태그를 수정할 수 있지만, 잘못된 채널(카테고리)에 올라온 글을 올바른 채널로 옮길 방법이 없다. `UpdatePostInput`에 `channel_id` 필드 자체가 없음.

## 요구사항
- **관리자만** 게시물 수정 시 채널 변경 가능. 일반 작성자는 본문/태그만 수정 가능 (채널 변경 시도 → 에러).
- 이동 대상 채널은 **존재해야** 하고, 기존 채널과 **같은 classroom** 소속이어야 함 (classroom 경계 보호).
- `channel_id` 미전송 시 기존 채널 유지 (하위 호환).
- 프론트: FeedPage 수정 다이얼로그에 관리자 전용 카테고리 selector 노출, 기존 channels 목록 재사용, 성공 시 로컬 post의 channel 갱신.

## 작업 내역 (TDD)
- [x] backend 회귀 테스트 먼저 작성: 권한(작성자 불가/관리자 가능)·존재하지 않는 채널·classroom 경계·미전송 시 유지
- [x] `UpdatePostInput.ChannelID *int` 추가 + usecase 검증 + repo `UpdatePostChannel`
- [x] frontend: 수정 다이얼로그 관리자 전용 selector + PUT payload `channel_id` + 로컬 갱신
- [x] frontend 테스트: selector 노출 권한, payload, 로컬 갱신, 관리자의 학생 글 수정 버튼 노출
- [x] smoke + backend/frontend 테스트 + 빌드 통과
