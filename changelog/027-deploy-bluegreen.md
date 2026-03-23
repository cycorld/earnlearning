# 027. 로컬 빌드 + GHCR + Blue-Green 무중단 배포

> **날짜**: 2026-03-23
> **태그**: `배포`, `Docker`, `Blue-Green`, `GHCR`, `무중단`

## 무엇을 했나요?

기존에 EC2 서버에서 직접 Docker 이미지를 빌드하던 방식을 **로컬 빌드 + GHCR(GitHub Container Registry) push + Blue-Green 무중단 배포**로 전환했습니다.

## 왜 필요했나요?

EC2 t3.small(2GB RAM)에서 Docker 빌드 시 메모리가 부족해서:
- SSH 연결이 끊기거나
- 서비스가 다운되는 문제가 발생

빌드를 로컬 Mac으로 옮기면 서버는 이미지를 pull만 하면 되어 리소스 걱정이 없습니다.

## 어떻게 만들었나요?

### 핵심 발견: Stage/Prod 동일 이미지 가능
VAPID 키(웹 푸시용)가 이미 런타임 API(`/api/push/vapid-public-key`)로 주입되고 있었습니다. 프론트엔드 빌드 시 `VITE_VAPID_PUBLIC_KEY`는 사용되지 않는 dead code였습니다. → **하나의 이미지로 Stage/Prod 모두 사용 가능!**

### 배포 아키텍처
```
[로컬 Mac]                    [GHCR]                [EC2 t3.small]
 docker buildx               docker push            docker pull
 --platform amd64   ───────►  이미지 저장  ───────►  blue-green 전환
                                                     Host Nginx가 라우팅
```

### Blue-Green 배포란?
두 개의 동일한 환경(Blue, Green)을 준비해두고, 한쪽에서 서비스하는 동안 다른 쪽에 새 버전을 배포합니다. 새 버전이 정상이면 Nginx가 트래픽을 전환합니다.

```
Host Nginx (port 80)
  ├── earnlearning.com       → active slot (blue:8180 또는 green:8181)
  └── stage.earnlearning.com → stage (8182)
```

전환은 `/etc/nginx/earnlearning-active-slot.conf` 파일 한 줄만 바꾸고 `nginx -s reload` — 약 2초면 완료됩니다.

### 롤백
문제 발생 시 이전 slot으로 Nginx만 전환하면 즉시 롤백됩니다. 이미지를 다시 빌드할 필요가 없어서 2초면 복구됩니다.

### 변경된 파일들
- `deploy/build-and-push.sh` — 로컬에서 amd64 이미지 빌드 + GHCR push
- `deploy-remote.sh` — 원커맨드 배포 (빌드→push→SSH 배포)
- `deploy/deploy.sh` — 서버 blue-green 배포 스크립트
- `deploy/docker-compose.blue.yml`, `green.yml` — Blue/Green slot compose
- `deploy/nginx-host.conf` — upstream + active-slot 방식

### 삭제된 파일들
- `.github/workflows/deploy-stage.yml` — GitHub Actions 자동 배포 (로컬 배포로 대체)
- `deploy/docker-compose.prod.yml` — blue/green으로 대체

## 사용한 프롬프트

```
배포 개선: 로컬 빌드 + GHCR + Blue-Green 무중단 배포
```

## 배운 점

1. **Dead code 발견의 가치**: VAPID 키가 빌드 arg로 전달되지만 실제로는 런타임 API로 주입되고 있었습니다. 이 발견 덕분에 Stage/Prod 동일 이미지 전략이 가능해졌습니다.
2. **Blue-Green 배포**: 복잡한 쿠버네티스 없이도 Nginx + Docker Compose만으로 무중단 배포를 구현할 수 있습니다.
3. **서버 리소스 분리**: 빌드(CPU/메모리 집중)와 실행(경량)을 분리하면 작은 서버에서도 안정적으로 운영할 수 있습니다.
4. **stdout vs stderr 분리**: 스크립트가 다른 스크립트에 의해 캡처(`$(...)`)될 때, 로그 메시지는 stderr로 보내고 반환값만 stdout으로 출력해야 합니다. 그렇지 않으면 로그가 반환값에 섞입니다.
5. **크로스 빌드 vs 네이티브 빌드**: Mac에서 `--platform linux/amd64` 크로스 빌드는 QEMU 에뮬레이션으로 30분+ 걸리지만, x86_64 서버에서 네이티브 빌드하면 1분이면 됩니다. 빌드 전용 서버를 분리하면 개발 경험이 크게 개선됩니다.
