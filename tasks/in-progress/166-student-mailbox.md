---
id: 166
title: 학생별 이메일 수신함 — Cloudflare Email Routing + 앱 내 메일함 (수신/답장)
priority: high
type: feat
branch: feat/166-student-mailbox
created: 2026-07-20
---

## 목표
earnlearning.com 도메인으로 학생별 개인 이메일 주소를 만들고, 앱 안에 메일함(수신 + SES 답장)을 제공.

## 아키텍처
```
수신: MX → Cloudflare Email Routing (catch-all) → Email Worker (postal-mime 파싱)
      → POST /api/mail/inbound (공유 시크릿) → SQLite 저장 → 앱 메일함 UI + 알림
발송(답장): 메일함 UI → 기존 SES 경로 (from = <로컬파트>@earnlearning.com, In-Reply-To 스레딩)
```

## 스코프 (사용자 확정)
- 주소 체계: 학생별 개인 주소 — 학생이 메일함 첫 진입 시 로컬파트 클레임 (영문소문자+숫자, 유니크)
- 권한: 본인 메일만 열람, admin은 전체 열람
- 수신 + 답장 모두 지원
- 알림 연동: 새 메일 수신 시 CreateNotification (getReferencePath/getNotifIcon 매핑 추가 필수)

## 구성 요소
1. Cloudflare: Email Routing 활성화 (MX/SPF 자동), catch-all → Email Worker (`workers/email-inbound/`)
2. 백엔드: `mail_addresses`·`emails` 마이그레이션(ALTER/CREATE, DEFAULT 필수), 웹훅, 메일함 CRUD API, SES 답장
3. 프론트: 메일함 페이지(목록/읽기/답장/새 메일), 주소 클레임 UI, 더보기 메뉴, 알림 매핑

## 주의
- DNS 현재 MX 없음 (수신 충돌 없음). SES 발송은 DKIM CNAME이라 MX와 무관.
- 첨부: private uploads 재사용 (인증 다운로드).
- 공개 repo — 시크릿은 env로만.
