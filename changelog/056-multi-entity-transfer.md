---
slug: 056-multi-entity-transfer
title: 송금 기능 확장 — 개인↔법인 송금 + 법인 전용 지갑 페이지
date: 2026-04-17
tags: [feat, 지갑, 법인, 교육, 백엔드, 프론트엔드]
---

# 송금 기능 확장 — 개인↔법인 송금 + 법인 전용 지갑 페이지

## 왜 필요했는가

우리 LMS는 "스타트업 창업 시뮬레이션"이 핵심 교육 목표입니다. 그런데 지금까지 송금 기능은 **개인 → 개인** 만 가능했어요.

현실에서 돈은 4가지 방향으로 흐릅니다:
- 개인 → 개인 (월급 받은 친구한테 밥 사주기)
- 개인 → 법인 (스폰서십, 선납금, 제품 구매)
- 법인 → 개인 (월급, 프리랜서 비용, 배당 대신 상여)
- 법인 → 법인 (B2B 결제, 계열사 자금 이동)

학생들이 "법인 자산과 개인 자산은 다르다" 는 핵심 개념을 체감하려면, 4가지 흐름 모두가 가능해야 합니다. 특히 **창업가가 법인 계좌에서 개인 계좌로 함부로 돈을 빼가면 횡령** 이라는 것도 자연스럽게 배울 수 있죠.

게다가 현재 회사 상세 페이지는 법인 잔액을 "읽기 전용" 숫자 하나로만 보여줘서, 학생들이 "내 회사에 돈이 얼마나 있지?" 를 확인하기 어려웠고, 거래 내역도 볼 수 없었습니다.

## 무엇을 했는가

### 1. 백엔드 — 송금 매트릭스 4종 모두 지원

이전에는 `POST /api/wallet/transfer` 에서 `target_type=company` 를 **명시적으로 거부** 하고 있었어요:

```go
if input.TargetType == "company" {
    return fmt.Errorf("회사 송금은 회사 ID가 아닌 대표의 user_id로 전달해야 합니다")
}
```

이걸 걷어내고, `target_type` 에 따라 분기하도록 고쳤습니다. `target_user_id` 필드는 이름 그대로 두되, `target_type="company"` 일 때는 **company_id** 로 해석합니다.

- **개인 → 법인**: `Debit(개인 지갑)` + `CreditCompanyWallet(법인 지갑)`. 실패 시 Credit 으로 환불해서 돈이 사라지지 않게 했습니다.
- **법인 → 개인 / 법인**: 새 엔드포인트 `POST /api/companies/:id/transfer`. 대표(OwnerID) 만 호출할 수 있고, 아니면 `403 NOT_OWNER` 를 돌려줍니다.

### 2. 법인 수신자 검색 — `SearchRecipients` 가 법인도 반환

`/api/wallet/recipients?q=...` 가 이제 개인(`type: "user"`) + 법인(`type: "company"`) 을 함께 반환합니다. 청산된 법인은 수신자 목록에서 제외.

프론트엔드 `Recipient` 타입은 이미 `type: 'user' | 'company'` 였기 때문에, 백엔드만 확장하면 UI는 자연스럽게 법인을 표시할 수 있었습니다.

### 3. 법인 전용 지갑 페이지 신설

`/company/:id/wallet` 경로를 만들어서, **법인 지갑만 위한 독립된 페이지**를 추가했습니다. 개인 지갑(`/wallet`) 과 **시각적으로 확실히 구분** 되도록 했어요:

- 개인 지갑: primary 컬러(블루) gradient
- **법인 지갑: purple → indigo gradient + Building2 아이콘 + "법인 계좌" 배지**

상단에 잔액, 하단에 거래 내역(페이지네이션), 중간에 "법인에서 송금하기" 버튼(대표만 노출) 이 있습니다. 대표가 아닌 사람에게는 "법인 송금은 대표만 할 수 있습니다" 안내를 보여줘서 권한 모델을 학습적으로 드러냈습니다.

회사 상세 페이지 (`/company/:id`) 하단 "법인 계좌 관리" 버튼으로 진입합니다. 버튼에 잔액도 같이 표시해서 "그 회사의 지갑에 얼마가 있는지" 바로 볼 수 있어요.

### 4. 거래내역 — 법인 거래 API 추가

