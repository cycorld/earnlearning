# 114. 투자 페이지에 회사 정보 노출 (대표자 · 소개 · 서비스 URL)

**날짜**: 2026-05-08
**태그**: 투자, UX, 회사정보, dogfood

## 배경
투자 페이지에서 학생이 어디 회사에 투자할지 결정해야 하는데, 라운드 정보(목표 금액 / 지분 / 주당 가격)만 보이고 **회사 자체에 대한 정보 가시성이 부족**. 회사 description·service_url 이 DB 에는 있지만 API 응답에 안 실렸음.

## 추가

### Backend (`backend/internal/...`)
- `domain/investment/entity.go` — `RoundCompany` 에 `Description`, `ServiceURL` 필드 추가 (`omitempty`)
- `infrastructure/persistence/investment_repo.go` — `FindRoundByID` / `ListRounds` 두 곳 모두 SELECT 에 `c.description`, `c.service_url` 추가하고 `RoundCompany` 에 채워 반환

### Frontend
- `types/index.ts` — `InvestmentRound.company` 타입에 `description?`, `service_url?` 추가
- **`routes/invest/InvestPage.tsx`** (리스트):
  - 카드에 **대표자 이름** (`displayName(round.owner)`) 노출
  - 회사 소개 1~2줄 snippet (80자 truncate, `line-clamp-2`)
  - 서비스 URL 클릭 시 새 창 열기 (외부 링크 아이콘)
- **`routes/invest/InvestDetailPage.tsx`** (상세):
  - 라운드 헤더에 service_url 클릭 가능한 링크
  - 별도 **"회사 소개" 카드** 추가 — markdown 원문 그대로 (`MarkdownContent`, maxLines 20)

## 회귀 테스트
`InvestPage.test.tsx` 5 tests:
- 대표자 이름 표시
- 서비스 URL 바로가기 버튼
- description snippet 표시
- 80자 초과 시 `…` truncate
- description / service_url 없으면 안 그림 (안전)

전체: frontend 155 pass · backend 39 pass (smoke + investment).

## 미포함 (의도)
- 회사 description 편집 UI 변경 — 이미 회사 페이지에서 수정 가능. 별도 티켓 X.
- 투자 알고리즘 변경 X.
- 라운드 카드 디자인 다른 변경 X — surgical change.
