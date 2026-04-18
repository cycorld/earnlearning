---
id: 071
title: 챗봇 조교 (전 페이지 플로팅 위젯 + RAG + 스킬 + 관리자 편집)
priority: high
type: feat
branch: feat/chatbot-ta
created: 2026-04-18
---

## 개요
모든 학생 페이지 우하단에 **말풍선 플로팅 버튼**을 띄워 질문할 수 있는 챗봇 조교.
Qwen3.6 (llm.cycorld.com) 를 엔진으로 쓰고, 마크다운 기반 **LLM Wiki** 를 RAG 로
참조. 관리자는 **대화를 통해** 새 스킬 (페르소나 + 툴 조합) 을 만들 수 있음.
노션 "언러닝 가이드" 12개 페이지를 지식베이스로 초기 임포트.

## 사용자 확정 사항 (2026-04-18)
- **스킬 정의**: 페르소나(시스템 프롬프트 + wiki scope) + **툴 호출 가능** 모두
- **비용**: 학생 부담 없음 (학교 운영 예산, 서버 공용 admin 키 사용)
- **저장소 전략**: Git `docs/llm-wiki/*.md` source-of-truth + SQLite FTS5 인덱스 캐시
- **대화 히스토리**: LMS DB 에 영구 저장 (학습 히스토리 용도)
- **스코프**: 풀스코프 한 PR — MVP + 관리자 대화형 스킬 생성 + 임베딩 여지

## 모델 사용 정책
- **학생 기본**: `qwen-chat` (빠름, ~2K max_tokens)
- **학생 "깊이 생각하기" 토글**: `qwen-reasoning` + `reasoning_effort=medium`
- **관리자 기본**: `qwen-reasoning` + `reasoning_effort=high` (스킬 설계 등 추론 중심)
- **스트리밍**: SSE 로 `delta.content` 전달, `reasoning_content` 는 별도 숨김 필드

## 아키텍처

### 프론트엔드
- `<ChatDock />` — `MainLayout` 에 마운트, 모든 인증 페이지에 플로팅
  - 우하단 말풍선 FAB (44px, `bg-primary`)
  - 클릭 시 우측 패널 슬라이드업 (모바일: 전체 화면, 데스크탑: 420px 폭)
  - 스트리밍 메시지, 마크다운 렌더 (기존 `MarkdownContent` 재사용)
  - 상단: 현재 활성 스킬 이름 + "다른 스킬" 드롭다운, 우측에 "깊이 생각하기" 토글
  - 하단: 입력창 + 전송 + (관리자) "+ 새 스킬" 버튼
- `/chat/history` 라우트 — 과거 세션 목록 (최근 50)
- `/admin/chat` 라우트 — 스킬 관리, 위키 문서 관리, 노션 동기화

### 백엔드
- 새 도메인: `backend/internal/domain/chat/`
  - `session.go` — `Session{ID, UserID, Title, ActiveSkillID, CreatedAt, LastMessageAt, TokensUsed}`
  - `message.go` — `Message{ID, SessionID, Role, Content, Model, PromptTokens, CompletionTokens, ToolCalls JSON, CreatedAt}`
  - `skill.go` — `Skill{ID, Slug, Name, Description, SystemPrompt, DefaultModel, DefaultReasoningEffort, ToolsAllowed []string, WikiScope []string (glob), Enabled, CreatedByUserID, UpdatedAt}`
  - `wiki.go` — `WikiDoc{Slug, Path, Title, BodyMD, FrontMatter, UpdatedAt}`
- 새 인프라: `backend/internal/infrastructure/ragindex/`
  - `bm25.go` — SQLite FTS5 기반 BM25 검색 (`MATCH` 쿼리)
  - `loader.go` — `docs/llm-wiki/**/*.md` 를 읽어 FTS5 에 upsert
  - `embeddings.go` (stub) — 나중에 외부 임베딩 붙일 인터페이스
- 새 인프라: `backend/internal/infrastructure/llmproxy/chat_stream.go`
  - `/v1/chat/completions` SSE 스트리밍 + tool-call parsing
