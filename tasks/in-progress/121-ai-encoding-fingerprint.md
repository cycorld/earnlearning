---
id: 121
title: AI 생성물에 자주 포함되는 비-인간 인코딩 검출 (#120 후속)
priority: medium
type: feat
branch: feat/student-milestone-dashboard
created: 2026-06-05
---

## 배경
#120 의 heuristic 점수에 **사람은 거의 입력하지 않는 유니코드 문자** 검출을 추가.
사용자 지적: "인간은 사용하지 않는 인코딩이 AI 생성물에 포함되는 경우도 있어"

## 검출 대상
| 카테고리 | 코드포인트 | 가중치 | 근거 |
|---------|-----------|--------|------|
| Zero-width chars | U+200B/C/D, U+FEFF, U+2060 | +30 | 사람이 키보드로 입력 불가, 거의 100% AI evidence |
| Mathematical alphanumeric | U+1D400~U+1D7FF (𝐀, 𝒂 등) | +25 | 특수 폰트 변환 필요 — 학생 입력 불가 |
| Narrow/non-breaking space | U+00A0, U+202F | +15 | 일반 keyboard 미지원 — typesetting tool/AI 출처 |
| Smart quotes 한 묶음 | U+2018/19/1C/1D (" " ' ') | +10 (조건부) | 사람도 일부 씀; 5개 이상 빈도일 때만 |
| Em/En dash | U+2014, U+2013 | +10 (조건부) | 한국어 글에서 1000자당 3개+ 일 때만 |

각 항목은 시그널로 노출 + 학생용 hint ("AI 흔적: 폭 0 문자가 N개 포함됨").

## TDD
- detectSuspiciousChars 단위 테스트 (각 카테고리)
- 사람이 손으로 쓴 평범한 한국어 → 시그널 0
- ChatGPT 풍 sample (em dash + smart quotes + zero-width) → 모든 시그널 발화
