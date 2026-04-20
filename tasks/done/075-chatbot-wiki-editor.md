---
id: 075
title: 챗봇 위키 — 관리자 인라인 에디터 (MVP)
priority: medium
type: feat
branch: feat/chatbot-wiki-editor
created: 2026-04-19
---

## 배경
원래 #075 는 "Notion 자동 동기화 + 관리자 편집 UI" 였음. Notion 동기화는 git push
권한 / 파일 vs DB persistence 등 결정사항이 많아 분리 (#082 로 분리).

이 PR 은 **관리자 인라인 에디터 MVP** 만 다룸. 학생 질문에 잘못 답변하는 위키
조항을 관리자가 즉석에서 수정 → 저장 → 다음 질문부터 반영.

## 스코프
1. `WikiRepository.GetDocBody(slug)` — FTS5 에서 본문 읽기
2. `application.AdminGetWikiDoc(slug)` / `AdminUpdateWikiDoc(slug, title, body)` —
   FTS5 + meta 둘 다 갱신, 가능하면 파일도 덮어씀 (dev 환경 영구화)
3. 라우트:
   - `GET /admin/chat/wiki/:slug` — 본문 + 메타 반환
   - `PUT /admin/chat/wiki/:slug` — body/title 업데이트
4. 프론트: AdminChatPage 위키 행에 "편집" 버튼 → 모달 + textarea + 저장

## 트레이드오프
프로덕션 컨테이너의 `docs/llm-wiki/*.md` 파일은 이미지에 패키징되어 있어
재배포 시 사라짐. 즉, **에디터 변경은 다음 재배포까지만 유효** (DB FTS5 는
유지되지만 부팅 시 ragindex.Sync() 가 파일로 DB 를 다시 덮어씀).

→ 영구화하려면 .md 파일을 git 에 커밋해야 함. 에디터 모달에 안내 문구 노출.

## 후속
- #082: Notion 자동 동기화 + git commit + push 자동화 (deploy key 필요)
