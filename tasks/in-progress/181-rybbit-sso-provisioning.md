---
id: 181
title: EarnLearning SSO 기반 Rybbit 자동 연동
priority: high
type: feat
branch: feat/earnlearning-rybbit-sso
created: 2026-07-27
---

부트캠프 승인 학생이 EarnLearning 계정으로 캠프 전용 Rybbit에 로그인하도록 한다.

- 회사 설립 후 서비스 URL 등록
- 회사 화면의 `Rybbit 연동하기`로 사이트 자동 provisioning
- 사용자 역할은 Member로 고정하고 본인 회사 사이트만 접근
- OAuth Authorization Code + PKCE 및 불변 user ID 연결
- 공개 가입 차단, fail-closed 권한, 캠프 소속 검증
- Stage 및 학생 A/B 교차 접근 회귀 테스트
- 비밀값과 학생 개인정보는 저장소에 포함하지 않음

## 설계 결정 (2026-07-27, EarnLearning 측 1차 구현)

**범위**: 이 브랜치는 EarnLearning 쪽 서비스 레지스트리 + URL 검증 + 연동 상태 관리 + provisioning 포트까지. Rybbit 인스턴스 쪽 OIDC 클라이언트 설정·Member 역할 강제는 외부 작업(가정 명시).

- **데이터**: 신규 테이블 `company_services` (additive; 레거시 `companies.service_url` 쉼표 방식 #115 불변). `UNIQUE(company_id, normalized_url)`.
- **상태 모델**: `validation_status ∈ {unvalidated, valid, invalid}` × `rybbit_status ∈ {not_connected, connected, needs_reconnect}`. URL 수정 → 검증 리셋 + connected→needs_reconnect(드리프트).
- **검증**: HTTPS 전용, credentials/fragment 거부, 정규화(host 소문자·:443 제거·꼬리 슬래시 제거). SSRF 가드: IP 리터럴/DNS 결과 전부 공인 IP 필수(사설·루프백·링크로컬·CGNAT·예약 대역 거부), dial 시점 IP 재검사(리바인딩 방어), 타임아웃 5s, 리다이렉트 ≤3회·매 홉 재검증, 최종 2xx만 valid. `checked_at`/`detail` 서버 기록, 클라이언트 주장 불신.
- **연동 게이트**: `connect_ready = valid && checked_at < 24h` (TTL 상수). 초과 시 `VALIDATION_STALE`.
- **권한**: 기존 회사 정책 그대로 — 소유자 전용(`OwnerID != userID → NOT_OWNER`), admin 우회 없음(기존 명시 정책에 admin 회사 수정 권한 없음). connect는 추가로 활성 강의실 == company.ClassroomID 요구. ApprovedOnly + read/write:company 스코프 재사용.
- **멱등·fail-closed**: connected+site_id 존재 시 재호출은 provisioner 호출 없이 200. 실패 시 상태 무변경(성공 후에만 site_id/status 기록). 미설정 시 503 `RYBBIT_NOT_CONFIGURED`.
- ~~**Rybbit API 계약 가정** (미검증): `POST /api/admin/sites` Bearer 토큰~~ → 2차에서 실제 계약으로 교체 (아래).
- **삭제**: `DELETE /companies/:id/services/:serviceId` 소유자 전용. 로컬 레코드만 삭제, Rybbit 사이트는 외부 파괴적 작업이라 호출하지 않음(주석 명시). 다음 연동 클릭의 재조정 집합에서 자연히 빠지면서 접근이 회수된다.
- **userinfo 캠프 자격 클레임**: `/api/oauth/userinfo` 응답에 `role, status, approved, active_classroom_id, camp_eligible` 추가 — Rybbit SSO 가 fail-closed 로 로그인 게이트할 근거. 조회 실패 시 eligible=false.

## 계약 확정 (2026-07-27 2차, Rybbit 커밋 6cc9ee30 / EARNLEARNING_SSO.md 기준)

- **프로비저닝 엔드포인트**: `POST {RYBBIT_API_BASE_URL}/api/earnlearning/provision`. 인증은 HMAC 서명만: `x-earnlearning-timestamp`(unix 초) + `x-earnlearning-signature` = `hex(HMAC-SHA256(secret, "<ts>.<raw_body>"))`. 전송 바이트 그대로 서명(재직렬화 금지), Origin 헤더 미전송, 요청/응답 스키마 strict. 계약 문서의 서명 재현 벡터를 유닛 테스트로 고정.
- **매핑**: `company.id = <companies.id 십진>`, `site.key = service:<company_services.id>` (AUTOINCREMENT PK → 영구 안정), `site.domain = 검증된 도메인`, `user.id = OAuth userinfo sub = user.OAuthSubject(소유자 id)` (완전 일치, 계정 연결 키), `user.role = "member"` 고정.
- **grants 재조정 의미론**: Rybbit 은 `grants.siteKeys` 를 누적이 아니라 재조정으로 처리 → 연동 클릭마다 "회사의 이미 프로비저닝된 전 서비스(RybbitSiteID 보유, needs_reconnect 포함) ∪ 클릭 서비스" 전체 집합을 id 오름차순으로 전송. 부분 전송 = 기존 접근 회수 사고. 한 번도 연동 안 된 서비스는 Rybbit 에 키가 없어 포함 시 전체 403 → 제외. 교차 회사 키는 회사 범위 조회로 원천 차단.
- **응답 검증**: 200 strict 파싱(모르는 필드 거부) + `siteId ∈ grantedSiteIds` + `len(grantedSiteIds) == 요청 키 수` 확인 후에만 site id 저장. 401/403/404/409/5xx 는 상태 코드+고정 힌트만 노출(본문·시크릿 비노출), 로컬 상태 무변경.
- **userinfo 계약 필드 추가**: `sub`(안정 문자열, user.OAuthSubject), `active`(= 승인 상태) — Rybbit 로그인 필수 클레임. `camp_eligible = approved && role==student && active_classroom != 0` (admin 은 명시 정책상 캠프 자격 없음).
- **ConnectRybbit 자격 게이트**: 소유자 + role student + status approved(DB 재조회, JWT 스테일 무관) + 활성 강의실 == 회사 강의실 + 24h 내 검증.
- **설정**: `RYBBIT_ADMIN_TOKEN` → `RYBBIT_PROVISION_SECRET` (32자 미만/부분 설정 = fail-closed Noop + 경고 로그, 시크릿 비로깅).
