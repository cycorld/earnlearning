---
title: "CI/CD와 보안: 자동화된 배포와 안전한 운영"
date: "2026-03-15"
tags: ["CI/CD", "GitHub Actions", "보안", "자동화", "배포"]
---

## 무엇을 했나요?

개발 프로세스의 마지막 단추를 채웠습니다. 코드를 푸시하면 자동으로 테스트하고 배포하는 CI/CD 파이프라인을 구축하고, 보안 취약점을 정리했습니다:

- **배포 최적화**: deploy.sh 스크립트 작성 + Stage에서 Prod로 프로모트하는 워크플로우
- **GitHub Actions CI/CD**: PR 생성 시 자동 테스트, 머지 시 자동 배포
- **빌드 정보 표시**: 현재 배포 버전을 UI에서 확인 가능
- **보안 정리**: 코드에 하드코딩된 민감 정보 제거, .env 참조로 전환

## 왜 필요했나요?

### 수동 배포의 위험성

005에서 만든 배포 방식은 수동이었습니다:

```bash
# 수동 배포 과정 (매번 이걸 해야 함)
ssh earnlearning
cd /home/ubuntu/lms
git pull
cd deploy
sudo docker compose -f docker-compose.prod.yml -p earnlearning-prod \
  --env-file .env.prod up -d --build --force-recreate
```

이 방식의 문제:

```
수동 배포의 위험:
1. 명령어를 잘못 입력할 수 있음 (prod 대신 stage 파일 사용)
2. git pull을 까먹을 수 있음 (이전 코드로 배포)
3. 테스트를 안 돌리고 배포할 수 있음 (버그 있는 코드 배포)
4. 누가 언제 배포했는지 기록이 없음
5. 배포 자체를 까먹을 수 있음

자동 배포의 장점:
1. 코드를 push하면 자동으로 배포 (실수 불가)
2. 자동으로 테스트 실행 (통과해야만 배포)
3. 배포 이력이 GitHub에 자동 기록
4. 누구든 같은 방식으로 배포 (일관성)
```

### 보안이 왜 중요한가?

개발 중에는 편의를 위해 비밀번호나 API 키를 코드에 직접 넣는 경우가 있습니다:

```go
// 이런 코드가 GitHub에 올라가면...
const JWT_SECRET = "super-secret-key-12345"
const ADMIN_PASSWORD = "admin123!"
```

이게 왜 위험한가?

```
GitHub 저장소가 public이면:
→ 전 세계 누구나 비밀번호를 볼 수 있음
→ 해커가 관리자 계정으로 로그인 가능
→ JWT 시크릿을 알면 아무 사용자로 위장 가능

GitHub 저장소가 private이라도:
→ 팀원 전원이 비밀번호를 알게 됨
→ 나중에 public으로 바뀌면 노출됨
→ Git 히스토리에 영구 기록됨 (삭제해도 남아있음!)
```

## 어떻게 만들었나요?

### CI/CD 파이프라인 이해하기

CI/CD는 두 가지 개념의 합성어입니다:

```
CI (Continuous Integration) - 지속적 통합
= 코드 변경 시 자동으로 테스트하고 빌드

CD (Continuous Deployment) - 지속적 배포
= 테스트 통과한 코드를 자동으로 서버에 배포
```

전체 흐름:

```
개발자가 코드 수정
    ↓
git push (GitHub에 업로드)
    ↓
GitHub Actions 자동 실행
    ↓
┌─────────────────────────┐
│  1. 코드 체크아웃          │
│  2. Go 빌드              │
│  3. 스모크 테스트          │
│  4. 전체 테스트            │
│  5. 프론트엔드 빌드        │
│  6. Docker 이미지 빌드     │
└─────────────────────────┘
    ↓
모두 통과?
    ├── Yes → Stage에 자동 배포
    └── No  → 실패 알림 → 수정 필요
```

### GitHub Actions 워크플로우

GitHub Actions는 GitHub에서 제공하는 무료 CI/CD 도구입니다.

