# 11. Infrastructure — Docker, Nginx, 보안

## 1. Docker 구성

### docker-compose.yml

```yaml
version: '3.8'
services:
  backend:
    build: ./backend
    ports:
      - "8080:8080"
    volumes:
      - db_data:/data/db
      - upload_data:/data/uploads
    environment:
      - DB_PATH=/data/db/earnlearning.db
      - UPLOAD_PATH=/data/uploads
      - JWT_SECRET=${JWT_SECRET}
      - ADMIN_EMAIL=cyc@snu.ac.kr
      - ADMIN_PASSWORD=test1234

  frontend:
    build:
      context: ./frontend
      args:
        - VITE_API_URL=/api
        - VITE_WS_URL=
    ports:
      - "5173:5173"

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - backend
      - frontend

volumes:
  db_data:
  upload_data:
```

### Backend Dockerfile

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o server ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates sqlite-libs
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

### Frontend Dockerfile

```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_API_URL=/api
ARG VITE_WS_URL=
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx-frontend.conf /etc/nginx/conf.d/default.conf
EXPOSE 5173
CMD ["nginx", "-g", "daemon off;"]
```

### nginx.conf

```nginx
events { worker_connections 1024; }
http {
    upstream backend  { server backend:8080; }
    upstream frontend { server frontend:5173; }

    server {
        listen 80;

        location /api/ {
            proxy_pass http://backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
        }

        location /ws {
            proxy_pass http://backend;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }

        location /uploads/ {
            proxy_pass http://backend;
        }

        location / {
            proxy_pass http://frontend;
        }
    }
}
```

---

## 2. SQLite 설정

```sql
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;
```

- **WAL 모드**: 읽기/쓰기 동시 접근 허용
- **busy_timeout**: 락 대기 5초
- **IMMEDIATE 트랜잭션**: 금융 거래(주문 매칭, 잔고 변동)에 사용

---

## 3. 보안 체크리스트

| 항목 | 구현 | 비고 |
|------|------|------|
| 비밀번호 해싱 | bcrypt (cost 10+) | 평문 저장 금지 |
| JWT | 만료 24시간 | Authorization: Bearer |
| SQL Injection | Prepared Statement 전용 | raw query 금지 |
| XSS | HTML sanitize | 게시글 content |
| CORS | 프론트엔드 도메인만 허용 | |
| Rate Limiting | 로그인 5회/분 | |
| 파일 업로드 | MIME 검증 + 크기 10MB | |
| Admin API | role 검증 미들웨어 | |
| 금액 조작 방지 | 서버 사이드 잔고 검증 | 클라이언트 값 무시 |
| 동시성 | WAL + IMMEDIATE 트랜잭션 | 주문 매칭 |

---

## 4. 환경 변수

| 변수 | 설명 | 기본값 |
|------|------|--------|
| `DB_PATH` | SQLite DB 경로 | `/data/db/earnlearning.db` |
| `UPLOAD_PATH` | 업로드 파일 경로 | `/data/uploads` |
| `JWT_SECRET` | JWT 서명 키 | (필수) |
| `ADMIN_EMAIL` | Admin 시드 이메일 | `cyc@snu.ac.kr` |
| `ADMIN_PASSWORD` | Admin 시드 비밀번호 | `test1234` |
| `PORT` | 서버 포트 | `8080` |

---

## 5. 비기능 요구사항

| 항목 | 기준 |
|------|------|
| 동시 접속 | 50명 (수강생 규모) |
| API 응답 시간 | < 200ms |
| 실시간 반영 | WebSocket (자산, 알림, 시세) |
| DB 백업 | 일별 자동 (Docker volume) |
| 파일 저장 | Docker volume 로컬 (`/data/uploads/`) |
| UI/UX | 모바일 퍼스트, 반응형 |

---

## 6. 구현 우선순위

### Phase 1: MVP (Week 1-2)
1. 인증 (이메일 회원가입/로그인 + Admin 승인제)
2. 강의실 생성 & 참여 (초기 자본 지급)
3. 회사 설립
4. 지갑 & 자산 현황
5. SNS 피드 (채널, 게시글, 댓글)
6. 과제 시스템

### Phase 2: 경제 시스템 (Week 3-4)
7. 외주 마켓
8. 투자 시스템 (IR, 펀딩)
9. KPI 소득 & 배당
10. 명함 생성 (PDF)

### Phase 3: 금융 시스템 (Week 5-6)
11. 주식 거래소
12. 은행 (대출/이자)
13. 실시간 알림

### Phase 4: 고도화 (Week 7+)
14. 자산 랭킹 & 리더보드
15. 통계 대시보드 (Admin)
16. 모바일 최적화
