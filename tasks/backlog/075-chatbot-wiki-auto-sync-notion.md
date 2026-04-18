---
id: 075
title: 챗봇 위키 — 노션 자동 동기화 + 관리자 편집 UI
priority: low
type: feat
branch: feat/chatbot-wiki-sync
created: 2026-04-19
---

## 배경
현재 `docs/llm-wiki/notion-manuals/*.md` 13편은 **수동 임포트** (Claude 가 노션 fetch
→ md 작성). 노션 문서가 업데이트돼도 자동으로 따라오지 않음.

## 스코프
1. **Notion API 클라이언트** (`internal/infrastructure/notion/`) — 페이지 ID 로 마크다운
   추출. `NOTION_INTEGRATION_TOKEN` env 주입.
2. **관리자 "노션 동기화" 버튼** — `/admin/chat` 페이지에 추가. 클릭 시 등록된
   `notion_page_id` 를 가진 모든 위키 문서를 최신본으로 refetch + 파일 rewrite + 재인덱싱.
3. **서버에서 git commit + push 자동화** (선택적) — deploy key 기반. 없으면 파일만 쓰고
   재배포 시 영구화.
4. **관리자 위키 에디터** — `/admin/chat/wiki/:slug` 에서 마크다운 직접 편집 + 저장.

## 확인 필요
- Notion API 는 rich text → markdown 변환이 완벽하지 않음. 콜아웃 / 토글 / 테이블
  등 어떻게 처리할지 매핑 규칙 필요
- git push 권한을 프로덕션 서버에 주는 것의 보안 영향

## 제외
- 실시간 웹훅 동기화 (노션이 변경될 때 즉시 반영) 는 더 큰 작업 — Phase 3
