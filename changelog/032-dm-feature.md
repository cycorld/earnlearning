# 032. 유저 간 1:1 DM(메시지) 기능

> **날짜**: 2026-04-02
> **태그**: `DM`, `WebSocket`, `실시간`, `UX`

## 무엇을 했나요?

유저 간 1:1 다이렉트 메시지(DM) 기능을 추가했습니다. 실시간 채팅이 가능하며, 피드에서 작성자를 클릭하면 바로 메시지를 보낼 수 있습니다.

## 왜 필요했나요?

학생들 간 직접 소통할 수 있는 채널이 없어서 피드 댓글이나 오프라인에서만 연락할 수 있었습니다. 과제 협업, 질문, 거래 등의 소통을 위해 DM 기능이 필요했습니다.

## 어떻게 만들었나요?

### 백엔드
- `dm_messages` 테이블 생성 (sender_id, receiver_id, content, is_read)
- 5개 REST API: 메시지 전송, 대화 목록, 메시지 조회, 읽음 처리, 미읽음 수
- WebSocket으로 sender/receiver 양쪽에 실시간 이벤트 전송

### 프론트엔드
- `/messages` — 대화 상대 목록 (마지막 메시지 미리보기, 미읽음 뱃지)
- `/messages/:userId` — 채팅 화면 (말풍선 UI, Enter 전송, 실시간 수신)
- Header에 채팅 아이콘 + 미읽음 뱃지
- 피드 작성자 아바타 클릭 시 드롭다운 (프로필 보기 / 메시지 보내기)
- 유저 프로필 페이지에 "메시지 보내기" 버튼

### 실시간 처리
- 기존 WebSocket Hub의 `SendToUser` 메서드를 재사용
- 메시지 전송 시 sender/receiver 모두에게 WS 이벤트 발송 (다중 탭 동기화)
- ConversationPage에서 id 중복 체크로 WS 에코 방지

## 배운 점

- **기존 인프라 재사용**: WebSocket Hub가 이미 `SendToUser` 메서드를 제공하고 있어서, DM을 위해 새로운 실시간 인프라를 만들 필요가 없었습니다.
- **vite proxy 순서**: `/api/ws`가 `/api` HTTP 프록시보다 먼저 매칭되어야 WebSocket 업그레이드가 정상 동작합니다.
- **대화 목록 쿼리**: 별도 conversations 테이블 없이 `dm_messages`를 GROUP BY + MAX(id) 집계로 대화 목록을 생성할 수 있습니다 (MVP 단순성).
