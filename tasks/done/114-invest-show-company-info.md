---
id: 114
title: 투자 페이지에서 회사 정보 노출 (대표자·소개·URL)
priority: medium
type: feat
branch: feat/invest-show-company-info
created: 2026-05-08
---

## 배경
투자 페이지(`/invest`, `/invest/:id`) 에서 **회사에 대한 정보가 거의 안 보임**. 학생이 어디 회사에 투자할지 결정해야 하는데 라운드 정보(target/percent/price)만 보이고 회사 소개·대표자·서비스 URL 가시성이 부족.

## 작업
### `/invest/:id` (InvestDetailPage)
- 회사 description (사용자 markdown) 을 **전문 그대로** 렌더
- 대표자 이름·프로필 링크
- service_url 클릭 가능한 링크
- (이미 있으면) 표시 위치만 정리

### `/invest` (InvestPage list)
- 카드별로 회사명 + **대표자 이름** + service_url(있으면 외부 링크 아이콘)
- 회사 소개 1~2줄 (description 첫 N자 truncate)

## 백엔드 (필요시)
- 투자 라운드 목록·상세 API 응답에 company.description / owner_name / service_url 포함되는지 확인
- 누락 시 join 추가

## 테스트
- Playwright 또는 vitest 로 라운드 카드에 회사 정보 표시되는지 회귀
- 스모크 통과 필수

## 미포함
- 회사 description 편집 UI 변경 (별도 티켓)
- 투자 알고리즘 변경
