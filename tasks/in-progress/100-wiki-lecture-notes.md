---
id: 100
title: 강의 노트 + 강의계획서 wiki 추가 + lecture_helper 스킬
priority: high
type: feat
branch: feat/wiki-lecture-notes
created: 2026-04-20
---

## 작업
1. `docs/llm-wiki/lecture-notes/` 새 서브디렉토리
2. 상위 폴더의 8개 md 복사 + frontmatter 추가:
   - syllabus-actual.md → syllabus.md
   - week01-orientation.md ~ week06-vibe-coding-backend-deploy.md
   - week06-side-quest.md
3. 새 챗봇 스킬 `lecture_helper` 추가 — 강의 내용/주차/과제/평가 질문 특화 (wiki_scope: lecture-notes/*)
4. `general_ta` system prompt 도 강의 자료 우선 참조 안내 추가
5. 빌드 + 테스트 + Stage + Prod + 재인덱싱
6. 챗봇으로 "1주차 뭐했어?", "PR&FAQ 가 뭐야?" 등 검증
