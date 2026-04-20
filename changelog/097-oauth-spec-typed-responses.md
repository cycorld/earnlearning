# 097. OAuth API 스펙 — typed response wrapper (#바운티 후속)

**날짜**: 2026-04-19
**태그**: API, OAuth, OpenAPI, 문서, 바운티

## 배경
[4주차] OAuth 연동 버그바운티 (grant 9) 신청 4건을 검토하면서, **Student-#267** 학생의
지적 중 하나가 실제 코드 레벨 결함이었다:

> OpenAPI 스펙의 모든 응답 `data` 필드가 빈 객체 `{}` 로만 정의되어 있어
> SDK 자동 생성/타입 추론이 불가능.

원인: 모든 핸들러가 `@Success {object} APIResponse` 로 표시되어 있었고,
`APIResponse.Data` 는 `interface{}` → swag 가 `{}` 로 출력.

## 수정
실제로 외부 통합 빈도가 가장 높은 OAuth 엔드포인트 두 개에 typed wrapper 도입:

### `swagger_models.go`
- `OAuthTokenData` — `access_token`, `refresh_token`, `token_type`, `expires_in`, `scopes`
- `OAuthTokenResponse` — `{success, data: OAuthTokenData, error}`
- `OAuthUserInfoData` — `id, email, name, department, bio, avatar_url`
- `OAuthUserInfoResponse` — 동일 envelope
- `OAuthTokenRequest` — `grant_type`, `code`, `client_id`, `client_secret(옵셔널)`, `redirect_uri`, `code_verifier`, `refresh_token`

### `oauth_handler.go`
- `Token` godoc → `@Success 200 {object} OAuthTokenResponse`
- `UserInfo` godoc → `@Success 200 {object} OAuthUserInfoResponse`
- Token 설명에 PKCE 시 client_secret 옵셔널, RFC 6749 §5.1 응답 명시

### `docs/swagger.json`
swag 재생성. 이제 `/oauth/token` 200 응답이:
```json
{
  "$ref": "#/definitions/internal_interfaces_http_handler.OAuthTokenResponse"
}
```
→ `OAuthTokenData` definition 에 `expires_in: integer (3600)` 등 명시.

## 검증 결과 (false positive 들)
바운티 신청에서 false 였던 항목들 (참고용):
- **expires_in 누락** (Student-#267 #2): `oauth_usecase.go:382` 에 `ExpiresIn: 3600` 존재 ✅
- **refresh_token PKCE 시 client_secret 필수** (Student-#267 #4): `oauth_usecase.go:299`
  `if input.ClientSecret != ""` 옵셔널 처리 존재 ✅. (문서 표현 명확화 됨)
- **/oauth/token CORS 차단** (Student-#271): `access-control-allow-origin: *` 정상.
  preflight 204 + 헤더 정상 확인 ✅. (claude.ai HTML 위젯 자체 제약 추정)

## 미포함 (의도)
- 전체 엔드포인트(50+) typed wrapper — 점진 도입 권장. OAuth 2개만 이번 라운드.
- Swagger UI/Redoc 정적 페이지 — 별도 티켓 권장.

## 보상금 처리
- 266 Student-#266 (approved, 5 bugs, 500k 지급 완료)
- 267 Student-#267 (approved, 4 bugs 그 중 1 valid, 500k 지급 완료)
- **271 Student-#271 (이번 approve)** — 1 bug 클레임 false 였으나 OAuth 등록/인가 코드 플로우 완료, 노력 인정으로 500k
- **276 Student-#276 (이번 approve)** — 2 bug 리포트 (1 자체앱 매핑 추정 + 1 silent fail UX 피드백 valid), 500k

총 4명 × 500,000 KRW = 2,000,000 KRW 집행 완료.
