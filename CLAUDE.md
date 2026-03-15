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
cd /home/ubuntu/lms/deploy

# Stage 배포 (빌드 + 배포)
./deploy.sh stage

# Production 배포 (빌드 + 배포 + 헬스체크)
./deploy.sh prod

# Stage → Production 프로모트 (빌드 스킵, ~5초)
./deploy.sh promote

# Production 풀 빌드 (캐시 무시, Dockerfile 변경 시)
./deploy.sh prod --full
```

**권장 배포 플로우**: `stage` → 스테이지 확인 → `promote`
```bash
./deploy.sh stage           # Stage 배포 (~30s 캐시 히트, ~3분 풀빌드)
# stage.earnlearning.com 에서 확인
./deploy.sh promote         # Stage 이미지를 Prod로 프로모트 (~5초)
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
- **TDD 방식 필수**: 버그 수정 및 새 기능 개발 시 반드시 TDD로 진행한다.
  1. 실패하는 테스트를 먼저 작성한다 (Red)
  2. 테스트를 통과시키는 최소한의 코드를 작성한다 (Green)
  3. 필요 시 리팩토링한다 (Refactor)
- **회귀 테스트 필수**: 버그 수정 시 반드시 해당 버그를 재현하는 회귀 테스트를 남겨 재발을 방지한다.
- **스모크 테스트 필수**: 커밋 또는 다른 테스트 실행 전에 반드시 스모크 테스트를 통과해야 한다.
  ```bash
  cd backend && go test ./tests/integration/ -run TestSmoke -timeout 60s
  ```
- 스모크 테스트 실패 시 커밋하지 않고 원인을 먼저 수정한다.
- **Backend 테스트**: `go test ./tests/integration/ -timeout 60s`
- **Frontend 테스트**: `cd frontend && npm test`

## 트러블슈팅

### Prod → Stage DB 복사
SQLite WAL 모드 + Docker 볼륨 마운트 차이로 인한 주의사항.

**핵심 함정**: Docker Compose에서 `stage_db:/data/db`로 마운트하면 볼륨 루트가 `/data/db/`에 매핑됨. temp container에서 같은 볼륨을 `/data`에 마운트하면 파일 경로가 달라진다.

| 컨테이너 | 마운트 | DB 경로 |
|----------|--------|---------|
| backend | `stage_db:/data/db` | `/data/db/earnlearning.db` |
| temp alpine | `stage_db:/data` | `/data/earnlearning.db` ← **여기에 복사** |

```bash
ssh earnlearning

# 1. Stage 중지 (DB 락 해제)
sudo docker stop earnlearning-stage-backend-1

# 2. Prod WAL checkpoint (WAL 데이터를 main DB로 병합)
sudo docker exec earnlearning-prod-backend-1 sh -c \
  'apk add sqlite > /dev/null 2>&1; sqlite3 /data/db/earnlearning.db "PRAGMA wal_checkpoint(TRUNCATE);"'

# 3. Prod DB를 호스트로 복사
sudo docker cp earnlearning-prod-backend-1:/data/db/earnlearning.db /tmp/prod_earnlearning.db

# 4. Stage 볼륨에 복사 (경로 주의: /data/earnlearning.db)
sudo docker run --rm \
  -v earnlearning-stage_stage_db:/data \
  -v /tmp:/host \
  alpine sh -c '
    rm -f /data/earnlearning.db /data/earnlearning.db-wal /data/earnlearning.db-shm
    cp /host/prod_earnlearning.db /data/earnlearning.db
    chmod 666 /data/earnlearning.db
  '

# 5. Stage 재시작 & 검증
sudo docker start earnlearning-stage-backend-1
sudo docker exec earnlearning-stage-backend-1 sh -c \
  'apk add sqlite > /dev/null 2>&1; sqlite3 /data/db/earnlearning.db "SELECT COUNT(*) FROM users;"'
```

**체크리스트**:
- [ ] WAL checkpoint 먼저 실행 (안 하면 최신 데이터 누락)
- [ ] temp container에서 `/data/earnlearning.db`에 복사 (NOT `/data/db/earnlearning.db`)
- [ ] WAL/SHM 파일 삭제 (구 stage 잔여 파일이 충돌 유발)
- [ ] 복사 후 `SELECT COUNT(*)` 로 데이터 확인

### SQLite WAL 관련 문제
- **증상**: DB 파일 크기는 작은데 데이터가 있어야 할 때 → WAL 파일에 데이터가 있음
- **해결**: `PRAGMA wal_checkpoint(TRUNCATE)` 로 WAL을 main DB에 병합
- **증상**: "disk I/O error" → 오래된 WAL/SHM 파일 삭제 후 재시작

### Frontend 빌드 실패 (테스트 파일)
- **증상**: `tsc -b` 에서 `vi` not found, `test` property 에러
- **원인**: 테스트 파일이 프로덕션 빌드에 포함됨
- **해결**: `tsconfig.app.json`에서 테스트 파일 exclude, `vite.config.ts`에서 `vitest/config` import

## 개발 워크플로우 (PR 기반)
- **main 직접 푸시 금지**: 모든 개발은 feature 브랜치에서 진행한다.
- **PR 생성 필수**: 작업 완료 후 PR을 생성하고 사용자가 리뷰 후 머지한다.
- **CI/CD**: main에 머지되면 GitHub Actions가 자동으로 Stage에 배포한다.
- **브랜치 네이밍**: `feat/기능명`, `fix/버그명`, `chore/작업명` 형식 사용.
- **Production 배포**: Stage 확인 후 `./deploy.sh promote`로 수동 프로모트.

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
