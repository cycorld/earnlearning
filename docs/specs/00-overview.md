# EarnLearning Technical Specification — Overview

## 1. 프로젝트 개요

**제품명**: EarnLearning
**목적**: 게임화된 창업 교육 시뮬레이션 LMS
**대상**: 이화여자대학교 "스타트업을 위한 코딩입문" 수강생

학생들이 **가상 자본 5,000만원**으로 실제 창업 생태계를 시뮬레이션한다.
학기 말 **최종 자산가치가 성적에 반영**된다.

---

## 2. 기술 스택

| 레이어 | 기술 | 비고 |
|--------|------|------|
| Backend | Go (Echo) | DDD 아키텍처 |
| Database | SQLite (WAL 모드) | Docker volume 영속화 |
| Frontend | Vite + React 18 | TypeScript + Tailwind CSS + shadcn/ui |
| Realtime | WebSocket | 자산/알림/시세 실시간 반영 |
| Push | Web Push (FCM) | PWA 오프라인 푸시 알림 |
| Auth | JWT | 이메일 회원가입 + Admin 승인제 |
| Deploy | Docker + Nginx | docker-compose 단일 구성 |
| PWA | Vite PWA Plugin | 홈 화면 설치, 오프라인 캐시, 푸시 알림 |

---

## 3. DDD 아키텍처

### 3.1 Bounded Contexts

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│   Identity   │  │  Classroom  │  │   Company    │
│  (인증/사용자) │  │   (강의실)   │  │  (회사/명함)  │
└──────┬───────┘  └──────┬───────┘  └──────┬───────┘
       │                 │                 │
┌──────┴───────┐  ┌──────┴───────┐  ┌──────┴───────┐
│    Wallet    │  │     Feed     │  │  Investment  │
│  (지갑/자산)  │  │ (SNS/과제)   │  │ (투자/배당)   │
└──────┬───────┘  └──────┬───────┘  └──────┬───────┘
       │                 │                 │
┌──────┴───────┐  ┌──────┴───────┐  ┌──────┴───────┐
│   Exchange   │  │   Market     │  │     Bank     │
│  (주식거래소)  │  │  (외주마켓)   │  │   (대출)     │
└──────────────┘  └──────────────┘  └──────────────┘
                  ┌──────────────┐
                  │ Notification │
                  │   (알림)      │
                  └──────────────┘