- 유스케이스: `backend/internal/application/chat_usecase.go`
  - `Ask(userID, sessionID, userMessage, mode) → stream` — 메시지 처리 + 스킬 선택 + 도구 실행 + LLM 호출
  - `ListSessions(userID, page)`, `GetSession(userID, id)`
  - `ListSkills(userRole)` — 학생/관리자별 필터
  - `AdminCreateSkill(...)`, `AdminUpdateSkill(...)`, `AdminDeleteSkill(...)`
  - `AdminImportNotion()` — 주요 가이드 12개 임포트
  - `AdminReindexWiki()` — git 파일 → FTS5 재로드
- 스킬 라우터: 질문 텍스트 → 기본은 `general_ta`, 관리자가 "skill:" 접두사로 특정 스킬 선택 가능. 2차에 LLM-as-router 도입 고려.
- 툴 레지스트리: `backend/internal/application/chat_tools.go`
  - 학생 도구: `get_my_wallet_balance`, `get_my_recent_transactions`, `get_my_companies`, `get_my_grant_applications`, `get_my_llm_usage_summary`, `search_wiki(query)`
  - 관리자 도구: 위 + `save_skill_draft(...)`, `list_recent_users`, `search_user_by_name`
  - 각 도구는 Go 함수로 구현하고 스킬의 `ToolsAllowed` 허용 목록에 있을 때만 노출

### HTTP 엔드포인트 (`/api/chat/*`)
- `POST /api/chat/sessions` — 새 세션 생성 (title auto-generated)
- `GET /api/chat/sessions?page=1&limit=20` — 내 세션 목록
- `GET /api/chat/sessions/:id` — 세션 메타 + 최근 메시지 50
- `POST /api/chat/sessions/:id/ask` — 스트리밍 응답 (SSE). body: `{message, mode?: "fast"|"deep", skill_slug?}`
- `GET /api/chat/skills` — 공개 스킬 목록
- `DELETE /api/chat/sessions/:id` — 세션 삭제 (본인만)

### 관리자 HTTP (`/api/admin/chat/*`)
- `GET /api/admin/chat/skills` — 전체 스킬 (비활성 포함)
- `POST /api/admin/chat/skills` — 스킬 수동 생성 (JSON body)
- `PUT /api/admin/chat/skills/:id`, `DELETE /api/admin/chat/skills/:id`
- `GET /api/admin/chat/wiki` — 위키 문서 목록
- `PUT /api/admin/chat/wiki/:slug` — 위키 문서 편집 (server write + git commit + reindex)
- `POST /api/admin/chat/wiki/import-notion` — 노션 동기화
- `POST /api/admin/chat/wiki/reindex` — 수동 재인덱싱

