---
title: "전체 기능 구현: 설계 문서를 동작하는 코드로"
date: "2026-03-13"
tags: ["구현", "풀스택", "Go", "React", "API"]
---

## 무엇을 했나요?

PRD와 SPEC 문서를 기반으로 EarnLearning LMS의 **전체 기능을 한 번에 구현**했습니다:

- **백엔드 (Go + Echo)**: 인증, 회사, 과제, 마켓, 피드, KPI 등 전체 API
- **프론트엔드 (React + TypeScript)**: 학생 대시보드, 회사 관리, 마켓, 과제 제출 등 전체 UI
- **관리자 페이지**: KPI 관리, 사용자 관리, 강의실 관리 등 Admin 전용 기능
- **데이터베이스**: SQLite 스키마 설계 및 마이그레이션
- **인증 시스템**: JWT 기반 로그인 + 관리자 승인제

## 왜 필요했나요?

### 기획이 끝나면 구현을 시작해야 한다

아무리 좋은 PRD와 SPEC이 있어도, 코드가 없으면 사용자에게 가치를 전달할 수 없습니다. 스타트업에서는 이 과정을 **MVP(Minimum Viable Product)** 개발이라고 합니다.

MVP란 "사용자가 핵심 가치를 경험할 수 있는 최소한의 제품"입니다. 완벽할 필요는 없지만, **핵심 기능이 동작**해야 합니다.

### 왜 한 번에 전체를 구현했나?

일반적으로는 기능을 하나씩 나눠서 구현하는 것이 좋습니다. 하지만 이 프로젝트에서는 다음 이유로 한 번에 구현했습니다:

1. **SPEC 문서가 충분히 상세**했기 때문에 기능 간 의존성을 미리 파악할 수 있었음
2. **Claude AI를 활용**해 SPEC을 입력하고 전체 코드를 생성했기 때문
3. **단일 개발자** 프로젝트이므로 커뮤니케이션 오버헤드가 없었음

이것은 AI 시대의 새로운 개발 패턴입니다. 과거에는 불가능했던 "전체 구현"이 상세한 스펙 문서 + AI 코드 생성으로 가능해졌습니다.

## 어떻게 만들었나요?

### 프로젝트 구조

```
lms/
├── backend/                  # Go 백엔드
│   ├── main.go              # 서버 시작점 (엔트리포인트)
│   ├── handlers/            # API 요청 처리 (컨트롤러)
│   │   ├── auth.go         # 회원가입, 로그인
│   │   ├── company.go      # 회사 CRUD
│   │   ├── assignment.go   # 과제 관리
│   │   ├── market.go       # 마켓 거래
│   │   ├── feed.go         # 피드 (소셜)
│   │   └── admin.go        # 관리자 기능
│   ├── models/              # 데이터 구조 정의
│   ├── middleware/          # 인증 등 공통 처리
│   └── database/            # DB 연결 및 마이그레이션
├── frontend/                 # React 프론트엔드
│   ├── src/
│   │   ├── pages/           # 페이지 컴포넌트
│   │   ├── components/      # 재사용 가능 UI
│   │   ├── api/             # API 호출 함수
│   │   ├── contexts/        # 전역 상태 관리
│   │   └── types/           # TypeScript 타입 정의
│   └── index.html
└── docker-compose.yml        # 컨테이너 설정
```

### 백엔드 구조 이해하기

Go의 Echo 프레임워크는 웹 서버를 만드는 도구입니다. 핵심 개념을 식당에 비유해볼게요:

```
클라이언트 요청     →  라우터(Router)     →  핸들러(Handler)     →  응답
"피자 주세요"       "피자는 주방2번"       "피자를 만듭니다"       "여기요!"
```

실제 코드에서는 이렇게 동작합니다:

```go
// 라우터: URL과 핸들러를 연결
e.POST("/api/auth/register", handlers.Register)    // 회원가입
e.POST("/api/auth/login", handlers.Login)           // 로그인
e.GET("/api/companies", handlers.GetCompanies)      // 회사 목록
e.POST("/api/companies", handlers.CreateCompany)    // 회사 설립
```

각 핸들러는 다음 순서로 동작합니다:

```
1. 요청 데이터 파싱 (JSON → Go 구조체)
2. 유효성 검증 (필수 필드 확인, 권한 확인)
3. 비즈니스 로직 실행 (DB 조회, 계산 등)
4. 응답 반환 (Go 구조체 → JSON)
```

### 인증 시스템

JWT(JSON Web Token)는 "디지털 신분증"입니다:

```
로그인 성공 시:
서버 → JWT 토큰 발급 → 클라이언트에 전달
      ┌──────────────────────────┐
      │ Header: 알고리즘 정보      │
      │ Payload: 사용자 ID, 권한   │
      │ Signature: 서버만 아는 서명 │
      └──────────────────────────┘

이후 API 요청 시:
클라이언트 → Authorization: Bearer <토큰> → 서버가 토큰 검증 → 요청 처리
```

이 프로젝트에서는 추가로 **관리자 승인제**를 도입했습니다:

```
회원가입 → 계정 생성 (status: pending) → 관리자 승인 → 로그인 가능
```

왜 승인제인가? 대학 강의용이므로 수강생만 접근해야 하기 때문입니다.

### 프론트엔드 구조

React의 핵심은 **컴포넌트**입니다. 레고 블록처럼 작은 UI 조각을 조합해서 페이지를 만듭니다.

