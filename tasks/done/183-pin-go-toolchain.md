---
id: 183
title: Go toolchain 및 Backend 테스트 계약 고정
priority: high
type: chore
branch: chore/go-1.24.13-toolchain
created: 2026-07-28
---

Go toolchain을 1.24.13으로 고정하고 SQLite FTS5가 필요한 Backend 테스트를 단일 스크립트로 실행한다. 로컬 개발, Docker build, CI가 같은 Go patch 버전과 build tag를 사용하도록 맞춘다.
