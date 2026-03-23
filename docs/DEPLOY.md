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

### 아키텍처

```
[로컬 Mac]                    [GHCR]                [EC2 t3.small]
 docker buildx               docker push            docker pull
 --platform amd64   ───────►  이미지 저장  ───────►  blue-green 전환
                                                     Host Nginx가 라우팅
```

### 서버 구성 (Blue-Green + Stage)
```
Host Nginx (port 80)
  ├── earnlearning.com       → active slot (blue:8180 또는 green:8181)
  └── stage.earnlearning.com → stage slot (8182)

Blue slot:  port 8180 (nginx) → backend + frontend
Green slot: port 8181 (nginx) → backend + frontend
Stage slot: port 8182 (nginx) → backend + frontend
```

Active slot은 `/etc/nginx/earnlearning-active-slot.conf` 파일로 결정.
전환 = 파일 내용 변경 + `nginx -s reload` (무중단, ~2초).

### 이미지 레지스트리 (GHCR)
```
ghcr.io/cycorld/earnlearning-backend:<sha>    # Go 백엔드
ghcr.io/cycorld/earnlearning-frontend:<sha>   # React 프론트 (stage/prod 공유)
```

Stage와 Prod가 동일 이미지를 사용 (VAPID 키는 런타임 API로 주입).

## 배포 명령어 (로컬에서 실행)

```bash
# 1. 빌드 → GHCR push → Stage 배포
./deploy-remote.sh

# 2. stage.earnlearning.com 에서 확인 후 Prod 배포
./deploy-remote.sh promote

# 3. 문제 발생 시 즉시 롤백 (~2초)
./deploy-remote.sh rollback

# 4. 서버 상태 확인
./deploy-remote.sh status
```

**권장 배포 플로우**:
```
PR 머지 → 로컬에서 ./deploy-remote.sh → Stage 확인 → ./deploy-remote.sh promote
```

### 서버에서 직접 실행 (SSH)

```bash
ssh earnlearning
cd /home/ubuntu/lms/deploy

IMAGE_TAG=<sha> ./deploy.sh stage       # Stage 배포
IMAGE_TAG=<sha> ./deploy.sh prod        # Prod blue-green 배포
./deploy.sh rollback                    # 즉시 롤백
./deploy.sh status                      # 상태 확인
```

## 서버 초기 설정 (1회)

```bash
ssh earnlearning

# GHCR 인증
echo "<GITHUB_PAT>" | sudo docker login ghcr.io -u cycorld --password-stdin

# Active slot 초기 파일 생성
echo "server 127.0.0.1:8180;" | sudo tee /etc/nginx/earnlearning-active-slot.conf

# Host Nginx 설정 교체
sudo cp /home/ubuntu/lms/deploy/nginx-host.conf /etc/nginx/sites-available/earnlearning
sudo nginx -t && sudo nginx -s reload
```

## 환경변수 (서버: /home/ubuntu/lms/deploy/)
- `.env.prod` / `.env.stage` — JWT_SECRET 등 키 분리됨
- `JWT_SECRET`, `ADMIN_EMAIL`, `ADMIN_PASSWORD`
- `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY` (Web Push용)
- AWS SES 인증 정보 (이메일 발송용)

## VAPID 키 생성 (Web Push 알림용)

```bash
npx web-push generate-vapid-keys
```

**주의**: VAPID 키는 한번 생성 후 변경하지 않아야 합니다. 변경 시 기존 푸시 구독이 모두 무효화됩니다.

## 롤백

```bash
# 즉시 롤백 — nginx 전환만, ~2초
./deploy-remote.sh rollback

# 특정 버전으로 롤백 — 이미지 pull 필요, ~30초
ssh earnlearning "cd /home/ubuntu/lms/deploy && IMAGE_TAG=<이전sha> ./deploy.sh prod"
```

## 메모리 제약 (t3.small 2GB)

| 컴포넌트 | 예상 메모리 |
|----------|-----------|
| Active prod (backend+frontend+nginx) | ~200MB |
| Stage (backend+frontend+nginx) | ~200MB |
| 전환 중 비활성 slot | ~200MB |
| OS + Docker | ~400MB |
| **합계** | ~1000MB / 2048MB |

## 트러블슈팅

### Prod → Stage DB 복사

```bash
ssh earnlearning

# 1. Stage 중지
sudo docker compose -f /home/ubuntu/lms/deploy/docker-compose.stage.yml -p earnlearning-stage stop backend

# 2. Prod WAL checkpoint
CONTAINER=$(./deploy.sh status 2>&1 | grep -oP 'earnlearning-(blue|green)' | head -1)-backend-1
sudo docker exec $CONTAINER sh -c \
  'apk add sqlite > /dev/null 2>&1; sqlite3 /data/db/earnlearning.db "PRAGMA wal_checkpoint(TRUNCATE);"'

# 3. Prod DB를 호스트로 복사
sudo docker cp $CONTAINER:/data/db/earnlearning.db /tmp/prod_earnlearning.db

# 4. Stage 볼륨에 복사
sudo docker run --rm \
  -v earnlearning-stage_stage_db:/data \
  -v /tmp:/host \
  alpine sh -c '
    rm -f /data/earnlearning.db /data/earnlearning.db-wal /data/earnlearning.db-shm
    cp /host/prod_earnlearning.db /data/earnlearning.db
    chmod 666 /data/earnlearning.db
  '

# 5. Stage 재시작
sudo docker compose -f /home/ubuntu/lms/deploy/docker-compose.stage.yml -p earnlearning-stage start backend
```

### SQLite WAL 관련 문제
- **증상**: DB 파일 크기는 작은데 데이터가 있어야 할 때 → WAL 파일에 데이터가 있음
- **해결**: `PRAGMA wal_checkpoint(TRUNCATE)` 로 WAL을 main DB에 병합
- **증상**: "disk I/O error" → 오래된 WAL/SHM 파일 삭제 후 재시작

### Frontend 빌드 실패 (테스트 파일)
- **증상**: `tsc -b` 에서 `vi` not found, `test` property 에러
- **원인**: 테스트 파일이 프로덕션 빌드에 포함됨
- **해결**: `tsconfig.app.json`에서 테스트 파일 exclude, `vite.config.ts`에서 `vitest/config` import