`GET /api/companies/:id/transactions?page=X&limit=Y` 로 법인 거래 내역을 페이지네이션해서 조회합니다. 투자/배당금/KPI 수익/공시 보상/청산/법인 송금 등 모든 법인 돈의 흐름이 여기 기록됩니다.

### 5. TDD — 회귀 테스트 8종 추가

`backend/tests/integration/multi_entity_transfer_test.go`:

1. `TestTransfer_UserToCompany_Success` — 개인 → 법인 송금 성공, 양쪽 잔액 변동 확인
2. `TestTransfer_UserToCompany_InsufficientFunds_RollsBack` — 잔액 부족 시 롤백 (양쪽 지갑 변동 없음)
3. `TestTransfer_CompanyToUser_ByOwner_Success` — 대표가 법인 → 개인 송금 성공
4. `TestTransfer_CompanyToCompany_Success` — 법인 → 법인 송금 성공
5. `TestTransfer_CompanyToUser_NonOwner_Forbidden` — 대표 아닌 유저의 법인 송금 시도 → 403
6. `TestTransfer_CompanyToUser_InsufficientFunds` — 법인 잔액 부족 시 거부
7. `TestSearchRecipients_IncludesCompanies` — 수신자 검색 결과에 법인 포함 (`type: "company"`)
8. `TestGetCompanyWallet_ReturnsBalanceAndTransactions` — 법인 지갑 조회 + 거래내역 조회

## 사용한 프롬프트

> "우리 송금 보내는 기능이 있잖아. 개인<->개인, 개인<->법인, 법인<->법인 도 가능해? 그리고 법인을 위한 계좌 관리 페이지가 따로 있거나. 기존 지갑에서 내 법인 계좌로 체인지하는 기능이 필요할거 같은데."
>
> "바로 3가지 갭을 구현해줘. 그리고 회사 페이지에 wallet 페이지를 만들어서 개인과 법인의 재정이 분리됨을 보여줘. (나중에 편의를 위해 기존 지급에서 드랍다운으로 쉽게 계좌들 관리하게 하자.)"

## 배운 점 / 설계 메모

### 왜 "target_user_id 이름은 그대로 두고 target_type 으로 분기" 로 갔는가

이미 기존 API 가 이 필드명을 쓰고 있었고, 프론트 `Recipient.type` 도 이미 `'user' | 'company'` 로 디자인되어 있어서 **API 형태를 바꾸지 않고 백엔드만 해석 규칙을 확장** 하는 게 가장 저렴했습니다. 만약 `target_id` 로 개명했다면 모든 호출자를 고쳐야 했을 거예요.

### 왜 법인 송금은 별도 엔드포인트(`/companies/:id/transfer`) 로 갔는가

개인 송금은 "누가 보내는지" 가 JWT 의 user_id 로 자연스럽게 결정됩니다. 하지만 법인 송금은 "한 유저가 여러 법인의 대표일 수 있고, 그 중 어느 법인에서 보내는지" 가 명시적이어야 합니다. URL path 에 companyID 를 박아두면 권한 체크(`c.OwnerID == actorID`) 가 명확해지고, 라우팅 레벨에서부터 "법인 → X" 임이 드러나서 로그 읽기도 쉬워져요.

### 왜 UI 톤을 다르게 했는가 (purple vs primary blue)

교육용 LMS 에서 가장 위험한 시나리오는 학생이 **"법인 돈이 곧 내 돈"** 이라고 오해하는 것입니다. 실제 세계에선 법인 계좌에서 개인 계좌로 함부로 돈을 빼면 횡령이에요. UI 톤을 시각적으로 확실히 다르게 해서, 개인 지갑과 법인 지갑이 **다른 주머니** 라는 걸 보자마자 알 수 있게 했습니다. 배경 그라디언트 + "법인 계좌" 배지 + 명시적 경고 문구까지 3단계로 신호를 줬어요.

### 향후 과제

- 기존 개인 지갑 페이지(`/wallet`) 상단에 **계좌 스위처 드롭다운** 추가: "개인 계좌 / ○○법인 계좌 / △△법인 계좌" 간 빠른 전환 (다음 티켓)
- 법인 송금 시 개인과 마찬가지로 수신자에게 알림 (`CreateNotification` 연동)
- 대표뿐 아니라 공동창업자(co-founder) 역할도 법인 송금 권한을 가지도록 확장 (지금은 단독 OwnerID 만 체크)
