# EarnLearning OAuth 연동 예제

EarnLearning API에 OAuth2를 통해 연동하는 방법을 보여주는 예제입니다.

## 빠른 시작 (5분)

### 1. 앱 등록

EarnLearning에 로그인한 후 API를 호출하여 앱을 등록합니다:

```bash
curl -X POST https://earnlearning.com/api/oauth/clients \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "내 앱",
    "description": "테스트용 앱",
    "redirect_uris": ["http://localhost:3000/callback"],
    "scopes": ["read:profile", "read:wallet"]
  }'
```

응답에서 `client_id`와 `client_secret`을 저장하세요.

### 2. 예제 실행

```bash
# 단순히 HTML 파일을 브라우저에서 열기
open index.html

# 또는 간단한 서버로 실행
python3 -m http.server 3000
```

### 3. 연동 테스트

1. `index.html`을 열어 `client_id`와 `redirect_uri` 입력
2. "EarnLearning으로 로그인" 클릭
3. 인가 → 토큰 교환 → API 호출!

## 주요 개념

### OAuth2란?

OAuth2는 사용자가 **비밀번호를 공유하지 않고** 외부 앱에 제한된 접근 권한을 부여하는 프로토콜입니다.

```
사용자 → 외부 앱 → EarnLearning (인가) → 외부 앱 (토큰) → API 호출
```

### Authorization Code + PKCE 플로우

이 예제에서 사용하는 플로우입니다:

```
1. 외부 앱이 code_verifier(비밀 랜덤 문자열)를 생성
2. code_challenge = SHA256(code_verifier)를 계산
3. 사용자를 인가 페이지로 리다이렉트 (code_challenge 포함)
4. 사용자가 "허용" 클릭 → redirect_uri?code=xxx로 돌아옴
5. 외부 앱이 code + code_verifier로 토큰 교환 요청
6. 서버가 SHA256(code_verifier) == code_challenge 검증 후 토큰 발급
```

**왜 PKCE인가요?**
- SPA(브라우저 앱)는 `client_secret`을 안전하게 보관할 수 없음
- PKCE는 `client_secret` 없이도 인가 코드 탈취 공격을 방지

### 스코프

| 스코프 | 설명 |
|--------|------|
| `read:profile` | 프로필 조회 |
| `write:profile` | 프로필 수정 |
| `read:wallet` | 지갑 잔액/거래 조회 |
| `write:wallet` | 송금 |
| `read:posts` | 게시물/댓글 조회 |
| `write:posts` | 게시물/댓글 작성, 좋아요 |
| `read:company` | 회사 정보 조회 |
| `write:company` | 회사 정보 수정 |
| `read:market` | 프리랜서/거래소/투자 조회 |
| `write:market` | 프리랜서 등록, 주문, 투자 |
| `read:notifications` | 알림 조회 |

### 토큰 관리

- **Access Token**: API 호출용, 1시간 유효
- **Refresh Token**: 액세스 토큰 갱신용, 30일 유효
- 만료된 액세스 토큰 → `refresh_token` grant로 갱신
- 더 이상 필요 없으면 → `/api/oauth/revoke`로 폐기

## 코드 설명

### PKCE 생성 (`index.html`)

```javascript
// 1. 랜덤 code_verifier 생성 (43~128자)
function generateCodeVerifier() {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return btoa(String.fromCharCode(...array))
    .replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}

// 2. SHA-256 해시로 code_challenge 생성
async function generateCodeChallenge(verifier) {
  const data = new TextEncoder().encode(verifier);
  const hash = await crypto.subtle.digest('SHA-256', data);
  return btoa(String.fromCharCode(...new Uint8Array(hash)))
    .replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}
```

### 토큰 교환

```javascript
// 인가 코드를 토큰으로 교환
const response = await fetch(API_BASE + '/api/oauth/token', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    grant_type: 'authorization_code',
    code: authorizationCode,      // 인가 서버에서 받은 코드
    client_id: clientId,
    redirect_uri: redirectUri,
    code_verifier: codeVerifier,  // PKCE 원본 verifier
  }),
});
```

### API 호출

```javascript
// Bearer 토큰으로 API 호출
const response = await fetch(API_BASE + '/api/oauth/userinfo', {
  headers: {
    'Authorization': 'Bearer ' + accessToken,
  },
});
```

### 에러 처리

```javascript
// 401 에러 시 토큰 갱신 시도
if (response.status === 401) {
  const newToken = await refreshAccessToken();
  // 새 토큰으로 재시도
}
```

## API 레퍼런스

전체 API 문서는 https://earnlearning.com/docs 에서 확인할 수 있습니다.
