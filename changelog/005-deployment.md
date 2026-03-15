---
title: "배포: 내 컴퓨터에서 세상으로 나가기"
date: "2026-03-14"
tags: ["배포", "AWS", "Docker", "Nginx", "Cloudflare", "백업"]
---

## 무엇을 했나요?

로컬에서 개발하던 LMS를 실제 인터넷에서 접근할 수 있도록 배포했습니다:

- **AWS EC2 인스턴스 설정**: 클라우드 서버(t3.small) 생성 및 환경 구성
- **Docker Compose 구성**: stage(테스트)와 prod(운영) 환경 분리
- **Nginx 리버스 프록시**: 도메인별로 다른 환경으로 트래픽 분배
- **Cloudflare DNS + SSL**: 도메인 연결 및 HTTPS 보안 적용
- **DB 백업 스크립트**: 데이터 손실 방지를 위한 자동 백업 시스템
- **External Volume**: Docker 컨테이너 삭제 시에도 DB 보존
- **요청 로깅 미들웨어**: 누가 언제 어떤 요청을 했는지 기록
- **--force-recreate**: Nginx 캐시 문제 해결을 위한 배포 옵션 추가

## 왜 필요했나요?

### 개발 환경과 운영 환경의 차이

```
개발 환경 (내 컴퓨터)              운영 환경 (AWS 서버)
────────────────────            ────────────────────
나만 접근 가능                    전 세계에서 접근 가능
컴퓨터 꺼면 서비스 중단             24시간 365일 운영
http://localhost:3000            https://earnlearning.com
데이터 날아가도 괜찮음              데이터 손실 = 사고
에러 나도 나만 불편                 에러 = 학생들이 못 씀
```

코드를 아무리 잘 짜도 사용자에게 전달하지 못하면 소용없습니다. 배포는 **"내 코드가 사용자에게 도달하는 과정"**입니다.

### 왜 Stage와 Prod를 분리하는가?

```
위험한 방식:
코드 수정 → 바로 운영 서버에 적용 → 버그 발생 → 학생들 피해!

안전한 방식:
코드 수정 → Stage에 배포 → 테스트 → 문제 없음 → Prod에 배포
```

Stage(스테이징) 환경은 "운영 환경과 동일하지만 사용자가 없는" 환경입니다. 여기서 먼저 테스트하고 문제가 없을 때만 운영에 배포합니다.

### 데이터 백업은 왜 필수인가?

```
백업 없이 운영하다 DB가 날아가면:
- 학생들의 과제 제출물 전부 사라짐
- 회사 정보, 거래 내역 모두 소실
- 한 학기 데이터를 복구할 수 없음
- 교수님의 평가 자료도 사라짐

→ 재앙!
```

## 어떻게 만들었나요?

### 1단계: AWS EC2 서버 생성

AWS(Amazon Web Services)는 클라우드 컴퓨팅 서비스입니다. EC2(Elastic Compute Cloud)는 "인터넷에 연결된 가상 컴퓨터"입니다.

```
EC2 인스턴스 사양:
- 타입: t3.small (vCPU 2개, 메모리 2GB)
- OS: Ubuntu 24.04
- 지역: ap-northeast-2 (서울)
- 비용: 약 월 $20-25
```

왜 t3.small인가?
- 학생 50명 규모에 충분
- 비용 대비 성능 최적
- 필요하면 나중에 업그레이드 가능 (이것이 "클라우드"의 장점)

SSH로 서버에 접속:
```bash
ssh earnlearning
# 실제로는 ssh -i ~/.ssh/earnlearning.pem ubuntu@16.184.23.204
```

### 2단계: Docker Compose 구성

Docker는 "앱을 상자에 담아서 실행하는 기술"입니다.

```
Docker 없이 배포               Docker로 배포
─────────────────            ─────────────
1. Go 설치                    1. docker compose up
2. Node.js 설치               끝!
3. Nginx 설치
4. 각각 설정
5. 각각 실행
6. 환경마다 달라서 에러 발생
```

Docker Compose는 여러 Docker 컨테이너를 한 번에 관리합니다:

