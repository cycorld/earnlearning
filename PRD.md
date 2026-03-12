# EarnLearning PRD (Product Requirements Document)

## 1. 제품 개요

**제품명**: EarnLearning
**목적**: 게임화된 창업 교육 시뮬레이션 LMS
**대상**: 이화여자대학교 "스타트업을 위한 코딩입문" 수강생
**운영자**: 최용철 (강의자, 중앙은행, 유동성 공급자)

### 핵심 컨셉
학생들이 **가상 자본 5,000만원**으로 실제 창업 생태계를 시뮬레이션한다.
바이브 코딩으로 서비스를 만들고, 투자를 받고, 외주를 주고받으며, 학기 말 **최종 자산가치가 성적에 반영**된다.

### 기술 스택
- **Backend**: Go (Echo) + SQLite (Docker volume persistent)
- **Frontend**: Vite + React 18 + TypeScript + Tailwind CSS + shadcn/ui
- **Realtime**: WebSocket (자산/알림 실시간 반영)
- **Auth**: JWT (이메일 회원가입 + Admin 승인제)
- **Deploy**: Docker + Nginx (SQLite DB는 Docker volume으로 영속화)

---

## 2. 사용자 역할

| 역할 | 설명 | 권한 |
|------|------|------|
| **Admin (최용철)** | 강의자, 중앙은행, 유동성 공급자 | 강의실 관리, 유동성 공급, 대출 승인, KPI 기반 소득 부여, 과제 출제, 배당 관리, **학생 승인**, 전체 학번 열람 |
| **Student** | 수강생, 창업가 | 회사 설립, 명함 생성, 외주 등록/수주, 투자, 주식 거래, 게시글 작성, 서비스 런칭 |
| **Pending** | 가입했지만 미승인 | 승인 대기 안내 화면만 노출 (문의: cyc@snu.ac.kr) |

---

## 3. 핵심 기능 모듈

### 3.1 온보딩 & 회사 설립

#### 3.1.0 회원가입 & 승인 시스템
- **가입 정보**: 이메일(학교 이메일), 이름, 학과, 학번(전체)
- 비밀번호 설정
- 가입 즉시 **pending** 상태
- Admin이 승인해야 서비스 접근 가능
- 미승인 시 **승인 대기 안내 페이지** 표시
  - "관리자 승인을 기다리고 있습니다."
  - "문의: cyc@snu.ac.kr"
- **학번 표시 규칙**:
  - 학생 간: 앞 2자리(입학년도)만 표시 (예: "24학번")
  - Admin: 전체 학번 열람 가능

#### 3.1.1 강의실 등록
- 강의실 코드 입력으로 참여 (첫 강의실: "2026 스타트업을위한코딩입문")
- 등록 시 **초기 자본 5,000만원** 자동 지급
- 강의자가 강의실 생성 및 초대 코드 발급

#### 3.1.2 회사 설립
- 학생 1명이 **여러 회사** 설립 가능 (1회사 = 1프로젝트)
- **회사명** 설정 (중복 검사)
- **최소 자본금**: 100만원 이상 납입 필수 (금액은 설립자 자유)
  - 자본금은 개인 지갑에서 회사 지갑으로 이동
- 설립자는 **지분 100%**로 시작 (총 발행 주식: 10,000주)
- 이후 투자 유치(신주 발행) 또는 거래소 매각으로 주주 구성 변경 가능
- **회사 로고** 업로드 또는 자동 생성
- **명함 디자이너**: 템플릿 기반 명함 생성
  - 회사명, 이름, 직함, 연락처, 로고
  - PDF/이미지 다운로드 가능
  - 여러 디자인 템플릿 제공
- 새 바이브코딩 프로젝트 = 새 회사 설립

#### 3.1.3 투자 시 신주 발행 방식
- 투자 라운드는 **신주 발행** 방식으로 진행
  - 예: 설립자 10,000주(100%) → 1,000만원 모집, 지분 20% 제안
  - 신주 2,500주 발행 → 총 주식 12,500주
  - 설립자 10,000주(80%), 투자자 2,500주(20%)
