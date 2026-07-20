---
id: 168
title: 메일 발송 IAM AccessDenied 폴백 + 발송 실패 로깅
priority: high
type: fix
branch: fix/168-mail-send-fallback
created: 2026-07-21
---

## 증상 (prod 실메일 E2E에서 발견)
`POST /api/mail/send` → 502. IAM 유저 `earnlearning-ses`가 `identity/noreply@earnlearning.com`만 허용되어
학생 주소 From 발송이 `AccessDeniedException`으로 거부됨. `isUnverifiedIdentity`가
"Email address is not verified"/"MessageRejected"만 매칭해 폴백 미발동 → 즉시 502.
발송 실패가 로그에 안 남아 원인 추적에 EC2 aws cli 재현이 필요했음.

## 수정
1. 폴백 트리거 확대: AccessDenied/not authorized 도 "From 신원 사용 불가"로 간주 →
   설정 From(noreply@) + Reply-To=학생주소 폴백.
2. 발송 실패 시 에러 로그 (usecase Send).

## 선택 업그레이드 (사용자 IAM 콘솔 작업 필요)
IAM 정책에 `ses:SendEmail` resource `arn:aws:ses:ap-northeast-2:...:identity/earnlearning.com` 추가
+ SES에 earnlearning.com 도메인 identity 검증 → 진짜 학생주소 From 발송.
