---
id: 128
title: 이메일 기반 비밀번호 찾기 (forgot/reset password)
priority: high
type: feat
branch: feat/128-password-reset
created: 2026-06-12
---

## 배경
서비스에 비밀번호 찾기 기능이 없음. 비밀번호를 잊은 사용자는 복구 수단이 전무 (admin도 비번 설정 API 없음).

## 작업 내용
- [x] DB: `password_reset_tokens` 테이블 (CREATE TABLE IF NOT EXISTS, sqlite.go)
- [x] Backend: `POST /api/auth/forgot-password` — 토큰 발급 + SES 이메일 발송 (이메일 존재 여부 비노출)
- [x] Backend: `POST /api/auth/reset-password` — 토큰 검증(1회용, 1시간 TTL) + bcrypt 재해싱
- [x] SES 비활성(dev) 시 reset URL 서버 로그 fallback
- [x] Frontend: `/forgot-password`, `/reset-password` 페이지 + 로그인 페이지 링크
- [x] 통합 테스트 (TDD): 전체 플로우, 토큰 재사용/만료/위조, 약한 비번