- 설립자 주식 수는 유지되나 지분율이 희석됨 (실제 스타트업과 동일)

#### 3.1.4 기업가치 산정
- **기업가치** = 최근 거래 주가 × 총 발행 주식 수
- 거래 이력이 없는 경우: 기업가치 = 납입 자본금
- 투자 라운드 체결 시: 투자 금액 ÷ 양도 지분율 = Post-money 기업가치
  - 예: 1,000만원 ÷ 20% = 5,000만원 기업가치
- 거래소에서 주식 거래 시: 최종 체결가 기준 시가총액 갱신

#### 3.1.5 프로필
- 보유 회사 목록 & 각 회사 기업가치
- 자산 현황, 보유 주식(타사 포함), 대출 현황 한눈에 확인

---

### 3.2 자산 관리 시스템

#### 3.2.1 지갑 (Wallet)
- **현금 잔고**: 실시간 표시
- **총 자산가치**: 현금 + 보유 주식 시가 - 부채
- **거래 내역**: 모든 입출금 로그 (필터/검색)

#### 3.2.2 유동성 공급 (Admin)
- 강의자가 특정 학생 또는 전체에게 자금 지급
- 사유 기록 (과제 보상, 이벤트 등)

#### 3.2.3 자산 대시보드
- 자산 추이 그래프 (일별/주별)
- 자산 구성 비율 (현금/주식/기타)
- **전체 랭킹**: 총 자산가치 기준 실시간 순위

---

### 3.3 SNS형 게시판 시스템

> UI는 **Instagram/Twitter 하이브리드** 형태. 채널로 기능 구분.

#### 3.3.1 채널 구조
| 채널 | 용도 | 특징 |
|------|------|------|
| `#공지` | 강의자 공지사항 | Admin만 작성 |
| `#자유` | 자유 게시판 | 잡담, 질문, 소통 |
| `#과제` | 과제 제출함 | Admin이 과제 게시 → 학생이 댓글/첨부로 제출 |
| `#쇼케이스` | 바이브코딩 결과물 공유 | 데모 링크, 스크린샷, 진행 과정 |
| `#외주마켓` | 외주 일거리 | 크몽 스타일 (별도 3.4 참조) |
| `#투자라운지` | 투자 유치 & IR | 사업 소개, 투자 요청 (별도 3.5 참조) |
| `#거래소` | 주식 거래 | 지분 매매 (별도 3.6 참조) |

#### 3.3.2 게시글 기능
- 텍스트 + 이미지 + 링크 + 파일 첨부
- 좋아요, 댓글, 공유
- 해시태그
- 멘션 (@사용자)

#### 3.3.3 과제 시스템 (Admin)
- 과제 게시글 생성 (마감일, 설명, 첨부파일)
- 학생은 해당 게시글에 **제출물 댓글**로 첨부
- 제출 현황 대시보드 (Admin)
- 채점 및 보상 지급 (가상화폐)

---

### 3.4 외주 마켓 (크몽 스타일)

> **개인 간 거래**: 외주는 회사가 아닌 개인 지갑 간 거래 (회사 설립 전에도 이용 가능)

#### 3.4.1 일거리 등록
- **의뢰자**: 프로젝트 설명, 예산, 마감일, 필요 스킬
- **수주자**: 지원서 제출 (포트폴리오, 예상 기간, 견적)

#### 3.4.2 계약 & 정산
- 양측 합의 시 **에스크로** 방식으로 금액 동결 (의뢰자 개인 지갑에서 차감)
- 작업 완료 → 의뢰자 승인 → 수주자 개인 지갑으로 자동 정산
- 분쟁 시 Admin 중재

#### 3.4.3 강의자 일거리
- Admin(최용철)도 일거리를 등록하여 학생들에게 수익 기회 제공
- 예: "랜딩 페이지 만들기", "데이터 정리" 등

