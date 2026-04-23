# 110. Qwen3.6-27B Dense 비교 환경 구축 (운영은 35B-A3B 유지)

**날짜**: 2026-04-23
**태그**: 운영, LLM, 모델, infra, 의사결정

## 배경
[news.hada.io/topic?id=28797](https://news.hada.io/topic?id=28797) 에서 Qwen3.6-27B Dense 공개. SWE-bench Verified 77.2 등 코딩 벤치 향상 → 학생 챗봇 / 코드 도움이 좋아질 가능성.

운영 모델은 현재 **Qwen3.6-35B-A3B** (MoE, 활성 3B). RTX 4090 24GB 한 장에서 **학생 동시성 4 slot** 으로 굴리는 중.

## 의사결정 — 운영 교체 ❌
27B Dense 는 토큰마다 모든 27B 파라미터를 활성화. 같은 GPU 에선:
- VRAM: Q4_K_XL 17GB + KV cache → 운영 컨텍스트(64K)·동시성 confine 시 **slot 1~2 개**로 떨어짐
- 추론 속도: Dense 27B 는 활성 파라미터 9배 ↑ → 토큰/s 5~9× 느려짐

→ **운영은 35B-A3B 유지** (학생 동시성 우선). 27B 는 비교·벤치 용도로만 보관.

## 구축한 것 (cycorld 서버)

### 모델 파일
- `~/models/Qwen3.6-27B-UD-Q4_K_XL.gguf` (~17GB)
- `~/models/mmproj-BF16.gguf` (vision projector — 35B 와 공유)

### 비교용 systemd unit
`~/.config/systemd/user/llama-server-27b.service`
- 포트 **8098** (운영 8099 와 분리)
- `-c 65536 -ctk q8_0 -ctv q8_0 --flash-attn on -ngl 99 --jinja`
- `WantedBy` 빠짐 → **수동 시작 only**

### swap 스크립트 (VRAM 24GB 공유 → 동시 불가)
- `~/bin/swap-to-27b.sh` — 35B (운영) stop → 27B start. **운영 다운타임 발생**
- `~/bin/swap-to-35b.sh` — 27B stop → 35B 운영 복귀

### 비교 벤치 스크립트
`~/bin/compare-llm.sh` — 동일 prompt 4개를 양 모델에 던져 응답 시간·토큰 측정:
1. 인사 (warmup)
2. 아이디어 생성 (창의)
3. Python 버그 찾기 (코딩)
4. Go deadlock 설명 (서술 + 기술)

결과: `~/models/compare-results-YYYYMMDD-HHMMSS/{model,prompt,latency,tokens,response}`. 자동 35B 복귀.

## 문서화
- **`/home/cycorld/llm-proxy/ARCHITECTURE.md`** — "보조 모델 (비교용)" 섹션 추가. swap 위험 / 격리 의도 명시.
- **`docs/LLM_ARCHITECTURE.md`** (이 PR) — LMS 관점 보조 모델 요약.

## 학습 포인트
- **MoE vs Dense 트레이드오프**: 같은 파라미터 총량이라도 MoE 는 활성 파라미터만 계산 → 동시성·속도 유리. 학생 4명이 동시에 챗봇 켜는 환경에선 Dense > MoE 단순 비교 무의미.
- **벤치 점수 ≠ 운영 적합성**: SWE-bench 같은 단일 task 점수는 throughput 무시. "내 학생들이 동시에 4명 쓸 때 어느 쪽이 빠른가" 가 진짜 지표.
- **격리된 비교 환경의 가치**: 운영 안 망가뜨리고 다음 세대 모델 후보 측정 가능. 다음 모델 (Qwen3.7?, GPT-OSS?) 나올 때도 같은 패턴 재사용.

## 미포함 (의도)
- **외부 노출 X**: FastAPI proxy `UPSTREAM=:8099` 고정. 27B (`:8098`) 는 nginx 경로에 없음. 외부 비교가 필요하면 별도 alias + 분기 로직 필요 (현재 미구현).
- **자동 복귀 타이머 X**: swap-to-27b 후 깜빡 잊으면 운영 다운 지속. 사용자 책임. 수업 중 절대 금지.
- **벤치 자동화 X**: `compare-llm.sh` 는 수동 실행. CI 화 가치 낮음 (모델 교체는 분기·반기 단위).
