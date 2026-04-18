---
id: 077
title: 챗봇 긴 context + reasoning 체인 타임아웃 완화
priority: medium
type: fix
branch: fix/chatbot-long-context
created: 2026-04-19
---

## 증상
복잡한 개발 질문에서 `context7_search → context7_docs → ...` 4-hop 이 지나고 최종
응답 합성 단계에서 ~24s 후 500. Qwen reasoning 이 누적된 context (5k+ tokens + tool
responses) 에서 시간 초과.

## 원인
- maxToolHops=6 이지만 실제 문제는 **각 hop 마다 context 가 누적**
- Qwen 35B 의 reasoning 모드 decode 속도 100~145 tok/s → 긴 prompt 처리에 시간 필요
- Cloudflare edge timeout (100s) 은 여유 있지만 Qwen 내부 slot timeout 이 있는 듯

## 가능한 대응
1. **도구 응답 요약 필수화**: context7_docs `tokens=1000` (현 3000 → 축소)
2. **maxToolHops 4 로 되돌리고**, 스킬 프롬프트에서 "한 번에 가장 연관성 있는 쿼리 하나만" 강제
3. **fast mode 기본화**: dev_helper 를 reasoning medium → chat (non-reasoning) 으로 내리고 학생이 필요 시 "깊이 생각" 토글
4. **스트리밍(#072) 구현**: 긴 생성도 중간중간 내려주어 CF 타임아웃 회피

## 제안 순서
우선 (1)+(2) 를 1 PR — 응답 크기만 축소하고 hop 제한. 이걸로도 부족하면 (4) 스트리밍 착수.
