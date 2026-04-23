# LLM 서버 아키텍처 (cycorld 외부 인프라)

EarnLearning 챗봇 + 학생 LLM API 키가 호출하는 외부 LLM 서버는 **cycorld** 머신에서 운영됨.
이 문서는 LMS 측 (이 repo) 관점의 요약. 서버 내부 상세는 `cycorld:/home/cycorld/llm-proxy/ARCHITECTURE.md`.

## 요청 경로
```
EarnLearning (LMS prod)
        │
        ▼  Authorization: Bearer <학생 API 키>
https://llm.cycorld.com   (Cloudflare proxy OFF)
        │
        ▼
[nginx] → FastAPI llm-proxy (127.0.0.1:8100)
                 │
                 └→ llama-server (127.0.0.1:8099, Qwen3.6-35B-A3B + mmproj, vision)
```

## 변경 위치 (서버 SSH)
| 하고 싶은 것 | 파일 | 재시작 |
|---|---|---|
| 허용 모델 / alias | `/home/cycorld/llm-proxy/main.py` (`MODEL_ALIASES`) | `systemctl --user restart llm-proxy.service` |
| 요청 검증 / 응답 가공 | 동일 (`proxy_chat_completions`) | 동일 |
| 학생 usage 로깅·과금 | 동일 (`_log_request`, `llm-proxy.db`) | 동일 |
| nginx / 도메인 | `/etc/nginx/...` | `sudo systemctl reload nginx` |

## ⚠️ 경로에 **없는** 것 (실수 주의 — #108)
- `/home/cycorld/bin/llama-proxy.ts` (`:8080`) — `llama-proxy.service` 로 돌긴 하지만 **요청 경로 밖**.
  여기만 고치면 효과 0. 파일 상단에 경고 배너 박아둠.
- `llama-qwen.service` — exit 1 반복 중 (모델 파일명 불일치). 실제 llama-server 는 별도 프로세스 (`ps aux | grep llama-server`).

## 검증 프로토콜
```bash
# 1. service 살아있나
ssh cycorld 'systemctl --user status llm-proxy.service --no-pager | head -4'

# 2. 학생 키로 end-to-end (PROD)
KEY="sk-stu-..."  # /api/llm/me/rotate 로 발급
curl -s https://llm.cycorld.com/v1/chat/completions \
  -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
  -d '{"model":"qwen-chat","messages":[{"role":"user","content":"ping"}],"max_tokens":20}'

# 3. 에러 케이스 (미지원 model → 400)
curl -s https://llm.cycorld.com/v1/chat/completions \
  -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
  -d '{"model":"claude-opus-4-7","messages":[{"role":"user","content":"x"}]}'
```

## 응답 옵션
- 기본: `reasoning_content` 필드는 클라이언트 응답에서 strip (#108)
- 명시적 노출 원할 시: 요청 body 에 `"include_reasoning": true`

## 보조 모델 (비교용 — 운영 X) — #110
RTX 4090 24GB 한 장에서 운영 모델은 **Qwen3.6-35B-A3B (MoE, 활성 3B)**. 학생 동시성 4 slot 유지가 최우선.

비교·벤치용으로 **Qwen3.6-27B Dense** (Q4_K_XL, ~17GB) 도 보관. Dense 라 같은 VRAM 에선 동시성이 1~2 slot 으로 떨어짐 → 운영 교체 안 함.

- 모델 파일: `cycorld:~/models/Qwen3.6-27B-UD-Q4_K_XL.gguf`
- 비교용 systemd: `llama-server-27b.service` (port 8098, 평소 stop)
- swap 스크립트: `~/bin/swap-to-{27b,35b}.sh` (VRAM 공유라 동시 불가, **운영 다운타임 발생**)
- 벤치 스크립트: `~/bin/compare-llm.sh`
- LMS proxy `UPSTREAM=:8099` 고정 → 27B 는 외부 노출 X (의도된 격리)

상세: `cycorld:/home/cycorld/llm-proxy/ARCHITECTURE.md` 의 "보조 모델" 섹션.

## 변경 이력 (LMS 관점)
- **#108 (2026-04-21)**: model 검증 강화 + reasoning_content strip + Student-#325 바운티 #1~#3 valid
- **#110 (2026-04-23)**: Qwen3.6-27B Dense 비교 환경 구축 (운영은 35B-A3B 유지)
