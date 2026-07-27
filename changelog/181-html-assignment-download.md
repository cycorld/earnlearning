# HTML 사업계획서 첨부 — 열지 않고 내려받게 만들기

**날짜**: 2026-07-27 · **티켓**: #176

## 무엇을 했나

정부지원금(사업계획서) 제출 첨부에 `.html` 파일을 허용했다. 단, 브라우저에서 **바로 열리지 않고 항상 다운로드**되게 만들었다.

제출 경로는 `POST /api/milestones/files` (`type=business_plan`)다. 정부지원금 신청 자체에는 첨부 필드가 없고, 사업계획서 첨부가 곧 제출 파일이다.

## 왜 필요했나

학생들이 사업계획서를 HTML 로 만들어 내는 경우가 있는데 지금까지는 확장자 검증에서 막혔다.

그런데 HTML 을 그냥 열어주면 위험하다. HTML 안에는 `<script>` 가 들어갈 수 있고, 그 스크립트가 **우리 사이트와 같은 출처(same-origin)** 에서 실행되면 로그인한 다른 사용자의 토큰을 훔칠 수 있다. 이걸 저장형 XSS(Stored Cross-Site Scripting)라고 한다.

핵심 아이디어는 간단하다. **파일은 받되, 절대 실행될 기회를 주지 않는다.**

## 어떻게 만들었나

### 1. 공개 정적 경로에는 넣지 않는다

기존 `/api/upload` → `/uploads/...` 는 nginx 가 그대로 서빙하는 **공개 경로**다. 여기에 `.html` 을 추가하면 곧바로 same-origin 렌더링이 된다. 그래서 이 경로는 손대지 않았다. `.html` 은 비공개 저장소(`private_uploads/`)에만 저장되고, 정적 서빙이 없는 별도 엔드포인트로만 내려간다.

### 2. 다운로드 응답 헤더로 실행을 막는다

```
Content-Disposition: attachment; filename="plan.html"
Content-Type: application/octet-stream
X-Content-Type-Options: nosniff
Content-Security-Policy: default-src 'none'; sandbox
```

- `attachment` — 브라우저가 렌더링 대신 저장하게 한다.
- `application/octet-stream` — "이건 그냥 바이트 덩어리"라는 뜻. HTML 로 해석하지 않는다.
- `nosniff` — 브라우저가 내용을 훔쳐보고 "이거 HTML 같은데?" 하고 추측(MIME sniffing)하는 걸 금지한다.
- CSP `sandbox` — 만약의 경우에도 스크립트가 못 돌게 하는 마지막 방어선.

MIME 은 DB 에 저장된 값(업로드할 때 클라이언트가 보낸 값이라 신뢰 불가) 대신 **확장자로 서버가 다시 판단**한다.

### 3. 내용 검증 — 하지만 과하지 않게

10MB 상한 안에서 파일 전체를 읽어 두 가지만 본다:

- UTF-8 로 올바른 텍스트인가
- NUL 바이트(`\x00`)가 없는가

`<script>` 같은 태그는 **막지 않는다**. 어차피 실행될 일이 없고, 평범한 사업계획서 HTML(스타일·차트 스크립트 포함)을 괜히 거부하면 기능이 쓸모없어지기 때문이다. 위험을 "내용 검사"가 아니라 "응답 헤더"로 막는 게 훨씬 확실하다.

여기서 작은 함정 하나: 앞 512바이트만 보고 UTF-8 검사를 하면 **한글 한 글자가 경계에 걸쳐 잘려서** 멀쩡한 파일이 거부된다. 그래서 전체를 버퍼에 담아 한 번에 검사한다(10MB 로 상한이 있으니 메모리도 안전).

### 4. 파일명 안전 검사와 정리(cleanup)

`../../etc/passwd.html` 같은 경로 탈출, 개행 문자(`\r\n`)로 응답 헤더를 조작하는 헤더 인젝션, 따옴표로 `Content-Disposition` 을 깨는 시도를 모두 거부한다. 한글 파일명은 그대로 허용한다.

또 업로드 도중 실패하거나 크기를 초과하면 **디스크에 남은 조각 파일을 반드시 지운다**. 예전 코드는 DB 실패 때만 지웠다.

### 5. 프론트엔드 — blob 새 탭 열기 금지

의외의 구멍이 여기 있었다. 기존 코드는 파일을 받아서 `URL.createObjectURL()` 로 blob 주소를 만든 뒤 `window.open()` 으로 새 탭에 열었다. **blob 주소는 만든 페이지와 같은 출처**라서, HTML blob 을 새 탭에 열면 서버가 붙인 헤더와 무관하게 우리 도메인에서 스크립트가 돈다.

그래서 PDF·이미지만 새 탭 열기를 허용하고, 나머지는 `<a download>` 로 저장하게 바꿨다. blob 타입도 항상 `application/octet-stream` 으로 강제한다.

## 사용한 프롬프트

> government-grant submissions must accept .html business-plan files, but HTML must be download-only and never rendered same-origin ... bounded 10MB full-content validation (UTF-8, reject NUL; do not unnecessarily reject normal business-plan HTML because it will never execute), safe filename checks, cleanup, and a protected/download endpoint that always sends Content-Disposition attachment plus safe Content-Type and nosniff

## 배운 점

- **차단 지점을 고르는 게 설계다.** "위험한 내용 찾기"는 항상 우회당한다. "실행될 수 없는 경로로만 내보내기"는 우회가 안 된다.
- 보안 헤더는 서버만 잘 붙인다고 끝이 아니다. 프론트엔드가 blob 으로 다시 열면 다 무너진다. **끝에서 끝까지** 봐야 한다.
- 부분 버퍼 검증은 멀티바이트 문자에서 오탐이 난다. 상한이 있으면 전체를 읽는 게 오히려 안전하다.
