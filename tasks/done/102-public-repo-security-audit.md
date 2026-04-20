---
id: 102
title: 공개 GitHub 저장소 민감정보 점검 — 키, 비밀번호, 학생 PII
priority: high
type: chore
branch: chore/public-repo-security-audit
created: 2026-04-20
---

## 배경
`github.com/cycorld/earnlearning` 가 **public 저장소**.
이전 작업에서 키 노출 사고 (#076) 발생 이력 있음.
프로덕션 학생 데이터(이름·이메일·학번) 도 점차 늘어나고 있어 정기 점검 필요.

## 점검 항목
1. **API 키 / 시크릿**
   - LLM_ADMIN_API_KEY, NOTION_INTEGRATION_TOKEN, JWT_SECRET, VAPID_PRIVATE_KEY
   - `sk-`, `Bearer `, AWS access keys, GitHub tokens, OpenAI keys
2. **`.env*` 파일**
   - `.env`, `.env.local`, `.env.production` 등 commit 여부
3. **DB 덤프 / 백업**
   - `*.sql`, `*.db`, `*.sqlite` 파일
4. **학생 개인정보 (PII)**
   - 실명, 이메일 (cycorld.com 외), 학번, 전화번호
   - 단, syllabus·강의자료 내 강사 본인 정보는 제외
5. **PR/Issue 본문**
   - 첨부된 토큰·로그·대화 내역 등

## 도구
- `git log --all -p` 패턴 grep
- `gh pr list --state all` + body 검사
- `gitleaks` 또는 `trufflehog` 가능하면 활용

## 발견 시
- 즉시 키 회전 + 커밋 rewrite 권한 확인 (force push 필요 시 사용자에 reconfirm)
- 학생 PII 는 case-by-case 판단 (LMS 운영 컨텍스트 vs 외부 노출)

## 산출물
- 점검 리포트 (이 티켓에 결과 기록)
- 필요한 fix 발견 시 별도 후속 티켓
