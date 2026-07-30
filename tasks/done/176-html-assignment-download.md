---
id: 176
title: 사업계획서 첨부에 .html 지원 (다운로드 전용)
priority: high
type: feat
branch: feat/176-html-assignment-download
created: 2026-07-27
---

정부지원금(사업계획서) 제출 첨부에 `.html` 파일을 허용한다.

- 제출 경로: `POST /api/milestones/files` (`type=business_plan`). grant 모듈 자체에는 첨부 필드가 없고, 사업계획서 첨부가 곧 정부지원금 제출 첨부다.
- HTML 은 **절대 same-origin 렌더링되면 안 된다**: 공개 `/uploads` 정적 경로에는 노출하지 않고, 비공개 저장소(`private_uploads`) + 권한 검증 다운로드 엔드포인트로만 제공한다.
- 다운로드 응답은 항상 `Content-Disposition: attachment` + `application/octet-stream` + `X-Content-Type-Options: nosniff` (+ CSP).
- 서버 검증: 확장자 허용, 안전한 파일명, 10MB 상한, 전체 내용 UTF-8 검사 및 NUL 거부, 실패 시 파일 정리.
- 프론트엔드: accept 목록에 `.html` 추가, HTML 은 blob 새 탭 열기 금지(다운로드 강제).
