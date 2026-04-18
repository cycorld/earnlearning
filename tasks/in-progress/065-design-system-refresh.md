---
id: 065
title: 디자인 시스템 리프레시 (Ewha Green + Warm Orange + Warm Paper)
priority: medium
type: chore
branch: claude/infallible-mahavira-ac2836
created: 2026-04-18
---

## 배경
국내외 LMS 디자인 리서치 후 우리 LMS만의 색을 찾는 작업. 칸 아카데미의 친근한 톤
+ 이화여대 공식 브랜드 그린 + 게임화 창업 교육의 활기를 결합한 디자인 시스템.

## 변경 사항
**Primary**: `#005F69` Deep Dive Teal → **`#00643E` Ewha Green** (이화여대 공식 CI, Pantone 336C)
**Background**: `#F8F9FA` Cloud White → **`#F8F7F4` Warm Paper** (따뜻한 종이 톤, 눈 피로 감소)
**Highlight (신규)**: **`#FF6C00` Warm Orange** — 단일 브랜드 악센트 (코인/액션/게임성)
**Coral** `#FF6B6B`: 기존 강조색 → **destructive/손실/마이너스 수익률 전용**으로 역할 분리

## 참고
- 이화여대 공식 컬러: http://www.ewha.ac.kr/ewha/intro/ui-si04.do
- 레퍼런스 톤: Khan Academy (Green + Orange)
- 게임화 요소(3D 버튼, 스킬트리, 배지 등)는 후속 티켓으로 분리

## 영향 범위
- `frontend/src/index.css` — CSS 변수 전면 재정의 (light + dark)
- 모든 페이지에 자동 반영 (CSS variable 기반)
- `bg-coral` 클래스 유지 (Header 알림 뱃지 등) — 의미만 "손실/경고"로 재정의
