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
[빌드서버 cycorld]           [GHCR]                [EC2 t3.small]
 docker build              docker push            docker pull
 네이티브 amd64      ───────►  이미지 저장  ───────►  blue-green 전환
 (x86_64 16코어)                                    Host Nginx가 라우팅
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

## 배포 명령어 (빌드서버 cycorld에서 실행)

`deploy-remote.sh`는 빌드서버 cycorld 자신에서 실행된다 (레포: `/home/cycorld/Workspace/earnlearning`). 빌드는 로컬에서 `./deploy/build-and-push.sh`로 수행하고, Stage/Prod 배포·롤백·상태 확인만 `ssh earnlearning`으로 원격 실행한다.

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

### 배포 전 안전 검증

`deploy` 서브커맨드는 빌드를 시작하기 전에 세 가지 조건을 모두 확인하고, 하나라도 어긋나면 빌드하지 않고 실패한다:

1. **현재 브랜치가 `main`** 이어야 한다.
2. **작업 트리가 clean** 해야 한다 (커밋되지 않은 변경 없음).
3. `git fetch origin main` 후 **HEAD == origin/main** 이어야 한다 (원격과 동기화됨).

스크립트는 **자동으로 pull 하거나 브랜치를 바꾸지 않는다.** origin/main과 어긋나면 사용자에게 직접 `git pull --ff-only`를 실행하라고 안내한다 — 오래됐거나 무관한 코드를 조용히 빌드/배포하는 사고를 막기 위해서다.

`promote`는 EC2의 git HEAD를 추측하지 않는다. **Stage에서 실제로 돌고 있는 컨테이너의 이미지 태그**를 직접 읽어 (`docker inspect earnlearning-stage-backend-1`) 그 태그를 Prod blue-green으로 승격한다. 태그가 비어 있거나 `latest`이면 거부한다. 특정 태그로 강제하려면 `IMAGE_TAG=<sha> ./deploy-remote.sh promote`.

회귀 테스트: `deploy/tests/test-deploy-remote.sh` (git/ssh/build-and-push를 mock으로 대체해 위 동작을 검증). 실행:

```bash
bash deploy/tests/test-deploy-remote.sh
```

**권장 배포 플로우**:
```
PR 머지 → 빌드서버(cycorld)에서 main을 git pull --ff-only 로 최신화 → ./deploy-remote.sh → Stage 확인 → ./deploy-remote.sh promote
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
