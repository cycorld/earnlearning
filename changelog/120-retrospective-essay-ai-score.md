# #120 · 회고 에세이 입력 + AI 작성 확률 평가기

**날짜:** 2026-06-05
**브랜치:** `feat/student-milestone-dashboard` (#119 와 같이 묶임)
**티켓:** [tasks/in-progress/120-retrospective-essay-ai-score.md](../tasks/in-progress/120-retrospective-essay-ai-score.md)

---

## 무엇을 했나

#119 의 회고(`retrospective`) 평가지표를 **자유 텍스트 → 전용 에세이 UI** 로 바꾸고,
제출되는 글이 **AI(ChatGPT, Claude 등) 로 작성됐을 확률**을 0~100점으로 평가합니다.

- 학생용: 큰 textarea + 글자수 카운트(800자 권장) + "AI 작성 확률 셀프체크" 버튼
- 학생용: 점수에 따른 색상 chip + 개선 가이드 (예: "1인칭과 본인 경험을 더 써주세요")
- 관리자용: 매트릭스 회고 셀에 점수 chip 자동 노출 + 상세 다이얼로그에 LLM 평가 근거

## 왜 필요했나

회고 발표는 "한 학기 동안 어떤 경험을 했고 무엇을 배웠나" 를 본인 말투로 쓰는 글인데,
그냥 ChatGPT 에 "한 학기 회고 써줘" 하면 5초 안에 한 페이지 분량의 균질한 글이 나옵니다.
**평가의 진정성** 때문에 AI 글을 자동으로 의심해주는 보조 도구가 필요했습니다.

자동 반려는 안 합니다 — false positive 위험. 교수님이 점수를 참고해서 직접 판단합니다.

## 어떻게 만들었나

### 1. Heuristic 점수 (Go pure, 한국어 특화)

5개 시그널을 합산해서 0~100 점수:

| 시그널 | 검출 방법 | 가중치 |
|--------|----------|--------|
| 문장 길이 변동계수 | stdev / mean < 0.3 → AI 의심 | +20 |
| 어휘 다양성 (TTR) | < 0.35 → 단조로움 | +15 |
| AI 특유 구문 빈도 | "~을 통해", "결론적으로", "다음과 같다" 등 | +25 |
| 1인칭/구체적 경험 | "내가", "느꼈다", "주차", "교수님" 부재 | +30 |
| 감정 표현(이모지·ㅋㅋ·!!) 부재 | 정규식 | +10 |

각 시그널엔 학생용 **개선 힌트** ("문장 길이를 다양하게 섞어주세요") 동봉.

[backend/internal/domain/milestone/ai_score.go](backend/internal/domain/milestone/ai_score.go)

### 2. LLM 보조 평가 (cycorld llm.cycorld.com)

`ChatLLMClient` (#071 챗봇이 쓰는 같은 어댑터) 를 milestone usecase 에도 주입.
시스템 프롬프트로 한국어 평가관 역할 + JSON 출력 강제:

```
{"score": <0~100>, "reasoning": "<50자 이내 한국어>"}
```

`qwen-chat` 모델로 호출, 30초 타임아웃, 4000자 잘라서 전송.
LLM 실패는 silent — heuristic 만으로 평가 진행.

### 3. 통합 점수 = LLM 60% + Heuristic 40%

LLM 이 좀 더 정확해서 가중치 높임. 둘 다 학생에게 노출(투명성).

### 4. 자동 평가 + 셀프체크 두 갈래

- **셀프체크** (저장 안 함): `POST /milestones/essay/score` — 학생이 다듬기 위해
- **자동 평가** (저장): retrospective 제출 시 200자 이상이면 자동으로 평가 후 `ai_score` 등에 저장

### 5. DB 마이그레이션

`student_milestones` 에 4개 컬럼 추가:
- `ai_score INTEGER`  (NULL = 미평가)
- `ai_reasoning TEXT`  (LLM 한 줄 평)
- `ai_signals TEXT`  (heuristic 시그널 JSON)
- `ai_evaluated_at DATETIME`

기존 DB 호환을 위해 `ALTER TABLE` + `CREATE TABLE` 양쪽 다 적용 (errors ignored 패턴).

## 사용한 프롬프트

> 한 학기 회고는 에세이로 제출할거야. ai 작성 확률 평가기도 만들어줘.

설계 결정 2가지를 사용자에게 물어봄:
- **점수 노출**: "학생도 본다 (제출 전 셀프체크)" 선택
- **평가 엔진**: "Heuristic + LLM 보조" 선택

## 배운 점

### Heuristic 만으로는 부족

처음엔 Heuristic 만으로 가려고 했는데, 단위 테스트에서 사람 글 vs AI 글 점수 차이가
**30점밖에 안 벌어졌습니다**. 학생들이 한국어 essay 를 쓸 때 "~을 통해" 같은 표현을 진짜로
쓰기도 하니까 false positive 우려. LLM 보조를 합쳐서 더 robust 한 점수 가능.

### LLM 실패는 silent fallback

LLM 호출이 타임아웃·rate limit 등으로 실패하면 학생 제출이 막히면 안 됨.
heuristic 점수만으로라도 진행해서 학생 경험 보호.

### TDD 흐름

1. **Go pure heuristic test**: 사람 풍/AI 풍 sample 텍스트 작성 → 점수 분리 (5개 단위 테스트)
2. **Fake LLM 통합 테스트**: `fakeChatLLM` 이 deterministic JSON 반환 → 통합 점수 계산 검증 (3개)
3. **vitest**: `aiScoreMeta` 점수 → tone 색상 매핑 (1개)

## 변경된 파일

### Backend
- `backend/internal/domain/milestone/ai_score.go` (신규 — heuristic scorer)
- `backend/internal/domain/milestone/ai_score_test.go` (신규 — 6개 단위 테스트)
- `backend/internal/domain/milestone/{entity,repository}.go` (ai_* 필드 + UpdateAIScore 추가)
- `backend/internal/infrastructure/persistence/{milestone_repo,sqlite}.go` (스키마 + scan 확장)
- `backend/internal/application/milestone_usecase.go` (LLM 주입 + EvaluateEssay + ScoreAndStoreEssay)
- `backend/internal/interfaces/http/handler/milestone_handler.go` (ScoreEssay endpoint)
- `backend/internal/interfaces/http/router/router.go` (`/milestones/essay/score` 등록)
- `backend/cmd/server/main.go` (`milestoneUC.SetLLM(chatLLM, ...)`)
- `backend/tests/integration/setup_test.go` (`injectMilestoneFakeLLM` helper)
- `backend/tests/integration/milestone_test.go` (+3 — 셀프체크 / 너무 짧음 / 제출 시 자동 평가)

### Frontend
- `frontend/src/lib/milestone.ts` (Milestone 인터페이스 + EssayScoreResult + aiScoreMeta)
- `frontend/src/lib/milestone.test.ts` (+1 — aiScoreMeta tone bucket)
- `frontend/src/routes/milestones/StudentMilestonesPage.tsx` (회고 전용 essay UI + 셀프체크 미리보기)
- `frontend/src/routes/admin/AdminMilestonesPage.tsx` (회고 셀에 AI 점수 mini-chip + 다이얼로그)