```yaml
# .github/workflows/ci.yml
name: CI/CD Pipeline

on:
  pull_request:          # PR 생성 시 테스트
    branches: [main]
  push:                  # main에 머지 시 배포
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      # 1. 코드 가져오기
      - uses: actions/checkout@v4

      # 2. Go 설치
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      # 3. Node.js 설치 (프론트엔드 빌드용)
      - uses: actions/setup-node@v4
        with:
          node-version: '20'

      # 4. 백엔드 테스트
      - name: Run smoke tests
        run: cd backend && go test ./tests/integration/ -run TestSmoke -timeout 60s

      - name: Run all tests
        run: cd backend && go test ./... -timeout 120s

      # 5. 프론트엔드 빌드
      - name: Build frontend
        run: cd frontend && npm ci && npm run build

  deploy:
    needs: test          # test가 통과해야만 실행
    if: github.ref == 'refs/heads/main'  # main 브랜치만
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to server
        uses: appleboy/ssh-action@v1
        with:
          host: ${{ secrets.SERVER_HOST }}
          username: ubuntu
          key: ${{ secrets.SSH_PRIVATE_KEY }}
          script: |
            cd /home/ubuntu/lms
            git pull
            cd deploy
            ./deploy.sh prod
```

핵심 포인트를 하나씩 살펴봅시다:

**`on: pull_request` / `on: push`**
- PR을 만들면 테스트만 실행 (코드 리뷰 전 품질 확인)
- main에 머지(push)하면 테스트 + 배포

**`needs: test`**
- deploy 작업은 test가 성공한 후에만 실행
- 테스트가 실패하면 배포되지 않음 (안전장치)

**`secrets.SSH_PRIVATE_KEY`**
- 서버 접속용 SSH 키를 GitHub Secrets에 저장
- 코드에 노출되지 않음 (보안)

### 배포 스크립트 (deploy.sh)

수동으로 길게 치던 명령어를 스크립트로 만들었습니다:

```bash
#!/bin/bash
# deploy.sh - 배포 자동화 스크립트

set -e  # 에러 발생 시 즉시 중단

ENV=${1:-stage}  # 인자가 없으면 stage

if [ "$ENV" = "prod" ]; then
    echo "🚀 Production 배포 시작..."
    COMPOSE_FILE="docker-compose.prod.yml"
    PROJECT_NAME="earnlearning-prod"
    ENV_FILE=".env.prod"
elif [ "$ENV" = "stage" ]; then
    echo "🧪 Staging 배포 시작..."
    COMPOSE_FILE="docker-compose.stage.yml"
    PROJECT_NAME="earnlearning-stage"
    ENV_FILE=".env.stage"
else
    echo "사용법: ./deploy.sh [stage|prod]"
    exit 1
fi

# 배포 실행
sudo docker compose \
    -f $COMPOSE_FILE \
    -p $PROJECT_NAME \
    --env-file $ENV_FILE \
    up -d --build --force-recreate

echo "✅ $ENV 배포 완료!"
```

이제 배포가 이렇게 간단해졌습니다:

```bash
# Stage 배포
./deploy.sh stage

# Stage에서 테스트 후 Prod 배포
./deploy.sh prod
```

### Stage → Prod 프로모트 워크플로우

안전한 배포 순서:

```
1. 코드 변경 → PR 생성
    ↓
2. GitHub Actions: 자동 테스트 ✅
    ↓
3. 코드 리뷰 → 머지
    ↓
4. GitHub Actions: Stage 자동 배포
    ↓
5. Stage에서 수동 확인 (stage.earnlearning.com)
    ↓
6. 문제 없으면 Prod 배포 (./deploy.sh prod)
```

### 빌드 정보 표시

"지금 서버에 어떤 버전이 배포되어 있지?"를 확인할 수 있도록 빌드 정보를 UI에 표시합니다:

```
페이지 하단:
┌──────────────────────────────────────┐
│ v1.0.0 | Build: abc1234 | 2026-03-15 │
└──────────────────────────────────────┘
```

빌드 시 Git 커밋 해시를 환경변수로 주입:

```bash
# Docker 빌드 시 커밋 정보 전달
docker build \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  --build-arg BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  .
```

이 정보가 있으면:
- "방금 배포한 코드가 반영되었나?" 즉시 확인 가능
- 버그 보고 시 "어떤 버전에서 발생했나?" 추적 가능

### 보안 정리

코드에 하드코딩된 민감 정보를 모두 환경변수 참조로 변경했습니다:

```go
// 수정 전 (위험!)
const JWT_SECRET = "my-super-secret-key"

// 수정 후 (안전)
var JWT_SECRET = os.Getenv("JWT_SECRET")
```

점검 항목:

