# EarnLearning LMS

## 프로젝트 개요
이화여자대학교 "스타트업을 위한 코딩입문" 강의용 게임화 창업 교육 LMS.

## 기술 스택
- **Backend**: Go (Echo) + SQLite (Docker volume persistent)
- **Frontend**: Vite + React 18 + TypeScript + Tailwind CSS + shadcn/ui
- **Realtime**: WebSocket + Web Push (VAPID)
- **PWA**: Vite PWA Plugin (홈 화면 설치, 오프라인 캐시, 웹 푸시)
- **Auth**: JWT (이메일 회원가입 + Admin 승인제)
- **Deploy**: AWS EC2 (t3.small) + Docker Compose + Nginx + Cloudflare (SSL/CDN)

## 배포 인프라

### AWS 리소스 (태그: Project=EarnLearning)
- **EC2**: `<EC2_INSTANCE_ID>` (t3.small, ap-northeast-2, Ubuntu 24.04)
- **IP**: <SERVER_IP>
- **SSH**: `ssh earnlearning` (~/.ssh/earnlearning.pem)
- **Security Group**: `<SECURITY_GROUP_ID>` (22, 80, 443)
- **Key Pair**: earnlearning

### 도메인 & DNS (Cloudflare)
- **Production**: https://earnlearning.com
- **Staging**: https://stage.earnlearning.com
- **Zone ID**: `<CLOUDFLARE_ZONE_ID>`
- **SSL**: Cloudflare Flexible + Always HTTPS

### 서버 구성
하나의 EC2에 stage/prod 두 환경을 Docker Compose로 운영:
```
호스트 Nginx (80) → earnlearning.com     → localhost:8080 (prod)
                  → stage.earnlearning.com → localhost:8081 (stage)

각 환경 = backend + frontend + nginx (Docker Compose)
데이터: Docker volume (prod_db, stage_db 분리)
```

### 배포 명령어
```bash
ssh earnlearning
cd /home/ubuntu/lms && git pull

# Production 배포 (--force-recreate로 nginx도 함께 재시작)
cd deploy && sudo docker compose -f docker-compose.prod.yml -p earnlearning-prod --env-file .env.prod up -d --build --force-recreate

# Staging 배포
cd deploy && sudo docker compose -f docker-compose.stage.yml -p earnlearning-stage --env-file .env.stage up -d --build --force-recreate
```

### 환경변수 (서버: /home/ubuntu/lms/deploy/)
- `.env.prod` / `.env.stage` — JWT_SECRET 등 키 분리됨
- `JWT_SECRET`, `ADMIN_EMAIL`, `ADMIN_PASSWORD`
- `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY` (Web Push용, 아직 미설정)

## 배포 시 필수 작업

### VAPID 키 생성 (Web Push 알림용)
배포 전에 반드시 VAPID 키 쌍을 생성하여 환경변수에 등록해야 합니다.
사용자(최용철)에게 아래 절차를 안내하세요:

```bash
# 방법 1: npx로 생성
npx web-push generate-vapid-keys

# 방법 2: Go로 생성 (서버에서)
# 프로젝트에 키 생성 CLI 포함 예정: go run cmd/keygen/main.go

# 출력 예시:
# Public Key:  BEl62iUYgUivxIkv69yViEuiBIa-Ib9...
# Private Key: UUxI4o8r2Hx_NbFkidF1L9Gi...
```

생성된 키를 `.env` 또는 docker-compose 환경변수에 등록:
```
VAPID_PUBLIC_KEY=BEl62iUYgUivxIkv69yViEuiBIa-Ib9...
VAPID_PRIVATE_KEY=UUxI4o8r2Hx_NbFkidF1L9Gi...
VAPID_SUBJECT=mailto:${CONTACT_EMAIL}
```

**주의**: VAPID 키는 한번 생성 후 변경하지 않아야 합니다. 변경 시 기존 푸시 구독이 모두 무효화됩니다.

## 테스트 규칙
- **스모크 테스트 필수**: 커밋 또는 다른 테스트 실행 전에 반드시 스모크 테스트를 통과해야 한다.
  ```bash
  cd backend && go test ./tests/integration/ -run TestSmoke -timeout 60s
  ```
- 스모크 테스트 실패 시 커밋하지 않고 원인을 먼저 수정한다.
- 새 기능 추가 시 관련 회귀 테스트도 함께 작성한다.

## 커밋 규칙
- 매 프롬프트 작업 완료 시 반드시 커밋한다.
- 커밋 전 반드시 스모크 테스트 통과 확인.
- 커밋 메시지 형식:
  ```
  [작업내용 요약 타이틀]

  prompt: [사용자가 입력한 원본 프롬프트]

  - [작업 내역 1]
  - [작업 내역 2]
  - ...
  ```
