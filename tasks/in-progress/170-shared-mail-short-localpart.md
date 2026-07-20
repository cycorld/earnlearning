---
id: 170
title: 공용 메일함 admin 생성 시 2자 로컬파트 허용 (01~15 팀 메일)
priority: high
type: feat
branch: feat/170-shared-mail-short-localpart
created: 2026-07-21
---

## 배경
부트캠프 팀 메일 `01@earnlearning.com` ~ `15@` 생성 요청. 현재 형식 검증이 최소 3자라 admin 공용 생성도 400.

## 수정
- admin 공용 메일함 생성 경로(ValidateLocalPartFormat)만 최소 2자 허용.
- 학생/회사 신청 경로는 기존 3자 유지 (사용자 스펙).

## 완료 기준
- admin이 "01" 생성 가능, 학생이 "01" 신청은 여전히 400.
- 회귀 테스트 포함.
