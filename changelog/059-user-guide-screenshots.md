---
slug: 059-user-guide-screenshots
title: 전체 이용가이드용 스크린샷 추가 — Notion 매뉴얼 임베드 자산
date: 2026-04-18
tags: [docs, 매뉴얼, 스크린샷, Notion]
---

# 전체 이용가이드용 스크린샷 추가

## 왜 필요했는가

EarnLearning은 학생 대상 게임화 창업 교육 플랫폼입니다. 이미 3개의 Notion 가이드(투자·법인·청산)가 있었지만, 학생이 **처음 접속했을 때** 어디부터 봐야 할지, 그리고 핵심 기능 7~8개(피드/마켓/회사/대출/그랜트 등)를 어떻게 쓰는지는 문서가 없었어요.

사용자 요청:

> "우리 서비스 전체 이용가이드(스크린샷 포함) 작성해줄 수 있어?"

전체 이용가이드를 Notion에 구축하려면 먼저 **스크린샷 자산**이 repo에 있어야 합니다 (Notion API는 외부 URL만 허용 → github raw URL 사용).

## 무엇을 했는가

### Stage에서 Playwright로 일괄 촬영

- 사용자 `user 43 (엔트로피패러독스 owner)` 으로 impersonate
- 하나의 Playwright 스크립트로 11개 섹션 · 25장 스크린샷 촬영
- 촬영 중 발견되는 에러는 `issues.json` 에 자동 기록 (이번에는 0건, post detail selector 매칭만 놓침)

### 스크린샷 구성 (`docs/manuals/user-guide/<section>/`)

| 섹션 | 장수 | 용도 |
|------|------|------|
| onboarding | 3 | 시작 가이드 — 로그인/가입/피드 홈 |
| feed | 2 | 피드 가이드 — 리스트, 글 작성 |
| wallet | 3 | 지갑 가이드 — 메인, 송금, 거래내역 |
| market | 3 | 마켓 가이드 — 리스트, 상세, 신규 |
| company | 6 | 회사 가이드 — 리스트/상세/지갑/명함/신규/공시 작성 |
| invest | 1 | 투자 가이드 — 라운드 리스트 |
| grant | 2 | 그랜트 가이드 — 리스트, 상세 |
| bank | 2 | 대출 가이드 — 메인, 신청 |
| proposal | 1 | 주주총회 가이드 — 일반 안건 다이얼로그 |
| notifications | 1 | 알림 페이지 |
| profile | 1 | 프로필 페이지 |

### 이어질 작업 (별도 PR/task)

- Notion `📢 언러닝 가이드` 아래에 진입 페이지 1개 + 세부 매뉴얼 8개 작성
- 각 페이지에서 `https://raw.githubusercontent.com/cycorld/earnlearning/main/docs/manuals/user-guide/<section>/<file>.png` 로 임베드

## 배운 점

**스크린샷 자동화는 가이드 지속성에 결정적**입니다. 손으로 찍으면 나중에 UI가 바뀌었을 때 "다시 찍기 귀찮아서" 매뉴얼이 낡아버려요. 스크립트 한 번 만들어두면 `node script.mjs` 한 방으로 모든 스크린샷이 최신화되고, CI에 올리면 자동 리프레시도 가능합니다.

**매뉴얼 자산도 코드 저장소 안에**. 이미지를 외부 호스팅(imgur 등)에 두면 정책 변경/계정 정지 등 위험이 있지만, repo에 commit해두면 수명이 훨씬 깁니다. GitHub raw URL은 수년간 안정적으로 유지돼요.

## 사용한 프롬프트

> 우리 서비스 전체 이용가이드(스크린샷 포함) 작성해줄 수 있어? → C로 진행해줘~ → 스크린샷 촬영중 발견되는 에러는 별도 티켓으로 만들어줘.