```

### 3.2 Backend 프로젝트 구조

```
backend/
├── cmd/
│   └── server/
│       └── main.go                    # 엔트리포인트
├── internal/
│   ├── domain/                        # 도메인 레이어 (핵심 비즈니스 규칙)
│   │   ├── user/
│   │   │   ├── entity.go             # User, Role, Status
│   │   │   ├── repository.go         # UserRepository interface
│   │   │   └── errors.go
│   │   ├── classroom/
│   │   │   ├── entity.go             # Classroom, ClassroomMember
│   │   │   └── repository.go
│   │   ├── company/
│   │   │   ├── entity.go             # Company, Shareholder, BusinessCard
│   │   │   ├── repository.go
│   │   │   ├── valuation.go          # 기업가치 계산 Value Object
│   │   │   └── errors.go
│   │   ├── wallet/
│   │   │   ├── entity.go             # Wallet, Transaction
│   │   │   ├── repository.go
│   │   │   └── errors.go
│   │   ├── post/
│   │   │   ├── entity.go             # Channel, Post, Comment, Assignment
│   │   │   └── repository.go
│   │   ├── freelance/
│   │   │   ├── entity.go             # Job, Application, Review
│   │   │   ├── repository.go
│   │   │   └── errors.go
│   │   ├── investment/
│   │   │   ├── entity.go             # InvestmentRound, Investment, Dividend
│   │   │   ├── repository.go
│   │   │   └── errors.go
│   │   ├── exchange/
│   │   │   ├── entity.go             # StockOrder, StockTrade
│   │   │   ├── repository.go
│   │   │   ├── matching.go           # 매칭 엔진 도메인 로직
│   │   │   └── errors.go
│   │   ├── loan/
│   │   │   ├── entity.go             # Loan, LoanPayment
│   │   │   ├── repository.go
│   │   │   └── errors.go
│   │   └── notification/
│   │       ├── entity.go             # Notification
│   │       └── repository.go
│   │
│   ├── application/                   # 애플리케이션 레이어 (유스케이스)
│   │   ├── auth_usecase.go           # 회원가입, 로그인, 승인
│   │   ├── classroom_usecase.go      # 강의실 생성, 참여
│   │   ├── company_usecase.go        # 회사 설립, 수정, 명함 생성
│   │   ├── wallet_usecase.go         # 잔고 조회, 이체, 랭킹
│   │   ├── post_usecase.go           # 게시글 CRUD, 과제
│   │   ├── freelance_usecase.go      # 외주 등록, 계약, 정산
│   │   ├── investment_usecase.go     # 투자 라운드, 투자, 배당
│   │   ├── exchange_usecase.go       # 주문, 매칭, 시세
│   │   ├── loan_usecase.go           # 대출 신청, 승인, 상환
│   │   ├── notification_usecase.go   # 알림 조회, 읽음 처리
│   │   └── valuation_usecase.go      # 자산가치 계산
│   │
│   ├── infrastructure/                # 인프라스트럭처 레이어
│   │   ├── persistence/
│   │   │   ├── sqlite.go             # SQLite 연결 (WAL 모드)
│   │   │   ├── migrations/
│   │   │   │   └── 001_init.sql
│   │   │   ├── seed.go               # Admin 시드 데이터
│   │   │   ├── user_repo.go          # UserRepository 구현
│   │   │   ├── classroom_repo.go
│   │   │   ├── company_repo.go
│   │   │   ├── wallet_repo.go
│   │   │   ├── post_repo.go
│   │   │   ├── freelance_repo.go
│   │   │   ├── investment_repo.go
│   │   │   ├── exchange_repo.go
│   │   │   ├── loan_repo.go
│   │   │   └── notification_repo.go
│   │   ├── pdf/
│   │   │   └── business_card.go      # PDF 생성 (명함)
│   │   ├── push/
│   │   │   └── webpush.go            # Web Push 발송 (VAPID)
│   │   └── config/
│   │       └── config.go             # 환경변수, 설정
│   │
│   └── interfaces/                    # 인터페이스 레이어
│       ├── http/
│       │   ├── handler/
│       │   │   ├── auth_handler.go
│       │   │   ├── admin_handler.go
│       │   │   ├── classroom_handler.go
│       │   │   ├── company_handler.go
│       │   │   ├── wallet_handler.go
│       │   │   ├── post_handler.go
│       │   │   ├── freelance_handler.go
│       │   │   ├── investment_handler.go
│       │   │   ├── exchange_handler.go
│       │   │   ├── loan_handler.go
│       │   │   ├── notification_handler.go
│       │   │   └── upload_handler.go
│       │   ├── middleware/
│       │   │   ├── auth.go            # JWT 인증
│       │   │   ├── cors.go            # CORS
│       │   │   └── approved.go        # 승인된 사용자만 통과
│       │   └── router/
│       │       └── router.go
│       └── ws/
│           ├── hub.go                 # WebSocket 허브
│           └── client.go
│
├── go.mod
├── go.sum
└── Dockerfile
```

### 3.3 Frontend 프로젝트 구조

```
frontend/
├── src/
│   ├── main.tsx                         # Vite 엔트리포인트
│   ├── App.tsx                          # React Router 설정
│   ├── routes/                          # 페이지 컴포넌트
│   │   ├── auth/
│   │   │   ├── LoginPage.tsx
│   │   │   ├── RegisterPage.tsx
│   │   │   └── PendingPage.tsx
│   │   ├── feed/
│   │   │   └── FeedPage.tsx
│   │   ├── wallet/
│   │   │   ├── WalletPage.tsx
│   │   │   └── TransactionsPage.tsx
│   │   ├── market/
│   │   │   ├── MarketPage.tsx
│   │   │   ├── MarketNewPage.tsx
│   │   │   └── MarketDetailPage.tsx
│   │   ├── company/
│   │   │   ├── CompanyListPage.tsx
│   │   │   ├── CompanyNewPage.tsx
│   │   │   ├── CompanyDetailPage.tsx
│   │   │   └── BusinessCardPage.tsx
│   │   ├── invest/
│   │   │   ├── InvestPage.tsx
│   │   │   └── InvestDetailPage.tsx
│   │   ├── exchange/
│   │   │   ├── ExchangePage.tsx
│   │   │   └── ExchangeDetailPage.tsx
│   │   ├── bank/
│   │   │   ├── BankPage.tsx
│   │   │   └── LoanApplyPage.tsx
│   │   ├── profile/
│   │   │   ├── ProfilePage.tsx
│   │   │   └── UserProfilePage.tsx
│   │   ├── notifications/
│   │   │   └── NotificationsPage.tsx
│   │   └── admin/
│   │       ├── AdminPage.tsx
│   │       ├── AdminUsersPage.tsx
│   │       ├── AdminClassroomPage.tsx
│   │       ├── AdminLoansPage.tsx
│   │       └── AdminKpiPage.tsx
│   ├── components/
│   │   ├── ui/                          # shadcn/ui
│   │   ├── layout/
│   │   │   ├── MainLayout.tsx           # 하단 네비게이션 포함
│   │   │   ├── AuthLayout.tsx
│   │   │   ├── BottomNav.tsx
│   │   │   └── Header.tsx
│   │   ├── guards/
│   │   │   ├── AuthGuard.tsx            # 인증 가드
│   │   │   ├── ApprovedGuard.tsx        # 승인 가드
│   │   │   └── AdminGuard.tsx           # Admin 가드
│   │   ├── feed/
│   │   ├── wallet/
│   │   ├── company/
│   │   ├── market/
│   │   ├── invest/
│   │   ├── exchange/
│   │   └── bank/
│   ├── lib/
│   │   ├── api.ts                       # API 클라이언트 (fetch/axios)
│   │   ├── auth.ts                      # JWT 관리 (localStorage)
│   │   ├── ws.ts                        # WebSocket 클라이언트
│   │   ├── push.ts                      # Web Push 구독/해제
│   │   └── utils.ts
│   ├── hooks/
│   │   ├── use-auth.ts
│   │   ├── use-wallet.ts
│   │   ├── use-ws.ts
│   │   └── use-push.ts                  # 푸시 알림 훅
│   └── types/
│       └── index.ts
├── public/
│   ├── manifest.json                    # PWA 매니페스트
│   ├── sw.js                            # Service Worker (Vite PWA 자동 생성)
│   └── icons/                           # PWA 아이콘 (192×192, 512×512)
├── index.html                           # Vite HTML 엔트리
├── vite.config.ts
├── tailwind.config.ts
├── tsconfig.json
├── package.json
└── Dockerfile
```

---

## 4. 공통 규약

### 4.1 API 공통

**Base URL**: `/api`

**인증**: `Authorization: Bearer <JWT>`

**JWT Payload**:
```json
{
  "user_id": 1,
  "email": "student@ewha.ac.kr",
  "role": "student",
  "status": "approved",
  "exp": 1234567890
}
```

**공통 응답 형식**:
```json
// 성공
{ "success": true, "data": { ... }, "error": null }

