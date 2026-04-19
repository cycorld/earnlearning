---
id: 082
title: 챗봇 위키 — Notion 자동 동기화 + git push (#075 후속)
priority: low
type: feat
branch: feat/chatbot-wiki-notion-sync
created: 2026-04-19
---

## 배경
#075 에서 관리자 인라인 에디터 MVP 만 구현. Notion 자동 동기화는 별도로 분리.

## 스코프
1. **Notion API 클라이언트** (`internal/infrastructure/notion/`) — 페이지 ID 로 마크다운
   추출. `NOTION_INTEGRATION_TOKEN` env 주입.
2. **관리자 "노션 동기화" 버튼** — `/admin/chat` 에 추가. 클릭 시 `notion_page_id` 가
   있는 모든 위키 문서를 최신본으로 refetch + 파일 rewrite + 재인덱싱.
3. **서버에서 git commit + push 자동화** — deploy key 필요. 보안 영향 검토 필요.
4. (선택) 실시간 웹훅 동기화.

## 확인 필요
- Notion → markdown 매핑 규칙 (콜아웃, 토글, 테이블 등)
- 프로덕션 서버에 git push 권한을 주는 것의 보안 영향
- 영구화 방안: git push 자동화 vs PR 자동 생성 vs DB 만 영구

## 트레이드오프
git push 자동화는 강력하지만 위험. 대안: 동기화는 DB 에만 적용 (재배포 시 사라지므로
admin 이 수동 commit) 또는 PR 생성 봇.
