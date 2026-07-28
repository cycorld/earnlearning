---
id: 182
title: 안전한 로컬 stdio MCP 서버
priority: high
type: feat
branch: feat/182-local-mcp-server-final
created: 2026-07-28
---

EarnLearning의 기존 읽기 전용 API를 로컬 AI 도구에서 안전하게 조회할 수 있는 stdio MCP 서버 MVP를 구현한다.

- 기본 설정은 실패 폐쇄(fail-closed)이며 API URL과 bearer token을 환경 변수로만 받는다.
- 기본적으로 localhost API만 허용하고 원격 연결은 명시적 opt-in에서만 허용한다.
- `initialize`, `tools/list`, `tools/call` JSON-RPC 메서드와 health/profile/grants/companies/posts 조회 도구를 제공한다.
- 입력과 조회량을 제한하고 토큰이 오류 및 로그에 노출되지 않도록 테스트한다.
- 배포 및 프로덕션 데이터 변경은 범위에서 제외한다.
