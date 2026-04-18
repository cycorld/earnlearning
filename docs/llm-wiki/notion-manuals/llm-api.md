---
title: LLM API 사용 완전 가이드 — 키 발급 + 코드 작성법
notion_page_id: 34668bb8-a660-8137-a27b-c15c4a1b7092
synced_at: 2026-04-19T00:00:00Z
---

# LLM API 사용 완전 가이드

## 30초 요약
1. LMS 하단 **더보기 → LLM 키** → 자동으로 평문 키 발급 (한 번만 보임!)
2. 복사해서 `Authorization: Bearer <키>` 로 `https://llm.cycorld.com/v1/chat/completions` 호출
3. 매일 03:33 KST 에 사용한 토큰만큼 **개인 지갑**에서 자동 차감

## API 키 받기

### 자동 발급 플로우
1. 로그인 후 화면 하단 **더보기** → **LLM 키** 메뉴
2. `/llm` 페이지가 열리면 **자동으로 키가 발급**되고 평문 키가 **딱 한 번** 표시됨
3. 즉시 복사해서 비밀번호 관리자에 저장 (창을 닫거나 새로고침하면 다시 볼 수 없음)
4. 이후 방문 시에는 prefix (앞 5글자) 만 보이고 평문은 숨겨짐

### 키 형식
```
sk-stu-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### 재발급
`/llm` 페이지의 **재발급** 버튼 → 기존 키 즉시 폐기 + 새 키 발급. 기존 키로 만들어두었던 앱/스크립트는 모두 재설정해야 함.

**절대 금지**: Git 저장소 커밋, Slack/Discord 에 복붙, 프론트엔드 코드 하드코딩.

## 요금 체계

Anthropic **Claude Opus 4.7** 공식 가격을 환율 **1 USD = 1,400원** 고정으로 환산.

| 구분 | USD (100만 토큰) | KRW (1,000 토큰) |
|---|---|---|
| 입력 토큰 | $15 | 21원 |
| 출력 토큰 | $75 | 105원 |
| 캐시 적중 입력 | $1.5 | 2.1원 (90% 할인) |

### 차감 타이밍
- **매일 새벽 03:33 KST** 에 전날 사용량 정산 → 개인 지갑에서 자동 차감
- 알림 1건 발송
- 잔액 부족 시 차감 가능한 만큼만 빠지고 나머지는 **부채로 누적** (다음 과금 때 우선 차감)

## 모델 & 엔드포인트

- 모델: `Qwen3.6-35B-A3B` (Mixture of Experts, 전체 35B / 활성 3B)
- 컨텍스트: 최대 **200,192 토큰** (요청+응답 합계)
- 하드웨어: RTX 4090 24GB × 4 슬롯 병렬
- 속도: prefill 3,500~6,000 tok/s, decode 100~145 tok/s

**Base URL**: `https://llm.cycorld.com/v1`

| 메서드 | 경로 | 설명 |
|---|---|---|
| POST | `/v1/chat/completions` | 대화 (OpenAI 호환, 스트리밍·이미지 지원) |
| GET | `/v1/models` | 모델 목록 |
| GET | `/health` | 상태 (인증 불필요) |

### 모델 변형

| 모델명 | Thinking | 기본 max_tokens | 용도 |
|---|---|---|---|
| `qwen` | ON | 16,384 | 기본, 추론 과제 |
| `qwen-chat` | OFF | 2,048 | 일반 대화 |
| `qwen-reasoning` | ON+보존 | 32,768 | 멀티턴 복잡한 추론 |

`reasoning_effort` 로 조절: `low`(1k), `medium`(4k), `high`(32k).

## 코드 작성 가이드

### cURL 기본
```bash
curl https://llm.cycorld.com/v1/chat/completions \
  -H "Authorization: Bearer $LLM_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen",
    "messages": [
      {"role": "system", "content": "너는 친절한 조교야."},
      {"role": "user", "content": "합성곱 신경망의 풀링 계층을 한 문단으로 설명해줘."}
    ],
    "max_tokens": 512
  }'
```

