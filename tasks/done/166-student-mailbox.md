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
- **(변경 2026-07-21)** 주소 생성은 **관리자 승인제**: 학생 신청(pending) → admin 승인 후 사용 가능. 반려 시 재신청. 승인 전 발송·수신 불가.
- **(확장 2026-07-21)** 멀티 메일함: 유저 개인 주소 + **회사 이메일** (회사 소유자가 등록, 동일 승인제). 메일함 화면에서 접근 가능한 메일함 선택. 이메일은 강의실 구분 없음. 이름(유저/회사)은 변경 가능하지만 이메일 주소는 승인 후 불변. 최소 3자. 금지어 목록 대폭 확장 (admin/billing/support/api/postmaster 등 90여 개).
- **(확장2 2026-07-21)** **공용 메일함**: admin이 생성(hello@ 등 — admin은 금지어 무시 가능) + 유저별 접근 권한 부여/회수(mail_address_grants, revoked 플래그). 셀렉터에 개인/회사/공용 구분. 공용 수신 시 권한자 전원 알림.
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
