# 076. context7 SearchResult.trustScore 타입 수정 (int → float64)

**날짜**: 2026-04-19
**태그**: 챗봇, Context7, 버그수정

## 버그
context7 활성화 후 첫 `context7_search` 호출 실패:
```
ctx7 decode: json: cannot unmarshal number 9.4 into Go struct field SearchResult.results.trustScore of type int
```

## 원인
Context7 API 가 `trustScore` 를 정수가 아닌 소수점 값으로 반환 (예: 9.4, 8.73).
초기 구현 시 int 로 선언한 게 잘못.

## 수정
`SearchResult.TrustScore int → float64`.

## 결과
재빌드 + 배포 후 TanStack Query 관련 질문에서 context7_search 정상 반환 확인 예정.
