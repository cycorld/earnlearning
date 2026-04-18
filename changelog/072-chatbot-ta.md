# 072. 챗봇 조교 — 전 페이지 플로팅 위젯 + RAG + 스킬 + 관리자 편집

**날짜**: 2026-04-19
**태그**: 풀스택, LLM, 챗봇, RAG, FTS5, 스킬, 관리자

## 무엇을 했나

LMS 의 모든 승인 학생 페이지 **우하단**에 말풍선 플로팅 버튼(`<ChatDock />`)을 띄워
Qwen3.6 기반 **챗봇 조교**에게 질문할 수 있게 했다. 학생이 "지갑 잔액이 얼마야?",
"청산하면 세금 얼마 빠져?", "LLM API 키 어떻게 발급받아?" 같은 질문을 하면:

1. **적절한 스킬**(일반 조교 / 지갑 도우미 / LLM API 도우미 등)이 자동으로 활성화
2. 필요하면 **도구**(wallet API, grant API, LLM usage API, 위키 검색) 를 실제 호출
3. **마크다운 기반 LLM Wiki**(언러닝 노션 가이드 12편 + LLM API 가이드 = 13 편 초기
   임포트) 에서 BM25 검색으로 근거 문서를 찾아 답변
4. 응답과 도구 호출 전부를 DB 에 영구 저장 (학생의 학습 히스토리)

관리자는 **"스킬 설계자"** 메타-스킬과 대화하면서 새 스킬을 자연어로 만들 수 있음.
확정되면 `save_skill_draft` 툴이 자동으로 DB 에 upsert.

## 왜 필요했나

1. **문서 검색 허들 제거** — 언러닝에는 이미 노션 가이드 12편이 있지만, 학생이
   질문이 떠오른 그 순간 노션에 가서 적절한 페이지를 찾아 읽는 건 허들이 있음.
   페이지 우하단에 말풍선 하나 누르면 자연어로 물어볼 수 있게 함.

2. **학생 개인 데이터 접근** — "내 지갑 잔액이 얼마지", "내가 지금까지 LLM API 에
   얼마 썼지" 같은 질문은 위키로 답할 수 없는 "계정 특화" 질문. 챗봇이 툴로
   실제 API 를 호출해서 답함.

3. **운영 확장성** — 관리자가 새로운 FAQ/특화 도우미를 만들고 싶을 때 개발자 없이
   대화로 스킬 초안을 뽑고 "저장" 버튼만 누르면 바로 반영. 개발자 병목 제거.

4. **수업 컨텍스트 적응** — 정식 강의 콘텐츠는 아직 시스템에 없지만, 노션 가이드를
   RAG 지식베이스로 당겨왔으므로 이미 첫날부터 쓸모 있는 답을 준다. 강의 콘텐츠는
   `docs/llm-wiki/course/` 아래에 md 로 추가하면 자동 인덱싱됨.

## 어떻게 만들었나

### 백엔드 계층 분리

| 레이어 | 파일 | 책임 |
|---|---|---|
| 도메인 | `internal/domain/chat/{entity,repository,errors}.go` | Session / Message / Skill / WikiDocMeta / 에러 |
| 퍼시스턴스 | `internal/infrastructure/persistence/chat_repo.go` | SQLite 4 리포지토리 + FTS5 래퍼 |
| RAG 인덱서 | `internal/infrastructure/ragindex/loader.go` | md 파일 → FTS5 동기화 (`Sync()`) |
| LLM 호출 | `internal/infrastructure/llmproxy/chat.go` + `chat_adapter.go` | `/v1/chat/completions` (non-streaming) |
| 툴 레지스트리 | `internal/application/chat_tools.go` | 7개 도구 (search_wiki / 지갑 / 회사 / 그랜트 / LLM / skill-designer) |
| 유스케이스 | `internal/application/chat_usecase.go` | `Ask()` 루프: 스킬 선택 → 툴 호출 루프 → 저장 |
| 시드 | `internal/application/chat_seed.go` | 기동 시 7개 기본 스킬 upsert |
| HTTP | `internal/interfaces/http/handler/chat_handler.go` | 학생/관리자 엔드포인트 |

