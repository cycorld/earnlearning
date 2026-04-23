---
id: 110
title: Qwen3.6-27B Dense 비교 환경 구축 (운영은 기존 35B-A3B 유지)
priority: low
type: chore
branch: chore/qwen-27b-compare-env
created: 2026-04-23
---

## 배경
news.hada.io/topic?id=28797 에서 Qwen3.6-27B Dense 공개. SWE-bench Verified 77.2 등 코딩 벤치 향상.
RTX 4090 24GB / 학생 동시성 4 slot 환경에서 27B Dense 로 교체하면 동시성 1~2 slot 으로 떨어질 위험 → **운영 교체는 안 함**.

대신 비교 환경만 구축해 추후 결정 근거 마련.

## 작업
### cycorld 서버
- `/home/cycorld/models/Qwen3.6-27B-Instruct-Q4_K_M.gguf` 다운로드 (Q4_K_M, ~17GB)
- 비교용 systemd unit `llama-qwen27b.service` (수동 시작, 포트 8083, 평소엔 stop)
- swap 스크립트 (35B ↔ 27B, VRAM 동시 불가):
  - `/home/cycorld/bin/swap-to-27b.sh` — 35B 정지 → 27B 시작
  - `/home/cycorld/bin/swap-to-35b.sh` — 27B 정지 → 35B 재시작
- 비교 벤치 스크립트 `/home/cycorld/bin/compare-llm.sh` — 동일 prompt 셋을 양 모델에 던져 응답 시간 + 출력 비교

### LMS repo (이 PR)
- `docs/LLM_ARCHITECTURE.md` 보조 모델 섹션 추가
- changelog 110

## 결정
- **운영은 기존 Qwen3.6-35B-A3B (MoE) 유지** — 학생 동시성 우선
- Qwen3.6-27B 는 코딩 task 비교·벤치용으로만 보관
