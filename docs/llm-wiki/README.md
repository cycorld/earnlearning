---
title: LLM Wiki — 챗봇 지식베이스
---

# LLM Wiki — 챗봇 지식베이스 (#071)

이 디렉토리는 **챗봇 조교 (#071)** 가 RAG 로 참조하는 마크다운 기반 지식베이스입니다.
서버 기동 시 `backend/internal/infrastructure/ragindex/` 로더가 모든 `.md` 파일을
읽어 SQLite FTS5 인덱스에 로드합니다.

## 구조

- `notion-manuals/` — 언러닝 노션 가이드 12편 (학생용 핵심 매뉴얼)
- 루트 `.md` — 별도 짧은 보조 문서 (챗봇 소개, 자주 쓰는 FAQ 등)

## 편집

1. `.md` 파일을 직접 수정 or 새로 추가 (관리자 권한 있는 개발자)
2. 서버에 배포하면 기동 시 자동 재인덱싱
3. 배포 없이 반영하려면 관리자 페이지의 `/admin/chat` → "위키 재인덱싱" 버튼

## frontmatter

각 파일은 YAML-lite 프런트매터를 지원합니다:

```yaml
---
title: 실제 제목
notion_page_id: 34668bb8-a660-...
synced_at: 2026-04-18T12:00:00Z
---
```

`title` 이 없으면 본문 첫 `# H1` 이 제목으로 쓰입니다.

## 챗봇에서 이 문서를 어떻게 쓰나

스킬마다 `wiki_scope` glob 으로 참조할 문서를 제한할 수 있어요 (예: `notion-manuals/wallet*`).
스킬 기본 도구 `search_wiki(query)` 가 FTS5 BM25 점수 순으로 결과를 반환.

## Skill 추가

관리자 페이지의 `/admin/chat` → "새 스킬" 버튼 또는 **`skill_designer`** 스킬과 대화.
