---
id: 034
title: EarnLearning 전체 이용가이드 (Notion, 스크린샷 포함)
priority: medium
type: chore
branch: docs/user-guide-complete
created: 2026-04-18
---

## 목표
언러닝 Notion 워크스페이스의 `📢 언러닝 가이드` 아래에 **언러닝 시작 가이드(진입 페이지)** + 기능별 상세 매뉴얼 7~8개를 추가해, 학생이 처음 접속했을 때부터 실제 운영까지 필요한 모든 흐름을 한 곳에서 볼 수 있게 한다.

## 구조
- **언러닝 시작 가이드** (진입 페이지, 신규): 개요 · 온보딩 흐름 · 주요 개념 · 목차
- 기능별 상세 매뉴얼 (각 독립 페이지):
  - 홈·피드·공지 (신규)
  - 회사 설립 & 기본 경영 (신규)
  - 주주총회 — 일반 안건 (신규, 청산 가이드 링크)
  - 프리랜스 마켓 (신규)
  - 대출 시스템 (신규)
  - 그랜트/지원사업 (신규)
  - 공시 (신규)
  - 알림 & 프로필 (신규)
- 기존 가이드 재활용:
  - 투자 라운드 완전 가이드
  - 개인 계좌 vs 법인 계좌 완전 가이드
  - 회사 청산 완전 가이드

## 작업
1. 프론트엔드 라우트 전수 조사 → 스크린샷 목록 확정
2. Playwright로 Stage에서 스크린샷 일괄 촬영 (admin/학생 2역할)
3. 스크린샷을 `docs/manuals/<topic>/images/` 에 commit (GitHub raw URL 임베드용)
4. Notion 페이지 작성: 진입 페이지 1개 + 세부 매뉴얼 7~8개
5. 메모리(reference_notion.md) 업데이트: 새 가이드 URL들 추가

## 주의
- 각 매뉴얼은 기존 3개 가이드와 동일한 포맷(callout, table, image, FAQ)
- 이미지는 `https://raw.githubusercontent.com/cycorld/earnlearning/main/...` URL
- 청산·투자·법인 가이드는 새 페이지에 mention-page로 링크만
- **스크린샷 촬영 중 발견되는 에러(404, 500, UI 깨짐, 한글 깨짐, 기능 누락 등)는 각각 `tasks/backlog/` 에 별도 티켓 생성**. 본 티켓에서는 수정하지 않고 문서화만.
