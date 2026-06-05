---
id: 119
title: 학생 4대 평가지표 제출 대시보드 + 관리자 승인 기능
priority: high
type: feat
branch: feat/student-milestone-dashboard
created: 2026-06-05
---

## 배경
syllabus-actual.md "평가 기준 (절대평가)" 4가지 항목으로 그룹(A/B/C/D) 결정:

| # | 평가지표 | 기한 |
|---|---------|------|
| 1 | 첫 번째 MVP 배포 | 7주차 |
| 2 | 두 번째 MVP 배포 | 12주차 |
| 3 | 사업계획서 제출 | 14주차 |
| 4 | 한 학기 회고 발표 | 보강 1주차 |

학생별 4개 항목 진행 현황을 한눈에 보고, 일부는 기존 데이터에서 자동 집계, 나머지는 직접 제출 + 관리자 승인.

## 요구사항
1. **자동 집계 소스**
   - 1·2차 MVP: 회사(`companies.service_url`) 의 다중 URL (#115) 에서 유효 URL 추출
   - 1·2차 MVP: 정부과제 응모 (`grant_applications`) 에 포함된 URL — TBD (proposal 텍스트 파싱 vs. 신규 컬럼)
2. **URL 유효성 필터**
   - 인정: `*.vercel.app`, 자체 도메인 (root domain that's not in 연습용 deny list)
   - 제외 (연습용): `aistudio.google.com`, `ai.studio`, `claude.ai`, `chatgpt.com`, `gemini.google.com`, `localhost`, `127.0.0.1`
3. **수동 제출 채널**
   - 사업계획서: 학생이 파일 첨부 or 링크 + 코멘트 제출
   - 회고 발표: 학생이 자료 링크 or 텍스트 제출
4. **관리자 승인**
   - 학생별 4개 항목 카드 → 관리자가 approve/reject + 코멘트
   - 상태: `pending` | `approved` | `rejected`
   - 승인 시 알림 발송
5. **대시보드 뷰**
   - 학생용: 본인의 4개 진행률 (X/4) + 미달 항목 안내
   - 관리자용: 전체 학생 매트릭스 + 그룹(A/B/C/D) 자동 분류

## TDD 계획
- backend integration test: 자동 집계 (vercel.app 인정 / aistudio 제외) + 승인 흐름
- frontend vitest: URL 필터 + 그룹 분류 로직

## 후속
- syllabus-actual.md week 7/12/14/보강1 의 deadline 과 연동된 강제 마감 (별도 티켓)