// 실패
{ "success": false, "data": null, "error": { "code": "ERROR_CODE", "message": "설명" } }
```

**에러 코드**:
| 코드 | HTTP | 설명 |
|------|------|------|
| `UNAUTHORIZED` | 401 | 미인증 |
| `FORBIDDEN` | 403 | 권한 없음 |
| `NOT_APPROVED` | 403 | 승인 대기 중 |
| `NOT_FOUND` | 404 | 리소스 없음 |
| `DUPLICATE` | 409 | 중복 (회사명 등) |
| `INSUFFICIENT_BALANCE` | 400 | 잔고 부족 |
| `INSUFFICIENT_SHARES` | 400 | 보유 주식 부족 |
| `MIN_CAPITAL` | 400 | 최소 자본금 미달 |
| `NOT_LISTED` | 400 | 비상장 회사 |
| `ROUND_CLOSED` | 400 | 투자 라운드 마감 |
| `VALIDATION` | 422 | 입력값 오류 |

**미들웨어 체인**:
```
Public    → [CORS]
Auth      → [CORS] → [JWT 검증]
Approved  → [CORS] → [JWT 검증] → [status == 'approved' 확인]
Admin     → [CORS] → [JWT 검증] → [status == 'approved'] → [role == 'admin']
```

**페이지네이션** (목록 API 공통):
```
GET /api/posts?page=1&limit=20&channel_id=1
```
```json
{
  "data": [...],
  "pagination": { "page": 1, "limit": 20, "total": 150, "total_pages": 8 }
}
```

### 4.2 시간 규칙

| 현실 | 게임 내 | 비고 |
|------|---------|------|
| 1주일 | 1년 | 이자/배당 계산 기준 |
| 학기 (15주) | 15년 | 전체 시뮬레이션 기간 |

### 4.3 성적 반영 공식

```
총 자산가치 = 현금 잔고
            + Σ(보유 주식 수 × 주당 가격)
            + Σ(회사 지갑 잔고 × 내 지분율)
            - Σ(미상환 대출 원금 + 미납 이자)
```

---

## 5. 도메인 스펙 문서 목록

| 파일 | 도메인 | 설명 |
|------|--------|------|
| `01-auth.md` | Identity | 회원가입, 로그인, Admin 승인 |
| `02-classroom.md` | Classroom | 강의실 생성, 참여, 초기 자본 |
| `03-company.md` | Company | 회사 설립, 명함, 기업가치 |
| `04-wallet.md` | Wallet | 지갑, 거래 내역, 자산, 랭킹 |
| `05-feed.md` | Feed | SNS 채널, 게시글, 댓글, 과제 |
| `06-market.md` | Market | 외주 마켓, 에스크로, 리뷰 |
| `07-investment.md` | Investment | 투자 라운드, 배당, KPI |
| `08-exchange.md` | Exchange | 주식 거래소, 주문 매칭 |
| `09-bank.md` | Bank | 대출, 이자, 상환 |
| `10-notification.md` | Notification | 알림, WebSocket |
| `11-infra.md` | Infrastructure | Docker, Nginx, 보안 |
