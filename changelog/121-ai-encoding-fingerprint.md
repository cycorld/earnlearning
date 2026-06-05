# #121 · AI 생성물 특유 인코딩 검출 (#120 보강)

**날짜:** 2026-06-05
**브랜치:** `feat/student-milestone-dashboard` (#119 + #120 + #121 묶음)
**티켓:** [tasks/in-progress/121-ai-encoding-fingerprint.md](../tasks/in-progress/121-ai-encoding-fingerprint.md)

---

## 무엇을 했나

#120 의 heuristic 점수에 **사람이 키보드로 거의 입력하지 않는 유니코드 문자** 검출을 추가했습니다.
ChatGPT/Claude/Gemini 출력에는 자주 들어가지만, 학생이 직접 타이핑한 글에는 거의 없습니다.

## 왜 필요했나

사용자 지적: "인간은 사용하지 않는 인코딩이 AI 생성물에 포함되는 경우도 있어"

LLM 출력에 자주 섞이는 유니코드 흔적들:
- **Zero-width 문자** (U+200B/C/D, U+FEFF, U+2060): 폭 0, 키보드로 입력 불가
- **Mathematical alphanumeric** (𝐀, 𝒂 등 U+1D400~U+1D7FF): 특수 폰트 변환 필요
- **Non-breaking space** (U+00A0, U+202F): typesetting tool / 일부 LLM 출력
- **Smart quotes** (" " ' ' U+2018-201D): 워드프로세서/LLM 출처
- **Em / en dash** (— – U+2014/2013): 한국어 글에서 학생이 잘 안 씀

heuristic 패턴(문장 길이 분산, 어휘 다양성 등) 보다 **훨씬 구체적인 forensic 증거** 라
점수에서 강한 가중치 부여 — zero-width 1개만 발견돼도 +30 점.

## 어떻게 만들었나

[ai_score.go](backend/internal/domain/milestone/ai_score.go) 에 `DetectSuspiciousChars` 함수 추가 +
`ScoreHeuristic` 에 5개 추가 시그널:

| 시그널 | 트리거 조건 | 가중치 | 학생용 힌트 |
|--------|------------|--------|------------|
| `ai_zero_width` | 1개라도 발견 | +30 | "ZWSP/ZWNJ 같은 보이지 않는 문자는 LLM 출력에서만 거의 나옵니다" |
| `ai_math_alpha` | 1개라도 발견 | +25 | "특수 폰트 변환을 거친 글자입니다. 일반 키보드로는 입력 불가능" |
| `ai_nbsp` | 1개라도 발견 | +15 | "일반 keyboard 의 space 가 아닙니다" |
| `ai_smart_quotes` | 5개 이상 (사람도 일부 씀) | +10 | "둥근 따옴표는 AI/워드프로세서 출처" |
| `ai_em_dash_density` | 1000자당 3개 이상 | +10 | "한국어 글에서 em dash 는 드뭅니다" |

### 버그 발견 + 수정: TrimSpace 가 NBSP 를 strip

처음엔 `clean := strings.TrimSpace(text)` 한 결과로 detection 했는데,
Go 의 `strings.TrimSpace` 는 NBSP(U+00A0) 도 whitespace 로 보고 strip 합니다.
**AI 가 trailing NBSP 를 넣은 경우 검출 못 함**. 

`DetectSuspiciousChars(text)` 로 원본 사용하도록 수정 + 회귀 테스트 락인.

### Frontend

추가 작업 없음 — `EssayScorePreview` 가 이미 `result.signals` 를 iterate 하므로
새 시그널들이 자동으로 학생에게 노출됩니다.

## TDD

- `TestDetectSuspiciousChars_*` 5개 (각 카테고리 검출 정확성)
- `TestScoreHeuristic_*TriggersSignal` 4개 (시그널 발화 + 가중치)
- `TestScoreHeuristic_AllAIEncodingFlagsPushScoreVeryHigh` (사람 글 + AI 흔적 → 점수 +30 이상 증가)
- `TestScoreHeuristic_NBSPTriggersSignal` — TrimSpace 회귀 방지

전체 milestone domain 단위 테스트 45개 통과 (29 URL + 16 AI score).

## 사용한 프롬프트

> 그리고 인간은 사용하지 않는 인코딩이 ai 생성물에 포함되는 경우도 있어. 그것도 체크해줘.

## 배운 점

### Go 소스에 BOM 박을 수 없음

`U+FEFF` (BOM) 를 string literal 에 직접 넣으면 Go 컴파일러가 "illegal byte order mark" 거부.
`﻿` escape 로 명시해야 함. ZWSP (`​`) 도 마찬가지로 escape 권장.

### "단정한 글이라 사람" 의 false positive

heuristic 만으로는 "정성껏 다듬어 쓴 진짜 사람 글" 도 AI 의심을 받을 수 있음.
zero-width / math alpha 같은 **물리적 증거** 는 false positive 거의 0 — 학생이
직접 타이핑했으면 절대 들어갈 수 없습니다. 이게 forensic 가치가 큰 이유.

## 변경된 파일

- `backend/internal/domain/milestone/ai_score.go`
  (`SuspiciousCharCounts`, `DetectSuspiciousChars`, 5개 시그널 추가, TrimSpace 버그 수정)
- `backend/internal/domain/milestone/ai_score_test.go` (+10 테스트)
