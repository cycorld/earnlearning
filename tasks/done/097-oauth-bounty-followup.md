---
id: 097
title: OAuth 버그바운티 후속 — pending 2건 보상지급 + spec data {} 개선
priority: high
type: chore
branch: chore/oauth-bounty-followup
created: 2026-04-19
---

OAuth 버그바운티(grant 9) 후속 처리.

## 신청 4건 검증
- 266 Student-#266 (approved, 500k 지급 완료) — 5 bugs
- 267 Student-#267 (approved, 500k 지급 완료) — 4 bugs (그 중 OpenAPI data {} 만 실 버그)
- 271 Student-#271 (pending) — 1 CORS bug 클레임 → **검증: CORS 정상 동작 확인** (`access-control-allow-origin: *` 정상). claude.ai 위젯 자체 제약 추정. 그래도 OAuth 인가 코드 플로우 + 등록 완료 → 노력 인정으로 보상 지급.
- 276 Student-#276 (pending) — 2 bugs: wallet 0 KRW 표시(자체 매핑 추정), Silent fail UX 피드백. 통합 + 상세 리포트 → 보상 지급.

## 실제 코드 개선
- Student-#267 #1: OpenAPI 의 `APIResponse.data` 가 `{}` 로만 정의됨 → SDK 자동 생성 어려움 → **OAuth 엔드포인트(/oauth/token, /oauth/userinfo) 에 typed response wrapper 추가**.

## 검증되어 false 였던 항목 (참고)
- Student-#267 #2 expires_in 누락 → 코드 line 382 `ExpiresIn: 3600` 존재 ✅
- Student-#267 #4 refresh PKCE 시 client_secret 필수 → line 299 `if input.ClientSecret != ""` 옵셔널 ✅
- Student-#271 CORS → `*` 로 허용 + 헤더 정상 ✅

## 작업
1. typed wrapper 추가 (OAuthTokenResponse, OAuthUserInfoResponse)
2. swag 재생성 + 검증
3. admin 토큰으로 271, 276 approve API 호출 → 자동 500k 지급
4. changelog 097
