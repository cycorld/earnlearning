# 백업 및 복원 가이드

## 개요

프로덕션 DB(SQLite)와 사용자 업로드 파일을 매일 자동으로 S3에 백업합니다.

## 인프라

| 항목 | 값 |
|------|------|
| S3 버킷 | `s3://earnlearning-backups` (ap-northeast-2) |
| IAM 유저 | `earnlearning-backup` (S3 전용 최소 권한) |
| 백업 스크립트 | `/home/ubuntu/lms/deploy/backup.sh` |
| 자동 실행 | 매일 KST 04:00 (root crontab) |
| DB 보존 | 90일 (S3 Lifecycle 자동 삭제) |
| Uploads 보존 | 무기한 (증분 동기화) |
| 로그 | `/var/log/earnlearning-backup.log` |

## S3 구조

```
s3://earnlearning-backups/
├── db/
│   ├── earnlearning-2026-04-02_1423.db.gz    ← 날짜별 DB 스냅샷
│   ├── earnlearning-2026-04-03_0400.db.gz
│   └── ...                                    ← 90일 후 자동 삭제
└── uploads/
    ├── f58bae97-...-.png                      ← 사용자 업로드 파일
    ├── 688535e8-...-.docx
    └── ...                                    ← 삭제 안 함
```

## 수동 백업

```bash
ssh earnlearning
sudo /home/ubuntu/lms/deploy/backup.sh
```

## 복원

### DB 복원

```bash
ssh earnlearning

# 1. 백업 목록 확인
aws s3 ls s3://earnlearning-backups/db/ --human-readable

# 2. 원하는 백업 다운로드
aws s3 cp s3://earnlearning-backups/db/earnlearning-2026-04-02_1423.db.gz /tmp/

# 3. 압축 해제
gunzip /tmp/earnlearning-2026-04-02_1423.db.gz

# 4. 서비스 중지 (데이터 무결성)
cd /home/ubuntu/lms/deploy
./deploy.sh status                    # 현재 active slot 확인
docker compose -f docker-compose-blue.yml stop    # 또는 green

# 5. DB 교체
sudo cp /tmp/earnlearning-2026-04-02_1423.db \
  /var/lib/docker/volumes/earnlearning-prod_prod_db/_data/earnlearning.db

# 6. 서비스 재시작
docker compose -f docker-compose-blue.yml up -d   # 또는 green
```

### Uploads 복원

```bash
ssh earnlearning

# 전체 복원 (S3 → 서버)
sudo aws s3 sync s3://earnlearning-backups/uploads/ \
  /var/lib/docker/volumes/earnlearning-prod_prod_uploads/_data/

# 특정 파일만 복원
sudo aws s3 cp s3://earnlearning-backups/uploads/uuid.png \
  /var/lib/docker/volumes/earnlearning-prod_prod_uploads/_data/
```

### 전체 재해 복구 (EC2 새로 생성한 경우)

```bash
# 1. EC2에 Docker, docker-compose 설치
# 2. 리포지토리 클론
git clone git@github.com:cycorld/earnlearning.git /home/ubuntu/lms

# 3. AWS CLI 설치 + credentials 설정
curl -s "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o /tmp/awscliv2.zip
cd /tmp && unzip awscliv2.zip && sudo ./aws/install
# ~/.aws/credentials, ~/.aws/config 설정

# 4. DB 복원
aws s3 ls s3://earnlearning-backups/db/ | tail -1       # 최신 백업 확인
aws s3 cp s3://earnlearning-backups/db/최신백업.db.gz /tmp/
gunzip /tmp/최신백업.db.gz

# 5. Docker 볼륨 생성 + 데이터 배치
docker volume create earnlearning-prod_prod_db
docker volume create earnlearning-prod_prod_uploads
sudo cp /tmp/최신백업.db /var/lib/docker/volumes/earnlearning-prod_prod_db/_data/earnlearning.db
sudo aws s3 sync s3://earnlearning-backups/uploads/ \
  /var/lib/docker/volumes/earnlearning-prod_prod_uploads/_data/

# 6. 배포
cd /home/ubuntu/lms/deploy
IMAGE_TAG=latest ./deploy.sh stage
IMAGE_TAG=latest ./deploy.sh prod

# 7. crontab 복원
echo "0 19 * * * /home/ubuntu/lms/deploy/backup.sh >> /var/log/earnlearning-backup.log 2>&1" | sudo crontab -
```

## 모니터링

```bash
# 최근 백업 로그 확인
ssh earnlearning "tail -20 /var/log/earnlearning-backup.log"

# S3 최신 백업 확인
aws --profile k s3 ls s3://earnlearning-backups/db/ | tail -3

# 백업 용량 확인
aws --profile k s3 ls s3://earnlearning-backups/ --recursive --summarize | tail -2
```

## crontab 관리

```bash
# 확인
ssh earnlearning "sudo crontab -l"

# 수정
ssh earnlearning "sudo crontab -e"

# 현재 설정: 매일 UTC 19:00 (KST 04:00)
# 0 19 * * * /home/ubuntu/lms/deploy/backup.sh >> /var/log/earnlearning-backup.log 2>&1
```
