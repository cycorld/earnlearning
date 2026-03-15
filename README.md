# EarnLearning LMS

이화여자대학교 "스타트업을 위한 코딩입문" 강의용 **게임화 창업 교육 LMS**입니다.
학생들이 가상 자본금으로 회사를 설립하고, 투자·외주·주식거래·대출 등 실제 스타트업 생태계를 체험합니다.

## 주요 기능

| 모듈 | 설명 |
|------|------|
| **인증** | 이메일 회원가입 + 관리자 승인제, JWT 인증 |
| **지갑** | 초기 자본금 5천만원, 잔액·거래내역·랭킹 |
| **회사 설립** | 다중 회사 설립, 기업가치 산정, 명함 PDF 생성 |
| **피드/게시판** | 채널별 게시판, 마크다운 에디터, 파일 업로드, 태그, 좋아요/댓글 |
| **과제** | 과제 출제·제출·채점 |
| **외주 마켓** | 의뢰 등록→지원→수락→완료 보고→승인, 에스크로 결제, 완료 보고서 자동 포스팅 |
| **투자** | 투자 라운드, 배당금 분배, KPI 기반 수익 |
| **주식 거래소** | 호가창, 매칭 엔진, 주문/취소 |
| **은행/대출** | 대출 신청·승인·상환, 주간 이자 |
| **알림** | WebSocket 실시간 + Web Push (PWA) |
| **관리자** | 유저 관리, KPI, 강의실 관리, 대출 관리 |

## 기술 스택

```
Backend:   Go (Echo) + SQLite (WAL mode)
Frontend:  Vite + React 18 + TypeScript + Tailwind CSS + shadcn/ui
Realtime:  WebSocket + Web Push (VAPID)
PWA:       Vite PWA Plugin (홈화면 설치, 오프라인 캐시)
Auth:      JWT
Deploy:    Docker + Nginx
```

## 프로젝트 구조

```
lms/
├── backend/
│   ├── cmd/server/          # 서버 엔트리포인트
│   └── internal/
│       ├── domain/          # 도메인 모델 (10개 모듈)
│       ├── application/     # 유스케이스
│       ├── infrastructure/  # DB, Push, PDF
│       └── interfaces/      # HTTP 핸들러, WebSocket
├── frontend/
│   └── src/
│       ├── routes/          # 페이지 컴포넌트 (28개)
│       ├── components/      # 공용 컴포넌트 + shadcn/ui
│       ├── hooks/           # 커스텀 훅
│       └── lib/             # API 클라이언트, 유틸
├── docs/                    # 스펙 문서
├── docker-compose.yml
└── nginx.conf
```

## 로컬 개발

### 사전 요구사항

- Go 1.22+
- Node.js 20+
- (선택) Docker & Docker Compose

### 백엔드

```bash
cd backend
cp ../.env.example .env     # 환경변수 설정
go run ./cmd/server/        # http://localhost:8080
```

### 프론트엔드

```bash
cd frontend
npm install
npm run dev                 # http://localhost:5173
```

Vite dev proxy가 `/api`, `/ws`, `/uploads`를 백엔드로 프록시합니다.

### Docker 배포

```bash
# VAPID 키 생성 (최초 1회)
npx web-push generate-vapid-keys

# .env 파일에 VAPID 키 설정 후
docker compose up -d
# http://localhost 에서 접속
```

## 환경 변수

| 변수 | 설명 | 기본값 |
|------|------|--------|
| `JWT_SECRET` | JWT 서명 키 | (필수) |
| `PORT` | 서버 포트 | 8080 |
| `DB_PATH` | SQLite DB 경로 | ./data/earnlearning.db |
| `VAPID_PUBLIC_KEY` | Web Push 공개키 | (선택) |
| `VAPID_PRIVATE_KEY` | Web Push 비공개키 | (선택) |
| `VAPID_SUBJECT` | Web Push 연락처 | mailto:admin@example.com |
| `ADMIN_EMAIL` | 초기 관리자 이메일 | admin@ewha.ac.kr |
| `ADMIN_PASSWORD` | 초기 관리자 비밀번호 | (필수) |

## 테스트

```bash
cd backend

# 스모크 테스트 (커밋 전 필수)
go test ./tests/integration/ -run TestSmoke -timeout 60s

# 전체 테스트
go test ./tests/integration/ -timeout 120s -v
```

## API 구조

모든 API 응답은 통일된 envelope 형식을 사용합니다:

```json
{
  "success": true,
  "data": { ... },
  "error": null
}
```

목록 API는 페이지네이션을 포함합니다:

```json
{
  "success": true,
  "data": {
    "data": [...],
    "pagination": { "page": 1, "limit": 20, "total": 42, "total_pages": 3 }
  }
}
```

## 문서

- [PRD.md](./PRD.md) - 제품 요구사항 정의서
- [docs/SPEC.md](./docs/SPEC.md) - 기술 스펙 문서
- [docs/specs/](./docs/specs/) - 모듈별 상세 스펙

## 라이선스

Private - 이화여자대학교 강의 전용