### DB 스키마 (신규 테이블 5)
```sql
CREATE TABLE chat_sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL REFERENCES users(id),
  title TEXT NOT NULL DEFAULT '',
  active_skill_id INTEGER,
  tokens_used INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_message_at DATETIME
);
CREATE INDEX idx_chat_sessions_user_recent ON chat_sessions(user_id, last_message_at DESC);

CREATE TABLE chat_messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id INTEGER NOT NULL REFERENCES chat_sessions(id),
  role TEXT NOT NULL CHECK (role IN ('system','user','assistant','tool')),
  content TEXT NOT NULL,
  reasoning_content TEXT DEFAULT '',
  model TEXT DEFAULT '',
  prompt_tokens INTEGER DEFAULT 0,
  completion_tokens INTEGER DEFAULT 0,
  cache_tokens INTEGER DEFAULT 0,
  tool_calls TEXT DEFAULT '[]',
  tool_call_id TEXT DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_chat_messages_session ON chat_messages(session_id, created_at);

CREATE TABLE chat_skills (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  system_prompt TEXT NOT NULL,
  default_model TEXT NOT NULL DEFAULT 'qwen-chat',
  default_reasoning_effort TEXT DEFAULT '',
  tools_allowed TEXT NOT NULL DEFAULT '[]',
  wiki_scope TEXT NOT NULL DEFAULT '[]',
  enabled INTEGER NOT NULL DEFAULT 1,
  admin_only INTEGER NOT NULL DEFAULT 0,
  created_by INTEGER REFERENCES users(id),
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- FTS5 가상 테이블 (외부-content, git md 파일이 source)
CREATE VIRTUAL TABLE chat_wiki_docs USING fts5(
  slug UNINDEXED,
  title,
  body,
  tokenize = 'unicode61 remove_diacritics 2'
);

CREATE TABLE chat_wiki_meta (
  slug TEXT PRIMARY KEY,
  path TEXT NOT NULL,
  title TEXT NOT NULL,
  notion_page_id TEXT DEFAULT '',
  synced_at DATETIME,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### 초기 스킬 (seed)
1. `general_ta` — 기본 조교. wiki scope: 전체. 도구: search_wiki
2. `wallet_helper` — 지갑 도우미. 도구: get_my_wallet_balance, get_my_recent_transactions
3. `company_helper` — 회사 경영 도우미. 도구: get_my_companies
4. `grant_helper` — 정부과제 도우미. 도구: get_my_grant_applications
5. `llm_api_helper` — LLM API 사용 도우미. 도구: get_my_llm_usage_summary
6. `code_review` — 과제 코드 리뷰 (도구 없음, 순수 질의응답)
7. `skill_designer` — 관리자 전용 메타-스킬. 도구: save_skill_draft

### 노션 임포트 (초기 실행)
- 스크립트 `scripts/import-notion-wiki.ts` (일회성)
- 대상: `reference_notion.md` 에 나열된 12개 가이드 페이지
- 각 페이지 → `docs/llm-wiki/notion-manuals/<slug>.md` 로 저장 (frontmatter: notion_id, title, synced_at)
- 스크립트가 자동으로 git add + commit — 또는 사람이 수동 커밋
- 관리자 페이지 "노션 동기화" 버튼도 같은 로직 서버 구현

## 테스트
- BM25 인덱서: 문서 로드·검색·업데이트 유닛 테스트
- 스킬 라우터: slug 매칭, 기본값 fallback, 비활성 스킬 필터
- 챗봇 유스케이스: fake LLM 클라이언트로 스트리밍 시뮬, 도구 호출 경로
- 관리자 스킬 CRUD: 권한 체크 (학생이 admin_only skill 사용 시 403)
- 통합: 학생 로그인 → 세션 생성 → 질문 → 응답 저장 확인
- 프론트: ChatDock 열기/닫기, 메시지 전송, 세션 전환

## 리스크 / 주의사항
- **단일 PR 이 상당히 큼** (예상 2,000~3,000 LOC + 프론트 1,000~1,500 LOC)
  - 커밋을 논리 섹션별로 쪼개서 리뷰 가능하게 할 것
- **Qwen function-calling 정확도**: Qwen3.6 은 tool-use 공식 지원하지만 현업 검증 필요
  - 1차: 도구 호출을 chat_completions 의 `tools` 파라미터로 시도
  - fallback: system prompt 에 도구 사용 가이드 포함 + JSON 출력 파싱 (사용자 명시 ReAct)
- **서버에서 git commit 권한**: 관리자가 위키 편집 시 `git add + commit + push` 가 필요한데,
  EC2 컨테이너가 deploy 키를 가지고 있어야 함. 초기엔 commit/push 없이 로컬 파일만 쓰고
  수동 배포 경로로 유지 (별도 후속 티켓으로 Git push 자동화)
- **비용 모니터링**: 학생 무료 → 운영진 부담 → 월간 총 비용 추적 필요.
  관리자 대시보드에 "이번 달 챗봇 토큰/원화" 요약 카드 추가
- **프롬프트 주입**: 학생이 system prompt 를 무력화하는 입력을 할 수 있음. 시스템 프롬프트를
  항상 "assistant_context" 보강 메시지로 쪼개서 보완
- **데이터 프라이버시**: LMS DB 영구 저장 + llm-proxy 원격 저장 이중. 학생 대화 조회 권한은
  본인 + 관리자만.

## 범위 밖 (별도 티켓)
- 외부 임베딩 (OpenAI text-embedding-3 등) 도입 — interface 만 열어두고 구현은 후순위
- 학생 탭 UI 에서 스킬 브라우징 — 현 범위는 관리자 드롭다운만
- 대화 내보내기 (PDF/JSON 다운로드)
- 팀 학습 세션 (여러 학생이 같은 세션 공유)
- 학생이 직접 만든 노트/과제 업로드해서 RAG 에 추가하는 기능
