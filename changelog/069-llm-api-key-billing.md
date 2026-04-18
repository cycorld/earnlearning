# 069. LLM API 키 학생 프로비저닝 + 매일 새벽 자동 과금

**날짜**: 2026-04-18
**태그**: 백엔드, LLM, 과금, cron, 프록시, TDD

## 무엇을 했나
이화여대 창업 수업 학생들이 LMS 로그인 한 번으로 `llm.cycorld.com` 의 자기 API 키를
받아 Claude Code / Cursor / curl 에서 쓸 수 있게 만들었다. 매일 새벽 03:33 KST 에
전날 사용한 토큰만큼 **지갑에서 자동으로 차감**되고, 잔액이 부족하면 **부채로**
쌓여 다음 과금 때 우선 차감된다.

### 새 페이지 `/llm`
좌하단 "더보기 → LLM 키" 에서 진입:
- **첫 방문**: 자동으로 llm-proxy 에 학생 등록 + 키 발급 → 평문 키를 **1회만** 노출
  (복사 후 새로고침하면 사라짐)
- **이후 방문**: 키 prefix + 발급일만 보임, 평문은 재조회 불가
- **재발급 버튼**: 기존 키 즉시 폐기 + 새 키 발급
- **요금 기준 카드**: Opus 4.7 공식가 × 환율 1 USD = 1,400원 기준
- **사용량 · 청구 내역 표**: 최근 30일 일별 입력/출력 토큰, 청구액, 부채

### 새 백엔드 엔드포인트
- `GET /api/llm/me` — 내 키 조회 (없으면 자동 발급, 평문은 첫 응답에서만)
- `POST /api/llm/me/rotate` — 키 재발급 (기존 revoke + 새 키 발급)
- `GET /api/llm/me/usage?days=N` — 일별 사용량 + 누적/주간 요약

### 자정(정확히는 KST 03:33) 과금 크론
`backend/internal/infrastructure/scheduler/billing.go` 에 `time.NewTicker` 대신
**정해진 시각에 다음 03:33 을 계산하고 sleep → 실행 → 반복** 하는 고루틴.
`main.go` 에서 `LLM_ADMIN_API_KEY` env 가 있을 때만 기동한다 (없으면 개발 환경처럼
조용히 비활성).

## 왜 필요했나
학생이 수업 시간에 매번 강의자 어드민 키를 공유받거나, Anthropic 콘솔에서 직접
결제·발급받게 두기엔 보안 / 책임 경계가 너무 흐려진다.

- **경계가 명확해짐**: 각 학생은 자기 이름으로 발급된 키만 사용 → 로그에도 분리
- **피드백 루프가 붙음**: 본인이 쓴 만큼만 본인 지갑에서 빠지니까 "LLM 을 얼마나
  썼는지 현실감" 이 생긴다. LMS 의 자본 게임이랑도 자연스레 붙는다
- **규율이 잡힘**: 토큰을 물쓰듯이 쓰면 진짜 돈이 사라진다 → 프롬프트 엔지니어링 /
  모델 효율 같은 주제를 수업에서 체험형으로 가르칠 수 있음

## 어떻게 만들었나

### 외부 API 탐색 (llm.cycorld.com Admin API)
```
POST /admin/api/students            # 학생 등록 (email unique)
POST /admin/api/students/{id}/keys  # 키 발급 (평문 1회 반환)
POST /admin/api/keys/{id}/revoke    # 키 폐기
GET  /admin/api/usage?days=N        # 최근 N일 rolling 집계
```
Swagger 스펙은 `https://llm.cycorld.com/admin/openapi.json` — 이 PR 의 Go 클라이언트
(`backend/internal/infrastructure/llmproxy/client.go`) 가 그대로 미러링한다.

### 백엔드 레이어별 책임
| 레이어 | 파일 | 책임 |
|---|---|---|
| 도메인 (순수 로직) | `internal/domain/llm/` | 엔티티, 가격 계산, 과금 시각 계산 |
| 외부 어댑터 | `internal/infrastructure/llmproxy/` | HTTP 클라이언트 + usecase adapter |
| 영속성 | `internal/infrastructure/persistence/llm_repo.go` | SQLite CRUD |
| 유스케이스 | `internal/application/llm_usecase.go` | `EnsureKey` / `RotateKey` / `BillAll` |
| 스케줄러 | `internal/infrastructure/scheduler/billing.go` | KST 03:33 루프 |
| HTTP | `internal/interfaces/http/handler/llm_handler.go` | 3개 엔드포인트 |

### 과금 공식
```
cost_usd =  prompt_tokens × (1 - cache_ratio) × 15 / 1M
         + prompt_tokens × cache_ratio × 1.50 / 1M   # 캐시 히트 90% 할인
         + completion_tokens × 75 / 1M
cost_krw = round(cost_usd × 1400)
cache_ratio = min(1, cache_hits / requests)
```
수업에서 설명하기 좋게 환율은 1 USD = 1,400원 고정. 캐시 할인은 LLM proxy 가
cached-token 수를 따로 주지 않아 **요청 수 비율로 근사**했다.

