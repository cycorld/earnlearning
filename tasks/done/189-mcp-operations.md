---
id: 189
title: MCP 운영 도구 P0~P2 확장
priority: high
type: feat
branch: feat/189-mcp-operations
created: 2026-07-29
---

기존 router.go API만 table-driven MCP 도구로 노출한다. 모든 변경 요청은 도구명과 정확히 같은 confirm을 요구하며, OAuth 차단·multipart·대량 파괴 작업은 제외한다.
