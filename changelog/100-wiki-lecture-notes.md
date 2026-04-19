# 100. 강의 노트 + 강의계획서 wiki 추가 + lecture_helper 스킬

**날짜**: 2026-04-20
**태그**: 챗봇, 위키, 강의자료, 스킬

## 배경
챗봇이 LMS 운영(지갑·회사·정부과제 등)은 잘 답하지만, **강의 내용 자체** ("PR&FAQ 가 뭐야?", "3주차 뭐했어?", "MVP 마감 언제?") 에는 답할 수 없었다. 챗봇 위키에 강의 자료가 없었기 때문.

## 추가
### `docs/llm-wiki/lecture-notes/`
8개 마크다운 파일 추가 (모두 frontmatter 포함: title, summary, source):
- `syllabus.md` — 실질 강의계획서 (2026-1학기)
- `week01-orientation.md` — 1주차 오리엔테이션
- `week02-problem-finding.md` — 2주차 문제 정의
- `week03-ai-driven-business-plan.md` — 3주차 PR&FAQ/RFP/PRD/SPEC
- `week04-spec-and-vibe-coding.md` — 4주차 SPEC + 바이브 코딩 시작
- `week05-vibe-coding-practice.md` — 5주차 웹 구조/도구/프롬프트 패턴
- `week06-vibe-coding-backend-deploy.md` — 6주차 CRUD/RBAC/배포
- `week06-side-quest.md` — 6주차 부록 Lv 0~6 자율 학습

원본은 상위 폴더 `/Users/cycorld/Workspace/ewha2026/week*.md`. 강의자가 직접 작성한 강의 노트 + 슬라이드 (week*.pdf) 의 마크다운 버전.

### `chat_seed.go` — 스킬 변경
**신규**: `lecture_helper`
- name: 강의 내용 도우미
- tools: `search_wiki` 만
- wiki_scope: `lecture-notes/*`
- 시스템 프롬프트: 모든 강의 질문은 search_wiki 먼저 → lecture-notes 참고 → 어느 주차 인용했는지 명시

**수정**: `general_ta`
- description: "강의 내용" 추가
- 시스템 프롬프트 우선순위 1번에 "강의 내용/주차/과제/평가" → lecture-notes/* 우선 추가
- 링크 규칙 예시에 lecture-notes slug 도 추가

## 인덱싱
서버 기동 시 `ragindex.Loader` 가 `docs/llm-wiki/**/*.md` 를 walk → SQLite FTS5 자동 인덱스. 별도 설정 불필요. 운영 중 변경 시 `/admin/chat → 위키 재인덱싱` 버튼 또는 `POST /api/admin/chat/wiki/reindex`.

## 미포함 (의도)
- 강의 슬라이드 (PDF) 자체 챗봇 노출 — 마크다운만 인덱싱
- Notion sync — 이 문서들은 Notion 에 없음 (강의자 로컬 파일)
- source_url 필드 추가 — Notion URL 만 자동 생성 가능. 강의 노트는 슬러그로만 표시 (학생이 admin/chat 에서 원본 보기 가능)

## 검증 시나리오
- "1주차 뭐했어?" → week01-orientation 인용
- "PR&FAQ 가 뭐야?" → week03-ai-driven-business-plan 인용
- "MVP 마감 언제야?" → syllabus 인용
- "SPEC 5요소" → week04-spec-and-vibe-coding 인용
- "프롬프트 카탈로그 6번" → week06-vibe-coding-backend-deploy 인용