```yaml
# docker-compose.prod.yml (간략화)
version: '3.8'

services:
  backend:
    build: ../backend
    environment:
      - DB_PATH=/data/earnlearning.db
      - JWT_SECRET=${JWT_SECRET}
    volumes:
      - prod_db:/data              # DB 파일을 외부 볼륨에 저장

  frontend:
    build: ../frontend
    depends_on:
      - backend

  nginx:
    image: nginx:alpine
    ports:
      - "8080:80"                  # 호스트의 8080 → 컨테이너의 80
    depends_on:
      - backend
      - frontend

volumes:
  prod_db:
    external: true                 # 컨테이너 삭제해도 데이터 보존
```

핵심 포인트:
- `external: true` 볼륨: 컨테이너를 삭제/재생성해도 DB가 사라지지 않음
- 환경변수: JWT_SECRET 같은 민감 정보는 `.env` 파일에서 주입
- depends_on: 서비스 시작 순서 지정 (backend → frontend → nginx)

### 3단계: Nginx 리버스 프록시

Nginx는 "교통 정리 경찰"입니다. 사용자의 요청을 올바른 서비스로 안내합니다.

```
인터넷 → Cloudflare → 호스트 Nginx(80) → earnlearning.com     → :8080 (prod)
                                       → stage.earnlearning.com → :8081 (stage)
```

하나의 서버에서 두 환경을 운영하는 구조:

```
                    ┌─── Port 8080 ──→ [Prod Docker Compose]
호스트 Nginx ──────┤                   backend + frontend + nginx
(Port 80)          │
                    └─── Port 8081 ──→ [Stage Docker Compose]
                                       backend + frontend + nginx
```

### 4단계: Cloudflare 연결

Cloudflare는 여러 역할을 합니다:

```
1. DNS: earnlearning.com → 16.184.23.204 (IP 주소로 변환)
2. SSL: HTTP → HTTPS (암호화 통신)
3. CDN: 전 세계에 캐시 서버 배치 (빠른 응답)
4. DDoS 방어: 악성 트래픽 차단
5. 캐싱: 정적 파일(이미지, CSS)을 캐시
```

Cloudflare의 "Flexible SSL" 모드:

```
사용자 ──HTTPS──→ Cloudflare ──HTTP──→ EC2 서버
       (암호화)               (내부 통신)

장점: EC2에 SSL 인증서를 따로 설치할 필요 없음
주의: Cloudflare ↔ EC2 구간은 암호화되지 않음
      (같은 네트워크이므로 보통 문제 없음)
```

### 5단계: DB 백업 시스템

데이터베이스 백업은 보험과 같습니다. 평소에는 필요 없지만, 사고가 나면 없어서는 안 됩니다.

백업 전략:

```
1. 로컬 백업: docker cp로 DB 파일을 호스트로 복사
2. 원격 백업: Cloudflare R2에 업로드 (이중화)
3. 스케줄링: cron으로 매일 자동 실행
```

```bash
#!/bin/bash
# 백업 스크립트 핵심 로직

# 1. Docker 컨테이너에서 DB 파일 복사
docker cp earnlearning-prod-backend-1:/data/earnlearning.db \
  /home/ubuntu/backups/earnlearning-$(date +%Y%m%d).db

# 2. Cloudflare R2에 업로드 (AWS S3 호환)
aws s3 cp /home/ubuntu/backups/earnlearning-$(date +%Y%m%d).db \
  s3://earnlearning-backup/ --endpoint-url $R2_ENDPOINT

# 3. 7일 이상 된 로컬 백업 삭제 (디스크 공간 관리)
find /home/ubuntu/backups/ -mtime +7 -delete
```

왜 이중화(로컬 + R2)?
- 로컬만 있으면: EC2가 망가지면 백업도 함께 사라짐
- R2만 있으면: 네트워크 문제 시 복구 불가
- 둘 다 있으면: 어느 한쪽이 실패해도 복구 가능

### 6단계: 요청 로깅

서비스를 운영하면 "누가 언제 무엇을 했는지" 알아야 합니다:

```go
// 요청 로깅 미들웨어
func RequestLogger(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        start := time.Now()

        // 요청 처리
        err := next(c)

        // 로그 기록
        log.Printf("[%s] %s %s - %d (%v)",
            c.RealIP(),           // 누가
            c.Request().Method,   // 어떤 방식으로 (GET/POST/...)
            c.Request().URL.Path, // 무엇을
            c.Response().Status,  // 결과
            time.Since(start),    // 얼마나 걸렸나
        )

        return err
    }
}
```

