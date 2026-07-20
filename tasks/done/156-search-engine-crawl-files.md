---
id: 156
title: 검색엔진 수집용 robots.txt 및 sitemap.xml 추가
priority: high
type: chore
branch: chore/156-search-engine-crawl-files
created: 2026-07-18
---

## 목표
- `/robots.txt`가 SPA HTML이 아닌 유효한 robots 문서로 응답
- `/sitemap.xml`이 유효한 XML sitemap으로 응답
- 공개 루트만 색인 대상으로 선언하고 API·관리·인증·학생 데이터 경로는 제외

## 검증
- 정적 파일 계약 테스트
- 프론트엔드 테스트·빌드
- 스테이지 및 프로덕션 HTTP 응답과 Content-Type 확인
