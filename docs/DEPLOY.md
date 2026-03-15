# 배포 가이드

## 인프라 구성

### AWS 리소스 (태그: Project=EarnLearning)
- **EC2**: t3.small, ap-northeast-2, Ubuntu 24.04
- **SSH**: `ssh earnlearning` (~/.ssh/earnlearning.pem)
- 상세 정보 (IP, 인스턴스 ID, SG 등)는 `.env.deploy` 참조

### 도메인 & DNS (Cloudflare)
- **Production**: https://earnlearning.com
- **Staging**: https://stage.earnlearning.com
- Zone ID 등 상세 정보는 `.env.deploy` 참조
- **SSL**: Cloudflare Flexible + Always HTTPS

### 서버 구성
하나의 EC2에 stage/prod 두 환경을 Docker Compose로 운영:
```
호스트 Nginx (80) → earnlearning.com     → localhost:8080 (prod)
                  → stage.earnlearning.com → localhost:8081 (stage)

각 환경 = backend + frontend + nginx (Docker Compose)
데이터: Docker volume (prod_db, stage_db 분리)
```

## CI/CD

main에 머지되면 GitHub Actions가 자동으로 Stage 배포 (SSH, ~33초)
- Secrets: `SERVER_HOST`, `SERVER_USER`, `SERVER_SSH_KEY` (gh secret으로 등록 완료)
- Workflow: `.github/workflows/deploy-stage.yml`

## 배포 명령어

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

**권장 배포 플로우**: PR 머지 → Stage 자동 배포 → 확인 → `promote`
```bash
# PR 머지 후 GitHub Actions가 stage 자동 배포 (~33초)
# stage.earnlearning.com 에서 확인
ssh earnlearning "cd /home/ubuntu/lms/deploy && ./deploy.sh promote"  # ~5초
```

## 환경변수 (서버: /home/ubuntu/lms/deploy/)
- `.env.prod` / `.env.stage` — JWT_SECRET 등 키 분리됨
- `JWT_SECRET`, `ADMIN_EMAIL`, `ADMIN_PASSWORD`
- `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY` (Web Push용, 아직 미설정)

## VAPID 키 생성 (Web Push 알림용)

배포 전에 반드시 VAPID 키 쌍을 생성하여 환경변수에 등록해야 합니다.

```bash
npx web-push generate-vapid-keys
```

생성된 키를 `.env`에 등록:
```
VAPID_PUBLIC_KEY=<생성된 공개키>
VAPID_PRIVATE_KEY=<생성된 비밀키>
VAPID_SUBJECT=mailto:<관리자 이메일>
```

**주의**: VAPID 키는 한번 생성 후 변경하지 않아야 합니다. 변경 시 기존 푸시 구독이 모두 무효화됩니다.

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