#### 3.4.4 평가 시스템
- 완료 후 상호 평점 (★ 1~5)
- 누적 평점 프로필에 표시

---

### 3.5 투자 시스템

#### 3.5.1 IR (투자 유치)
- 학생이 **사업 계획서** 형식의 게시글 작성
  - 서비스 소개, 데모, 목표 KPI
  - 모집 금액, 양도 지분율 설정
  - 투자 기간 설정
- 다른 학생들이 투자 참여
- **신주 발행** 방식: 투자금 유입 시 신주 발행, 기존 주주 지분 희석

#### 3.5.2 투자 실행
- **1라운드 = 1투자자**: 한 투자자가 모집 금액 전액을 투자
- 투자 시 즉시 **펀딩 성공** → 신주 발행 & 회사 지갑에 입금
- 부분 펀딩 없음

#### 3.5.3 KPI 기반 소득 (Admin)
- 런칭된 서비스에 대해 Admin이 **KPI 규칙** 설정
  - 예: "일일 방문자 100명당 10만원/주"
  - 예: "가입자 1명당 5만원"
- Admin이 매주 KPI 확인 후 **가상 소득** 부여
- 소득은 회사 지갑으로 입금

#### 3.5.4 배당 시스템
- **설립자가 수동 실행**: 설립자가 배당 금액/비율 결정 후 실행
- 회사 지갑 잔고 내에서 배당 → 주주들의 개인 지갑으로 지분율에 따라 분배
- **회사 지갑 → 개인 인출 불가**: 배당만이 유일한 현금 분배 수단 (주주 보호)
- 배당 내역 투명하게 공개
- 배당 이력 조회

---

### 3.6 주식 거래소

> **조건부 상장**: 회사 자본금(납입 자본금 + 투자금)이 **5,000만원 이상** 도달 시 거래소 상장
> (설립 시 5,000만원 풀베팅 또는 투자 유치로 도달)

#### 3.6.1 지분 표시
- 각 회사의 **지분 구조** 시각화
  - 파이 차트: 누가 몇 % 보유
  - 지분 변동 이력

#### 3.6.2 거래 시스템
- **매도 주문**: 보유 지분 중 일부를 가격 지정하여 매도 등록
- **매수 주문**: 원하는 회사의 지분을 가격 지정하여 매수 등록
- **체결**: 가격 매칭 시 자동 체결 (지정가 주문만 지원)

#### 3.6.3 시세 정보
- 회사별 지분 가격 차트
- 거래량, 시가총액
- 전체 회사 시가총액 랭킹

---

### 3.7 은행 시스템 (Admin 운영)

> 1주일 = 1년으로 환산

#### 3.7.1 대출
- 학생이 대출 신청 (금액, 용도)
- Admin 심사 후 승인/거절
- **이자율**: Admin이 설정 (예: 주당 5%)
- **이자 납부**: 매주 자동 차감 또는 수동 납부
- 미납 시 연체 표시 + **연체 이자** (기본 이자율의 2배 적용)

#### 3.7.2 대출 현황
- 원금, 잔여 원금, 이자율, 납부 일정
- 상환 이력
- 연체 경고

#### 3.7.3 예금 (향후 확장)
- 은행에 예치 → 이자 수령
- 예금 상품 다양화

---

### 3.8 알림 시스템

- 실시간 알림 (WebSocket)
  - 투자 체결, 외주 계약, 배당 입금, 이자 차감
  - 과제 등록, 새 게시글 (팔로우 채널)
  - 주식 체결, 대출 승인
- 알림 센터 (읽음/안읽음 관리)

---

## 4. 화면 구성 (IA)

