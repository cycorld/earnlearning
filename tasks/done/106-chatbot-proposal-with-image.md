---
id: 106
title: 챗봇 교수님께 제안하기 + 이미지 첨부
priority: high
type: feat
branch: feat/chatbot-proposal
created: 2026-04-20
---

## 핵심
학생이 챗봇과 대화로 제안/버그 정리 → admin (교수) 알림 + 확인.

## Backend
- `proposals` 테이블 (id, user_id, category[feature|bug|general], title, body, attachments JSON, status[open|reviewing|resolved|wontfix], admin_note, ticket_link, created_at, updated_at)
- `chat_messages.attachments` JSON 컬럼 추가 (ALTER TABLE)
- domain/proposal + application/proposal_usecase + persistence
- chat 도구 `save_proposal(category, title, body)` — 자동으로 최근 학생 메시지 attachments 수집해 proposal 에 저장
- chat 도구 `get_my_proposals` — 본인 제안 이력
- 신규 chat skill `feedback_helper` (도구: search_wiki, save_proposal, get_my_proposals)
- API
  - `GET /api/chat/proposals/mine` (본인)
  - `GET /api/admin/proposals` (필터: 카테고리, 상태)
  - `PATCH /api/admin/proposals/:id` (status, admin_note)
- 새 proposal → admin user_id=1 에게 NotifSystem (`notif_type=proposal`, link=/admin/proposals/:id)

## Frontend
- ChatDock 에 paperclip 첨부 버튼 + preview chips
- send 시 attachments[] 함께 POST
- chat_messages 렌더에서 attachment 이미지 표시
- 챗봇 스킬 셀렉터에 `feedback_helper` 노출
- `/admin/proposals` 목록 + 상세 (이미지 포함) + 상태 변경 + "티켓 markdown 복사" 버튼

## 미포함 (의도)
- LLM 멀티모달 (Qwen 모델 텍스트 only)
- prod 서버 git commit 자동화 — admin 이 markdown 복사 후 로컬에서 직접
- 학생 본인 별도 페이지 — 챗봇 안 도구로만
