# Go 버전이 자꾸 달라질 때 — 빌드와 테스트 환경 하나로 맞추기

- 날짜: 2026-07-28
- 관련 작업: #183

## 무엇을 했나요?

Backend가 사용하는 Go를 `1.24.13`으로 고정했습니다. Docker builder도 같은 버전을 사용하며, 로컬 테스트는 `scripts/test-backend.sh`로 실행합니다.

## 왜 필요한가요?

프로젝트는 Go 1.24를 요구하지만 로컬 Ubuntu 기본 Go는 1.22였습니다. 자동 toolchain이 patch 없는 `go1.24`를 찾다가 실패했고, SQLite FTS5 build tag를 빼면 migration 테스트도 실패했습니다.

## 어떻게 해결했나요?

- `go.mod`와 Backend Dockerfile을 Go 1.24.13으로 통일했습니다.
- 테스트 스크립트가 `GOTOOLCHAIN=go1.24.13`과 `sqlite_fts5`를 항상 적용합니다.
- 서버에는 Go를 설치하지 않고, 기존처럼 로컬에서 이미지를 빌드해 배포합니다.

## 사용한 프롬프트

> go 버전 고정 배포 마무리 해줘.

## 배운 점

언어의 major/minor 버전만 맞추는 것보다 patch 버전과 build tag까지 같은 진입점으로 고정해야 재현 가능한 테스트가 됩니다.