**import cycle 회피**: LLM 요청/응답 타입을 `application/chat_llm_types.go` 에 두고,
llmproxy 는 이 타입을 구현하는 어댑터를 제공 (기존 LLM 도메인과 같은 패턴).

### DB 스키마 (테이블 5 + FTS5 1)

```
chat_sessions          — 세션 메타 + 활성 스킬
chat_messages          — 메시지 전문 + tool_calls JSON + 사용 토큰
chat_skills            — 스킬 정의 (slug unique, system_prompt, tools_allowed, wiki_scope)
chat_wiki_meta         — 위키 문서 메타 (path, title, notion_page_id, synced_at)
chat_wiki_docs (FTS5)  — BM25 검색용 가상 테이블 (body 인덱싱)
chat_usage             — 학교 부담 사용료 일자별 집계 (관리자 모니터링용)
```

### FTS5 빌드 태그

`go-sqlite3` 의 FTS5 는 `sqlite_fts5` 빌드 태그가 필요함. `Dockerfile` 의 `go build`
에 `-tags "sqlite_fts5"` 추가. 통합 테스트도 `go test -tags sqlite_fts5 ./...` 로 실행.

### 스킬 라우터 (간단 버전)

1. 요청에 `skill_slug` 명시 → 그 스킬 사용
2. 세션에 active skill 있으면 → 재사용
3. 없으면 → `general_ta` fallback

LLM-as-router (질문 분류) 는 Phase 2. 현재는 사용자가 드롭다운으로 직접 스킬 선택.

### 도구 호출 루프

OpenAI-compatible `tools` 포맷으로 Qwen 에 전달. assistant 가 `tool_calls` 를
돌려주면 각 도구를 Go 에서 실행하고 `role=tool` 메시지로 다음 턴에 주입. 최대 4 hop
(`maxToolHops=4`) 후에는 강제로 최종 응답 처리.

**안전 장치**: 각 도구 실행 전에 스킬의 `tools_allowed` 재확인 (LLM 이 허용되지 않은
도구를 호출해도 Go 에서 거부).

### 지식베이스 — `docs/llm-wiki/`

루트에 13 개 파일 (노션 가이드 12 편 + LLM API 가이드 1 편). 각 파일:
```yaml
---
title: ...
notion_page_id: ...
synced_at: 2026-04-19T00:00:00Z
---
...본문...
```

서버 기동 시 `ragindex.Loader.Sync()` 가 전체 디렉토리 재귀 스캔 → 파일이 삭제됐으면
고아 정리 → FTS5 upsert. 관리자 `/admin/chat` 페이지에 "재인덱싱" 버튼으로 수동 재로드.

### 프론트엔드

- **`<ChatDock />`** (`components/chat/ChatDock.tsx`) — `MainLayout` 에 마운트 → 모든
  승인 페이지에서 우하단 FAB 표시. 클릭 시 슬라이드업 패널. 모바일은 하단 82vh,
  데스크탑은 우하단 420×600 고정창.
- 스킬 드롭다운 (학생용 제외 `admin_only` 필터), 모드 토글 (fast/deep), 관리자는
  "관리자 (깊이 자동)" 배지만 표시.
- 메시지 렌더:
  - user → 오른쪽 primary 버블
  - assistant → 왼쪽 muted 버블 + 마크다운 렌더 + 호출한 도구 칩(🔧)
  - tool → `<details>` 접힘 블록 (디버깅용)
- `/admin/chat` — 스킬 목록 (enable/disable 토글) + 위키 문서 테이블 + 재인덱싱 버튼
- 관리자 홈(`/admin`)에 "챗봇 관리" 카드 추가

### 과금 정책 (#071 Q2 학생 부담 없음)

- LMS 서버의 기존 `LLM_ADMIN_API_KEY` (학교 계정) 로 llm.cycorld.com 호출
- 학생 개인 지갑은 **건드리지 않음**
- `chat_usage` 테이블에 일자별·학생별 토큰·원화 집계 (운영진 모니터링용, 예산 추적)
- 학생 개인 API 키(`/llm/me`) 는 **직접 코드 짤 때** 용도로 별도 유지

