# 핫픽스 가이드

프로덕션에서 긴급 버그가 발견되었을 때의 대응 절차.

## 핫픽스 vs 일반 배포

| 구분 | 일반 배포 | 핫픽스 |
|------|----------|--------|
| 브랜치 | `feat/기능명` | `fix/버그명` |
| 테스트 | 전체 테스트 | 스모크 + 해당 기능 |
| Stage 확인 | 꼼꼼히 | 빠르게 (API 레벨) |
| Prod 배포 | Stage 확인 후 | Stage 통과 즉시 |

## 핫픽스 절차

### 1. 문제 확인
```bash
# 프로덕션 서버 로그 확인
ssh earnlearning "cd /home/ubuntu/lms/deploy && ./logs.sh prod"

# API 직접 호출로 에러 재현
curl -s -H "Authorization: Bearer $TOKEN" "https://earnlearning.com/api/문제API"
```

### 2. 로컬에서 원인 파악 및 수정
```bash
git checkout main && git pull
git checkout -b fix/버그명

# 수정 후 빌드 확인
cd backend && go build ./...
cd frontend && npx tsc --noEmit

# 스모크 테스트
cd backend && go test ./tests/integration/ -run TestSmoke -timeout 60s
```

### 3. PR → 머지 → 배포
```bash
git push -u origin fix/버그명
gh pr create --title "fix: 설명"
gh pr merge N --merge

# 빌드 → Stage 배포
./deploy-remote.sh

# Stage 확인 후 Prod 배포
./deploy-remote.sh promote
```

### 4. 검증
```bash
# 헬스체크
curl -s https://earnlearning.com/api/health
curl -s https://earnlearning.com/api/version

# 문제 API 재확인
curl -s -H "Authorization: Bearer $TOKEN" "https://earnlearning.com/api/문제API"
```

## 자주 발생하는 문제 유형

### 502 Bad Gateway (nginx upstream 연결 실패)
- **원인**: Docker 컨테이너 재생성 시 IP 변경, nginx가 이전 IP를 캐시
- **해결**: `sudo docker compose ... restart nginx`

### SQL 쿼리 에러 (500)
- **원인**: 로컬/테스트 DB와 프로덕션 DB 스키마 차이 (테이블명, 컬럼명)
- **예방**: 새 SQL 작성 시 `001_init.sql`의 실제 테이블명 확인
- **확인법**: 서버 로그에서 500 응답 확인 후 SQL 쿼리 검토

### 프론트엔드 빌드 에러
- **원인**: 테스트 파일이 프로덕션 빌드에 포함, 타입 불일치
- **예방**: `npx tsc --noEmit` 반드시 실행

## Changelog 규칙
- 핫픽스도 changelog 수정이 필요함 (CI hook 체크)
- 별도 파일 생성 대신 원본 changelog에 핫픽스 PR 링크를 추가하는 것이 적절
- `index.json`에도 `hotfix` 필드 추가 가능
