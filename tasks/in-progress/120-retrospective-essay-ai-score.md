---
id: 120
title: 회고 에세이 전용 입력 + AI 작성 확률 평가기 (#119 후속)
priority: high
type: feat
branch: feat/student-milestone-dashboard
created: 2026-06-05
---

## 배경
#119 에서 회고 milestone을 자유 텍스트로 받게 만들었는데, 학생들이 LLM 으로 갈겨쓰는 걸 방지하기 위해 AI 작성 확률을 평가해주는 기능 필요.

## 요구사항
1. **회고 milestone (retrospective)** 만 전용 에세이 입력 UI
   - 큰 textarea + 글자수 카운트 + 분량 가이드 (예: 800자 이상)
2. **AI 작성 확률 평가** — 두 엔진 합산
   - **Heuristic** (Go pure function, 한국어 특화):
     - 문장 길이 표준편차 (낮으면 AI 의심)
     - 어휘 다양성 (TTR)
     - AI 특유 구문 빈도: "~을 통해", "~에 대해", "~로 인해", "결론적으로", "다음과 같다"
     - 1인칭 + 구체적 표현 ("내가", "저는", 특정 인명/시간)
     - 이모지·줄임말·반말 부재 → AI 의심
   - **LLM 보조** (cycorld llm.cycorld.com via ChatLLMClient):
     - 시스템 프롬프트로 "AI 작성 확률 0~100 + 한 줄 이유" 요청
     - JSON 응답 강제
3. **학생도 점수 봄** — 제출 폼에서 "AI 점수 확인" 버튼 → 결과 노출 + 가이드
   - 자동 반려 없음, admin이 보고 판단
4. **저장** — student_milestones 에 ai_score (0-100), ai_reasoning, ai_signals (JSON) 추가
   - 제출/수정 시 자동 평가 + 저장
5. **관리자 매트릭스** — 회고 셀에 점수 chip 노출

## TDD
- Go heuristic 단위 테스트: AI 풍 vs 사람 풍 sample 텍스트 → 점수 차이 검증
- Integration test: /milestones (retrospective 제출) → ai_score 저장 + admin matrix 노출
- LLM 호출은 fake adapter 로 deterministic

## 후속
- 점수 캐싱 (같은 텍스트 hash → 동일 점수) — cost 모니터링 후 결정
