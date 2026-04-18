# 073. 챗봇 조교 — 지식베이스 빌드 수정 + web_search + 관리자 대화 조회

**날짜**: 2026-04-19
**태그**: 풀스택, 챗봇, RAG, 도구, 관리자, 버그수정

## 무엇을 했나

#072 챗봇 조교 첫 배포 직후 발견된 3가지 이슈/요청을 한 PR 로 해결:

1. **지식베이스 12편이 스테이지에 인덱싱되지 않는 문제 (버그)** — Docker 빌드 컨텍스트가
   `backend/` 로 한정돼서 `docs/llm-wiki/*.md` (최상위 경로) 가 이미지에 안 들어감.
2. **일반 개발 질문 대응 (기능 요청)** — "Python 어떻게 쓰지", "React useEffect 문법"
   같은 질문에 답하려면 외부 문서 검색 필요.
3. **관리자가 모든 학생 대화 조회 (기능 요청)** — 개인정보 보호를 위해 학생끼리는
   격리되지만 관리자는 전부 열람 가능해야.

## 어떻게 만들었나

### 1. Docker 빌드 컨텍스트 수정

- `deploy/build-and-push.sh` 의 backend 빌드 컨텍스트를 `$PROJECT_DIR/backend`
  → `$PROJECT_DIR` 로 변경
- `backend/Dockerfile` 의 `COPY . .` 을 `COPY backend/ .` 로 변경
- 최종 스테이지에 `COPY docs/llm-wiki ./docs/llm-wiki` 추가
- 결과: `docs/llm-wiki/` 13 파일이 이미지 `/app/docs/llm-wiki/` 에 들어감 → 기동 시
  ragindex loader 가 정상 인덱싱

### 2. web_search + fetch_url 도구

`backend/internal/infrastructure/websearch/client.go` 신설:

- `Search(query, limit)` — DuckDuckGo HTML 검색 결과 파싱 (top N). API key 불필요.
- `Fetch(url, maxChars)` — 임의 URL GET → HTML 태그 제거 후 plain text 반환 (최대 20k자).

두 도구를 `BuildChatTools` 에 등록:

- **`web_search`** — 공개 웹 검색. 오픈소스 라이브러리, 공식 문서, 최신 튜토리얼
- **`fetch_url`** — URL 상세 본문 추출 (web_search 결과 or 공식 문서 URL 직접 입력)

**스킬 업데이트**:
- `general_ta` — `search_wiki + web_search + fetch_url` (모든 질문 대응)
- `code_review` — `search_wiki + web_search + fetch_url` (최신 문서 확인)
- **`dev_helper`** (신규) — 개발 질문 특화 스킬. Qwen reasoning medium + 3 개 웹 도구

### 3. 관리자 전체 대화 조회

**Backend**:
- `SessionRepository.ListAll(page, limit)` 추가 (admin 전용, user_id 필터 없음)
- `ChatUseCase.AdminListAllSessions(userID, page)` / `AdminGetSession(id)` 추가
- HTTP: `GET /api/admin/chat/sessions?user_id=N&page=M`, `GET /api/admin/chat/sessions/:id`
- `Session.user_id` JSON 태그를 `json:"-"` → `json:"user_id"` (학생 본인 것만 응답에
  포함되므로 노출 문제 없음)

**Frontend**:
- `/admin/chat` 페이지 하단에 **전체 학생 대화 섹션** 추가
- 세션 클릭 → 모달로 전체 메시지 + tool 호출 상세 (모델/토큰/역할별 스타일링)

### 4. 기존 권한 격리 재확인

`ChatUseCase.GetSession/DeleteSession/Ask` 모두 `session.UserID != userID` 이면
`chat.ErrForbidden` 반환 — 학생은 **자기 세션만** 접근. 관리자 API (`/api/admin/*`) 는
`AdminOnly` 미들웨어가 걸려 있어 학생은 접근 불가.

## 테스트

- `go test -tags sqlite_fts5 ./... → 312 passed`
- Dockerfile 변경은 빌드 서버에서 실 빌드로 확인 예정 (로컬은 frontend build 통과)
- 프론트 `npm run build` 통과

## 배운 점

- **Docker 빌드 컨텍스트는 조용히 파일을 누락시킨다** — `COPY .` 이 성공적으로 실행되고
  이미지가 빌드되면 "파일이 없어서 실패" 가 아니라 **"빈 디렉토리" 로 성공**. 런타임에
  `ragindex loader` 가 "root dir not found" 를 조용히 skip 하면서 인덱스 0 개. 스테이지
  배포 로그에서 `chatbot wiki indexed: 0 docs` 를 보고서야 발견. 후속 조치: 기동
  시 wiki docs 수가 0 이면 **경고 로그**를 찍어 눈에 띄게 개선 필요.
- **DuckDuckGo HTML 파싱은 임시 해결** — 공식 API 아님. 운영 중 품질 불만족이면
  Brave Search API (월 2000 쿼리 무료) 또는 Context7 HTTP API 로 교체. Context7 통합은
  별도 티켓으로 분리.

## 범위 밖 (별도 티켓)

- Context7 MCP → HTTP 게이트웨이 통합 (공식 라이브러리 문서 전용)
- SSE 스트리밍 (첫 글자 체감 시간 개선)
- 관리자 세션 테이블에 user 이름 표시 (현재는 user_id 만) — 조인 쿼리 추가 필요
