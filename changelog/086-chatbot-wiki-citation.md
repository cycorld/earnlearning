# 086. 챗봇 출처 링크가 PWA 메인으로 가던 문제 수정

**날짜**: 2026-04-19
**태그**: 챗봇, 위키, PWA, 버그수정

## 증상
챗봇이 `search_wiki` 결과를 인용한 답변에서 "출처(노션 가이드)" 를 누르면
노션이 아니라 EarnLearning 메인(/feed) 으로 이동.

## 원인
1. `search_wiki` 도구는 결과에 **slug 만** 노출 (`## 제목 (notion-manuals/wallet)`)
2. LLM 이 그걸 보고 ` [제목](notion-manuals/wallet) ` 같은 **상대경로 markdown 링크** 생성
3. MarkdownContent 가 `<a target="_blank" href="notion-manuals/wallet">` 로 렌더
4. 클릭 → `https://earnlearning.com/notion-manuals/wallet` → 라우트 없음
   → SPA fallback (index.html) → 메인 화면 (PWA)

## 수정 (3중 안전망)

### 1) `search_wiki` 출력에 노션 절대 URL 포함
```
## 주주총회 완전 가이드
출처: https://www.notion.so/3466b8a660...
(slug: notion-manuals/shareholder-proposal)
... 본문 snippet ...
```
+ 도구 결과 끝에 "이 문서를 인용할 때는 위 '출처:' URL 만 사용" 안내 추가.
`notion_page_id` (UUID dashed) → 32-char hex → `https://www.notion.so/<id>`.

### 2) `general_ta` 시스템 프롬프트 강화
> **링크 규칙**: markdown 링크는 반드시 `https://` 또는 `http://` 로 시작하는
> 절대 URL 만 사용. slug 는 링크로 만들지 마 (텍스트로만 언급).

### 3) Frontend MarkdownContent 안전장치
href 가 `http(s)://`, `mailto:`, `tel:`, `/uploads/` 가 아니면 `<a>` 대신
`<span>` 으로 렌더 (점선 밑줄 + tooltip). LLM 이 어떻게든 잘못된 링크를 만들어도
PWA 메인으로 이동하는 사고는 차단.

## 트레이드오프
- Notion URL 은 노션 워크스페이스 접근 권한이 있어야 열림. 비공개 페이지면 학생이
  로그인 화면을 볼 수 있음. 후속 옵션:
  - LMS 자체 위키 뷰어 (`/wiki/:slug`) 만들기
  - Notion 페이지를 public 으로 전환

## 배운 점
1. **도구 출력은 LLM 의 입력**. 데이터 누락이 환각/잘못된 인용으로 직결됨.
2. **렌더 단계 안전장치 필수**. 프롬프트 + 도구만으로 100% 막을 수 없음 — 마지막
   라인으로 마크다운 렌더러에서 검증.
3. **PWA SPA fallback 의 함정** — `target="_blank"` + relative href = 사용자 혼란.