## 테스트

- **RAG 로더**: `ragindex/loader_test.go` — md 파일 → FTS5 3 시나리오 (sync 모두 /
  orphan 정리 / 누락 디렉토리 graceful skip)
- **전체 backend**: `go test -tags sqlite_fts5 ./... → 312 passed`
- **smoke**: 통합 smoke 24 passed (FTS5 태그 필요)
- **프론트**: 기존 125 tests 그대로 통과, chat dock 은 런타임 스모크로 검증

## 리스크 / 후속 과제

- **LLM tool-call 신뢰도** — Qwen3.6 의 tool-use 정확도는 Opus 수준은 아님. 부정확한
  호출이 나오면 스킬 `system_prompt` 를 더 엄격하게 조정하는 운영 과제로 풀어가야 함.
- **스트리밍 미구현** — 첫 응답까지 대기 체감이 있음. SSE 스트리밍을 다음 이터레이션.
- **Git push 자동화 미구현** — 관리자가 위키를 편집해도 git 커밋은 수동. 추후 deploy
  키 기반 `git commit + push` 자동화는 별도 티켓.
- **LLM-as-router 미구현** — 학생이 드롭다운으로 스킬을 선택. 질문 텍스트 분석해서
  자동 라우팅은 Phase 2.
- **프롬프트 인젝션** — 학생이 system prompt 를 뒤집으려는 입력 가능성. 현재는
  system 을 stringly-typed 로 주입하는데, 운영하면서 문제되는 케이스 관찰 필요.

## 배운 점

- **FTS5 는 빌드 태그가 필요** — `go-sqlite3` 디폴트에 FTS5 가 비활성. `-tags sqlite_fts5`
  를 Dockerfile 과 테스트 명령 양쪽에 명시하는 걸 잊지 말 것.
- **도메인-어댑터 분리로 import cycle 회피** — `application` 이 `llmproxy` 에 의존,
  `llmproxy` 가 다시 `application` 의 인터페이스에 의존하는 구조. 공용 타입을
  `application` 에 두고 `llmproxy` adapter 가 변환. 기존 LLM 도메인에서 이미 이 패턴을
  썼고, chat 도메인에도 동일 적용.
- **"스킬 = 페르소나+도구+범위 번들"** 개념이 잘 동작 — 관리자 전용 `skill_designer`
  스킬이 자기 자신 설계 작업을 수행하는 self-hosting 메타 구조가 흥미로움. 나중에 이
  스킬로 새 스킬 품질을 평가하는 순환도 만들 수 있을 것.

## 사용한 프롬프트

> "우리 서비스 전체에서 사용가능한 챗봇 조교를 오른쪽 아래에 말풍선 아이콘으로 넣어줘.
>  학생들이 질문하는 내용에 답을 해줘야해. 아직 우리 시스템에 강의 컨텐츠 관련을
>  올리지 않아서, LLM wiki 방식으로 md 파일 중심의 RAG 를 해야 해. skill 등을 불러다
>  쓸 수 있는 챗봇을 만들어서, 관리자 페이지에서 필요한 스킬들을 추가로 만들어 넣을
>  수 있어야해 (이것또한 대화로) 그리고 방금 세팅한 qwen 모델로 모든걸 다할거야
>  (관리자는 최고 성능, 학생들에게 답변할 때는 빠르게 또는 깊이 있는 답변은 충분히
>  생각하도록). 우리 노션 언러닝 메뉴얼도 llm wiki (지식베이스)에 가지고 있어야 겠지.
>  일단 구체적으로 기획한 후 티켓 생성해줘."

## 다음 이터레이션

- 스트리밍 (SSE) → 첫 글자 체감 시간 단축
- Notion API 직접 동기화 (관리자 "노션 동기화" 버튼)
- 학생용 대화 히스토리 페이지 (`/chat/history`)
- LLM-as-router (질문 분류해서 적합 스킬 자동 선택)
- 외부 임베딩 (OpenAI text-embedding-3-small) 플러그인 — 한국어 BM25 한계 보완
