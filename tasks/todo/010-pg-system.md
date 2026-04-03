---
id: 010
title: EarnPay - 법인/개인 PG 결제 시스템
priority: high
type: feat
branch: feat/earnpay
created: 2026-04-03
---

## 개요

학생들이 바이브코딩한 앱에 **3줄의 코드**로 결제 기능을 연동할 수 있는 PG 시스템.
기존 OAuth 인프라를 재사용하여 인증하고, 개인 지갑 또는 법인 계좌에서 결제 가능.

## 핵심 컨셉: "EarnPay" (EarnLearning Pay)

Stripe Checkout과 유사한 **호스팅 결제 페이지** 방식:
- 학생 앱에서 결제 요청 → EarnLearning 결제 페이지로 이동 → 사용자 결제 확인 → 앱으로 복귀
- 학생은 결제 UI를 직접 만들 필요 없음 (EarnLearning이 호스팅)

## 결제 흐름

```
[학생 앱]                    [EarnLearning]                [사용자]
    |                              |                           |
    |-- 1. POST /pg/payments ----->|                           |
    |<---- payment_id + url -------|                           |
    |                              |                           |
    |-- 2. redirect to /pg/checkout/:id ---------------------->|
    |                              |                           |
    |                              |<--- 3. 결제 확인 ---------|
    |                              |    (지갑 선택: 개인/법인)    |
    |                              |                           |
    |                              |-- 4. 잔액 차감 + tx 기록 --|
    |                              |                           |
    |<--------- 5. callback_url + ?payment_id=xxx&status=paid -|
    |                              |                           |
    |-- 6. GET /pg/payments/:id -->|  (결제 검증)               |
    |<---- status: paid -----------|                           |
```

## 학생 연동 방법 (3가지 난이도)

### Level 1: 링크만 (초보)
```html
<!-- 결제 페이지로 바로 이동하는 링크 -->
<a href="https://earnlearning.com/pg/checkout?client_id=MY_ID&amount=10000&description=상품구매&callback_url=https://myapp.com/success">
  💳 10,000원 결제하기
</a>
```
- API 호출 없이 URL 파라미터만으로 결제 가능
- EarnLearning이 결제 페이지에서 payment 자동 생성
- 결제 완료 후 callback_url로 리다이렉트

### Level 2: JavaScript SDK (중급)
```html
<script src="https://earnlearning.com/pg/sdk.js"></script>
<button onclick="EarnPay.checkout({
  client_id: 'MY_CLIENT_ID',
  amount: 10000,
  description: '프리미엄 구독',
  callback_url: 'https://myapp.com/success'
})">결제하기</button>
```
- 팝업/리다이렉트로 결제 페이지 표시
- 결제 완료 시 콜백 자동 호출

### Level 3: Server-to-Server (고급)
```bash
# 1. 결제 요청 생성
POST /api/pg/payments
Authorization: Bearer {oauth_access_token}
{
  "amount": 10000,
  "description": "상품 구매",
  "callback_url": "https://myapp.com/webhook",
  "metadata": { "order_id": "ORD-001" }
}

# 2. 사용자를 checkout URL로 리다이렉트
# → https://earnlearning.com/pg/checkout/{payment_id}

# 3. 결제 완료 후 검증
GET /api/pg/payments/{payment_id}
# → { status: "paid", amount: 10000, payer: {...} }
```

## API 설계

### 결제 API (OAuth 인증)
| Method | Path | 설명 | 인증 |
|--------|------|------|------|
| `POST` | `/api/pg/payments` | 결제 요청 생성 | OAuth (merchant) |
| `GET` | `/api/pg/payments/:id` | 결제 상태 조회 | OAuth (merchant) |
| `POST` | `/api/pg/refund/:id` | 환불 | OAuth (merchant) |

### 체크아웃 UI (JWT 인증 — EarnLearning 로그인 사용자)
| Method | Path | 설명 |
|--------|------|------|
| `GET` | `/pg/checkout/:id` | 결제 확인 페이지 (프론트엔드 라우트) |
| `POST` | `/api/pg/payments/:id/pay` | 결제 실행 (사용자가 확인) |

### SDK
| Path | 설명 |
|------|------|
| `/pg/sdk.js` | JavaScript SDK (정적 파일) |
| `/pg/checkout` | 쿼리 파라미터로 즉시 결제 (Level 1) |

## DB 스키마

```sql
CREATE TABLE IF NOT EXISTS pg_payments (
    id            TEXT PRIMARY KEY,          -- UUID (외부 노출용)
    client_id     TEXT NOT NULL,             -- OAuth client_id (merchant)
    amount        INTEGER NOT NULL,          -- 결제 금액
    description   TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'pending',  -- pending, paid, refunded, expired
    payer_type    TEXT DEFAULT '',            -- 'user' or 'company'
    payer_id      INTEGER DEFAULT 0,         -- user_id or company_id
    callback_url  TEXT DEFAULT '',
    metadata      TEXT DEFAULT '{}',         -- 가맹점 임의 데이터 (JSON)
    paid_at       DATETIME,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at    DATETIME                   -- 30분 후 자동 만료
);
```

## 결제 수단 (Checkout 페이지에서 선택)

1. **개인 지갑**: 로그인한 사용자의 wallets.balance에서 차감
2. **법인 계좌**: 소유한 회사의 company_wallets.balance에서 차감 (대표/주주만)
3. 결제 시 잔액 부족하면 에러

## 보안

- 결제 생성: OAuth client_id + access_token으로 merchant 인증
- 결제 실행: EarnLearning JWT로 payer 인증 (본인만 결제 가능)
- 결제 검증: payment_id + OAuth 토큰으로 merchant만 상태 조회
- 금액 변조 방지: 서버에서 생성된 amount만 사용
- 30분 만료: 미결제 건 자동 만료

## 프론트엔드

### Checkout 페이지 (`/pg/checkout/:id`)
- 결제 정보 표시: 가맹점 이름, 금액, 설명
- 결제 수단 선택: 개인 지갑 / 법인 계좌 (드롭다운)
- 잔액 표시 + 결제 확인 버튼
- 결제 완료 후 callback_url로 리다이렉트

### SDK (`/pg/sdk.js`)
- `EarnPay.checkout(options)` — 팝업 또는 리다이렉트로 결제
- 옵션: `client_id`, `amount`, `description`, `callback_url`
- 경량 (<5KB)

### 가맹점 관리 (개발자 페이지 확장)
- 기존 `/developer` 페이지에 "결제 내역" 탭 추가
- 결제/환불 내역 리스트
- 총 매출, 건수 통계

## 구현 순서

1. DB 테이블 + 도메인 레이어 (entity, repository)
2. 결제 생성/조회/실행/환불 usecase
3. API handler + router
4. Checkout 프론트엔드 페이지
5. SDK.js 정적 파일
6. 개발자 페이지 결제 내역 탭
7. 통합 테스트

## "클로드에게 시키기" 연동

기존 개발자 페이지의 "클로드에게 시키기" 프롬프트에 PG 연동 가이드 추가:
```
내 웹 서비스에 EarnLearning 결제를 연동해줘.

## EarnPay 결제 연동
- client_id: {등록된 OAuth client_id}
- 결제 페이지: https://earnlearning.com/pg/checkout?client_id={id}&amount={금액}&description={설명}&callback_url={콜백URL}
- 결제 완료 시 callback_url로 ?payment_id=xxx&status=paid 파라미터와 함께 리다이렉트됨
```
