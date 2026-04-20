# 106. 챗봇 교수님께 제안하기 + 이미지 첨부 + Qwen 멀티모달

**날짜**: 2026-04-20
**태그**: 챗봇, 제안, 이미지, 멀티모달, admin

## 배경
학생이 LMS/챗봇/강의 운영에 대한 의견·버그신고를 챗봇과 정리해서 교수에게 전달할 통로가 없었다. + Qwen 모델이 mmproj 로 멀티모달 (이미지 input) 가능함을 발견 → 첨부 이미지를 LLM 도 분석하도록 통합.

## 추가
### Backend (#106)
- **migration**: `proposals` 테이블 + `chat_messages.attachments` 컬럼 (ALTER TABLE 패턴)
- `domain/proposal/` (entity, repository), `infrastructure/persistence/proposal_repo.go`
- `application/chat_proposal_usecase.go` — Create / ListMine / AdminList / AdminUpdate
- `interfaces/http/handler/chat_proposal_handler.go` + routes:
  - `GET /api/chat/proposals/mine` (학생 본인)
  - `GET /api/chat/proposals/:id` (학생 본인 또는 admin)
  - `GET/PATCH /api/admin/proposals` (admin)
- 신규 알림 타입 `NotifProposalSubmitted` → 새 제안 시 admin (user_id=1) 에게 push
- `chat_seed.go` 신규 스킬 `feedback_helper` (system prompt: 카테고리 분류 → 정리 → 학생 확인 → save_proposal 호출)
- chat 도구 `save_proposal(category, title, body)` — 세션 모든 학생 메시지 attachments 자동 수집
- chat 도구 `get_my_proposals(limit)` — 본인 이력
- **Vision 통합** (`llmproxy/chat.go`, `chat_llm_types.go`, `chat_adapter.go`, `chat_usecase.go`):
  - `ChatMessage` 에 `ContentParts []ContentBlock` 추가 + custom `MarshalJSON` (string OR array 분기)
  - `LLMChatMessage` 도 동일 + `LLMContentBlock`
  - 학생 user 메시지에 attachments 있으면 OpenAI vision 형식으로 변환 (`type:"image_url"`)
  - `absoluteImageURL()`: `/uploads/xxx.png` → `PUBLIC_BASE_URL` 또는 `https://earnlearning.com` 으로 절대화 (llama-server 가 외부 fetch 가능한 URL 필요)

### Frontend
- `ChatDock.tsx`:
  - 첨부 paperclip 버튼 (input 좌하단) + preview chips + 제거 X 버튼
  - `streamAsk` 가 `attachments[]` body 로 전송
  - user message 렌더에 첨부 이미지 표시
- `AdminProposalsPage.tsx` (신규): 카테고리/상태 필터, 목록, 상세 모달, 상태 변경, **"티켓 markdown 복사" 버튼** (자동 git X — 안전)
- `App.tsx`: 신규 라우트 `/admin/proposals`
- `AdminPage.tsx`: "학생 제안" 카드 추가 (Lightbulb 아이콘)
- `NotificationsPage.tsx`: `proposal` reference_type → `/admin/proposals` 매핑
- `lib/api.ts`: `api.patch` 메서드 추가

## 작동 흐름
1. 학생 챗봇 → 스킬 "교수님께 제안하기" 선택
2. (선택) paperclip → 이미지 첨부 (5MB 이하, image/* 만)
3. 챗봇과 대화로 정리 → AI 가 카테고리/제목/본문 정리 → 학생 확인
4. AI 가 `save_proposal` 호출 → DB 저장 + admin 에게 push 알림
5. admin: 알림 클릭 → `/admin/proposals` → 상세 → 상태 변경 / 티켓 markdown 복사
6. 학생은 `get_my_proposals` 로 진행 상황 확인 (admin_note 보임)

## 미포함 (의도)
- prod 서버 git commit 자동화 — admin 이 markdown 복사 후 로컬에서 직접 ticket 생성 (안전)
- shareholder proposal (주주총회) vs chat proposal — 모두 `reference_type=proposal` 사용 중. 충돌 가능. 별도 정리 티켓 권장.
- 학생용 별도 페이지 — 챗봇 안 도구로만 (간소화)

## Qwen 멀티모달 활성 확인
cycorld 서버에서 `llama-server --mmproj mmproj-BF16-qwen3.6.gguf` 로 실행 중. proxy 는 그대로 통과. OpenAI vision API (`content: [{type:"text"}, {type:"image_url"}]`) 형식 사용 가능.
