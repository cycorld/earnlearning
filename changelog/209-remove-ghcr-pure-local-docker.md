# 209. GHCR 의존성 완전 제거 및 Pure Local Docker 이미지 전환

## 무엇을 했나요?

- 배포 환경의 Docker 이미지 명칭에서 `ghcr.io/cycorld/` 외부 레지스트리 접두어를 전면 제거하고 `earnlearning-backend`, `earnlearning-frontend` 순수 로컬 명칭으로 정규화했습니다.
- `deploy/build-and-push.sh`에서 외부 레지스트리 푸시(`docker push`) 로직을 완전히 제거했습니다.
- `deploy/deploy.sh`에서 외부 레지스트리 풀(`docker compose pull`) 로직을 완전히 제거했습니다.
- `deploy-remote.sh`에서 순수 로컬 태그 기반의 SSH 스트리밍 파이프라인으로 전환했습니다.
- `deploy/tests/test-deploy-remote.sh` 회귀 테스트 스위트를 업데이트하여 33개 테스트 전수 통과를 검증했습니다.

## 왜 필요한가요?

- 외부 컨테이너 레지스트리(GHCR)에 토큰 만료, 패키지 권한(PAT scope), 네트워크 레이턴시 등 불필요한 외부 의존성이 발생했습니다.
- 빌드는 로컬 머신(`cycorld`)에서 5초 만에 완료되고, EC2 서버로는 SSH 파이프로 3.8초 만에 직배송되므로 외부 레지스트리 저장이 불필요합니다.

## 어떻게 안전하게 만들었나요?

- `docker-compose.blue.yml`, `docker-compose.green.yml`, `docker-compose.stage.yml` 이미지 경로 일괄 전환.
- 로컬 및 EC2 호스트의 `~/.docker/config.json`에서 만료된 `ghcr.io` 인증 정보 삭제.
- 배포 스크립트 회귀 테스트 33건 전수 검증 통과.

## 검증

`test-deploy-remote.sh` 33/33 PASS, 로컬 네이티브 빌드 및 도커 컴포즈 구문 정합성 확인.