```
✅ JWT_SECRET → .env 파일로 이동
✅ ADMIN_EMAIL, ADMIN_PASSWORD → .env 파일로 이동
✅ 데이터베이스 경로 → 환경변수 참조
✅ .env 파일이 .gitignore에 포함되어 있는지 확인
✅ Git 히스토리에서 민감 정보 흔적 확인
```

**.env 파일과 .gitignore의 관계:**

```
.gitignore:
─────────
.env
.env.prod
.env.stage
*.pem
*.key

→ 이 파일들은 Git에 커밋되지 않음
→ 서버에서만 직접 생성/관리
```

### GitHub Secrets 사용

CI/CD에서 필요한 비밀 정보는 GitHub Secrets에 저장합니다:

```
GitHub 저장소 → Settings → Secrets and variables → Actions

등록된 Secrets:
- SERVER_HOST: 서버 IP 주소
- SSH_PRIVATE_KEY: 서버 접속용 SSH 키
- DEPLOY_USER: 배포 사용자명
```

이렇게 하면:
- 코드에 비밀 정보가 노출되지 않음
- 저장소 관리자만 Secrets를 열람/수정 가능
- 로그에도 마스킹되어 표시됨 (****)

## 사용한 프롬프트

### CI/CD 프롬프트
```
GitHub Actions CI/CD 파이프라인을 구축해줘.
PR 시 자동 테스트, main 머지 시 자동 배포.
배포 스크립트(deploy.sh)도 만들어서 stage→prod 프로모트 가능하게.
빌드 정보(커밋 해시, 날짜)를 UI에 표시해줘.
```

### 보안 정리 프롬프트
```
코드에 하드코딩된 모든 민감 정보를 찾아서 .env 참조로 변경해줘.
JWT_SECRET, 비밀번호, API 키 등 모든 시크릿을 점검해줘.
.gitignore도 확인하고, Git 히스토리에 민감 정보가 있으면 알려줘.
```

CI/CD 프롬프트의 핵심은 **"PR 시 테스트, 머지 시 배포"**라는 워크플로우를 명확히 하는 것입니다. 자동화 수준을 구체적으로 지정해야 원하는 파이프라인을 얻을 수 있습니다.

## 배운 점

### 1. 자동화는 실수를 줄인다
사람은 반복 작업에서 실수합니다. 배포 명령어를 100번 치면 1~2번은 실수합니다. 자동화하면 실수가 0이 됩니다. CI/CD는 "사람의 실수를 시스템이 방지하는 것"입니다.

### 2. 테스트 없는 CI/CD는 무의미하다
자동 배포가 있어도 테스트가 없으면 "버그를 자동으로 배포"하는 것에 불과합니다. CI/CD의 진짜 가치는 **"테스트를 통과한 코드만 배포된다"**는 보장입니다.

### 3. 보안은 처음부터
프로젝트 초기에 .env 패턴을 확립하면 민감 정보가 코드에 들어갈 일이 없습니다. 나중에 정리하면 Git 히스토리에 흔적이 남아 더 복잡해집니다.

### 4. PR 기반 워크플로우
```
기능 브랜치 생성 → 개발 → PR 생성 → 자동 테스트 → 코드 리뷰 → 머지 → 자동 배포
```
이 흐름은 거의 모든 현업 소프트웨어 팀에서 사용합니다. 이 경험은 취업 후 바로 적용할 수 있습니다.

### 5. 비밀 관리의 계층 구조
```
코드에 직접 작성 (최악) ❌
→ .env 파일 (.gitignore) (기본)
→ GitHub Secrets (CI/CD용)
→ AWS Secrets Manager (대규모 서비스)
```
프로젝트 규모에 맞는 비밀 관리 방식을 선택하세요.

### 6. "배포 가능한 상태"를 유지하자
CI/CD가 있으면 main 브랜치는 항상 배포 가능한 상태여야 합니다. 이를 위해 PR + 테스트 + 코드 리뷰를 거쳐야만 main에 합칠 수 있도록 합니다.

### 7. 빌드 정보는 디버깅의 시작점이다
"이 버그가 어느 버전에서 발생했지?"를 빌드 정보로 바로 확인할 수 있습니다. 단순한 기능이지만 운영 시 매우 유용합니다.

---

## GitHub 참고 링크
- [PR #1: CI/CD 파이프라인 구축](https://github.com/cycorld/earnlearning/pull/1)
- [커밋 372f6d9: 보안: 민감 정보 제거 및 .env 참조로 전환](https://github.com/cycorld/earnlearning/commit/372f6d9)
