---
id: 090
title: 챗봇 자주 묻는 질문 캐싱 — system prompt 무거운 짧은 인사 즉시 응답
priority: low
type: feat
branch: feat/chatbot-faq-cache
created: 2026-04-19
---

## 배경
"안녕", "고마워", "잘가" 같은 인사조차 system prompt 800 토큰 + 도구 spec 수백
토큰을 LLM 에 매번 보냄. LLM 슬롯 1개 점유 + 비용 발생. 부하 테스트에서 인사
하나에 4–10초 걸리는 게 확인됨.

## 스코프
1. **FAQ 매칭** — Ask 진입 시 message 를 normalize 하고 FAQ 사전과 비교
2. **사전** — `chat_faq` 테이블 또는 hardcoded JSON
   - `안녕` / `안녕하세요` / `hi` → "안녕하세요! 이화여대 LMS 조교입니다 😊..."
   - `고마워` / `감사합니다` → "도움이 됐다면 다행이에요!"
   - `잘가` / `bye` → "다음에 또 봐요!"
3. **즉시 응답** — LLM 호출 안 하고 바로 SSE text_delta + done event
4. **DB 기록** — chat_messages 에 저장 (assistant content + model="faq")

## 확인 필요
- 매칭 너무 공격적이면 진짜 질문도 가로챌 수 있음 ("안녕하세요 회사가 뭐예요?" 같은)
- → exact match 또는 짧은 (< 6자) 만 매칭, 부분 매칭은 안 함
- 다국어 / 변형 (안녕!, 안녕~~, 안녕!! 등) 정규화

## 효과 추정
- 인사 / 감사 비율 ~10% 가정 → 슬롯 점유 10% 감소
- 비용 동일 비율 절약
- 인사 응답 latency 5초 → 50ms

## 후속
- LLM-generated 캐시 (자주 받는 질문 자동 학습)
- Notion 가이드 일부도 캐시