로그 예시:
```
[203.0.113.42] POST /api/auth/login - 200 (15ms)
[203.0.113.42] GET /api/companies - 200 (8ms)
[203.0.113.42] POST /api/companies - 201 (23ms)
[198.51.100.7] GET /api/feed - 200 (12ms)
```

이 로그로 할 수 있는 것:
- 서비스 장애 원인 분석
- 사용 패턴 파악 (어떤 기능이 가장 많이 쓰이는지)
- 보안 이상 탐지 (비정상적인 요청 패턴)

### 배포 트러블슈팅: --force-recreate

배포 중 발견한 문제와 해결:

```
문제: 코드를 수정하고 재배포했는데 이전 버전이 계속 보임

원인: Nginx가 이전 backend 컨테이너의 IP를 캐시하고 있음
      Docker가 컨테이너를 재생성하면 IP가 바뀌는데,
      Nginx는 이전 IP로 계속 요청을 보냄

해결: --force-recreate 옵션으로 Nginx도 함께 재시작
```

```bash
# 수정 전
docker compose up -d --build

# 수정 후 (Nginx도 강제 재생성)
docker compose up -d --build --force-recreate
```

## 사용한 프롬프트

### 배포 구성 프롬프트
```
AWS EC2에 EarnLearning LMS를 배포해줘.
Docker Compose로 stage/prod를 분리하고,
호스트 Nginx로 도메인별 라우팅을 설정해줘.
Cloudflare DNS + Flexible SSL도 적용해줘.
```

### 백업 시스템 프롬프트
```
prod DB 백업 시스템을 만들어줘.
docker cp로 로컬 백업 + Cloudflare R2로 원격 백업.
external volume으로 컨테이너 삭제 시에도 DB 보존.
```

### 로깅 프롬프트
```
Go Echo 서버에 요청 로깅 미들웨어를 추가해줘.
IP, 메서드, 경로, 상태코드, 응답시간을 기록하고,
서버에서 로그를 쉽게 확인할 수 있는 스크립트도 만들어줘.
```

배포 프롬프트에서 핵심은 **인프라 요구사항을 구체적으로 명시**하는 것입니다. "배포해줘"가 아니라 "EC2, Docker Compose, stage/prod 분리, Nginx, Cloudflare"처럼 각 구성요소를 명시해야 원하는 결과를 얻을 수 있습니다.

## 배운 점

### 1. 배포는 개발만큼 중요하다
많은 개발자들이 코딩에만 집중하고 배포를 소홀히 합니다. 하지만 사용자에게 가치를 전달하는 것은 코드가 아니라 **배포된 서비스**입니다.

### 2. Docker는 현대 배포의 기본이다
"내 컴퓨터에서는 되는데..." 문제를 Docker가 해결합니다. 개발 환경과 운영 환경의 차이를 없애주므로, 배포 관련 문제가 크게 줄어듭니다.

### 3. 데이터 백업은 보험이다
백업은 귀찮고 비용이 들지만, 데이터를 잃었을 때의 피해와 비교하면 아무것도 아닙니다. **백업 없는 운영은 안전벨트 없는 운전과 같습니다.**

### 4. Stage 환경이 시간을 절약한다
운영 서버에서 직접 테스트하면 학생들에게 피해가 갑니다. Stage에서 먼저 확인하는 습관이 사고를 예방합니다.

### 5. 로그는 운영의 눈이다
서비스가 제대로 동작하는지, 문제가 발생했는지 알 수 있는 유일한 방법이 로그입니다. 처음부터 로깅을 잘 설계해놓으면 나중에 큰 도움이 됩니다.

### 6. 트러블슈팅 능력이 경쟁력이다
배포하면 반드시 예상치 못한 문제가 발생합니다. Nginx 캐시 문제처럼 코드가 아닌 인프라에서 발생하는 문제를 해결하는 능력이 실무에서 매우 중요합니다.

### 7. 비용 의식을 갖자
스타트업에서 인프라 비용은 중요합니다. t3.small($25/월)로 시작해서 사용자가 늘면 업그레이드하는 것이 현명합니다. 처음부터 큰 서버를 쓰면 돈 낭비입니다.
