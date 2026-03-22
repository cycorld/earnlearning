# 025. API 문서화 + OAuth2 시스템

**날짜**: 2026-03-22
**태그**: `API`, `OAuth`, `보안`, `문서화`

## 무엇을 했나요?

1. **API 문서 자동생성**: swaggo로 70+ 엔드포인트에 어노테이션 추가, Scalar UI로 `/docs`에서 API 문서를 볼 수 있게 했습니다.
2. **OAuth2 시스템**: 외부 앱이 사용자 데이터에 안전하게 접근할 수 있는 OAuth2 Authorization Code + PKCE 플로우를 구현했습니다.
3. **프론트엔드**: 개발자 설정 페이지(앱 등록/삭제)와 OAuth 인가 동의 화면을 만들었습니다.
4. **예제 앱**: `examples/oauth-demo/`에 학생들이 참고할 수 있는 단일 HTML 데모를 만들었습니다.

## 왜 필요했나요?

학생들이 EarnLearning API를 활용해 자기만의 연동 서비스를 만들고 싶어했습니다. 하지만 API 문서가 없어서 어떤 엔드포인트가 있는지 알 수 없었고, OAuth가 없어서 외부 앱이 사용자 데이터에 안전하게 접근할 방법이 없었습니다.

## 어떻게 만들었나요?

### API 문서화
- Go의 `swaggo` 라이브러리로 각 핸들러 함수에 주석 형태의 어노테이션 추가
- `swag init`으로 OpenAPI JSON 스펙 자동 생성
- Scalar CDN으로 모던 API 문서 UI 서빙

### OAuth2 시스템
- **Clean Architecture**: domain → repository → usecase → handler 레이어 분리
- **PKCE (S256)**: SPA/모바일 앱에서 client_secret 없이도 안전하게 인증
- **스코프 11종**: read:profile, write:wallet 등 세분화된 권한 제어
- **토큰 수명**: access_token 1시간, refresh_token 30일, 인가코드 10분

### 프론트엔드
- `/developer` — 앱 등록/목록/삭제 + AI 연동 프롬프트 자동생성
- `/oauth/authorize` — 인가 동의 화면 (스코프별 권한 표시, write 권한 경고)

## 사용한 프롬프트

```
내 웹 서비스에 EarnLearning OAuth 로그인을 연동해줘.
(개발자 설정 페이지에서 앱 정보가 자동 반영된 프롬프트를 복사할 수 있습니다)
```

## 배운 점

- **OAuth2 플로우**: Authorization Code + PKCE가 SPA에서 가장 안전한 인증 방식입니다. client_secret을 브라우저에 노출하지 않으면서도 인가 코드 탈취를 방지합니다.
- **API 문서화**: swaggo 어노테이션은 코드와 문서를 같은 곳에서 관리해서 문서가 코드와 따로 노는 문제를 줄여줍니다.
- **스코프 설계**: read/write를 분리하면 사용자가 앱에 최소 권한만 부여할 수 있어 보안에 유리합니다.
