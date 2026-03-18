---
id: 1
title: AWS SES 이메일 알림 시스템
priority: high
type: feat
created: 2026-03-16
updated: 2026-03-16
---

## 설명
AWS SES로 earnlearning.com 도메인 이메일 발송 시스템 구축.
기존 Push 알림과 함께 이메일로도 알림 발송. 사용자별 on/off 설정.

## 완료 내역
- SES 도메인 인증 (DKIM, SPF, DMARC)
- SES 전용 IAM 유저 (earnlearning-ses)
- backend: SES 발송 서비스 + HTML 템플릿
- frontend: 프로필 페이지 이메일 알림 토글
- PR #13 머지