```
EarnLearning
├── 🏠 홈 (피드 - SNS 타임라인)
│   ├── 채널 필터 탭
│   ├── 게시글 목록 (무한 스크롤)
│   └── 게시글 작성 (FAB)
│
├── 💰 자산 (My Wallet)
│   ├── 잔고 & 총 자산가치
│   ├── 자산 추이 차트
│   ├── 거래 내역
│   └── 랭킹
│
├── 🏪 마켓 (외주)
│   ├── 일거리 목록 (필터: 카테고리, 예산, 상태)
│   ├── 일거리 상세 & 지원
│   ├── 내 의뢰/수주 관리
│   └── 일거리 등록
│
├── 🏢 내 회사
│   ├── 보유 회사 목록
│   ├── 회사 설립 (자본금 납입)
│   ├── 회사 상세 (지분 구조, 기업가치, 명함)
│   └── 배당 실행 (설립자)
│
├── 📈 투자
│   ├── IR 목록 (투자 가능한 회사)
│   ├── IR 상세 & 투자하기
│   ├── 내 투자 포트폴리오
│   └── 배당 수령 내역
│
├── 📊 거래소 (자본금 5,000만원 이상 회사만)
│   ├── 상장 회사 목록 & 시세
│   ├── 호가창 & 주문
│   ├── 체결 내역
│   └── 내 보유 지분
│
├── 🏦 은행
│   ├── 대출 신청
│   ├── 내 대출 현황
│   └── 상환 일정
│
├── 🔔 알림
│
├── 👤 프로필
│   ├── 개인 정보 & 자산 요약
│   ├── 외주 평점 & 이력
│   └── 활동 이력
│
└── ⚙️ 관리자 (Admin Only)
    ├── 학생 승인 관리
    ├── 강의실 관리
    ├── 유동성 공급
    ├── 과제 관리
    ├── KPI 설정 & 소득 부여
    ├── 대출 심사
    └── 시스템 설정
```

---

## 5. 데이터 모델 (핵심 엔티티)

```
User
├── id, email, name, department, student_id (full)
├── role (admin/student), status (pending/approved/rejected)
├── bio
└── created_at, updated_at
# 학번 표시: API 응답 시 role에 따라 student_id 마스킹
# student → 앞 2자리만 (예: "24학번"), admin → 전체 학번

Company (1 User → N Companies, 1 Company = 1 Project)
├── id, owner_id (설립자), name (unique), logo, description
├── initial_capital (≥ 1,000,000), total_shares (초기 10,000, 신주 발행 시 증가)
├── listed (거래소 상장 여부: 자본금 ≥ 5,000만원 시 상장)
├── valuation (기업가치 = 최근 주가 × total_shares)
├── business_card_data (JSON)
└── created_at, status (active/dissolved)

Classroom
├── id, name, code, created_by (admin)
├── initial_capital (default: 50,000,000)
└── settings (JSON)

Wallet
├── user_id, balance
└── TransactionLog[] (amount, type, description, timestamp)

Channel
├── id, classroom_id, name, type
└── permissions

Post
├── id, channel_id, author_id
├── content, media[], tags[]
├── type (normal/assignment/showcase/ir)
└── Comment[]

Assignment (extends Post)
├── deadline, reward_amount
└── Submission[] (student_id, content, files, grade, rewarded)

FreelanceJob
├── id, client_id, title, description
├── budget, deadline, required_skills[]
├── status (open/in_progress/completed/disputed)
├── Application[] (freelancer_id, proposal, price)
└── escrow_amount

CompanyWallet (회사별 지갑 - 자본금, KPI 소득 등)
├── company_id, balance
└── TransactionLog[] (amount, type, description, timestamp)

ShareHolder (지분 구조)
├── company_id, user_id, shares, percentage
└── acquired_at, acquisition_type (founding/investment/trade)

KpiRule (회사별 KPI 규칙 - Admin 설정)
├── company_id, rule_description, formula
└── weekly_revenue, active

InvestmentRound
├── company_id, target_amount, offered_percent (양도 지분율)
├── new_shares (발행 신주 수), price_per_share
├── status (open/funded/failed/cancelled)
└── Investment[] (investor_id, amount, shares)

StockOrder
├── company_id, user_id
├── type (buy/sell), order_type (limit only — 지정가만 지원)
├── shares, price_per_share, remaining_shares
├── status (open/filled/cancelled)
└── matched_order_id

Loan
├── borrower_id, amount, interest_rate
├── status (pending/approved/rejected/active/paid)
├── repayment_schedule[]
└── LoanPayment[] (amount, date, type)

Notification
├── user_id, type, title, body
├── reference_type, reference_id
└── read_at
```

