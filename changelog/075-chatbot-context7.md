# 075. 챗봇 Context7 통합 + AskOutput JSON 태그 + maxToolHops 6

**날짜**: 2026-04-19
**태그**: 챗봇, 도구, Context7, 버그수정

## 무엇을 했나

### Context7 HTTP API 통합
`context7.com/api/v1/*` 호출용 클라이언트 + 2개 도구:
- `context7_search(query, limit)` — 라이브러리 검색 → id 반환
- `context7_docs(library_id, topic, tokens)` — 공식 문서 + 코드 예제

`general_ta` / `code_review` / `dev_helper` 스킬이 이 도구로 React / TanStack Query / Next.js / Go / Python 공식 문서를 정확하게 인용.

### 기존 이슈 수정
- `AskOutput` 구조체에 `json:"message"` / `json:"tool_logs"` 태그 추가 (프론트에서 PascalCase 로 역직렬화되던 버그)
- `maxToolHops` 4 → 6 (여러 도구 호출 루프 완료 전 cutoff 되던 문제)
- `web_search` 가 DDG 봇 탐지로 비어있을 때 "알고 있는 공식 URL 로 fetch_url 호출" 안내 추가

### 설정
- `CONTEXT7_API_KEY` env 로 주입 (없으면 context7 도구 비활성)
- 키는 context7.com 대시보드에서 발급. 운영 서버 stage/.env.stage + prod/.env.prod 에 추가 필요
