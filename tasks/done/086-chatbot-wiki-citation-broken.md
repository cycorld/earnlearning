---
id: 086
title: 챗봇 출처 링크 클릭 시 PWA 메인으로 이동하는 문제
priority: high
type: fix
branch: fix/chatbot-wiki-citation
created: 2026-04-19
---

## 증상
챗봇이 `search_wiki` 결과를 인용한 답변에서 출처 링크를 누르면 노션이 아니라
EarnLearning 메인(/feed) 으로 이동.

## 원인
- `search_wiki` 도구가 결과에 slug 만 포함 (`## 제목 (notion-manuals/wallet)`)
- LLM 이 그걸 보고 `[제목](notion-manuals/wallet)` 같은 상대 markdown 링크 생성
- MarkdownContent 는 그대로 `<a target="_blank" href="notion-manuals/wallet">` 렌더
- 클릭 시 `https://earnlearning.com/notion-manuals/wallet` 로 이동 → 라우트 없음
  → SPA fallback (index.html) → 메인 화면

## 수정
1. **`search_wiki` 출력에 노션 URL 포함**
   - `notion_page_id` (UUID 형식) → `https://www.notion.so/{32자hex}` 변환
   - 결과 헤더에 클릭 가능한 절대 URL 노출
2. **시스템 프롬프트 보강**
   - "도구가 반환한 URL 만 출처로 인용. 임의로 만들지 마."
3. **(선택) MarkdownContent 안전장치**
   - href 가 `http(s)://` 또는 `/uploads/` 또는 `mailto:` 가 아니면 클릭 비활성

## 검증
- "주주총회 가결 기준 출처와 함께 알려줘" → 출처 링크가 `notion.so/...` 로 시작
- 클릭 시 새 탭에서 노션 페이지 열림
