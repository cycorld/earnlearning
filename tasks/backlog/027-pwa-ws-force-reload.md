---
id: 027
title: WebSocket force-reload broadcast로 구버전 PWA 즉시 복구
priority: medium
type: feat
branch: feat/pwa-ws-force-reload
created: 2026-04-13
---

## 배경

`use-version-check.ts` 훅은 60초 폴링 + focus/visibility 이벤트로 새 버전을 감지한다. 일반 상황에서는 잘 동작하지만, 다음 경우에는 실패할 수 있다:
- 폴링 타이밍을 놓치는 사이 사용자가 중요 동작 수행
- 토스트를 놓치거나 무시함
- 더 심각한 케이스: 구버전에 폴링 훅 자체가 없음 (#206 사례 — #018 이전 빌드)

이 때 **서버가 능동적으로 "지금 reload해주세요"를 푸시**할 수 있으면 구조적으로 더 견고해진다. 이미 WebSocket hub가 구축되어 있어 구현 난이도 낮음.

## 핵심 아이디어

배포 직후 관리자가 (또는 배포 스크립트가) 다음 명령을 보내면, 현재 접속 중인 모든 클라이언트가 `forceRefresh()`를 실행한다:

```bash
POST /api/admin/notifications/force-reload
```

WS hub → 전체 broadcast → 프론트 훅이 받아서 `forceRefresh()` 호출.

## 작업 내용

### 백엔드

1. **WS 메시지 타입 추가**
   `backend/internal/interfaces/ws/` 에 `force_reload` 메시지 타입 추가.

2. **Admin API 추가**
   ```
   POST /api/admin/force-reload
   Body: { "target": "all" | "user:<id>", "reason": "청산 기능 롤아웃" }
   ```
   - `AdminOnly()` 미들웨어
   - WS hub의 `Broadcast()` 호출 또는 사용자 지정 push

3. **(선택) 배포 스크립트 연동**
   `./deploy-remote.sh promote` 성공 직후 자동으로 force-reload 발사 옵션:
   ```bash
   ./deploy-remote.sh promote --force-reload
   ```

### 프론트엔드

1. **WS 메시지 핸들러**
   `frontend/src/lib/websocket.ts` (또는 WS hook)에 `force_reload` 타입 리스너 추가.
   
2. **forceRefresh 재사용**
   `use-version-check.ts` 에 이미 있는 `forceRefresh()` 함수를 export해서 WS 핸들러에서 호출.

3. **사용자 경험 설계**
   - 즉시 hard reload? → 사용자가 입력 중이면 데이터 유실 위험
   - 토스트 + 5초 카운트다운 후 자동 reload (기본값)
   - 사용자가 토스트를 닫으면 다음 포커스 때 재시도
   - 중요 페이지(결제 등 없다면 전체 적용)

### 안전장치

- **Rate limiting**: admin 실수로 연타 방지 → 1분에 1회
- **Audit log**: 누가 언제 왜 force-reload했는지 DB 기록
- **Dry-run 모드**: `--dry-run` 옵션으로 영향받는 연결 수만 확인

## 한계

WS 연결이 살아있는 클라이언트만 영향받는다. 완전히 백그라운드에 있는 PWA는 여전히 #026(index.html no-cache)에 의존한다. 두 장치를 함께 쓰는 게 완전한 방어.

## 검증

- [ ] 스테이지에서 두 탭 열고 admin API 호출 → 두 탭 모두 5초 후 reload
- [ ] 카운트다운 중 "취소" 가능 여부 (결정 필요)
- [ ] rate limit 동작 확인
- [ ] audit log 기록 확인
- [ ] 프로덕션 배포 시 작동 시연

## 관련

- #018 PWA 자동 업데이트 (선행)
- #026 index.html no-cache (상보적 방어 — 두 가지 모두 필요)
- 기존 WS hub: `backend/internal/interfaces/ws/hub.go`