---

## 6. API 설계 (주요 엔드포인트)

```
Auth
  POST   /api/auth/register           (이메일, 이름, 학과, 학번, 비밀번호)
  POST   /api/auth/login
  GET    /api/auth/me

Admin - 학생 승인
  GET    /api/admin/users/pending      (승인 대기 목록)
  PUT    /api/admin/users/:id/approve  (승인)
  PUT    /api/admin/users/:id/reject   (거절)

Classroom
  POST   /api/classrooms              (admin)
  POST   /api/classrooms/join         (student)
  GET    /api/classrooms/:id

Users/Profile
  GET    /api/users/:id/profile
  GET    /api/users/me/companies       (내 보유 회사 목록)

Company (1 User → N Companies)
  POST   /api/companies                (회사 설립: 이름, 자본금 ≥ 100만원)
  GET    /api/companies/:id
  PUT    /api/companies/:id            (회사 정보 수정)
  GET    /api/companies/:id/shareholders (지분 구조)
  GET    /api/companies/:id/valuation  (기업가치)
  POST   /api/companies/:id/business-card (명함 생성)

Wallet
  GET    /api/wallet
  GET    /api/wallet/transactions
  POST   /api/wallet/transfer         (admin: 유동성 공급, target_user_ids 또는 target_all)
  GET    /api/wallet/ranking

Posts (SNS)
  GET    /api/channels
  GET    /api/posts                   (?channel_id=1&page=1&limit=20)
  POST   /api/posts
  POST   /api/posts/:id/comments
  POST   /api/posts/:id/like

Assignments
  POST   /api/assignments             (admin)
  POST   /api/assignments/:id/submit
  PUT    /api/assignments/:id/grade   (admin)

Freelance Market
  GET    /api/jobs
  POST   /api/jobs
  POST   /api/jobs/:id/apply
  PUT    /api/jobs/:id/accept/:appId
  PUT    /api/jobs/:id/complete       (수주자: 작업 완료 알림)
  PUT    /api/jobs/:id/approve        (의뢰자: 승인 → 정산)
  PUT    /api/jobs/:id/cancel         (의뢰자: 취소, open일 때만)
  PUT    /api/jobs/:id/dispute        (양쪽: 분쟁 신고)

Investment
  POST   /api/companies/:id/rounds    (투자 라운드 생성)
  POST   /api/rounds/:id/invest       (투자 참여)
  GET    /api/portfolio               (내 투자 포트폴리오)

Stock Exchange
  GET    /api/exchange/companies                (상장 회사 목록 & 시세)
  GET    /api/exchange/companies/:id/orderbook  (호가창)
  POST   /api/exchange/orders                   (매수/매도 주문)
  DELETE /api/exchange/orders/:id               (주문 취소)
  GET    /api/exchange/my-orders                (내 주문 내역)

Bank
  POST   /api/bank/loans/apply
  GET    /api/bank/loans
  PUT    /api/bank/loans/:id/approve  (admin)
  POST   /api/bank/loans/:id/repay

Dividend & KPI
  POST   /api/companies/:id/kpi-rules  (admin: KPI 규칙 설정)
  POST   /api/companies/:id/revenue    (admin: KPI 소득 부여)
  POST   /api/companies/:id/dividend   (배당 실행)
  GET    /api/dividends/my

Notifications
  GET    /api/notifications
  PUT    /api/notifications/:id/read
```

---

## 7. 시간 규칙

