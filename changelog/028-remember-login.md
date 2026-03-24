# 028. 아이디 저장 & 로그인 유지 + 댓글 작성자 버그 수정

> **날짜**: 2026-03-24
> **태그**: `인증`, `UX`, `JWT`, `버그수정`

## 무엇을 했나요?

1. 로그인 페이지에 **아이디 저장**(기본 켜짐)과 **로그인 유지**(6개월) 체크박스를 추가했습니다.
2. 댓글을 작성하면 작성자 이름이 `?`로 표시되던 버그를 수정했습니다.

## 왜 필요했나요?

### 아이디 저장 & 로그인 유지
기존에는 매번 이메일을 직접 입력해야 했고, JWT 토큰이 24시간이면 만료되어 매일 다시 로그인해야 했습니다. 학생들이 자주 접속하는 서비스인 만큼 편의성이 중요합니다.

### 댓글 작성자 버그
댓글을 달면 API가 작성자 이름을 빈 문자열로 반환해서, 프론트엔드의 `displayName("")` 함수가 `?`를 표시했습니다. 새로고침하면 댓글 목록 조회 API가 DB JOIN으로 작성자 정보를 채워서 정상 표시되었지만, 작성 직후에는 깨져 보이는 문제였습니다.

## 어떻게 만들었나요?

### 아이디 저장 (프론트엔드만)
- `localStorage`에 이메일과 체크 상태를 저장
- 다음 방문 시 저장된 이메일을 자동으로 입력 필드에 채움
- 체크 해제하면 저장된 이메일 삭제

```
사용한 프롬프트: "아이디 저장(디폴트 활성화), 로그인 유지 기능 추가해줘. 로그인 유지는 반기 정도는 되면 좋겠어."
```

### 로그인 유지 (프론트엔드 + 백엔드)
- 백엔드: `LoginInput`에 `remember_me` 필드 추가
- `remember_me=true`이면 JWT 만료를 **180일**(약 6개월)로 설정
- `remember_me=false`이면 기존과 동일하게 **24시간**

```go
// 백엔드 코드 핵심 부분
duration := 24 * time.Hour
if input.RememberMe {
    duration = 180 * 24 * time.Hour // 6개월
}
token, err := uc.generateTokenWithDuration(u, duration)
```

### 댓글 작성자 버그 수정
- `PostUsecase`에 `userRepo`(사용자 정보 저장소) 의존성을 추가
- `CreateComment`에서 댓글 생성 후 `userRepo.FindByID()`로 작성자 정보를 조회하여 응답에 포함

```go
// 수정 전: author 정보 비어있음
c.ID = commentID
c.CreatedAt = time.Now()

// 수정 후: author 정보 채워넣음
if u, err := uc.userRepo.FindByID(userID); err == nil {
    c.AuthorName = u.Name
    c.AuthorAvatar = u.AvatarURL
    c.AuthorStudentID = u.StudentID
    c.AuthorDepartment = u.Department
}
```

## 배운 점

- **API 응답의 일관성이 중요하다**: 목록 조회(GET)에서는 JOIN으로 관련 데이터를 채워주지만, 생성(POST) 응답에서는 빠뜨리기 쉽습니다. 프론트엔드가 응답을 그대로 상태에 추가하므로, 생성 API도 조회 API와 동일한 형태의 데이터를 반환해야 합니다.
- **JWT 만료 시간은 용도에 따라 다르게**: 보안이 중요한 서비스는 짧게, 편의성이 중요한 교육 서비스는 길게 설정할 수 있습니다. `remember_me` 옵션으로 사용자에게 선택권을 주는 것이 좋습니다.
- **회귀 테스트의 가치**: 기존 테스트가 `author.name`의 존재만 확인하고 값이 비어있는지는 검증하지 않아 버그를 잡지 못했습니다. 값의 유효성까지 검증하는 회귀 테스트를 추가했습니다.
