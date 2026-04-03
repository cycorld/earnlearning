---
id: 011
title: OAuth 버그바운티 2차 제보 수정 (김나연)
priority: high
type: fix
branch: fix/oauth-bugfix-2
created: 2026-04-03
---

## 제보자: 김나연 (grant_application #267)
앱: Swipe2Eat (음식점 매칭 서비스)

## 유효 버그 4건

### 버그 1: OpenAPI 스펙 data 필드 빈 객체
- swagger 어노테이션에 구체적 응답 타입 정의 필요
- 현재는 모든 엔드포인트가 `APIResponse{data: object}` 공통 래퍼만 사용
- 작업량 큰 문서 개선 → 핵심 엔드포인트만 우선 (userinfo, wallet, posts)

### 버그 2: 토큰 응답에 expires_in 없음 (RFC 6749 위반)
- POST /api/oauth/token 응답에 expires_in 추가
- 현재 1시간 고정 → `3600` 반환

### 버그 3: GET /api/posts 필수 파라미터 미전달 시 동작 미정의
- classroom_id 없이 호출 시 빈 배열 반환 → 400 에러로 변경

### 버그 4: refresh_token에 client_secret 요구 (퍼블릭 클라이언트 불가)
- PKCE로 발급된 토큰은 client_secret 없이 갱신 가능하도록 수정
