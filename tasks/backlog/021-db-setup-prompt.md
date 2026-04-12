---
id: 021
title: PostgreSQL 프로젝트 생성 시 DB 연동 프롬프트 제공
priority: medium
type: feat
branch: feat/db-setup-prompt
created: 2026-04-12
---

## 설명
학생이 프로필에서 PostgreSQL 프로젝트를 생성하면, env 정보(DB_HOST, DB_PORT, DB_NAME 등) 아래에 DB 세팅 관련 프롬프트를 추가로 보여준다.

## 목적
학생들이 해당 프롬프트를 복붙하면 자기 서비스(바이브코딩 앱)에 DB를 쉽게 연동할 수 있도록 한다.

## 작업 내용
- PostgreSQL 프로젝트 생성 완료 화면에서 env 정보 아래에 "DB 연동 프롬프트" 섹션 추가
- 프롬프트 내용: 사용자의 DB 접속 정보를 포함한 연동 가이드 (연결 문자열, ORM 설정 등)
- 복사 버튼으로 원클릭 복사 가능
- 프레임워크별 예시 (Node.js/Python/Go 등) 포함 고려
