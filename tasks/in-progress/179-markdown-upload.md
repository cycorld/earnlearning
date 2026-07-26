---
id: 179
title: Markdown 문서 업로드 지원
priority: high
type: feat
branch: feat/markdown-upload
created: 2026-07-24
---

PRD와 SPEC 같은 `.md` 문서를 에디터에서 선택하고 안전하게 업로드할 수 있도록 한다.
서버에서 확장자, MIME 형식, 파일 크기, 파일명을 검증하고 기존 업로드 형식의 동작을 회귀 테스트한다.