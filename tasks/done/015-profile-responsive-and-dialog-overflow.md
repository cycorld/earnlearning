---
id: 015
title: 프로필 페이지 반응형 + DB 다이얼로그 모바일 오버플로우 수정
priority: medium
type: fix
branch: fix/profile-responsive-dialog-overflow
created: 2026-04-10
---

## 배경

스테이지 배포 후 두 가지 UX 문제 발견:

1. **PC (데스크톱) 좁음**: `ProfilePage.tsx` 가 `max-w-lg` (32rem ≈ 512px) 로
   고정되어 데스크톱에서도 모바일 폭으로 보임. 한 줄에 한 카드만 들어가서
   세로로 길게 늘어져 공간 낭비.

2. **모바일 다이얼로그 오버플로우**: 학생이 DB 를 만들면 뜨는 "접속 정보"
   다이얼로그에서 host/database/username/password 값 (특히 password 24자) 과
   `.env`/`psql` 스니펫이 좁은 폰 화면에 들어가지 않고 가로로 삐져나오면서
   복사 버튼이 화면 밖으로 밀려나 사용 불가. (스크린샷 첨부 예정)

## 목표

### 데스크톱 반응형
- `lg` 이상 화면에서 컨테이너 폭을 넓혀 카드들이 숨 쉬도록
- 가능하면 다단(grid) 레이아웃: 좌측 user 정보 + 자산 요약, 우측 내 DB + 네비게이션
- 모바일 (sm 이하) 은 기존처럼 단일 컬럼 유지

### 모바일 다이얼로그
- 긴 값(password 24자, DATABASE_URL, psql 명령) 이 다이얼로그 폭을 넘지 않도록
- 복사 버튼은 항상 화면 안에 있어야 함
- 사용성 해치지 않기:
  - 값이 잘려도 복사 버튼은 항상 동작
  - 토글로 펼쳐 보거나 가로 스크롤 등 보완책 필요
- 옵션:
  - A) 라벨 위/값 아래 (수직 stack) 로 mobile, sm: 이상은 가로
  - B) 가로 스크롤 + 복사 버튼 sticky right
  - C) 값 자체에 `select-all` + 모달 자체 폭은 viewport 제약

## 작업

### 데스크톱
- [ ] `ProfilePage.tsx`: `max-w-lg` → `max-w-lg lg:max-w-5xl xl:max-w-6xl`
  + `lg:grid lg:grid-cols-2 lg:gap-6` 로 2열 배치
  + 각 섹션의 위치 재조정 (사용자/자산 좌, DB/네비 우)

### 모바일 다이얼로그 (`UserDatabasesSection.tsx`)
- [ ] `KV` 컴포넌트:
  - flex parent 에 `min-w-0` 추가 (truncate 동작 보장)
  - 값을 `truncate` 가 아니라 `break-all` 로 변경하거나 가로 스크롤 처리
- [ ] `CopyBlock` 컴포넌트:
  - `min-w-0` 추가, `overflow-x-auto` 가 실제로 동작하게
- [ ] 다이얼로그 자체 폭: `max-w-[calc(100vw-1rem)] sm:max-w-lg`
- [ ] 복사 버튼은 `shrink-0`, sticky 효과로 항상 우측 노출

## 검증

- [ ] 데스크톱 (1920x1080): 프로필이 좌우로 펼쳐져 보임
- [ ] 태블릿 (768x1024): 단일 컬럼, 컨텐츠는 적당한 폭
- [ ] 모바일 (375x812): 기존과 동일하게 보이되 다이얼로그가 깨지지 않음
- [ ] 다이얼로그 안의 password / .env / psql 모두 복사 버튼 사용 가능
- [ ] 가로 스크롤이 다이얼로그 안에서만 발생, body 자체는 스크롤 없음
- [ ] 회귀: 빈 상태, 1개/2개/3개 카드 모두 정상

## 비-목표

- 새 기능 추가 (검색/필터/정렬 등)
- 폰트/색상 변경
- 모바일 네비 변경
