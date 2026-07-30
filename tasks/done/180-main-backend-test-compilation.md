---
id: 180
title: main 백엔드 전체 테스트 컴파일 복구
priority: high
type: fix
branch: fix/main-test-failures
created: 2026-07-24
---

프로덕션 인터페이스 변경을 따라가지 못한 테스트 fake와 호출부를 갱신해 Go 1.24 및 `sqlite_fts5` 태그 환경의 전체 백엔드 테스트를 복구한다.