### Python (OpenAI SDK)
```python
from openai import OpenAI
import os

client = OpenAI(
    base_url="https://llm.cycorld.com/v1",
    api_key=os.environ["LLM_API_KEY"],
)
resp = client.chat.completions.create(
    model="qwen",
    messages=[{"role": "user", "content": "안녕!"}],
)
print(resp.choices[0].message.content)
print(f"입력 토큰: {resp.usage.prompt_tokens}, 출력: {resp.usage.completion_tokens}")
```

### Node.js
```javascript
import OpenAI from "openai";
const client = new OpenAI({
  baseURL: "https://llm.cycorld.com/v1",
  apiKey: process.env.LLM_API_KEY,
});
const resp = await client.chat.completions.create({
  model: "qwen",
  messages: [{ role: "user", content: "async/await 초보자용 설명" }],
});
```

### 스트리밍
```python
stream = client.chat.completions.create(
    model="qwen", stream=True,
    stream_options={"include_usage": True},
    messages=[{"role": "user", "content": "소수 정리 설명"}],
)
for chunk in stream:
    if chunk.choices and chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)
```

### 이미지 입력 (Vision)
OpenAI Vision 표준 포맷. JPEG/PNG/WEBP 지원.
```python
resp = client.chat.completions.create(
    model="qwen",
    messages=[{"role": "user", "content": [
        {"type": "text", "text": "이 그림 설명"},
        {"type": "image_url", "image_url": {"url": "https://example.com/photo.jpg"}},
    ]}],
)
```

## 캐싱으로 비용 줄이기

**Prefix caching** 지원: 이전 요청 앞부분과 같은 토큰 시퀀스면 KV 캐시 재사용 → 속도 29배, 비용 90% 할인.

**실측 예**: 140K 토큰 프롬프트 캐시 적중 시 18.6초 → 0.64초.

### 캐시 적중률 높이는 4가지 원칙
1. **긴 고정 부분은 맨 앞에** — 시스템 프롬프트, 긴 매뉴얼, 코드베이스 컨텍스트
2. **타임스탬프 프롬프트 금지** — 매번 달라지면 전체 miss
3. **랜덤 ID 금지** — 같은 이유
4. **멀티턴 메시지 순서 고정** — messages 배열 앞부분 바꾸지 말 것

## 금지사항

- **개인정보**(주민번호, 전화번호, 계좌), 비밀번호, 의료 기록, 기업 기밀을 프롬프트에 넣지 마세요. **모든 대화는 서버에 저장됩니다.**
- 키를 Git/Slack/Discord/공개 포럼에 노출 금지
- 수업과 무관한 콘텐츠 대량 생성 금지 (과금 폭탄의 원인)

## FAQ

**Q. 키를 잃어버렸어요.**
`/llm` 페이지 **재발급** 버튼. 기존 키는 즉시 무효화.

**Q. 지갑 잔액 0 인데 계속 API 호출하면?**
호출은 계속 성공. 다음 03:33 에 정산될 때 가능한 만큼만 빠지고 나머지는 **부채 누적**. 지갑 충전하면 다음 과금 때 부채부터 우선 갚음.

**Q. 현재 사용량을 실시간으로 볼 수 있나요?**
LMS `/llm` 페이지 **사용량 · 청구 내역** 표. 당일 사용량은 다음날 03:33 이후 반영. 즉시 확인하려면 API 응답의 `usage.prompt_tokens` / `usage.completion_tokens` 직접 합산.

**Q. thinking 모드를 끄고 바로 답만 받고 싶어요.**
`model="qwen-chat"` 을 쓰거나, `qwen` + `reasoning_effort="low"`.

**Q. 서비스 상태 확인?**
LMS `/llm` 페이지 맨 위 "서비스 상태" 카드. 직접은 `curl https://llm.cycorld.com/health`.
