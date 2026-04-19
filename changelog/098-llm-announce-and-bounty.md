# 098. LLM API 사용 안내 공지 + LLM 연동 버그바운티 (운영)

**날짜**: 2026-04-19
**태그**: 운영, 공지, LLM, 바운티

## 배경
강의에서 학생 개인 LLM API 키 발급 기능(`/llm`)을 출시했지만 **공식 안내 공지가 없었음**.
학생들이 키 페이지가 어디 있는지, 어떤 도구에 어떻게 붙여 쓰는지, 비용이 어떻게 빠지는지를
모르고 있었기 때문에 사용량이 거의 0 상태로 정체.

또한 OAuth 바운티(grant 9) 가 마무리되어 다음 라운드의 실무 연동 학습 동기가 필요했다.

## 작업
운영성 작업 — **코드 변경 0줄**, 외부 시스템 데이터만 변경.

### 1) Stage 캡처
Playwright 로 `https://stage.earnlearning.com/llm` 흐름 자동 캡처:
- `llm-more-menu.png` — 하단바 더보기 → LLM 키 메뉴 위치
- `llm-page-full.png` — LLM API 키 페이지 전체 (서비스 상태 / 내 키 / 요금 / 사용량)

### 2) Prod 업로드
`/api/upload` 로 admin JWT 사용해 prod 에 업로드:
- `/uploads/c24c864d-6b78-4ba4-9ab2-16664bd0a2c3.png`
- `/uploads/152c6208-f42d-4ced-98ba-4ea43ab0be3a.png`

### 3) 공지 게시 (post 102)
`POST /api/channels/1/posts` 로 공지 채널에 마크다운 게시.
구성: 메뉴 진입 → 발급/재발급 → curl/Claude Code/Cursor 사용 예시 → 요금(Opus 4.7 환산) → 03:33 KST 자동 차감 → 서비스 상태 카드 안내 → 바운티 공모 안내.

### 4) Grant 14 생성 (post 103 자동 게시)
`POST /api/admin/grants` — `[5주차] LLM API 연동 버그바운티`, 5명, 각 500,000원.

제출 양식: 연동 도구, 정상/막힌 기능, 발견 버그(재현법 포함), 개선 제안.
선착순 5명 자동 마감. AutoPoster 가 과제 채널(3) 에 자동 게시.

## 미포함 (의도)
- 코드 변경 — 이번 라운드는 사용 활성화가 우선. 바운티 결과 보고 다음 PR 에서 fix
- 강의 시간 데모 — 다음 수업에서 라이브 데모 예정 (별도 처리)

## 다음 액션
바운티 신청이 오면:
1. proposal 검토 후 admin API `POST /api/admin/grants/14/approve/{appId}` 로 자동 500k 지급
2. 발견된 실 버그가 있으면 별도 fix 티켓 생성
