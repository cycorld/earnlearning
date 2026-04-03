# 038. OAuth 버그바운티 2차 제보 수정

> **날짜**: 2026-04-03
> **태그**: `OAuth`, `보안`, `버그수정`, `버그바운티`
> **기여**: Student-#267

## 무엇을 했나요?

Student-#267 학생의 OAuth 연동 버그바운티 제보(App-#267 앱)를 기반으로 2건의 서버 측 버그를 수정했습니다.

## 수정된 버그

1. **GET /api/posts 필수 파라미터 미전달 시 동작 미정의**: classroom_id/channel_id 없이 호출 시 빈 배열 대신 400 에러 반환하도록 변경
2. **refresh_token 갱신 시 client_secret 강제 요구**: PKCE 퍼블릭 클라이언트(SPA)에서 client_secret 없이도 갱신 가능하도록 수정

## 이미 정상이었던 항목
- `expires_in` 필드: 이미 토큰 응답에 `3600` 포함
- OpenAPI 스펙 개선: 별도 문서 작업으로 진행 예정
