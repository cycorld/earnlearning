---
id: 159
title: 멀티 강의(코호트) 지원 — 초대 코드 입장 + 강의별 지갑·데이터 분리
priority: high
type: feat
branch: feat/159-multi-course-invite
created: 2026-07-20
---

## 배경

지난 스코입 강의 이후 새로운 강의를 진행하게 됨. 요구사항:

1. 회원가입 후 초대 코드로 특정 강의에 입장
2. 기존 유저도 새 강의에 초대 가능
3. 강의별로 지갑 등 데이터 분리 (한 유저가 여러 강의에 속해도 잔액·자산 격리)
4. 관리자는 최초 화면에서 강의실을 선택해 들어가서 해당 강의를 관리

## 검증 결과 (2026-07-20)

- 강의실 엔티티 + 6자리 조인 코드 + 강의실별 채널/게시글/과제: **이미 구현됨**
- 지갑·금융 데이터: 유저 전역 (wallets.user_id UNIQUE) — 두 번째 강의 조인 시 같은 지갑에 초기자본 중복 지급 + 잔액 혼합 **버그**
- 활성 강의실 컨텍스트, 관리자 강의실 선택 진입: 없음

## Phase 1 — 지갑 격리 + 활성 강의실 컨텍스트 (이 PR, 완료)

- `wallets` 리빌드: `UNIQUE(user_id)` → `UNIQUE(user_id, classroom_id)`. id 보존 복사로 transactions FK 유효, 구 테이블 `wallets_legacy_159` 보존, 멤버십 기반 classroom_id 백필.
- `users.active_classroom_id` 추가 + 백필. `FindByUserID` = 활성 강의실 지갑 해석 (fallback: 미배정 0 → 최소 classroom_id) — 기존 usecase 13곳 시그니처 무변경.
- `JoinClassroom`: (user, classroom) 지갑 확보(미배정 지갑 귀속 or 신규) + 최초 귀속 시에만 초기자본 지급 + 활성 전환.
- `POST /classrooms/:id/activate` (멤버만), 랭킹·자산 cash·관리자 대시보드 강의실 스코프.

### Phase 1 알려진 한계 (Phase 2 대상)
- 포스트/댓글/좋아요 보상, 외주, 투자, 거래소, 대출, 지원금 등 도메인 행위는 **활성 강의실 지갑**으로 입금 — 행위가 일어난 강의실과 다를 수 있음.
- `GetAssetBreakdown`의 주식가치/회사지분/부채는 아직 전역 (companies 등 미스코프).

## Phase 2 — 금융 도메인 강의실 스코핑 (다음 PR)

companies·investments·stock_*·loans·freelance_jobs·grants 에 classroom_id 추가 + 조회/보상 경로를 행위 발생 강의실로 스코프.

## Phase 3 — UX (다음 PR)

- 가입 승인 후 온보딩에서 초대 코드 입력
- 학생 강의실 스위처 UI
- 관리자 최초 화면 강의실 선택 진입
