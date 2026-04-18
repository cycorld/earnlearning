---
slug: 064-admin-ws-force-reload
title: 관리자 WebSocket 강제 새로고침 브로드캐스트 (#027)
date: 2026-04-18
tags: [feat, PWA, WebSocket, 관리자, 배포, 회귀테스트]
---

# 관리자 WebSocket 강제 새로고침 브로드캐스트

## 왜 필요했는가

프론트 폴링 기반 버전 체크(#018/#028)는 **클라이언트가 스스로** 새 버전을 감지하는 방식이라 두 가지 구멍이 있습니다:

1. **폴링 타이밍을 놓친 구간** 에 중요 동작을 하면 구버전으로 처리됨
2. **더 심각한 케이스**: 구버전 자체에 새 폴링 로직이 없음 — 배포 이전 빌드의 PWA는 영원히 stuck

예를 들어 청산 기능(#033) 같은 DB 스키마 변경이 동반된 배포 직후에는, 구버전 클라이언트가 새 필드 없이 요청하면 이상 동작. **서버가 능동적으로 "지금 새로고침" 을 푸시**할 수 있으면 이 구멍을 바로 메꿀 수 있습니다.

이미 WS 허브가 구축되어 있어서 구현 난이도는 낮았어요. #028 (폴링 자동 refresh) 과 상보적인 방어선입니다.

## 무엇을 했는가

### 1. 백엔드: Admin API `POST /api/admin/force-reload`

`backend/internal/interfaces/http/handler/admin_handler.go` 의 `AdminHandler` 에 `hub *ws.Hub` 필드 추가 + `ForceReload(c echo.Context) error` 핸들러 신규:

```go
// 1. Rate limit — 관리자 실수 연타 방지 (1분/1회)
h.frMu.Lock()
if !h.frLast.IsZero() && time.Since(h.frLast) < forceReloadRateLimit {
    h.frMu.Unlock()
    return errorResponse(c, http.StatusTooManyRequests, "RATE_LIMITED", ...)
}
h.frLast = time.Now()
h.frMu.Unlock()

// 2. Body: { reason: string } (선택)
// 3. Audit log
log.Printf("admin: force-reload triggered by user %d, reason=%q", actorID, body.Reason)

// 4. WS 브로드캐스트
h.hub.Broadcast(map[string]interface{}{
    "event": "force_reload",
    "data": map[string]interface{}{
        "reason": body.Reason, "at": time.Now().Unix(), "actor_id": actorID,
    },
})
```

라우트: `admin.POST("/force-reload", h.Admin.ForceReload)` — `AdminOnly()` 미들웨어 하위.

**목적: 단순함 > 완벽함.** DB 테이블 없는 `log.Printf` 감사 로그, `sync.Mutex` 보호 인메모리 rate limit. dry-run 모드 / per-user target 은 티켓에 있었지만 실제 요구는 "배포 후 전체 refresh" 하나라 v1 에서는 뺐습니다. 필요해지면 추가 가능.

### 2. 프론트엔드: `useForceReload` 훅

`frontend/src/hooks/use-force-reload.ts` 신규. `wsClient.on('force_reload', handler)` 로 구독하고 수신 시:

```ts
// 5초 카운트다운 토스트 + 취소 버튼
toast('⚠️ 관리자 강제 새로고침', {
  description: `${remaining}초 후 자동으로 새로고침됩니다${reason ? ` — ${reason}` : ''}`,
  action: { label: '취소', onClick: () => { /* 카운트다운 중단 */ } },
})

// setInterval 로 1초마다 remaining 감소 → 0 되면 forceRefresh()
```

- **취소 버튼**: 사용자가 중요한 입력 중이면 취소할 수 있음 (데이터 유실 방지)
- **중복 브로드캐스트 방어**: 이미 카운트다운 중이면 2번째 이벤트는 무시

`MainLayout` 에서 훅을 활성화:
```tsx
useVersionCheck()  // 기존
useForceReload()   // #027 추가
```

### 3. `forceRefresh` 공용화

기존에는 `use-version-check.ts` 내부 private 함수였던 `forceRefresh` 가 이제 `use-force-reload` 도 사용해야 해서 `frontend/src/lib/force-refresh.ts` 로 추출했습니다. SW 캐시 삭제 + SW unregister + 캐시 버스팅 쿼리 + `window.location.replace` 로직은 동일.

### 4. 테스트

**백엔드 통합 테스트 3건** (`admin_force_reload_test.go`):
- 일반 사용자 → 403
- 관리자 → 200
- 1분 내 2회차 → 429 RATE_LIMITED

**프론트 단위 테스트 7건** (`use-force-reload.test.tsx`):
- WS 구독 성립
- 이벤트 수신 시 토스트 노출 (reason 포함)
- 5초 카운트다운 후 `window.location.replace` 호출
- "취소" 클릭 시 새로고침 보류
- 중복 브로드캐스트 무시
- 매초 description 감소
- 언마운트 시 구독 해제

## 사용한 프롬프트

```
#27, # 28 은 이미 반영된거 아닌가? 아직 안되었으면 반영해줘.
```

AI 가 실제 반영 여부를 grep 으로 확인 → 둘 다 미반영. #028 먼저 (프론트만) → #027 (백+프론트+테스트) 순서로 PR 분리.

## 배운 점

- **방어는 여러 레이어로 겹쳐야 한다.** #026(전송 no-cache) + #028(클라이언트 폴링) + #027(서버 푸시) 이 세 층이 모두 다른 실패 모드를 커버. 하나만 있으면 결국 stuck 케이스가 남음.
- **v1 은 "운영자가 실제로 당장 쓰는 기능만" 구현하자.** 티켓에 있던 `--dry-run`, per-user target, DB audit 테이블은 지금 필요 없음. `log.Printf` + rate limit 만으로 배포 직후 브로드캐스트라는 핵심 가치는 다 잡혔다.
- **Rate limit 은 외부로부터의 공격 방어가 아니라 관리자 자기 자신으로부터의 보호.** 관리자가 실수로 버튼을 연타하면 수백 명이 반복적으로 reload 당함. 1분 제한은 그 비용 대비 거의 무비용.
- **강제 행동은 반드시 "취소 가능성" 을 준다.** 5초 카운트다운 + 취소 버튼이 그 역할. 유저가 무조건 자동 수행은 나쁜 UX.

## 관련 티켓

- #027 (backlog → in-progress) — 이 PR
- #028 — 같은 문제를 클라이언트 쪽에서 방어 (상보적)
- #026 — index.html no-cache (전송 레이어 방어)
- #018 — PWA 자동 업데이트 원본
