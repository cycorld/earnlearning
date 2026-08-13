---
id: 201
title: canonical 관리자의 강의실 Analytics 프로젝트 접근
priority: high
type: feat
branch: feat/admin-all-analytics-projects
created: 2026-08-13
---
서버가 DB에서 승인된 admin role, canonical 관리자 이메일, 활성 강의실을 검증한 뒤 해당 강의실 회사의 기존 Rybbit 사이트를 회사별로 재조정한다. 일반 관리자와 학생 권한은 확대하지 않으며 stable OAuth subject를 identity key로 사용한다.

보안: email 단독 account linking 금지, 회사별 organization 경계 유지, secret/학생 개인정보 공개 저장소 기록 금지.
