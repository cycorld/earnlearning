# 검색엔진이 읽을 수 있는 안내 파일 추가

## 무엇을 했나요?

검색엔진용 `robots.txt`와 `sitemap.xml`을 추가했습니다. 이전에는 두 주소 모두 애플리케이션 HTML을 반환해 검색엔진이 사이트 구조를 정확히 이해하기 어려웠습니다.

## 왜 필요한가요?

- `robots.txt`는 검색로봇에 공개 범위와 sitemap 위치를 알려줍니다.
- `sitemap.xml`은 검색엔진에 대표 공개 URL을 명확히 전달합니다.
- EarnLearning에는 학생·관리자·금융 시뮬레이션 데이터가 있으므로 로그인 이후 화면은 검색 대상에서 제외했습니다.

## 어떻게 만들었나요?

Vite의 `public/` 정적 파일로 두 문서를 제공하고, 계약 테스트에서 canonical URL과 비공개 경로 차단 규칙을 검증합니다.

## 사용한 프롬프트

> 등록은 내가 할테니 robots sitemap 파일들 작업해줘

## 배운 점

SPA fallback이 HTTP 200을 반환한다고 해서 `robots.txt`나 `sitemap.xml`이 정상인 것은 아닙니다. 응답 본문과 Content-Type까지 확인해야 합니다.
