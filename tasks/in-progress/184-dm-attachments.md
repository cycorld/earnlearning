---
id: 184
title: DM 파일/이미지 첨부
priority: medium
type: feat
branch: feat/184-dm-attachments
created: 2026-07-28
---

## 배경
DM(1:1 쪽지)은 텍스트만 보낼 수 있다. 팀 과제·계약 협의 중 이미지/문서를 보내려면
피드에 올리거나 메일을 써야 해서 흐름이 끊긴다.

## 요구사항
- 보내는 사람은 **텍스트만 / 첨부만 / 둘 다** 모두 보낼 수 있다.
- 첨부 목록·다운로드는 **해당 DM의 발신자·수신자만** 가능하다. (관리자 우회 없음)
- 저장은 **비공개 경로**(`PRIVATE_UPLOAD_PATH/dm/`)에만. 공개 `/uploads` 정적 경로 노출 금지.
- 파일명 안전화, 확장자·MIME·실제 바이트(매직넘버) 3중 검증.
- 파일당 **10MB**, 메시지당 **최대 4개**, 합계 **20MB**.
- 디스크 기록 후 DB 실패 시 파일 정리(고아 파일/고아 행 금지).
- 비이미지 응답은 `Content-Disposition: attachment` + `application/octet-stream`
  + `X-Content-Type-Options: nosniff` + `CSP: default-src 'none'; sandbox`.
- 검증된 안전한 이미지(jpg/jpeg/png/gif/webp)만 inline 렌더 허용.
- HTML은 애초에 허용 확장자에서 제외 → 실행 경로 자체가 없다.
- 마이그레이션은 additive(`CREATE TABLE IF NOT EXISTS`)만.
- 기존 `new_dm` 알림은 그대로 동작(첨부만 보낸 경우 본문 대신 "사진/파일" 요약).

## 설계 결정 (근거)
1. **별도 테이블 `dm_attachments`** — JSON 컬럼은 `/dm/attachments/:id` 로 라우팅할 id가 없다.
   또한 `sqlite.go` 의 `alterStatements` 는 `dmTables` 블록보다 **먼저** 실행되므로
   `ALTER TABLE dm_messages` 는 새 DB에서 무음 실패한다. `dmTables` 에 `CREATE TABLE IF NOT EXISTS` 추가가 안전.
2. **허용 확장자에서 `.md/.txt/.json` 제외** — 이 텍스트 타입들은 전체 버퍼(TeeReader) 검증이 필요한데
   (512바이트 Peek 은 한글 멀티바이트를 잘라 오탐), DM 첨부에 필수가 아니라 검증 분기를 줄였다.
   허용: `.jpg .jpeg .png .gif .webp .pdf .doc .docx .xls .xlsx .ppt .pptx .zip .mp4 .mp3`
3. **개수 4개 / 합계 20MB** — nginx `client_max_body_size 50M` 한도 아래. 5×10MB 는 테스트는 통과하고
   프로덕션에서만 413 이 난다.
4. **`RejectOAuth()`** — DM 라우트는 "internal only" 주석과 달리 스코프 검사가 없다.
   비공개 파일 바이트를 OAuth 토큰에 노출하지 않도록 **새 다운로드 라우트와 전송(업로드) 라우트만**
   메일(`router.go:273`) 선례대로 `RejectOAuth()` 그룹에 넣는다. 기존 읽기 라우트 3개는 그대로 둔다(비대칭 명시).
5. 메시지 행 + 첨부 행은 **하나의 트랜잭션**으로 insert (`MaxOpenConns=1` 이라 tx 안에서는 `tx.Exec` 만 사용).

## API
- `POST /api/dm/messages`
  - `application/json` (기존 그대로): `{receiver_id, content}`
  - `multipart/form-data`: `receiver_id`, `content`(선택), `files`(반복)
  - 201 → `data: Message{..., attachments: [...]}`
- `GET /api/dm/messages/:userId` → 각 메시지에 `attachments: []`
- `GET /api/dm/attachments/:id` → 파일 바이트 (발신자/수신자만, 그 외 403 / 없으면 404)

## 완료 기준
- 백엔드 통합 테스트: 권한(제3자·관리자 403), 위조 확장자/MIME, 초과 크기·개수, 실패 시 정리,
  빈 메시지(텍스트만/첨부만/둘 다), 재조회 영속성, 다운로드 헤더.
- 프론트: 첨부 선택 UI + 전송 전 미리보기 + 이미지 인라인 + 그 외 다운로드, vitest 통과.
- `./scripts/test-backend.sh smoke` / `integration`, `npm test`, `npm run build` 전부 통과.
