# EarnLearning LMS

## 프로젝트 개요
이화여자대학교 "스타트업을 위한 코딩입문" 강의용 게임화 창업 교육 LMS.

## 기술 스택
- **Backend**: Go (Echo) + SQLite (Docker volume persistent)
- **Frontend**: Vite + React 18 + TypeScript + Tailwind CSS + shadcn/ui
- **Realtime**: WebSocket + Web Push (VAPID)
- **PWA**: Vite PWA Plugin (홈 화면 설치, 오프라인 캐시, 웹 푸시)
- **Auth**: JWT (이메일 회원가입 + Admin 승인제)
- **Deploy**: Docker + Nginx (SQLite는 volume 마운트로 영속화)

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

## 커밋 규칙
- 매 프롬프트 작업 완료 시 반드시 커밋한다.
- 커밋 메시지 형식:
  ```
  [작업내용 요약 타이틀]

  prompt: [사용자가 입력한 원본 프롬프트]

  - [작업 내역 1]
  - [작업 내역 2]
  - ...
  ```