```
페이지 구성 예시 (대시보드):

┌─────────────────────────────────────┐
│  Header (로고, 네비게이션, 로그아웃)    │
├──────────┬──────────────────────────┤
│ Sidebar  │  Dashboard              │
│          │  ┌───────┐ ┌───────┐    │
│ - 대시보드 │  │ 기업가치 │ │ 과제현황│    │
│ - 회사    │  │ 카드    │ │ 카드   │    │
│ - 과제    │  └───────┘ └───────┘    │
│ - 마켓    │  ┌───────────────────┐  │
│ - 피드    │  │ 최근 활동 피드      │  │
│          │  │                   │  │
│          │  └───────────────────┘  │
└──────────┴──────────────────────────┘
```

### 데이터 흐름

프론트엔드와 백엔드가 데이터를 주고받는 흐름:

```
사용자 클릭 "회사 설립"
    ↓
React 컴포넌트: 폼 데이터 수집
    ↓
API 함수: fetch('/api/companies', { method: 'POST', body: JSON })
    ↓
Go 서버: handlers.CreateCompany() 실행
    ↓
SQLite: INSERT INTO companies (...) VALUES (...)
    ↓
응답: { id: 1, name: "우리회사", valuation: 0 }
    ↓
React: 상태 업데이트 → UI 리렌더링
```

### Admin KPI 관리

교수(관리자)가 학생들의 성과를 추적할 수 있는 KPI 대시보드를 구현했습니다:

```
KPI 항목:
- 과제 제출률 (팀별/개인별)
- 마켓 활동 지수
- 피드 참여도
- 기업가치 변동 추이
```

이 데이터는 실시간으로 집계되어 관리자 페이지에 표시됩니다.

### SQLite 데이터베이스

SQLite는 파일 하나에 모든 데이터를 저장하는 데이터베이스입니다:

```
일반 DB (PostgreSQL, MySQL)     SQLite
─────────────────────────     ──────────
별도 서버 프로세스 필요           파일 하나 (data.db)
네트워크 통신                   직접 파일 접근
복잡한 설정                     설정 거의 없음
수천 명 동시 접속 가능           수백 명까지 적합
```

50명 규모의 교육용 LMS에는 SQLite가 최적입니다.

테이블 구조 예시:

```sql
-- 사용자 테이블
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    role TEXT DEFAULT 'student',
    status TEXT DEFAULT 'pending',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 회사 테이블
CREATE TABLE companies (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    owner_id INTEGER REFERENCES users(id),
    valuation REAL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## 사용한 프롬프트

### 전체 구현 프롬프트
```
SPEC.md를 기반으로 EarnLearning LMS 전체 기능을 구현해줘.
백엔드는 Go + Echo + SQLite, 프론트엔드는 React + TypeScript + Tailwind.
인증은 JWT + 관리자 승인제로 구현하고,
모든 API는 RESTful 원칙을 따라줘.
```

### Admin KPI 페이지 프롬프트
```
Admin 전용 KPI 관리 페이지를 구현해줘.
팀별 과제 제출률, 마켓 활동, 기업가치 변동을 차트로 보여주고,
CSV 내보내기 기능도 추가해줘.
```

핵심은 **SPEC 문서의 존재**입니다. 상세한 스펙이 있으면 AI가 일관성 있는 코드를 생성할 수 있습니다. 스펙 없이 "LMS 만들어줘"라고 하면 매우 일반적이고 불완전한 결과를 얻게 됩니다.

## 배운 점

### 1. SPEC의 품질이 코드의 품질을 결정한다
상세한 SPEC 문서 덕분에 전체 기능을 한 번에 구현할 수 있었습니다. SPEC이 모호하면 AI도 모호한 코드를 생성합니다. "입력 → 처리 → 출력"이 명확히 정의된 SPEC을 작성하세요.

### 2. 풀스택은 "연결"이 핵심이다
백엔드와 프론트엔드를 각각 만드는 것보다 **둘을 연결하는 것**이 더 어렵습니다. API 규격(요청/응답 형식)을 먼저 정하고, 양쪽이 그 규격을 따르도록 하는 것이 중요합니다.

### 3. MVP는 "최소"이지 "대충"이 아니다
MVP의 각 기능은 완성도가 있어야 합니다. 회원가입이 있으면 로그아웃도 있어야 하고, 회사를 만들 수 있으면 수정/삭제도 가능해야 합니다. "절반만 동작하는 10개 기능"보다 "완전히 동작하는 5개 기능"이 낫습니다.

### 4. AI 코드 생성의 현실
AI가 전체 코드를 생성해도 **그 코드를 이해하고 수정할 수 있는 능력**이 필수입니다. 다음 단계(003)에서 볼 수 있듯이, 생성된 코드에는 반드시 버그가 있고 테스트와 디버깅이 필요합니다.

### 5. 관리자 기능의 중요성
학생이 사용하는 기능만 만들면 안 됩니다. 교수(관리자)가 학생들의 활동을 모니터링하고 평가할 수 있어야 교육 도구로서 가치가 있습니다. KPI 대시보드는 이 역할을 합니다.

### 6. 데이터 모델이 비즈니스를 반영한다
테이블 구조를 보면 그 서비스가 무엇을 하는지 알 수 있습니다. users, companies, assignments, market_items 같은 테이블 이름이 곧 서비스의 핵심 개념입니다. 좋은 데이터 모델은 비즈니스 로직을 자연스럽게 담아냅니다.

---

## GitHub 참고 링크
- [커밋 1e19397: EarnLearning LMS 전체 기능 구현](https://github.com/cycorld/earnlearning/commit/1e19397)
- [커밋 304438a: Admin KPI 관리 페이지 구현](https://github.com/cycorld/earnlearning/commit/304438a)
