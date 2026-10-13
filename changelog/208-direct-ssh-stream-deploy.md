# 208. SSH 파이프 직배송 배포 파이프라인 구축 (GHCR 의존성 우회)

## 무엇을 했나요?

로컬 빌드 머신(`cycorld`)에서 빌드한 Docker 이미지를 SSH 스트리밍 파이프(`docker save | gzip | ssh earnlearning "gunzip | docker load"`)를 통해 EC2 로컬 Docker 데몬으로 직접 전송하는 파이프라인을 원커맨드 배포 스크립트에 탑재했습니다.

## 왜 필요한가요?

- EC2 인스턴스는 `t3.small`(2GB RAM) 사양으로 서버 내 직접 빌드 시 OOM 메모리 고갈로 실서비스 크래시 위험이 있습니다.
- GitHub Personal Access Token(PAT) 만료 또는 패키지 스코프 누락 시 GHCR `denied` 에러로 배포가 중단되는 취약점이 있었습니다.
- 외부 컨테이너 레지스트리를 경유하지 않고 로컬 머신에서 EC2로 4초 만에 압축 직배송하여 토큰/네트워크 의존성을 원천 차단합니다.

## 어떻게 안전하게 만들었나요?

- `deploy/build-and-push.sh`: GHCR 푸시 실패 시 경고 후 로컬 빌드 이미지 보존 및 스트리밍 단계로 안전하게 폴백.
- `deploy-remote.sh`: 로컬 빌드 후 EC2로 `docker save | gzip -1 | ssh earnlearning "gunzip | docker load"` 파이프라인 기본 탑재.
- `deploy/deploy.sh`: `docker compose pull` 실패 시에도 로컬에 적재된 이미지를 감지하여 정상 컨테이너 기동.

## 검증

- 로컬 이미지(backend 32MB, frontend 66MB) 대상 SSH 스트리밍 전송 실측 (전송 시간 3.8초).
- Stage 및 Production Blue-Green 무중단 전환 테스트 완료.