### 새벽 03:33 에 과금 (자정이 아닌 이유)
- LLM proxy 의 로그 flush 타이밍 여유
- LMS 의 다른 자정 로직과 섞이지 않음
- "일일 지표" 로 취급하되 전날 분으로 귀속 — `llm.BillingDate(now)` 이 자동으로
  전날 KST 달력일자를 돌려줌

### 지갑 부족 → 부채 원장
기존 `wallet.Debit` 은 잔액 부족 시 reject. 여기서는 그 규칙을 뚫고 싶지 않아서,
usecase 레벨에서:
```
debit = min(cost, balance)
debt  = cost - debit
```
으로 쪼갠 뒤, `debit > 0` 일 때만 `Debit` 을 호출. `llm_daily_usage` 테이블에
(user_id, usage_date) UNIQUE 로 저장해서 **재실행에 안전**.

### 알림
매 과금마다 `NotifLLMBilled` 알림을 생성 — 부채가 발생하면 제목에 "(부채 N원)" 이
붙고 본문에 "지갑 충전 후 다음 과금 주기에 우선 차감됩니다" 안내가 추가됨.

## 테스트
- **pure 도메인 로직**: `CostKRW` + `NextBillingTime` 14 개 케이스 (환율, 캐시 비율
  경계, KST timezone edge, 음수 clamp 등)
- **proxy 클라이언트**: `httptest.Server` 로 7 개 — Bearer 주입, 요청 바디 직렬화,
  4xx 에러, 쿼리 파싱
- **usecase (fake proxy/repo/wallet)**: 9 개 — 최초 발급 / 재조회 plaintext 차단 /
  이메일 없음 / 회전 / zero usage noop / 정상 과금 / 부분 차감+부채 / 잔액 0 /
  idempotent upsert
- **통합 (SQLite + 실제 HTTP 라우트 + fake proxy)**: 5 개 — 자동 발급, 재조회, rotate
  revoke 확인, 정상 과금, 부채 기록
- 총 신규 35 테스트, 전체 backend 306 테스트 통과

## 설계 메모 / 시행착오
- **proxy 가 key_id 를 발급 응답에 안 줌**: Swagger 스펙상 `IssuedKey` 는 `{key,
  prefix, label, warning}` 만 반환. 그래서 adapter 에서 발급 직후 `ListKeys(studentID)`
  로 다시 훑어 방금 만든 prefix 를 매칭해 id 를 뽑아내도록 했다.
- **SQLite 가 DATE 컬럼을 "2026-04-17T00:00:00Z" 로 돌려줌**: `UsageDate` 를 INSERT
  할 땐 10자 문자열로 넣었는데, SELECT 시엔 RFC3339 로 복구됨. 파싱 헬퍼
  `parseUsageDate(raw)` 에서 길이에 따라 분기 + 양쪽 포맷 수용.
- **`onSelect` 충돌 (React)**: 프론트 `SkillTree` 컴포넌트에서 겪었던 건데,
  `onClick` / `onSubmit` / `onSelect` 같은 네이티브 HTML 이벤트와 이름이 겹치면
  `extends React.HTMLAttributes` 가 타입 체킹을 거부함. 기억해둘 것.

## 배운 점
- **"자정 크론" 은 보통 자정이 아니다**: 학교 수업 맥락에서 자정은 학생들이 과제
  제출하느라 바쁜 시간. 03:33 처럼 **아무도 깨있지 않을 시각** 을 잡는 게 운영
  관점에서 훨씬 안전. 이름만 "midnight billing" 이지 실제 시각은 맥락별로 옮길 수
  있게 상수로 뺐다 (`BillingHour` / `BillingMinute` in `domain/llm/schedule.go`).
- **외부 API 는 usecase 가 아니라 adapter 에 가두라**: `llmproxy.Client` 가 HTTP 를
  알고, `llmproxy.UseCaseAdapter` 가 도메인 타입으로 번역한다. usecase 테스트는
  가짜 ProxyClient 만 주입하면 끝 — 네트워크 없어도 모든 경로 커버됨.
- **환율·가격 상수는 domain 안에**: `llm/pricing.go` 안에 `USDToKRW = 1400.0`,
  `InputPricePerMTokUSD = 15.0` 같은 숫자를 두고 테스트로 강제함. 가격 정책이
  바뀔 때 수정 지점이 한 파일로 모임.

## 다음 단계
- 관리자 대시보드에 "전체 학생의 LLM 사용량 / 미회수 부채" 뷰 추가
- 월별 집계 리포트 PDF 출력 (회계 소스)
- 예산 한도 기능 (학생당 월 최대 원화 초과 시 키 자동 폐기)