| 현실 | 게임 내 | 비고 |
|------|---------|------|
| 1주일 | 1년 | 이자/배당 계산 기준 |
| 1일 | ≈52일 | 대략적 환산 |
| 학기 (15주) | 15년 | 전체 시뮬레이션 기간 |

- 이자: 주당 이율 적용 (예: 주당 5% = 연 5%)
- 배당: 매주 1회 (= 매년 1회)
- KPI 정산: 매주 Admin이 수동 반영

---

## 8. 구현 우선순위

### Phase 1: MVP (Week 1-2)
1. 인증 (이메일 회원가입/로그인 + Admin 승인제)
2. 강의실 생성 & 참여 (초기 자본 지급)
3. 회사 설립 & 명함 생성
4. 지갑 & 자산 현황
5. SNS 피드 (채널, 게시글, 댓글)
6. 과제 시스템

### Phase 2: 경제 시스템 (Week 3-4)
7. 외주 마켓
8. 투자 시스템 (IR, 펀딩)
9. KPI 소득 & 배당

### Phase 3: 금융 시스템 (Week 5-6)
10. 주식 거래소
11. 은행 (대출/이자)
12. 실시간 알림

### Phase 4: 고도화 (Week 7+)
13. 자산 랭킹 & 리더보드
14. 통계 대시보드 (Admin)
15. 예금 상품
16. 모바일 최적화

---

## 9. 비기능 요구사항

- **동시 접속**: 50명 기준 (수강생 규모)
- **응답 시간**: API < 200ms
- **실시간**: 자산 변동, 알림은 WebSocket으로 즉시 반영
- **보안**: JWT 인증, CORS, SQL Injection 방지, XSS 방지
- **DB**: SQLite (WAL 모드 + 트랜잭션, Docker volume 영속화, 일별 백업)
- **파일 저장**: 이미지/첨부파일은 Docker volume 로컬 저장 (`/data/uploads/`)
- **Admin 시드**: 초기 기동 시 Admin 계정 자동 생성 (cyc@snu.ac.kr)
- **이메일 제한 없음**: 아무 이메일로 가입 가능 (Admin 승인이 필터 역할)
- **UI/UX**: 모바일 퍼스트, 깔끔하고 fancy한 디자인

---

## 10. 성적 반영 공식

```
최종 성적 기여도 = f(총 자산가치)

총 자산가치 = 현금 잔고
            + Σ(보유 회사 지분 × 최종 주가)   # 내가 설립한 회사 + 타사 투자분
            + Σ(보유 회사 지갑 잔고 × 내 지분율) # 회사 현금 중 내 지분만큼
            - Σ(미상환 대출 원금 + 미납 이자)
```

---

## 11. 핵심 사용자 시나리오

### 시나리오 1: 바이브 코딩 → 투자 유치 → 런칭
1. 학생 A가 바이브 코딩으로 웹앱 개발 과정을 `#쇼케이스`에 공유
2. 완성 후 `#투자라운지`에 IR 게시 (1,000만원 모집, 지분 20%)
3. 학생 B가 1,000만원 전액 투자 → 펀딩 성공 (1라운드 = 1투자자)
4. Admin이 KPI 규칙 설정 (가입자 1명 = 5만원/주)
5. 런칭 후 매주 Admin이 KPI 확인 → 소득 부여
6. 매주 배당 실행 → 투자자 B에게 지분율만큼 분배

### 시나리오 2: 외주로 자본 축적
1. 학생 D가 "로고 디자인" 외주 등록 (30만원)
2. 학생 E가 지원 → 계약 체결 → 에스크로 동결
3. 작업 완료 → D 승인 → E에게 30만원 정산
4. E는 축적한 자본으로 다른 프로젝트에 투자

### 시나리오 3: 대출 → 투자 → 수익
1. 학생 F가 은행에서 1,000만원 대출 (주당 이자 3%)
2. 유망 프로젝트에 1,000만원 투자
3. 매주 배당 수익이 이자보다 크면 레버리지 수익 실현
4. 원금 상환 후 순수익 확보
